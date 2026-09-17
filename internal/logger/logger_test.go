package logger_test

import (
	"context"
	"testing"

	"github.com/GagarinRu/avatars/internal/logger"
)

func TestInitialize(t *testing.T) {
	t.Parallel()
	if err := logger.Initialize("info"); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	logger.InfoContext(context.Background(), "test message", "key", "value")
}

func TestInitializeInvalidLevel(t *testing.T) {
	t.Parallel()
	if err := logger.Initialize("not-a-level"); err == nil {
		t.Fatal("expected error")
	}
}
