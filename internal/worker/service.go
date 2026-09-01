package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/processor"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
)

type ObjectStore interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, keys ...string) error
}

type Service struct {
	store   storage.Storage
	objects ObjectStore
}

func NewService(store storage.Storage, objects ObjectStore) *Service {
	return &Service{store: store, objects: objects}
}

func (s *Service) Handle(ctx context.Context, msg queue.AvatarProcessMessage, _ amqp.Delivery) error {
	if err := s.store.UpdateAvatarStatus(ctx, msg.UserID, models.StatusProcessing, ""); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	data, err := s.objects.Download(ctx, msg.StagingKey)
	if err != nil {
		_ = s.store.UpdateAvatarStatus(ctx, msg.UserID, models.StatusFailed, err.Error())
		return err
	}

	processed, err := processor.Process(bytes.NewReader(data))
	if err != nil {
		_ = s.store.UpdateAvatarStatus(ctx, msg.UserID, models.StatusFailed, err.Error())
		return err
	}

	originalKey := fmt.Sprintf("avatars/%s/original.png", msg.UserID)
	thumbKey := fmt.Sprintf("avatars/%s/thumb.png", msg.UserID)

	if err := s.objects.Upload(ctx, originalKey, bytes.NewReader(processed.OriginalPNG), "image/png"); err != nil {
		_ = s.store.UpdateAvatarStatus(ctx, msg.UserID, models.StatusFailed, err.Error())
		return err
	}
	if err := s.objects.Upload(ctx, thumbKey, bytes.NewReader(processed.ThumbnailPNG), "image/png"); err != nil {
		_ = s.store.UpdateAvatarStatus(ctx, msg.UserID, models.StatusFailed, err.Error())
		return err
	}

	if err := s.store.MarkAvatarReady(ctx, msg.UserID, originalKey, thumbKey, processed.ContentType, processed.SizeBytes, processed.Width, processed.Height); err != nil {
		return err
	}

	_ = s.objects.Delete(ctx, msg.StagingKey)
	return nil
}
