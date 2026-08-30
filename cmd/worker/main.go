package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/GagarinRu/avatars/internal/config"
	"github.com/GagarinRu/avatars/internal/logger"
	"github.com/GagarinRu/avatars/internal/objectstore"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
	"github.com/GagarinRu/avatars/internal/worker"
	"go.uber.org/zap"
)

func shutdownSignals() []os.Signal {
	sigs := []os.Signal{os.Interrupt, syscall.SIGTERM}
	if runtime.GOOS != "windows" {
		sigs = append(sigs, syscall.SIGQUIT)
	}
	return sigs
}

func main() {
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

	if err := logger.Initialize(opts.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	defer func() { _ = logger.Log.Sync() }()

	if opts.DatabaseDSN == "" {
		logger.Log.Fatal("Database DSN is required", zap.String("hint", "use -d or DATABASE_DSN"))
	}

	store, err := storage.NewPostgresStorage(opts.DatabaseDSN)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() { _ = store.Close() }()

	objects, err := objectstore.NewClient(objectstore.Config{
		Endpoint:       opts.S3Endpoint,
		PublicEndpoint: opts.S3PublicEndpoint,
		Bucket:         opts.S3Bucket,
		Region:    opts.S3Region,
		AccessKey: opts.S3AccessKey,
		SecretKey: opts.S3SecretKey,
		UseSSL:    opts.S3UseSSL,
	})
	if err != nil {
		logger.Log.Fatal("Failed to create S3 client", zap.Error(err))
	}

	conn, ch, err := queue.Connect(opts.RabbitMQURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer func() {
		_ = ch.Close()
		_ = conn.Close()
	}()

	svc := worker.NewService(store, objects)
	consumer := queue.NewConsumer(ch, queue.NewIdempotency(store))

	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()

	logger.Log.Info("Worker started")
	if err := consumer.Consume(ctx, svc.Handle); err != nil && err != context.Canceled {
		logger.Log.Fatal("Worker stopped with error", zap.Error(err))
	}
	logger.Log.Info("Worker stopped gracefully")
}
