package handler

import (
	"net"
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

type uploadRateLimitRegistry struct {
	mu                sync.Mutex
	buckets           map[string]*uploadRateLimiter
	requestsPerMinute int
}

func newUploadRateLimitRegistry(requestsPerMinute int) *uploadRateLimitRegistry {
	return &uploadRateLimitRegistry{
		buckets:           make(map[string]*uploadRateLimiter),
		requestsPerMinute: requestsPerMinute,
	}
}

func (r *uploadRateLimitRegistry) allow(clientKey string) bool {
	r.mu.Lock()
	limiter, ok := r.buckets[clientKey]
	if !ok {
		limiter = newUploadRateLimiter(r.requestsPerMinute)
		r.buckets[clientKey] = limiter
	}
	r.mu.Unlock()
	return limiter.allow()
}

func clientKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func UploadRateLimitMiddleware(requestsPerMinute int, next http.Handler) http.Handler {
	registry := newUploadRateLimitRegistry(requestsPerMinute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && isAvatarUploadPath(r.URL.Path) {
			if !registry.allow(clientKey(r)) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
