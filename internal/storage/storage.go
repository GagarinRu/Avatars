// Package storage implements avatar metadata persistence.
package storage

import (
	"context"

	"github.com/GagarinRu/avatars/internal/models"
)

// Storage defines persistence operations for avatars and idempotency keys.
type Storage interface {
	UpsertAvatar(ctx context.Context, avatar *models.Avatar) error
	GetAvatar(ctx context.Context, userID string) (*models.Avatar, error)
	DeleteAvatar(ctx context.Context, userID string) error
	UpdateAvatarStatus(ctx context.Context, userID string, status models.ProcessingStatus, errMsg string) error
	MarkAvatarReady(ctx context.Context, userID string, originalKey, thumbKey, contentType string, size int64, width, height int) error

	IsMessageProcessed(ctx context.Context, messageID string) (bool, error)
	MarkMessageProcessed(ctx context.Context, messageID string) error

	TotalStorageBytes(ctx context.Context) (int64, error)

	Ping(ctx context.Context) error
	Close() error
}
