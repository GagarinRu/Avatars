// Package logger configures structured application logging.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Initialize configures JSON logging to stderr. Used in tests and local runs without OTLP.
func Initialize(level string) error {
	lvl, err := parseLevel(level)
	if err != nil {
		return err
	}
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
	return nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errInvalidLevel
	}
}

var errInvalidLevel = &levelError{}

type levelError struct{}

func (e *levelError) Error() string { return "invalid log level" }

// InfoContext logs an info message with request context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

// ErrorContext logs an error message with request context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}
