package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/observability"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
	"github.com/shabohin/photo-tags/services/gateway/internal/handler"
	"github.com/shabohin/photo-tags/services/gateway/internal/telegram"
)

func retry(attempts int, delay time.Duration, logger *logging.Logger, operationName string, fn func() error) error {
	for i := 1; i <= attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		logger.Error(fmt.Sprintf("Attempt %d/%d failed for %s", i, attempts, operationName), err)
		if i < attempts {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("all %d attempts failed for %s", attempts, operationName)
}

func main() {
	fmt.Println("Starting Gateway Service...")
	log.Println("Gateway Service is up and running")

	// Create context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		log.Printf("Received signal: %v", sig)
		cancel()
	}()

	// Load configuration
	cfg := config.LoadConfig()

	// Create logger
	logger := logging.NewLogger("gateway")
	logger.Info("Starting Gateway Service v1.0.0 at "+time.Now().Format(time.RFC3339), nil)

	// Initialize OpenTelemetry tracing
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "localhost:4317"
	}
	
	if err := observability.InitTracing("gateway", "1.0.0", otlpEndpoint); err != nil {
		logger.Error("Failed to initialize tracing", err)
		// Continue without tracing - not critical for basic functionality
	} else {
		logger.Info("OpenTelemetry tracing initialized", map[string]interface{}{
			"endpoint": otlpEndpoint,
		})
		
		// Setup graceful tracing shutdown
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := observability.Shutdown(shutdownCtx); err != nil {
				logger.Error("Failed to shutdown tracing", err)
			}
		}()
	}

	// Initialize dependencies
	minioClient, rabbitmqClient, err := initializeDependencies(ctx, cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize dependencies", err)
		os.Exit(1)
	}
	defer rabbitmqClient.Close()

	// Create and start HTTP handler
	httpHandler := handler.NewHandler(logger, cfg)
	go func() {
		if err := httpHandler.StartServer(ctx); err != nil {
			logger.Error("HTTP server error", err)
		}
	}()
	logger.Info("HTTP server started", nil)

	// Create and start Telegram bot if token is provided
	if cfg.TelegramToken != "" {
		bot, err := telegram.NewBot(cfg, logger, minioClient, rabbitmqClient)
		if err != nil {
			logger.Error("Failed to create Telegram bot", err)
			os.Exit(1)
		}

		go func() {
			if err := bot.Start(ctx); err != nil {
				logger.Error("Telegram bot error", err)
			}
		}()
		logger.Info("Telegram bot started", map[string]interface{}{
			"bot_username": bot.GetUsername(),
		})
	} else {
		logger.Info("Telegram token not provided, bot will not be started", nil)
	}

	// Block until context is canceled
	logger.Info("Gateway service running. Press Ctrl+C to stop.", nil)
	<-ctx.Done()
	logger.Info("Shutting down Gateway Service", nil)

	// Allow some time for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		logger.Info("Shutdown timed out, forcing exit", nil)
	case <-time.After(2 * time.Second):
		logger.Info("Gateway Service shutdown complete", nil)
	}
}

// initializeDependencies initializes MinIO and RabbitMQ clients
func initializeDependencies(
	ctx context.Context,
	cfg *config.Config,
	logger *logging.Logger,
) (storage.MinIOInterface, messaging.RabbitMQInterface, error) {
	var minioClient storage.MinIOInterface
	var rabbitmqClient messaging.RabbitMQInterface

	logger.Info("Initializing MinIO client", map[string]interface{}{
		"endpoint": cfg.MinIOEndpoint,
		"use_ssl":  cfg.MinIOUseSSL,
	})

	err := retry(5, 2*time.Second, logger, "MinIO client creation", func() error {
		client, clientErr := storage.NewMinIOClient(
			cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOUseSSL)
		if clientErr != nil {
			return clientErr
		}
		minioClient = client
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create MinIO client after retries: %w", err)
	}

	err = retry(5, 2*time.Second, logger, "MinIO bucket check", func() error {
		return minioClient.EnsureBucketExists(ctx, storage.BucketOriginal)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check MinIO connection after retries: %w", err)
	}

	logger.Info("Initializing RabbitMQ client", map[string]interface{}{
		"url": cfg.RabbitMQURL,
	})

	err = retry(5, 2*time.Second, logger, "RabbitMQ client creation", func() error {
		client, clientErr := messaging.NewRabbitMQClient(cfg.RabbitMQURL)
		if clientErr != nil {
			return clientErr
		}
		rabbitmqClient = client
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RabbitMQ client after retries: %w", err)
	}

	logger.Info("Declaring RabbitMQ queues", nil)

	if _, err := rabbitmqClient.DeclareQueue(messaging.QueueImageUpload); err != nil {
		return nil, nil, fmt.Errorf("failed to declare queue %s: %w", messaging.QueueImageUpload, err)
	}

	if _, err := rabbitmqClient.DeclareQueue(messaging.QueueImageProcessed); err != nil {
		return nil, nil, fmt.Errorf("failed to declare queue %s: %w", messaging.QueueImageProcessed, err)
	}

	logger.Info("Dependencies initialized successfully", nil)
	return minioClient, rabbitmqClient, nil
}
