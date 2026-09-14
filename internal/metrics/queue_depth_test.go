package metrics_test

import (
	"testing"

	"github.com/GagarinRu/avatars/internal/metrics"
)

func TestRabbitManagementURL(t *testing.T) {
	t.Parallel()
	got, err := metrics.RabbitManagementURLForTest("amqp://guest:guest@avatars.rabbitmq:5672/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "http://guest:guest@avatars.rabbitmq:15672"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
