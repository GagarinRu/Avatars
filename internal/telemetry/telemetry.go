// Package telemetry configures OpenTelemetry tracing and structured logging.
package telemetry

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Init configures tracing and logging for the service.
// When OTEL_EXPORTER_OTLP_ENDPOINT is unset, tracing is noop and logs go to stderr as JSON.
func Init(ctx context.Context, serviceName, logLevel string) (func(), error) {
	lvl := parseLevel(logLevel)
	shutdowns := make([]func(context.Context) error, 0, 2)

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))

	var tp *sdktrace.TracerProvider
	if endpoint != "" {
		traceExporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
	} else {
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	shutdowns = append(shutdowns, tp.Shutdown)

	stderrHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	if endpoint != "" {
		logExporter, err := otlploggrpc.New(ctx)
		if err != nil {
			return nil, err
		}
		loggerProvider := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		)
		otelHandler := otelslog.NewHandler(serviceName, otelslog.WithLoggerProvider(loggerProvider))
		handler := &multiHandler{handlers: []slog.Handler{stderrHandler, otelHandler}}
		slog.SetDefault(slog.New(handler).With("service", serviceName))
		shutdowns = append(shutdowns, loggerProvider.Shutdown)
	} else {
		slog.SetDefault(slog.New(stderrHandler).With("service", serviceName))
	}

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, shutdownFn := range shutdowns {
			_ = shutdownFn(shutdownCtx)
		}
	}, nil
}

// LoggerFromContext returns a logger enriched with trace identifiers when present.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	attrs := make([]any, 0, 4)
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		attrs = append(attrs,
			"trace_id", sc.TraceID().String(),
			"span_id", sc.SpanID().String(),
		)
	}
	if len(attrs) == 0 {
		return slog.Default()
	}
	return slog.Default().With(attrs...)
}

type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
