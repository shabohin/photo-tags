package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         string
	RabbitMQURL  string
	GatewayURL   string
	AnalyzerURL  string
	ProcessorURL string
	MinIOURL     string
	RabbitMQMgmt string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("DASHBOARD_PORT", "3000"),
		RabbitMQURL:  getEnv("RABBITMQ_URL", "amqp://user:password@rabbitmq:5672/"),
		GatewayURL:   getEnv("GATEWAY_URL", "http://gateway:8080"),
		AnalyzerURL:  getEnv("ANALYZER_URL", "http://analyzer:8081"),
		ProcessorURL: getEnv("PROCESSOR_URL", "http://processor:8082"),
		MinIOURL:     getEnv("MINIO_URL", "http://localhost:9001"),
		RabbitMQMgmt: getEnv("RABBITMQ_MGMT_URL", "http://localhost:15672"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
