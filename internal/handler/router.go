package handler

import (
	"net/http"

	"github.com/GagarinRu/avatars/internal/web"
)

func NewMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", web.Index)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /ping", h.Ping)
	mux.HandleFunc("POST /api/avatars/{user_id}", h.UploadAvatar)
	mux.HandleFunc("GET /api/avatars/{user_id}", h.GetAvatar)
	mux.HandleFunc("DELETE /api/avatars/{user_id}", h.DeleteAvatar)
	return mux
}
