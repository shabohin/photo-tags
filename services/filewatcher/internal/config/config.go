package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the filewatcher service
type Config struct {
	// Input/Output directories
	InputDir     string
	OutputDir    string
	ProcessedDir string

	// Scan configuration
	ScanInterval time.Duration
	UseFsnotify  bool

	// File validation
	MaxFileSizeMB     int64
	AllowedExtensions []string

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
		InputDir:          getEnv("INPUT_DIR", "/app/input"),
		OutputDir:         getEnv("OUTPUT_DIR", "/app/output"),
		ProcessedDir:      getEnv("PROCESSED_DIR", "/app/input/processed"),
		ScanInterval:      time.Duration(getEnvInt("SCAN_INTERVAL_SECONDS", 5)) * time.Second,
		UseFsnotify:       getEnvBool("USE_FSNOTIFY", true),
		MaxFileSizeMB:     int64(getEnvInt("MAX_FILE_SIZE_MB", 50)),
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".JPG", ".JPEG", ".PNG"},
		RabbitMQURL:       getEnv("RABBITMQ_URL", "amqp://user:password@localhost:5672/"),
		MinIOEndpoint:     getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:    getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:    getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOUseSSL:       getEnvBool("MINIO_USE_SSL", false),
		ServerPort:        getEnvInt("SERVER_PORT", 8081),
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
