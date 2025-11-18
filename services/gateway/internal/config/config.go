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

	// PostgreSQL configuration
	PostgresHost     string
	PostgresPort     int
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
	PostgresSSLMode  string

	// Server configuration
	ServerPort int
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	cfg := &Config{
		TelegramToken:    getEnv("TELEGRAM_TOKEN", ""),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://user:password@localhost:5672/"),
		MinIOEndpoint:    getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:   getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:   getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:      getEnvBool("MINIO_USE_SSL", false),
		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnvInt("POSTGRES_PORT", 5432),
		PostgresDB:       getEnv("POSTGRES_DB", "photo_tags"),
		PostgresUser:     getEnv("POSTGRES_USER", "photo_tags_user"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "photo_tags_password"),
		PostgresSSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),
		ServerPort:       getEnvInt("SERVER_PORT", 8080),
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
