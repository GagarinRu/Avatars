package config

const (
	defaultAddress       = ":8080"
	defaultLogLevel      = "info"
	defaultRabbitMQURL   = "amqp://guest:guest@localhost:5672/"
	defaultS3Endpoint    = "http://localhost:9000"
	defaultS3Bucket      = "avatars"
	defaultS3AccessKey   = "minioadmin"
	defaultS3SecretKey   = "minioadmin"
	defaultS3Region      = "us-east-1"
	defaultMaxUploadSize = 5 * 1024 * 1024 // 5 MiB
)

// AppJSON is the JSON config format shared by API and worker.
type AppJSON struct {
	Address        string `json:"address"`
	DatabaseDSN    string `json:"database_dsn"`
	LogLevel       string `json:"log_level"`
	RabbitMQURL    string `json:"rabbitmq_url"`
	S3Endpoint     string `json:"s3_endpoint"`
	S3Bucket       string `json:"s3_bucket"`
	S3AccessKey    string `json:"s3_access_key"`
	S3SecretKey    string `json:"s3_secret_key"`
	S3Region          string `json:"s3_region"`
	S3PublicEndpoint  string `json:"s3_public_endpoint"`
	S3UseSSL          *bool  `json:"s3_use_ssl"`
	MaxUploadBytes int64  `json:"max_upload_bytes"`
}

// AppOptions holds resolved application configuration.
type AppOptions struct {
	Address        string
	DatabaseDSN    string
	LogLevel       string
	RabbitMQURL    string
	S3Endpoint     string
	S3Bucket       string
	S3AccessKey    string
	S3SecretKey    string
	S3Region         string
	S3PublicEndpoint string
	S3UseSSL         bool
	MaxUploadBytes int64
}

func ReadAppJSON(path string) (AppJSON, error) {
	var file AppJSON
	if err := loadJSON(path, &file); err != nil {
		return file, err
	}
	return file, nil
}

func ApplyAppJSON(opts AppOptions, file AppJSON) AppOptions {
	opts = applyOptions(opts,
		nonEmptyStringOption(file.Address, func(o *AppOptions, v string) { o.Address = v }),
		nonEmptyStringOption(file.DatabaseDSN, func(o *AppOptions, v string) { o.DatabaseDSN = v }),
		nonEmptyStringOption(file.LogLevel, func(o *AppOptions, v string) { o.LogLevel = v }),
		nonEmptyStringOption(file.RabbitMQURL, func(o *AppOptions, v string) { o.RabbitMQURL = v }),
		nonEmptyStringOption(file.S3Endpoint, func(o *AppOptions, v string) { o.S3Endpoint = v }),
		nonEmptyStringOption(file.S3Bucket, func(o *AppOptions, v string) { o.S3Bucket = v }),
		nonEmptyStringOption(file.S3AccessKey, func(o *AppOptions, v string) { o.S3AccessKey = v }),
		nonEmptyStringOption(file.S3SecretKey, func(o *AppOptions, v string) { o.S3SecretKey = v }),
		nonEmptyStringOption(file.S3Region, func(o *AppOptions, v string) { o.S3Region = v }),
		nonEmptyStringOption(file.S3PublicEndpoint, func(o *AppOptions, v string) { o.S3PublicEndpoint = v }),
	)
	if file.S3UseSSL != nil {
		opts.S3UseSSL = *file.S3UseSSL
	}
	if file.MaxUploadBytes > 0 {
		opts.MaxUploadBytes = file.MaxUploadBytes
	}
	return opts
}

func ApplyAppEnv(opts AppOptions) AppOptions {
	opts.Address = envString("ADDRESS", opts.Address)
	opts.DatabaseDSN = envString("DATABASE_DSN", opts.DatabaseDSN)
	opts.LogLevel = envString("LOG_LEVEL", opts.LogLevel)
	opts.RabbitMQURL = envString("RABBITMQ_URL", opts.RabbitMQURL)
	opts.S3Endpoint = envString("S3_ENDPOINT", opts.S3Endpoint)
	opts.S3Bucket = envString("S3_BUCKET", opts.S3Bucket)
	opts.S3AccessKey = envString("S3_ACCESS_KEY", opts.S3AccessKey)
	opts.S3SecretKey = envString("S3_SECRET_KEY", opts.S3SecretKey)
	opts.S3Region = envString("S3_REGION", opts.S3Region)
	opts.S3PublicEndpoint = envString("S3_PUBLIC_ENDPOINT", opts.S3PublicEndpoint)
	opts.S3UseSSL = envBool("S3_USE_SSL", opts.S3UseSSL)
	opts.MaxUploadBytes = envInt64("MAX_UPLOAD_BYTES", opts.MaxUploadBytes)
	return opts
}

func DefaultAppOptions() AppOptions {
	opts := AppOptions{
		Address:        defaultAddress,
		LogLevel:       defaultLogLevel,
		RabbitMQURL:    defaultRabbitMQURL,
		S3Endpoint:     defaultS3Endpoint,
		S3Bucket:       defaultS3Bucket,
		S3AccessKey:    defaultS3AccessKey,
		S3SecretKey:    defaultS3SecretKey,
		S3Region:       defaultS3Region,
		S3UseSSL:       false,
		MaxUploadBytes: defaultMaxUploadSize,
	}
	return ApplyAppEnv(opts)
}
