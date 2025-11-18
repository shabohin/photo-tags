package config

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNew(t *testing.T) {
	// Test default values
	cfg := New()

	if cfg.RabbitMQ.URL != "amqp://guest:guest@rabbitmq:5672/" {
		t.Errorf("Expected default RabbitMQ URL, got %s", cfg.RabbitMQ.URL)
	}

	if cfg.RabbitMQ.ConsumerQueue != "metadata_generated" {
		t.Errorf("Expected consumer queue 'metadata_generated', got %s", cfg.RabbitMQ.ConsumerQueue)
	}

	if cfg.RabbitMQ.PublisherQueue != "image_processed" {
		t.Errorf("Expected publisher queue 'image_processed', got %s", cfg.RabbitMQ.PublisherQueue)
	}

	if cfg.MinIO.OriginalBucket != "original" {
		t.Errorf("Expected original bucket 'original', got %s", cfg.MinIO.OriginalBucket)
	}

	if cfg.MinIO.ProcessedBucket != "processed" {
		t.Errorf("Expected processed bucket 'processed', got %s", cfg.MinIO.ProcessedBucket)
	}

	if cfg.ExifTool.BinaryPath != "/usr/bin/exiftool" {
		t.Errorf("Expected exiftool path '/usr/bin/exiftool', got %s", cfg.ExifTool.BinaryPath)
	}

	if cfg.Worker.Concurrency != 3 {
		t.Errorf("Expected worker concurrency 3, got %d", cfg.Worker.Concurrency)
	}
}

func TestNewWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("RABBITMQ_URL", "amqp://test:test@localhost:5672/")
	os.Setenv("WORKER_CONCURRENCY", "5")
	os.Setenv("EXIFTOOL_TEMP_DIR", "/custom/temp")
	defer func() {
		os.Unsetenv("RABBITMQ_URL")
		os.Unsetenv("WORKER_CONCURRENCY")
		os.Unsetenv("EXIFTOOL_TEMP_DIR")
	}()

	cfg := New()

	if cfg.RabbitMQ.URL != "amqp://test:test@localhost:5672/" {
		t.Errorf("Expected custom RabbitMQ URL, got %s", cfg.RabbitMQ.URL)
	}

	if cfg.Worker.Concurrency != 5 {
		t.Errorf("Expected worker concurrency 5, got %d", cfg.Worker.Concurrency)
	}

	if cfg.ExifTool.TempDir != "/custom/temp" {
		t.Errorf("Expected temp dir '/custom/temp', got %s", cfg.ExifTool.TempDir)
	}
}

func TestConfigureLogger(t *testing.T) {
	cfg := &Config{}
	cfg.Log.Level = "debug"
	cfg.Log.Format = "json"

	logger := ConfigureLogger(cfg)

	if logger.Level != logrus.DebugLevel {
		t.Errorf("Expected debug level, got %v", logger.Level)
	}

	if _, ok := logger.Formatter.(*logrus.JSONFormatter); !ok {
		t.Error("Expected JSON formatter")
	}
}

func TestGetEnvAsDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "10s")
	defer os.Unsetenv("TEST_DURATION")

	duration := getEnvAsDuration("TEST_DURATION", 5*time.Second)
	if duration != 10*time.Second {
		t.Errorf("Expected 10s, got %v", duration)
	}

	// Test default
	duration = getEnvAsDuration("NON_EXISTENT", 5*time.Second)
	if duration != 5*time.Second {
		t.Errorf("Expected default 5s, got %v", duration)
	}
}
