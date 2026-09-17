package telemetry_test

import (
	"context"
	"testing"

	"github.com/GagarinRu/avatars/internal/telemetry"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestInitWithoutOTLPEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	shutdown, err := telemetry.Init(context.Background(), "avatars-test", "info")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	defer shutdown()

	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	span.End()

	if !trace.SpanFromContext(ctx).SpanContext().IsValid() {
		t.Fatal("expected valid span context")
	}
}

func TestLoggerFromContextWithoutSpan(t *testing.T) {
	t.Parallel()
	logger := telemetry.LoggerFromContext(context.Background())
	if logger == nil {
		t.Fatal("expected logger")
	}
}

func TestLoggerFromContextWithSpan(t *testing.T) {
	t.Parallel()
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	logger := telemetry.LoggerFromContext(ctx)
	if logger == nil {
		t.Fatal("expected logger")
	}
}
