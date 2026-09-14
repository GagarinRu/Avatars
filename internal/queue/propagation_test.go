package queue_test

import (
	"context"
	"testing"

	"github.com/GagarinRu/avatars/internal/queue"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestTraceContextPropagationRoundTrip(t *testing.T) {
	t.Parallel()

	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := otel.Tracer("test")
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent")
	defer parentSpan.End()

	headers := queue.InjectTraceContext(parentCtx, amqp.Table{})
	childCtx := queue.ExtractTraceContext(context.Background(), headers)

	childSpanCtx := trace.SpanFromContext(childCtx).SpanContext()
	parentSpanCtx := parentSpan.SpanContext()
	if childSpanCtx.TraceID() != parentSpanCtx.TraceID() {
		t.Fatalf("trace ids differ: %s vs %s", childSpanCtx.TraceID(), parentSpanCtx.TraceID())
	}
}
