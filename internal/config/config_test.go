package config_test

import (
	"testing"

	"github.com/GagarinRu/avatars/internal/config"
)

func TestApplyAppJSON(t *testing.T) {
	t.Parallel()
	opts := config.DefaultAppOptions()
	opts = config.ApplyAppJSON(opts, config.AppJSON{
		Address:     ":9090",
		DatabaseDSN: "postgres://x",
		LogLevel:    "debug",
	})
	if opts.Address != ":9090" || opts.DatabaseDSN != "postgres://x" || opts.LogLevel != "debug" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestApplyAppEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("S3_BUCKET", "test-bucket")
	opts := config.ApplyAppEnv(config.DefaultAppOptions())
	if opts.LogLevel != "debug" || opts.S3Bucket != "test-bucket" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}

func TestDefaultAppOptions(t *testing.T) {
	t.Parallel()
	opts := config.DefaultAppOptions()
	if opts.Address == "" || opts.S3Bucket == "" || opts.MaxUploadBytes <= 0 {
		t.Fatalf("unexpected defaults: %+v", opts)
	}
}
