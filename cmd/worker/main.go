package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/GagarinRu/avatars/internal/config"
	"github.com/GagarinRu/avatars/internal/metrics"
	"github.com/GagarinRu/avatars/internal/objectstore"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
	"github.com/GagarinRu/avatars/internal/telemetry"
	"github.com/GagarinRu/avatars/internal/worker"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func shutdownSignals() []os.Signal {
	sigs := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if runtime.GOOS != "windows" {
		sigs = append(sigs, syscall.SIGQUIT)
	}
	return sigs
}

func main() {
	os.Exit(run())
}

func run() int {
	printBuildInfo()

	opts := config.DefaultAppOptions()
	if configPath := config.ConfigPath(); configPath != "" {
		file, err := config.ReadAppJSON(configPath)
		if err != nil {
			panic("failed to load config: " + err.Error())
		}
		opts = config.ApplyAppJSON(opts, file)
	}

	var (
		logLevel  string
		database  string
		rabbitURL string
	)
	flag.StringVar(&logLevel, "l", opts.LogLevel, "Log level")
	flag.StringVar(&database, "d", opts.DatabaseDSN, "Database DSN")
	flag.StringVar(&rabbitURL, "rabbitmq-url", opts.RabbitMQURL, "RabbitMQ URL")
	flag.StringVar(new(string), "c", "", "Path to JSON config file")
	flag.StringVar(new(string), "config", "", "Path to JSON config file")
	flag.Parse()

	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if visited["l"] {
		opts.LogLevel = logLevel
	}
	if visited["d"] {
		opts.DatabaseDSN = database
	}
	if visited["rabbitmq-url"] {
		opts.RabbitMQURL = rabbitURL
	}
	opts = config.ApplyAppEnv(opts)

	ctx := context.Background()
	otelShutdown, err := telemetry.Init(ctx, serviceName(), opts.LogLevel)
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		return 1
	}
	defer otelShutdown()

	if opts.DatabaseDSN == "" {
		slog.Error("database DSN is required", "hint", "use -d or DATABASE_DSN")
		return 1
	}

	store, err := storage.NewPostgresStorage(opts.DatabaseDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return 1
	}
	defer func() { _ = store.Close() }()

	objects, err := objectstore.NewClient(objectstore.Config{
		Endpoint:       opts.S3Endpoint,
		PublicEndpoint: opts.S3PublicEndpoint,
		Bucket:         opts.S3Bucket,
		Region:         opts.S3Region,
		AccessKey:      opts.S3AccessKey,
		SecretKey:      opts.S3SecretKey,
		UseSSL:         opts.S3UseSSL,
	})
	if err != nil {
		slog.Error("failed to create S3 client", "error", err)
		return 1
	}

	conn, ch, err := queue.Connect(opts.RabbitMQURL)
	if err != nil {
		slog.Error("failed to connect to RabbitMQ", "error", err)
		return 1
	}
	defer func() {
		_ = ch.Close()
		_ = conn.Close()
	}()

	svc := worker.NewService(store, objects)
	consumer := queue.NewConsumer(ch, queue.NewIdempotency(store))

	sigCtx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()

	mgmt, err := metrics.RabbitMQManagementFromURL(
		opts.RabbitMQURL,
		opts.RabbitMQMgmtPort,
		opts.RabbitMQUser,
		opts.RabbitMQPassword,
	)
	if err != nil {
		slog.Error("failed to parse rabbitmq management config", "error", err)
		return 1
	}
	metrics.StartQueueDepthPoller(sigCtx, mgmt, queue.QueueProcessing, 15*time.Second)
	metrics.StartStorageUsagePoller(sigCtx, store, 30*time.Second)

	metricsServer := &http.Server{Addr: opts.MetricsAddress, Handler: promhttp.Handler()}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	slog.Info("worker started", "metrics_address", opts.MetricsAddress)
	if err := consumer.Consume(sigCtx, svc.Handle); err != nil && err != context.Canceled {
		slog.Error("worker stopped with error", "error", err)
		return 1
	}
	_ = metricsServer.Shutdown(context.Background())
	slog.Info("worker stopped gracefully")
	return 0
}

func serviceName() string {
	if name := os.Getenv("OTEL_SERVICE_NAME"); name != "" {
		return name
	}
	return "avatars-worker"
}
