package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GagarinRu/avatars/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestHTTPMiddlewareRecordsMetrics(t *testing.T) {
	t.Parallel()

	handler := metrics.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/avatars/user-1", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", rr.Code)
	}

	metric, err := collectCounterValue("http_requests_total")
	if err != nil {
		t.Fatalf("collect metric: %v", err)
	}
	if metric == 0 {
		t.Fatal("expected http_requests_total to increase")
	}
}

func TestNormalizePath(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"/api/avatars/alice": "/api/avatars/{user_id}",
		"/health":            "/health",
	}
	for input, want := range cases {
		if got := metrics.NormalizePath(input); got != want {
			t.Fatalf("normalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func collectCounterValue(name string) (float64, error) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return 0, err
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			return m.GetCounter().GetValue(), nil
		}
	}
	return 0, nil
}

func TestUploadMetricsRegistration(t *testing.T) {
	t.Parallel()
	metrics.UploadsTotal.WithLabelValues("success").Inc()
	metrics.UploadDuration.WithLabelValues("success").Observe(0.1)
	metrics.StorageUsage.WithLabelValues("user-1").Set(1024)

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() == "avatars_uploads_total" {
			return
		}
	}
	t.Fatal("avatars_uploads_total not registered")
}
