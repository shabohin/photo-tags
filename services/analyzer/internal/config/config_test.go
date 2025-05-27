package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// Save original environment variables
	originalEnv := map[string]string{
		"RABBITMQ_URL":             os.Getenv("RABBITMQ_URL"),
		"RABBITMQ_CONSUMER_QUEUE":  os.Getenv("RABBITMQ_CONSUMER_QUEUE"),
		"RABBITMQ_PUBLISHER_QUEUE": os.Getenv("RABBITMQ_PUBLISHER_QUEUE"),
		"RABBITMQ_PREFETCH_COUNT":  os.Getenv("RABBITMQ_PREFETCH_COUNT"),
		"MINIO_ENDPOINT":           os.Getenv("MINIO_ENDPOINT"),
		"MINIO_ACCESS_KEY":         os.Getenv("MINIO_ACCESS_KEY"),
		"MINIO_SECRET_KEY":         os.Getenv("MINIO_SECRET_KEY"),
		"MINIO_USE_SSL":            os.Getenv("MINIO_USE_SSL"),
		"MINIO_ORIGINAL_BUCKET":    os.Getenv("MINIO_ORIGINAL_BUCKET"),
		"OPENROUTER_API_KEY":       os.Getenv("OPENROUTER_API_KEY"),
		"OPENROUTER_MODEL":         os.Getenv("OPENROUTER_MODEL"),
		"OPENROUTER_MAX_TOKENS":    os.Getenv("OPENROUTER_MAX_TOKENS"),
		"OPENROUTER_TEMPERATURE":   os.Getenv("OPENROUTER_TEMPERATURE"),
		"LOG_LEVEL":                os.Getenv("LOG_LEVEL"),
		"LOG_FORMAT":               os.Getenv("LOG_FORMAT"),
		"WORKER_CONCURRENCY":       os.Getenv("WORKER_CONCURRENCY"),
	}

	// Restore original environment variables after test
	defer func() {
		for key, value := range originalEnv {
			if value != "" {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	// Clear environment variables before test
	for key := range originalEnv {
		os.Unsetenv(key)
	}

	// Test default values
	cfg := New()

	assert.Equal(t, "amqp://guest:guest@rabbitmq:5672/", cfg.RabbitMQ.URL)
	assert.Equal(t, "image_upload", cfg.RabbitMQ.ConsumerQueue)
	assert.Equal(t, "metadata_generated", cfg.RabbitMQ.PublisherQueue)
	assert.Equal(t, 1, cfg.RabbitMQ.PrefetchCount)
	assert.Equal(t, 5, cfg.RabbitMQ.ReconnectAttempts)
	assert.Equal(t, 5*time.Second, cfg.RabbitMQ.ReconnectDelay)

	assert.Equal(t, "minio:9000", cfg.MinIO.Endpoint)
	assert.Equal(t, "minioadmin", cfg.MinIO.AccessKey)
	assert.Equal(t, "minioadmin", cfg.MinIO.SecretKey)
	assert.Equal(t, false, cfg.MinIO.UseSSL)
	assert.Equal(t, "original", cfg.MinIO.OriginalBucket)
	assert.Equal(t, 30*time.Second, cfg.MinIO.DownloadTimeout)

	assert.Equal(t, "", cfg.OpenRouter.APIKey)
	assert.Equal(t, "openai/gpt-4o", cfg.OpenRouter.Model)
	assert.Equal(t, 500, cfg.OpenRouter.MaxTokens)
	assert.Equal(t, 0.7, cfg.OpenRouter.Temperature)
	assert.Equal(t,
		"Generate title, description and keywords for this image. "+
			"Return strictly in JSON format with fields 'title', 'description' and 'keywords'.",
		cfg.OpenRouter.Prompt)
	assert.Equal(t, false, cfg.OpenRouter.UseOpenRouterGoAdapter)

	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "json", cfg.Log.Format)

	assert.Equal(t, 3, cfg.Worker.Concurrency)
	assert.Equal(t, 3, cfg.Worker.MaxRetries)
	assert.Equal(t, 5*time.Second, cfg.Worker.RetryDelay)

	// Test custom values
	os.Setenv("RABBITMQ_URL", "amqp://user:pass@localhost:5672/")
	os.Setenv("RABBITMQ_CONSUMER_QUEUE", "test_queue")
	os.Setenv("RABBITMQ_PREFETCH_COUNT", "10")
	os.Setenv("MINIO_ENDPOINT", "localhost:9000")
	os.Setenv("MINIO_USE_SSL", "true")
	os.Setenv("OPENROUTER_MODEL", "anthropic/claude-3-opus")
	os.Setenv("OPENROUTER_MAX_TOKENS", "1000")
	os.Setenv("OPENROUTER_TEMPERATURE", "0.5")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("WORKER_CONCURRENCY", "5")
	os.Setenv("USE_OPENROUTERGO_ADAPTER", "true")

	cfg = New()
	assert.Equal(t, "amqp://user:pass@localhost:5672/", cfg.RabbitMQ.URL)
	assert.Equal(t, "test_queue", cfg.RabbitMQ.ConsumerQueue)
	assert.Equal(t, 10, cfg.RabbitMQ.PrefetchCount)
	assert.Equal(t, "localhost:9000", cfg.MinIO.Endpoint)
	assert.Equal(t, true, cfg.MinIO.UseSSL)
	assert.Equal(t, "anthropic/claude-3-opus", cfg.OpenRouter.Model)
	assert.Equal(t, 1000, cfg.OpenRouter.MaxTokens)
	assert.Equal(t, 0.5, cfg.OpenRouter.Temperature)
	assert.Equal(t, "debug", cfg.Log.Level)
	assert.Equal(t, "text", cfg.Log.Format)
	assert.Equal(t, true, cfg.OpenRouter.UseOpenRouterGoAdapter)
	assert.Equal(t, 5, cfg.Worker.Concurrency)
}

func TestConfigureLogger(t *testing.T) {
	// Test logger configuration with JSON format
	cfg := &Config{}
	cfg.Log.Level = "info"
	cfg.Log.Format = "json"

	logger := ConfigureLogger(cfg)
	assert.NotNil(t, logger)

	// Test logger configuration with text format
	cfg.Log.Format = "text"
	logger = ConfigureLogger(cfg)
	assert.NotNil(t, logger)

	// Test logger configuration with invalid level
	cfg.Log.Level = "invalid_level"
	logger = ConfigureLogger(cfg)
	assert.NotNil(t, logger)
}
