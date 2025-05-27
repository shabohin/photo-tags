package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup
	os.Setenv("TELEGRAM_TOKEN", "test-token")
	os.Setenv("RABBITMQ_URL", "amqp://test:test@localhost:9002/")
	os.Setenv("MINIO_ENDPOINT", "test-endpoint:9000")
	os.Setenv("MINIO_ACCESS_KEY", "test-access-key")
	os.Setenv("MINIO_SECRET_KEY", "test-secret-key")
	os.Setenv("MINIO_USE_SSL", "true")
	os.Setenv("SERVER_PORT", "9003")

	// Execute
	cfg := LoadConfig()

	// Verify
	if cfg.TelegramToken != "test-token" {
		t.Errorf("Expected TelegramToken to be 'test-token', got '%s'", cfg.TelegramToken)
	}
	if cfg.RabbitMQURL != "amqp://test:test@localhost:9002/" {
		t.Errorf("Expected RabbitMQURL to be 'amqp://test:test@localhost:9002/', got '%s'", cfg.RabbitMQURL)
	}
	if cfg.MinIOEndpoint != "test-endpoint:9000" {
		t.Errorf("Expected MinIOEndpoint to be 'test-endpoint:9000', got '%s'", cfg.MinIOEndpoint)
	}
	if cfg.MinIOAccessKey != "test-access-key" {
		t.Errorf("Expected MinIOAccessKey to be 'test-access-key', got '%s'", cfg.MinIOAccessKey)
	}
	if cfg.MinIOSecretKey != "test-secret-key" {
		t.Errorf("Expected MinIOSecretKey to be 'test-secret-key', got '%s'", cfg.MinIOSecretKey)
	}
	if !cfg.MinIOUseSSL {
		t.Errorf("Expected MinIOUseSSL to be true, got false")
	}
	if cfg.ServerPort != 9003 {
		t.Errorf("Expected ServerPort to be 9003, got %d", cfg.ServerPort)
	}

	// Cleanup
	os.Unsetenv("TELEGRAM_TOKEN")
	os.Unsetenv("RABBITMQ_URL")
	os.Unsetenv("MINIO_ENDPOINT")
	os.Unsetenv("MINIO_ACCESS_KEY")
	os.Unsetenv("MINIO_SECRET_KEY")
	os.Unsetenv("MINIO_USE_SSL")
	os.Unsetenv("SERVER_PORT")
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// Setup - clear all relevant environment variables
	os.Unsetenv("TELEGRAM_TOKEN")
	os.Unsetenv("RABBITMQ_URL")
	os.Unsetenv("MINIO_ENDPOINT")
	os.Unsetenv("MINIO_ACCESS_KEY")
	os.Unsetenv("MINIO_SECRET_KEY")
	os.Unsetenv("MINIO_USE_SSL")
	os.Unsetenv("SERVER_PORT")

	// Execute
	cfg := LoadConfig()

	// Verify defaults
	if cfg.TelegramToken != "" {
		t.Errorf("Expected TelegramToken to be empty, got '%s'", cfg.TelegramToken)
	}
	if cfg.RabbitMQURL != "amqp://user:password@localhost:9002/" {
		t.Errorf("Expected RabbitMQURL to be 'amqp://user:password@localhost:9002/', got '%s'", cfg.RabbitMQURL)
	}
	if cfg.MinIOEndpoint != "localhost:9000" {
		t.Errorf("Expected MinIOEndpoint to be 'localhost:9000', got '%s'", cfg.MinIOEndpoint)
	}
	if cfg.MinIOAccessKey != "minioadmin" {
		t.Errorf("Expected MinIOAccessKey to be 'minioadmin', got '%s'", cfg.MinIOAccessKey)
	}
	if cfg.MinIOSecretKey != "minioadmin" {
		t.Errorf("Expected MinIOSecretKey to be 'minioadmin', got '%s'", cfg.MinIOSecretKey)
	}
	if cfg.MinIOUseSSL {
		t.Errorf("Expected MinIOUseSSL to be false, got true")
	}
	if cfg.ServerPort != 9003 {
		t.Errorf("Expected ServerPort to be 9003, got %d", cfg.ServerPort)
	}
}
