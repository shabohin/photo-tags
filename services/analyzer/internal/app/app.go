package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/analyzer/internal/api/openrouter"
	"github.com/shabohin/photo-tags/services/analyzer/internal/api/openrouter/openroutergo_adapter"
	"github.com/shabohin/photo-tags/services/analyzer/internal/config"
	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/service"
	"github.com/shabohin/photo-tags/services/analyzer/internal/storage/minio"
	"github.com/shabohin/photo-tags/services/analyzer/internal/transport/rabbitmq"
)

type App struct {
	consumer    *rabbitmq.Consumer
	publisher   *rabbitmq.Publisher
	minioClient *minio.Client
	processor   *service.MessageProcessorService
	logger      *logrus.Logger
	workerCount int
	shutdown    chan struct{}
	shutdownWg  sync.WaitGroup
}

func New(cfg *config.Config) (*App, error) {
	logger := config.ConfigureLogger(cfg)

	logger.Info("Initializing Analyzer Service")

	// Initialize MinIO client
	minioClient, err := minio.NewClient(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.UseSSL,
		cfg.MinIO.OriginalBucket,
		logger,
		cfg.MinIO.ConnectAttempts,
		cfg.MinIO.ConnectDelay,
	)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize MinIO client")
		return nil, err
	}

	var openRouterClient openrouter.OpenRouterClient
	if cfg.OpenRouter.UseOpenRouterGoAdapter {
		openRouterClient = openroutergo_adapter.NewOpenRouterGoAdapter(
			cfg.OpenRouter.APIKey,
			cfg.OpenRouter.Model,
			cfg.OpenRouter.MaxTokens,
			cfg.OpenRouter.Temperature,
			cfg.OpenRouter.Prompt,
			logger,
		)
	} else {
		openRouterClient = openrouter.NewClient(
			cfg.OpenRouter.APIKey,
			cfg.OpenRouter.Model,
			cfg.OpenRouter.MaxTokens,
			cfg.OpenRouter.Temperature,
			cfg.OpenRouter.Prompt,
			logger,
		)
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

	// Initialize image analyzer service
	imageAnalyzer := service.NewImageAnalyzer(minioClient, openRouterClient, logger)

	// Initialize message processor
	processor := service.NewMessageProcessor(
		imageAnalyzer,
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
		publisher.Close()
		logger.WithError(err).Error("Failed to initialize RabbitMQ consumer")
		return nil, err
	}

	return &App{
		consumer:    consumer,
		publisher:   publisher,
		minioClient: minioClient,
		processor:   processor,
		logger:      logger,
		workerCount: cfg.Worker.Concurrency,
		shutdown:    make(chan struct{}),
	}, nil
}

func (a *App) Start() error {
	a.logger.WithField("worker_count", a.workerCount).Info("Starting workers")

	ctx, cancel := context.WithCancel(context.Background())

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
		go func() {
			defer a.shutdownWg.Done()
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

func (a *App) startWorker(ctx context.Context, id int) {
	logger := a.logger.WithField("worker_id", id)
	logger.Info("Worker started")

	// Define message handler function
	handler := func(message []byte) error {
		processingCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
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
