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
	"github.com/shabohin/photo-tags/services/filewatcher/internal/api"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/config"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/consumer"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/processor"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/statistics"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/watcher"
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
	fmt.Println("Starting File Watcher Service...")
	log.Println("File Watcher Service is up and running")

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
	logger := logging.NewLogger("filewatcher")
	logger.Info("Starting File Watcher Service v1.0.0 at "+time.Now().Format(time.RFC3339), nil)

	// Initialize statistics
	stats := statistics.NewStatistics()

	// Initialize dependencies
	minioClient, rabbitmqClient, err := initializeDependencies(ctx, cfg, logger)
	if err != nil {
		logger.Error("Failed to initialize dependencies", err)
		os.Exit(1)
	}
	defer rabbitmqClient.Close()

	// Create processor
	proc := processor.NewProcessor(cfg, logger, minioClient, rabbitmqClient, stats)

	// Create watcher
	watch := watcher.NewWatcher(cfg, logger, proc)

	// Create consumer
	cons := consumer.NewConsumer(cfg, logger, minioClient, rabbitmqClient, stats)

	// Create API server
	apiServer := api.NewServer(cfg, logger, watch, stats)

	// Start consumer
	if err := cons.Start(ctx); err != nil {
		logger.Error("Failed to start consumer", err)
		os.Exit(1)
	}
	logger.Info("Consumer started", map[string]interface{}{
		"queue": messaging.QueueImageProcessed,
	})

	// Start watcher
	go func() {
		if err := watch.Start(ctx); err != nil {
			logger.Error("Watcher error", err)
		}
	}()
	logger.Info("File watcher started", map[string]interface{}{
		"input_dir":     cfg.InputDir,
		"output_dir":    cfg.OutputDir,
		"use_fsnotify":  cfg.UseFsnotify,
		"scan_interval": cfg.ScanInterval,
	})

	// Start API server
	go func() {
		if err := apiServer.Start(ctx); err != nil {
			logger.Error("API server error", err)
		}
	}()
	logger.Info("API server started", map[string]interface{}{
		"port": cfg.ServerPort,
	})

	// Block until context is canceled
	logger.Info("File Watcher Service running. Press Ctrl+C to stop.", nil)
	<-ctx.Done()
	logger.Info("Shutting down File Watcher Service", nil)

	// Allow some time for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		logger.Info("Shutdown timed out, forcing exit", nil)
	case <-time.After(2 * time.Second):
		logger.Info("File Watcher Service shutdown complete", nil)
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

	// Ensure buckets exist
	err = retry(5, 2*time.Second, logger, "MinIO bucket check (original)", func() error {
		return minioClient.EnsureBucketExists(ctx, storage.BucketOriginal)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ensure original bucket exists: %w", err)
	}

	err = retry(5, 2*time.Second, logger, "MinIO bucket check (processed)", func() error {
		return minioClient.EnsureBucketExists(ctx, storage.BucketProcessed)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ensure processed bucket exists: %w", err)
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
