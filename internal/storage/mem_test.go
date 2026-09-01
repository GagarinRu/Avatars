package storage_test

import (
	"context"
	"testing"

	"github.com/GagarinRu/avatars/internal/models"
	"github.com/GagarinRu/avatars/internal/storage"
)

func TestMemStorageAvatarLifecycle(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()

	avatar := &models.Avatar{
		UserID:     "user-1",
		Status:     models.StatusPending,
		StagingKey: "staging/user-1/msg",
		MessageID:  "msg-1",
	}
	if err := store.UpsertAvatar(ctx, avatar); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := store.GetAvatar(ctx, "user-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Status != models.StatusPending {
		t.Fatalf("unexpected avatar: %+v", got)
	}

	if err := store.MarkAvatarReady(ctx, "user-1", "orig", "thumb", "image/png", 100, 10, 10); err != nil {
		t.Fatalf("mark ready: %v", err)
	}
	got, err = store.GetAvatar(ctx, "user-1")
	if err != nil {
		t.Fatalf("get ready: %v", err)
	}
	if got.Status != models.StatusReady || got.OriginalKey != "orig" {
		t.Fatalf("unexpected ready avatar: %+v", got)
	}

	if err := store.DeleteAvatar(ctx, "user-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := store.DeleteAvatar(ctx, "user-1"); err != storage.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestMemStorageUpdateStatus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()
	if err := store.UpdateAvatarStatus(ctx, "missing", models.StatusFailed, "err"); err != storage.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestMemStorageIdempotency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	store := storage.NewMemStorage()

	processed, err := store.IsMessageProcessed(ctx, "m1")
	if err != nil {
		t.Fatalf("is processed: %v", err)
	}
	if processed {
		t.Fatal("expected not processed")
	}
	if err := store.MarkMessageProcessed(ctx, "m1"); err != nil {
		t.Fatalf("mark: %v", err)
	}
	processed, err = store.IsMessageProcessed(ctx, "m1")
	if err != nil || !processed {
		t.Fatalf("expected processed, got %v err=%v", processed, err)
	}
}
