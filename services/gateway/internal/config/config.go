package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the gateway service
type Config struct {
	// Telegram Bot configuration
	TelegramToken string

	// RabbitMQ configuration
	RabbitMQURL string

	// MinIO configuration
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOUseSSL    bool

	// Server configuration
	ServerPort int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{
		TelegramToken:  getEnv("TELEGRAM_TOKEN", ""),
		RabbitMQURL:    getEnv("RABBITMQ_URL", "amqp://user:password@localhost:9002/"),
		MinIOEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:    getEnvBool("MINIO_USE_SSL", false),
		ServerPort:     getEnvInt("SERVER_PORT", 9003),
	}

	return cfg
}

// Helper functions to get environment variables
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		boolValue, err := strconv.ParseBool(value)
		if err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		intValue, err := strconv.Atoi(value)
		if err == nil {
			return intValue
		}
	}
	return defaultValue
}
