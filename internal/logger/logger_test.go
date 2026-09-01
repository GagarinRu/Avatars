package logger_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/avatars/internal/logger"
)

func TestInitializeAndRequestLogger(t *testing.T) {
	t.Parallel()
	if err := logger.Initialize("info"); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	called := false
	h := logger.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusCreated {
		t.Fatalf("handler not called correctly")
	}
}

func TestInitializeInvalidLevel(t *testing.T) {
	t.Parallel()
	if err := logger.Initialize("not-a-level"); err == nil {
		t.Fatal("expected error")
	}
}
