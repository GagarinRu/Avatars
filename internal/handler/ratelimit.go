package handler

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

func isAvatarUploadPath(path string) bool {
	return strings.HasPrefix(path, "/api/avatars/")
}

type uploadRateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	refill   float64
	last     time.Time
}

func newUploadRateLimiter(requestsPerMinute int) *uploadRateLimiter {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 60
	}
	capacity := float64(requestsPerMinute)
	return &uploadRateLimiter{
		tokens:   capacity,
		capacity: capacity,
		refill:   capacity / 60,
		last:     time.Now(),
	}
}

func (l *uploadRateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.last = now
	l.tokens += elapsed * l.refill
	if l.tokens > l.capacity {
		l.tokens = l.capacity
	}
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

func UploadRateLimitMiddleware(next http.Handler) http.Handler {
	limiter := newUploadRateLimiter(30)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && isAvatarUploadPath(r.URL.Path) {
			if !limiter.allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
