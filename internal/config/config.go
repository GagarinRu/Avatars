// Package config provides JSON configuration loading.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func ConfigPath() string {
	if path, ok := os.LookupEnv("CONFIG"); ok && path != "" {
		return strings.Trim(path, `"'`)
	}
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-c", arg == "-config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(arg, "-c="):
			return strings.TrimPrefix(arg, "-c=")
		case strings.HasPrefix(arg, "-config="):
			return strings.TrimPrefix(arg, "-config=")
		}
	}
	return ""
}

func loadJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	return nil
}

type Option[T any] func(*T)

func applyOptions[T any](base T, opts ...Option[T]) T {
	for _, opt := range opts {
		opt(&base)
	}
	return base
}

func nonEmptyStringOption[T any](value string, set func(*T, string)) Option[T] {
	return func(t *T) {
		if value != "" {
			set(t, value)
		}
	}
}

func envString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return strings.Trim(val, `"'`)
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := envString(key, "")
	if raw == "" {
		return fallback
	}
	var v int
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		return fallback
	}
	return v
}

func envInt64(key string, fallback int64) int64 {
	raw := envString(key, "")
	if raw == "" {
		return fallback
	}
	var v int64
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		return fallback
	}
	return v
}

func envBool(key string, fallback bool) bool {
	raw := strings.ToLower(envString(key, ""))
	switch raw {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return fallback
	}
}
