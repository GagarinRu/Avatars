package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/GagarinRu/avatars/internal/config"
	"github.com/GagarinRu/avatars/internal/handler"
	"github.com/GagarinRu/avatars/internal/logger"
	"github.com/GagarinRu/avatars/internal/objectstore"
	"github.com/GagarinRu/avatars/internal/queue"
	"github.com/GagarinRu/avatars/internal/storage"
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
		addr      string
		logLevel  string
		database  string
		rabbitURL string
	)
	flag.StringVar(&addr, "a", opts.Address, "Server address")
	flag.StringVar(&logLevel, "l", opts.LogLevel, "Log level")
	flag.StringVar(&database, "d", opts.DatabaseDSN, "Database DSN")
	flag.StringVar(&rabbitURL, "rabbitmq-url", opts.RabbitMQURL, "RabbitMQ URL")
	flag.StringVar(new(string), "c", "", "Path to JSON config file")
	flag.StringVar(new(string), "config", "", "Path to JSON config file")
	flag.Parse()

	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { visited[f.Name] = true })
	if visited["a"] {
		opts.Address = addr
	}
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
	if err := objects.EnsureBucket(context.Background()); err != nil {
		logger.Log.Fatal("Failed to ensure S3 bucket", zap.Error(err))
	}

	conn, ch, err := queue.Connect(opts.RabbitMQURL)
	if err != nil {
		logger.Log.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer func() {
		_ = ch.Close()
		_ = conn.Close()
	}()

	publisher := queue.NewPublisher(ch)
	h := handler.NewHandler(store, objects, publisher, opts.MaxUploadBytes)
	mux := handler.NewMux(h)

	server := &http.Server{Addr: opts.Address, Handler: logger.RequestLogger(mux)}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Server failed", zap.Error(err))
		}
	}()
	logger.Log.Info("API server started", zap.String("address", opts.Address))

	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()
	<-ctx.Done()
	logger.Log.Info("Received shutdown signal")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.Fatal("Server shutdown failed", zap.Error(err))
	}
	logger.Log.Info("API server stopped gracefully")
}
