package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/processor/internal/config"
	"github.com/shabohin/photo-tags/services/processor/internal/domain/service"
	"github.com/shabohin/photo-tags/services/processor/internal/exiftool"
	"github.com/shabohin/photo-tags/services/processor/internal/handler"
	"github.com/shabohin/photo-tags/services/processor/internal/storage/minio"
	"github.com/shabohin/photo-tags/services/processor/internal/transport/rabbitmq"
)

// App represents the processor application
type App struct {
	shutdown    chan struct{}
	consumer    *rabbitmq.Consumer
	publisher   *rabbitmq.Publisher
	minioClient *minio.Client
	exifTool    *exiftool.Client
	processor   *service.MessageProcessorService
	httpHandler *handler.Handler
	logger      *logrus.Logger
	shutdownWg  sync.WaitGroup
	workerCount int
	cfg         *config.Config
}

// New creates and initializes a new application
func New(cfg *config.Config) (*App, error) {
	logger := config.ConfigureLogger(cfg)

	logger.Info("Initializing Processor Service")

	// Initialize MinIO client
	minioClient, err := minio.NewClient(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.UseSSL,
		cfg.MinIO.OriginalBucket,
		cfg.MinIO.ProcessedBucket,
		logger,
		cfg.MinIO.ConnectAttempts,
		cfg.MinIO.ConnectDelay,
	)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize MinIO client")
		return nil, err
	}

	// Initialize ExifTool client
	exifToolClient := exiftool.NewClient(
		cfg.ExifTool.BinaryPath,
		cfg.ExifTool.CommandTimeout,
		logger,
	)

	// Check ExifTool availability
	if err := exifToolClient.CheckAvailability(); err != nil {
		logger.WithError(err).Warn("ExifTool availability check failed")
		// Don't fail startup, but log warning
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(cfg.ExifTool.TempDir, 0755); err != nil {
		logger.WithError(err).Error("Failed to create temp directory")
		return nil, err
	}

	// Initialize RabbitMQ publisher
	publisher, err := rabbitmq.NewPublisher(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.PublisherQueue,
		cfg.RabbitMQ.ReconnectAttempts,
		cfg.RabbitMQ.ReconnectDelay,
		logger,
	)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize RabbitMQ publisher")
		return nil, err
	}

	// Initialize image processor service
	imageProcessor := service.NewImageProcessor(
		minioClient,
		exifToolClient,
		cfg.ExifTool.TempDir,
		logger,
	)

	// Initialize message processor
	messageProcessor := service.NewMessageProcessor(
		imageProcessor,
		publisher,
		logger,
		cfg.Worker.MaxRetries,
		cfg.Worker.RetryDelay,
	)

	// Initialize RabbitMQ consumer
	consumer, err := rabbitmq.NewConsumer(
		cfg.RabbitMQ.URL,
		cfg.RabbitMQ.ConsumerQueue,
		cfg.RabbitMQ.PrefetchCount,
		cfg.RabbitMQ.ReconnectAttempts,
		cfg.RabbitMQ.ReconnectDelay,
		logger,
	)
	if err != nil {
		if closeErr := publisher.Close(); closeErr != nil {
			logger.WithError(closeErr).Error("Failed to close publisher during cleanup")
		}
		logger.WithError(err).Error("Failed to initialize RabbitMQ consumer")
		return nil, err
	}

	// Initialize HTTP handler for health checks
	httpHandler := handler.NewHandler(
		logger,
		cfg,
		consumer,
		publisher,
		minioClient,
		exifToolClient,
		cfg.Worker.Concurrency,
	)

	return &App{
		consumer:    consumer,
		publisher:   publisher,
		minioClient: minioClient,
		exifTool:    exifToolClient,
		processor:   messageProcessor,
		httpHandler: httpHandler,
		logger:      logger,
		workerCount: cfg.Worker.Concurrency,
		shutdown:    make(chan struct{}),
		cfg:         cfg,
	}, nil
}

// Start starts the application and worker pool
func (a *App) Start() error {
	a.logger.WithField("worker_count", a.workerCount).Info("Starting workers")

	ctx, cancel := context.WithCancel(context.Background())

	// Start HTTP server for health checks
	go func() {
		if err := a.httpHandler.StartServer(ctx); err != nil {
			a.logger.WithError(err).Error("HTTP server error")
		}
	}()

	// Start signal handler
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		sig := <-sigCh
		a.logger.WithField("signal", sig.String()).Info("Received shutdown signal")
		cancel()
		close(a.shutdown)
	}()

	// Start workers
	for i := 0; i < a.workerCount; i++ {
		workerID := i
		a.shutdownWg.Add(1)
		a.httpHandler.SetActiveWorkers(i + 1)
		go func() {
			defer a.shutdownWg.Done()
			defer func() {
				// Decrement active workers on exit
				activeWorkers := a.httpHandler.GetActiveWorkers()
				a.httpHandler.SetActiveWorkers(activeWorkers - 1)
			}()
			a.startWorker(ctx, workerID)
		}()
	}

	a.logger.Info("All workers started, waiting for shutdown signal")
	<-a.shutdown
	a.logger.Info("Shutdown signal received, waiting for workers to finish")
	a.shutdownWg.Wait()
	a.logger.Info("All workers stopped")

	return nil
}

// startWorker starts a single worker that processes messages
func (a *App) startWorker(ctx context.Context, id int) {
	logger := a.logger.WithField("worker_id", id)
	logger.Info("Worker started")

	// Define message handler function
	handler := func(message []byte) error {
		processingCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		err := a.processor.Process(processingCtx, message)
		if err != nil {
			logger.WithError(err).Error("Message processing failed")
			return err
		}
		return nil
	}

	// Start consuming messages
	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker shutting down")
			return
		case <-a.shutdown:
			logger.Info("Worker shutting down")
			return
		default:
			if err := a.consumer.Consume(ctx, handler); err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.WithError(err).Warn("Consumer disconnected, reconnecting...")
				time.Sleep(time.Second)
			}
		}
	}
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown() {
	a.logger.Info("Shutting down application")
	close(a.shutdown)
	a.shutdownWg.Wait()

	if err := a.consumer.Close(); err != nil {
		a.logger.WithError(err).Error("Error closing consumer")
	}

	if err := a.publisher.Close(); err != nil {
		a.logger.WithError(err).Error("Error closing publisher")
	}

	a.logger.Info("Application shutdown complete")
}
