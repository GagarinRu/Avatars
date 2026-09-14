// Package handler provides HTTP handlers for the avatars API.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/GagarinRu/avatars/internal/metrics"
	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
	"github.com/GagarinRu/avatars/internal/telemetry"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ObjectStore interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Delete(ctx context.Context, keys ...string) error
	PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type Publisher interface {
	PublishAvatarProcess(ctx context.Context, msg queue.AvatarProcessMessage) error
}

type Handler struct {
	store      storage.Storage
	objects    ObjectStore
	publisher  Publisher
	maxUpload  int64
	urlExpiry  time.Duration
}

func NewHandler(store storage.Storage, objects ObjectStore, publisher Publisher, maxUpload int64) *Handler {
	if maxUpload <= 0 {
		maxUpload = 5 * 1024 * 1024
	}
	return &Handler{
		store:     store,
		objects:   objects,
		publisher: publisher,
		maxUpload: maxUpload,
		urlExpiry: 15 * time.Minute,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type uploadResponse struct {
	UserID    string                 `json:"user_id"`
	Status    models.ProcessingStatus `json:"status"`
	MessageID string                 `json:"message_id"`
}

type avatarResponse struct {
	UserID        string                 `json:"user_id"`
	Status        models.ProcessingStatus `json:"status"`
	ContentType   string                 `json:"content_type,omitempty"`
	SizeBytes     int64                  `json:"size_bytes,omitempty"`
	Width         int                    `json:"width,omitempty"`
	Height        int                    `json:"height,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	OriginalURL   string                 `json:"original_url,omitempty"`
	ThumbnailURL  string                 `json:"thumbnail_url,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	status := "success"
	defer func() {
		metrics.UploadsTotal.WithLabelValues(status).Inc()
		metrics.UploadDuration.WithLabelValues(status).Observe(time.Since(start).Seconds())
	}()

	ctx, span := otel.Tracer("avatars-api").Start(r.Context(), "upload_avatar")
	defer span.End()
	log := telemetry.LoggerFromContext(ctx)

	userID := strings.TrimSpace(r.PathValue("user_id"))
	if userID == "" {
		status = "error"
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_id required"})
		return
	}
	const multipartOverhead = 1024
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload+multipartOverhead)
	if err := r.ParseMultipartForm(h.maxUpload); err != nil {
		status = "error"
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "request too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid multipart form"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		status = "error"
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "file field required"})
		return
	}
	defer file.Close()

	limited := io.LimitReader(file, h.maxUpload+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		status = "error"
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "failed to read file"})
		return
	}
	if len(data) == 0 {
		status = "error"
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "empty file"})
		return
	}
	if int64(len(data)) > h.maxUpload {
		status = "error"
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "file too large"})
		return
	}
	if !allowedExtension(header.Filename) {
		status = "error"
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "unsupported file extension"})
		return
	}

	messageID := uuid.NewString()
	stagingKey := fmt.Sprintf("staging/%s/%s", userID, messageID)
	contentType := detectContentType(data, header.Filename)
	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("file_name", header.Filename),
		attribute.Int64("file_size", int64(len(data))),
		attribute.String("mime_type", contentType),
	)
	log.InfoContext(ctx, "uploading avatar",
		"user_id", userID,
		"file_size", len(data),
		"mime_type", contentType,
	)

	if err := h.objects.Upload(ctx, stagingKey, bytes.NewReader(data), contentType); err != nil {
		status = "error"
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to store upload"})
		return
	}

	avatar := &models.Avatar{
		UserID:     userID,
		Status:     models.StatusPending,
		StagingKey: stagingKey,
		MessageID:  messageID,
	}
	if err := h.store.UpsertAvatar(ctx, avatar); err != nil {
		status = "error"
		_ = h.objects.Delete(ctx, stagingKey)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to save metadata"})
		return
	}

	msg := queue.AvatarProcessMessage{
		MessageID:  messageID,
		UserID:     userID,
		StagingKey: stagingKey,
	}
	if err := h.publisher.PublishAvatarProcess(ctx, msg); err != nil {
		status = "error"
		_ = h.objects.Delete(ctx, stagingKey)
		_ = h.store.UpdateAvatarStatus(ctx, userID, models.StatusFailed, "failed to enqueue processing")
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to enqueue processing"})
		return
	}

	log.InfoContext(ctx, "avatar upload accepted", "user_id", userID, "message_id", messageID)
	writeJSON(w, http.StatusAccepted, uploadResponse{
		UserID:    userID,
		Status:    models.StatusPending,
		MessageID: messageID,
	})
}

func (h *Handler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("user_id"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_id required"})
		return
	}
	avatar, err := h.store.GetAvatar(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get avatar"})
		return
	}
	if avatar == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "avatar not found"})
		return
	}
	resp, err := h.toResponse(r.Context(), avatar)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to build response"})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.PathValue("user_id"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "user_id required"})
		return
	}
	avatar, err := h.store.GetAvatar(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to get avatar"})
		return
	}
	if avatar == nil {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "avatar not found"})
		return
	}
	if err := h.objects.Delete(r.Context(), avatar.StagingKey, avatar.OriginalKey, avatar.ThumbnailKey); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete files"})
		return
	}
	if err := h.store.DeleteAvatar(r.Context(), userID); err != nil {
		if err == storage.ErrNotFound {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "avatar not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "failed to delete metadata"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "database unavailable"})
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) toResponse(ctx context.Context, avatar *models.Avatar) (avatarResponse, error) {
	resp := avatarResponse{
		UserID:       avatar.UserID,
		Status:       avatar.Status,
		ContentType:  avatar.ContentType,
		SizeBytes:    avatar.SizeBytes,
		Width:        avatar.Width,
		Height:       avatar.Height,
		ErrorMessage: avatar.ErrorMessage,
		CreatedAt:    avatar.CreatedAt,
		UpdatedAt:    avatar.UpdatedAt,
	}
	if avatar.Status == models.StatusReady {
		originalURL, err := h.objects.PresignedURL(ctx, avatar.OriginalKey, h.urlExpiry)
		if err != nil {
			return resp, err
		}
		thumbURL, err := h.objects.PresignedURL(ctx, avatar.ThumbnailKey, h.urlExpiry)
		if err != nil {
			return resp, err
		}
		resp.OriginalURL = originalURL
		resp.ThumbnailURL = thumbURL
	}
	return resp, nil
}

func allowedExtension(filename string) bool {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

func detectContentType(data []byte, filename string) string {
	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 {
		return "image/png"
	}
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
