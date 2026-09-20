package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GagarinRu/avatars/internal/handler"
	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
)

type mockStore struct {
	avatar *models.Avatar
}

func (m *mockStore) UpsertAvatar(_ context.Context, avatar *models.Avatar) error {
	cp := *avatar
	cp.CreatedAt = time.Now().UTC()
	cp.UpdatedAt = cp.CreatedAt
	m.avatar = &cp
	return nil
}
func (m *mockStore) GetAvatar(_ context.Context, userID string) (*models.Avatar, error) {
	if m.avatar == nil || m.avatar.UserID != userID {
		return nil, nil
	}
	cp := *m.avatar
	return &cp, nil
}
func (m *mockStore) DeleteAvatar(_ context.Context, userID string) error {
	if m.avatar == nil || m.avatar.UserID != userID {
		return storage.ErrNotFound
	}
	m.avatar = nil
	return nil
}
func (m *mockStore) UpdateAvatarStatus(_ context.Context, _ string, _ models.ProcessingStatus, _ string) error {
	return nil
}
func (m *mockStore) MarkAvatarReady(_ context.Context, userID, originalKey, thumbKey, contentType string, size int64, width, height int) error {
	if m.avatar == nil {
		return storage.ErrNotFound
	}
	m.avatar.Status = models.StatusReady
	m.avatar.OriginalKey = originalKey
	m.avatar.ThumbnailKey = thumbKey
	m.avatar.ContentType = contentType
	m.avatar.SizeBytes = size
	m.avatar.Width = width
	m.avatar.Height = height
	return nil
}
func (m *mockStore) TotalStorageBytes(_ context.Context) (int64, error) {
	if m.avatar != nil && m.avatar.Status == models.StatusReady {
		return m.avatar.SizeBytes, nil
	}
	return 0, nil
}
func (m *mockStore) IsMessageProcessed(context.Context, string) (bool, error) { return false, nil }
func (m *mockStore) MarkMessageProcessed(context.Context, string) error       { return nil }
func (m *mockStore) Ping(context.Context) error                               { return nil }
func (m *mockStore) Close() error                                             { return nil }

type mockObjects struct {
	uploaded map[string][]byte
}

func newMockObjects() *mockObjects {
	return &mockObjects{uploaded: make(map[string][]byte)}
}
func (o *mockObjects) Upload(_ context.Context, key string, reader io.Reader, _ string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	o.uploaded[key] = data
	return nil
}
func (o *mockObjects) Delete(_ context.Context, keys ...string) error {
	for _, key := range keys {
		delete(o.uploaded, key)
	}
	return nil
}
func (o *mockObjects) PresignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	if key == "" {
		return "", nil
	}
	return "http://example.com/" + key, nil
}

type mockPublisher struct {
	last queue.AvatarProcessMessage
}

func (p *mockPublisher) PublishAvatarProcess(_ context.Context, msg queue.AvatarProcessMessage) error {
	p.last = msg
	return nil
}

func TestUploadAndGetAvatar(t *testing.T) {
	t.Parallel()
	store := &mockStore{}
	objects := newMockObjects()
	publisher := &mockPublisher{}
	h := handler.NewHandler(store, objects, publisher, 1024*1024)
	mux := handler.NewMux(h)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}
	if _, err := part.Write(png); err != nil {
		t.Fatalf("write png header: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/avatars/user-1", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("upload status = %d body=%s", rr.Code, rr.Body.String())
	}
	if publisher.last.UserID != "user-1" {
		t.Fatalf("unexpected publish: %+v", publisher.last)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/avatars/user-1", nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("get status = %d", getRR.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(getRR.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != string(models.StatusPending) {
		t.Fatalf("unexpected status: %v", resp["status"])
	}
}

func TestDeleteAvatarSuccess(t *testing.T) {
	t.Parallel()
	store := &mockStore{
		avatar: &models.Avatar{
			UserID:       "user-1",
			Status:       models.StatusReady,
			OriginalKey:  "avatars/user-1/original.png",
			ThumbnailKey: "avatars/user-1/thumb.png",
		},
	}
	objects := newMockObjects()
	h := handler.NewHandler(store, objects, &mockPublisher{}, 1024)
	mux := handler.NewMux(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/avatars/user-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetAvatarNotFound(t *testing.T) {
	t.Parallel()
	h := handler.NewHandler(&mockStore{}, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)
	req := httptest.NewRequest(http.MethodGet, "/api/avatars/missing", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetAvatarReady(t *testing.T) {
	t.Parallel()
	store := &mockStore{
		avatar: &models.Avatar{
			UserID:       "user-1",
			Status:       models.StatusReady,
			OriginalKey:  "avatars/user-1/original.png",
			ThumbnailKey: "avatars/user-1/thumb.png",
			ContentType:  "image/png",
			SizeBytes:    100,
			Width:        64,
			Height:       64,
		},
	}
	h := handler.NewHandler(store, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)
	req := httptest.NewRequest(http.MethodGet, "/api/avatars/user-1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["original_url"] == "" || resp["thumbnail_url"] == "" {
		t.Fatalf("missing urls: %+v", resp)
	}
}

func TestDeleteNotFound(t *testing.T) {
	t.Parallel()
	h := handler.NewHandler(&mockStore{}, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)
	req := httptest.NewRequest(http.MethodDelete, "/api/avatars/missing", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	h := handler.NewHandler(&mockStore{}, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestUploadInvalidExtension(t *testing.T) {
	t.Parallel()
	h := handler.NewHandler(&mockStore{}, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "avatar.txt")
	_, _ = part.Write([]byte("hello"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/avatars/user-1", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestWebIndex(t *testing.T) {
	t.Parallel()
	h := handler.NewHandler(&mockStore{}, newMockObjects(), &mockPublisher{}, 1024)
	mux := handler.NewMux(h)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Avatars") {
		t.Fatal("expected html page")
	}
}
