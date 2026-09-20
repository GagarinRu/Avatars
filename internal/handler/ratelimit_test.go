package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUploadRateLimitMiddleware(t *testing.T) {
	handler := UploadRateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/avatars/user-1", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/avatars/user-1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/avatars/user-1", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET should not be rate limited, got %d", getRec.Code)
	}
}
