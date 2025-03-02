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
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
	"github.com/shabohin/photo-tags/services/gateway/internal/handler"
	"github.com/shabohin/photo-tags/services/gateway/internal/telegram"
)

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
	logger.Info(fmt.Sprintf("Starting Gateway Service v1.0.0 at %s", time.Now().Format(time.RFC3339)), nil)

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
func initializeDependencies(ctx context.Context, cfg *config.Config, logger *logging.Logger) (storage.MinIOInterface, messaging.RabbitMQInterface, error) {
	// Initialize MinIO client
	logger.Info("Initializing MinIO client", map[string]interface{}{
		"endpoint": cfg.MinIOEndpoint,
		"use_ssl":  cfg.MinIOUseSSL,
	})

	minioClient, err := storage.NewMinIOClient(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOUseSSL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Check MinIO connection
	if err := minioClient.EnsureBucketExists(ctx, storage.BucketOriginal); err != nil {
		return nil, nil, fmt.Errorf("failed to check MinIO connection: %w", err)
	}

	// Initialize RabbitMQ client
	logger.Info("Initializing RabbitMQ client", map[string]interface{}{
		"url": cfg.RabbitMQURL,
	})

	rabbitmqClient, err := messaging.NewRabbitMQClient(cfg.RabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	// Declare necessary queues
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
