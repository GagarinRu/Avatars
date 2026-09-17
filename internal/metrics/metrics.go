package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	UploadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatars_uploads_total",
			Help: "Total number of avatar uploads",
		},
		[]string{"status"},
	)

	UploadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "avatars_upload_duration_seconds",
			Help:    "Avatar upload duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	StorageUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "avatars_storage_bytes",
			Help: "Total storage used by avatars",
		},
		[]string{"user_id"},
	)

	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "avatars_db_connections_open",
			Help: "Number of open database connections",
		},
	)
)

func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		path := normalizePath(r.URL.Path)
		status := strconv.Itoa(rw.statusCode)
		labels := []string{r.Method, path, status}
		HTTPRequestsTotal.WithLabelValues(labels[0], labels[1], labels[2]).Inc()
		HTTPRequestDuration.WithLabelValues(labels[0], labels[1], labels[2]).Observe(time.Since(start).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func NormalizePath(path string) string {
	if strings.HasPrefix(path, "/api/avatars/") {
		return "/api/avatars/{user_id}"
	}
	return path
}

func normalizePath(path string) string {
	return NormalizePath(path)
}
