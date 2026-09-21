package storage

import (
	"context"

	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/resilience"
)

// BreakerStorage wraps Storage with a circuit breaker on database calls.
type BreakerStorage struct {
	inner   Storage
	breaker *resilience.Breaker
}

func NewBreakerStorage(inner Storage, breaker *resilience.Breaker) *BreakerStorage {
	return &BreakerStorage{inner: inner, breaker: breaker}
}

func (s *BreakerStorage) UpsertAvatar(ctx context.Context, avatar *models.Avatar) error {
	return s.breaker.Call(func() error {
		return s.inner.UpsertAvatar(ctx, avatar)
	})
}

func (s *BreakerStorage) GetAvatar(ctx context.Context, userID string) (*models.Avatar, error) {
	var avatar *models.Avatar
	var innerErr error
	err := s.breaker.Call(func() error {
		avatar, innerErr = s.inner.GetAvatar(ctx, userID)
		return innerErr
	})
	return avatar, err
}

func (s *BreakerStorage) DeleteAvatar(ctx context.Context, userID string) error {
	return s.breaker.Call(func() error {
		return s.inner.DeleteAvatar(ctx, userID)
	})
}

func (s *BreakerStorage) UpdateAvatarStatus(ctx context.Context, userID string, status models.ProcessingStatus, errMsg string) error {
	return s.breaker.Call(func() error {
		return s.inner.UpdateAvatarStatus(ctx, userID, status, errMsg)
	})
}

func (s *BreakerStorage) MarkAvatarReady(ctx context.Context, userID string, originalKey, thumbKey, contentType string, size int64, width, height int) error {
	return s.breaker.Call(func() error {
		return s.inner.MarkAvatarReady(ctx, userID, originalKey, thumbKey, contentType, size, width, height)
	})
}

func (s *BreakerStorage) IsMessageProcessed(ctx context.Context, messageID string) (bool, error) {
	var processed bool
	var innerErr error
	err := s.breaker.Call(func() error {
		processed, innerErr = s.inner.IsMessageProcessed(ctx, messageID)
		return innerErr
	})
	return processed, err
}

func (s *BreakerStorage) MarkMessageProcessed(ctx context.Context, messageID string) error {
	return s.breaker.Call(func() error {
		return s.inner.MarkMessageProcessed(ctx, messageID)
	})
}

func (s *BreakerStorage) TotalStorageBytes(ctx context.Context) (int64, error) {
	var total int64
	var innerErr error
	err := s.breaker.Call(func() error {
		total, innerErr = s.inner.TotalStorageBytes(ctx)
		return innerErr
	})
	return total, err
}

func (s *BreakerStorage) Ping(ctx context.Context) error {
	return s.breaker.Call(func() error {
		return s.inner.Ping(ctx)
	})
}

func (s *BreakerStorage) Close() error {
	return s.inner.Close()
}
