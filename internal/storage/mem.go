package storage

import (
	"context"
	"sync"
	"time"

	"github.com/GagarinRu/avatars/internal/models"
)

type MemStorage struct {
	mu        sync.RWMutex
	avatars   map[string]*models.Avatar
	processed map[string]time.Time
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		avatars:   make(map[string]*models.Avatar),
		processed: make(map[string]time.Time),
	}
}

func (m *MemStorage) Close() error { return nil }

func (m *MemStorage) Ping(_ context.Context) error { return nil }

func (m *MemStorage) UpsertAvatar(_ context.Context, avatar *models.Avatar) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	cp := *avatar
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = now
	}
	cp.UpdatedAt = now
	m.avatars[avatar.UserID] = &cp
	return nil
}

func (m *MemStorage) GetAvatar(_ context.Context, userID string) (*models.Avatar, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.avatars[userID]
	if !ok {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}

func (m *MemStorage) DeleteAvatar(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.avatars[userID]; !ok {
		return ErrNotFound
	}
	delete(m.avatars, userID)
	return nil
}

func (m *MemStorage) UpdateAvatarStatus(_ context.Context, userID string, status models.ProcessingStatus, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.avatars[userID]
	if !ok {
		return ErrNotFound
	}
	a.Status = status
	a.ErrorMessage = errMsg
	a.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemStorage) MarkAvatarReady(_ context.Context, userID, originalKey, thumbKey, contentType string, size int64, width, height int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.avatars[userID]
	if !ok {
		return ErrNotFound
	}
	a.Status = models.StatusReady
	a.OriginalKey = originalKey
	a.ThumbnailKey = thumbKey
	a.ContentType = contentType
	a.SizeBytes = size
	a.Width = width
	a.Height = height
	a.StagingKey = ""
	a.ErrorMessage = ""
	a.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *MemStorage) IsMessageProcessed(_ context.Context, messageID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.processed[messageID]
	return ok, nil
}

func (m *MemStorage) MarkMessageProcessed(_ context.Context, messageID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed[messageID] = time.Now().UTC()
	return nil
}
