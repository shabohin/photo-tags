package config

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	pkgstorage "github.com/shabohin/photo-tags/pkg/storage"
)

type Config struct {
	Log struct {
		Level  string
		Format string
	}

	RabbitMQ struct {
		URL               string
		ConsumerQueue     string
		PublisherQueue    string
		ReconnectDelay    time.Duration
		ReconnectAttempts int
		PrefetchCount     int
	}

	MinIO struct {
		Endpoint         string
		AccessKey        string
		SecretKey        string
		OriginalBucket   string
		ProcessedBucket  string
		DownloadTimeout  time.Duration
		UploadTimeout    time.Duration
		ConnectDelay     time.Duration
		ConnectAttempts  int
		UseSSL           bool
	}

	ExifTool struct {
		BinaryPath     string
		TempDir        string
		CommandTimeout time.Duration
	}

	Worker struct {
		RetryDelay  time.Duration
		Concurrency int
		MaxRetries  int
	}
}

func New() *Config {
	cfg := &Config{}

	// RabbitMQ Config
	cfg.RabbitMQ.URL = getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	cfg.RabbitMQ.ConsumerQueue = getEnv("RABBITMQ_CONSUMER_QUEUE", "metadata_generated")
	cfg.RabbitMQ.PublisherQueue = getEnv("RABBITMQ_PUBLISHER_QUEUE", "image_processed")
	cfg.RabbitMQ.PrefetchCount = getEnvAsInt("RABBITMQ_PREFETCH_COUNT", 1)
	cfg.RabbitMQ.ReconnectAttempts = getEnvAsInt("RABBITMQ_RECONNECT_ATTEMPTS", 5)
	cfg.RabbitMQ.ReconnectDelay = getEnvAsDuration("RABBITMQ_RECONNECT_DELAY", 5*time.Second)

	// MinIO Config
	cfg.MinIO.Endpoint = getEnv("MINIO_ENDPOINT", "minio:9000")
	cfg.MinIO.AccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinIO.SecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	cfg.MinIO.UseSSL = getEnvAsBool("MINIO_USE_SSL", false)
	cfg.MinIO.OriginalBucket = getEnv("MINIO_ORIGINAL_BUCKET", pkgstorage.BucketOriginal)
	cfg.MinIO.ProcessedBucket = getEnv("MINIO_PROCESSED_BUCKET", pkgstorage.BucketProcessed)
	cfg.MinIO.DownloadTimeout = getEnvAsDuration("MINIO_DOWNLOAD_TIMEOUT", 30*time.Second)
	cfg.MinIO.UploadTimeout = getEnvAsDuration("MINIO_UPLOAD_TIMEOUT", 30*time.Second)
	cfg.MinIO.ConnectAttempts = getEnvAsInt("MINIO_CONNECT_ATTEMPTS", 5)
	cfg.MinIO.ConnectDelay = getEnvAsDuration("MINIO_CONNECT_DELAY", 3*time.Second)

	// ExifTool Config
	cfg.ExifTool.BinaryPath = getEnv("EXIFTOOL_BINARY_PATH", "/usr/bin/exiftool")
	cfg.ExifTool.TempDir = getEnv("EXIFTOOL_TEMP_DIR", "/tmp/processor")
	cfg.ExifTool.CommandTimeout = getEnvAsDuration("EXIFTOOL_COMMAND_TIMEOUT", 10*time.Second)

	// Log Config
	cfg.Log.Level = getEnv("LOG_LEVEL", "info")
	cfg.Log.Format = getEnv("LOG_FORMAT", "json")

	// Worker Config
	cfg.Worker.Concurrency = getEnvAsInt("WORKER_CONCURRENCY", 3)
	cfg.Worker.MaxRetries = getEnvAsInt("WORKER_MAX_RETRIES", 3)
	cfg.Worker.RetryDelay = getEnvAsDuration("WORKER_RETRY_DELAY", 5*time.Second)

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func ConfigureLogger(cfg *Config) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set log format
	if cfg.Log.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}
