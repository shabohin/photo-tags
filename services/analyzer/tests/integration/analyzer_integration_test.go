package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shabohin/photo-tags/services/analyzer/internal/api/openrouter"
	"github.com/shabohin/photo-tags/services/analyzer/internal/config"
	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/service"
	"github.com/shabohin/photo-tags/services/analyzer/internal/storage/minio"
	"github.com/shabohin/photo-tags/services/analyzer/internal/transport/rabbitmq"
)

const (
	testRabbitMQURL    = "amqp://testuser:testpass@localhost:5673/"
	testMinIOEndpoint  = "localhost:9002"
	testMinIOAccessKey = "testuser"
	testMinIOSecretKey = "testpass123"
	testTimeout        = 30 * time.Second
	retryAttempts      = 5
	retryDelay         = 2 * time.Second
)

// MockOpenRouterClient is a mock implementation of OpenRouterClient for testing
type MockOpenRouterClient struct {
	mu            sync.Mutex
	callCount     int
	shouldFail    bool
	failCount     int
	failThreshold int
}

func NewMockOpenRouterClient() *MockOpenRouterClient {
	return &MockOpenRouterClient{
		shouldFail:    false,
		failCount:     0,
		failThreshold: 0,
	}
}

func (m *MockOpenRouterClient) SetFailureMode(failCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = true
	m.failThreshold = failCount
	m.callCount = 0
}

func (m *MockOpenRouterClient) AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++

	// Simulate failure for first N calls
	if m.shouldFail && m.callCount <= m.failThreshold {
		return model.Metadata{}, fmt.Errorf("mock error: simulated failure %d/%d", m.callCount, m.failThreshold)
	}

	// Return mock metadata
	return model.Metadata{
		Title:       "Test Image Title",
		Description: "This is a test image description",
		Keywords:    []string{"test", "image", "mock"},
	}, nil
}

func (m *MockOpenRouterClient) GetAvailableModels(ctx context.Context) ([]openrouter.Model, error) {
	return []openrouter.Model{
		{
			ID:   "test/model",
			Name: "Test Model",
		},
	}, nil
}

func (m *MockOpenRouterClient) SelectBestFreeVisionModel(models []openrouter.Model) (*openrouter.Model, error) {
	if len(models) > 0 {
		return &models[0], nil
	}
	return nil, fmt.Errorf("no models available")
}

func (m *MockOpenRouterClient) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// TestMain sets up and tears down test infrastructure
func TestMain(m *testing.M) {
	// Check if test infrastructure is running
	if !checkInfrastructure() {
		fmt.Println("Test infrastructure not ready. Run: docker-compose -f docker-compose.test.yml up -d")
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}

// checkInfrastructure verifies that RabbitMQ and MinIO are accessible
func checkInfrastructure() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check RabbitMQ
	conn, err := amqp.Dial(testRabbitMQURL)
	if err != nil {
		fmt.Printf("RabbitMQ not accessible: %v\n", err)
		return false
	}
	conn.Close()

	// Check MinIO
	minioClient, err := minio.New(testMinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(testMinIOAccessKey, testMinIOSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		fmt.Printf("MinIO client creation failed: %v\n", err)
		return false
	}

	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		fmt.Printf("MinIO not accessible: %v\n", err)
		return false
	}

	return true
}

// TestAnalyzerWithMockOpenRouter tests analyzer service with mock OpenRouter
func TestAnalyzerWithMockOpenRouter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create MinIO client
	minioClient, err := minio.NewClient(
		testMinIOEndpoint,
		testMinIOAccessKey,
		testMinIOSecretKey,
		false,
		"original",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure bucket exists
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create bucket")

	// Upload test image
	testImageData := []byte("fake image data")
	objectName := "test-image-" + time.Now().Format("20060102150405") + ".jpg"
	err = minioClient.UploadFile(ctx, "original", objectName, strings.NewReader(string(testImageData)), int64(len(testImageData)))
	require.NoError(t, err, "Failed to upload test image")

	// Create mock OpenRouter client
	mockClient := NewMockOpenRouterClient()

	// Create analyzer service
	analyzer := service.NewImageAnalyzer(minioClient, mockClient, logger)

	// Analyze image
	metadata, err := analyzer.AnalyzeImage(ctx, "original", objectName)
	require.NoError(t, err, "Failed to analyze image")

	// Verify metadata
	assert.Equal(t, "Test Image Title", metadata.Title)
	assert.Equal(t, "This is a test image description", metadata.Description)
	assert.Equal(t, []string{"test", "image", "mock"}, metadata.Keywords)

	// Verify mock was called
	assert.Equal(t, 1, mockClient.GetCallCount(), "Mock should be called once")

	logger.Info("Analyzer with mock OpenRouter test passed")
}

// TestAnalyzerRetryLogic tests retry logic with failing OpenRouter
func TestAnalyzerRetryLogic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create MinIO client
	minioClient, err := minio.NewClient(
		testMinIOEndpoint,
		testMinIOAccessKey,
		testMinIOSecretKey,
		false,
		"original",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure bucket exists
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create bucket")

	// Upload test image
	testImageData := []byte("fake image data for retry test")
	objectName := "test-retry-image-" + time.Now().Format("20060102150405") + ".jpg"
	err = minioClient.UploadFile(ctx, "original", objectName, strings.NewReader(string(testImageData)), int64(len(testImageData)))
	require.NoError(t, err, "Failed to upload test image")

	// Create mock OpenRouter client that fails first 2 times
	mockClient := NewMockOpenRouterClient()
	mockClient.SetFailureMode(2)

	// Create message processor with retry logic
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-output-queue",
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")
	defer publisher.Close()

	analyzer := service.NewImageAnalyzer(minioClient, mockClient, logger)
	processor := service.NewMessageProcessor(analyzer, publisher, logger, 3, 1*time.Second)

	// Create test message
	message := model.UploadMessage{
		BucketName: "original",
		ObjectName: objectName,
		TraceID:    "test-trace-id",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err, "Failed to marshal message")

	// Process message (should succeed after retries)
	err = processor.Process(ctx, messageBytes)
	require.NoError(t, err, "Message processing should succeed after retries")

	// Verify mock was called 3 times (2 failures + 1 success)
	assert.Equal(t, 3, mockClient.GetCallCount(), "Mock should be called 3 times (2 failures + 1 success)")

	logger.Info("Analyzer retry logic test passed")
}

// TestConcurrentMessageProcessing tests concurrent message processing
func TestConcurrentMessageProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create MinIO client
	minioClient, err := minio.NewClient(
		testMinIOEndpoint,
		testMinIOAccessKey,
		testMinIOSecretKey,
		false,
		"original",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure bucket exists
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create bucket")

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-concurrent-output-"+time.Now().Format("20060102150405"),
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")
	defer publisher.Close()

	// Create mock OpenRouter client
	mockClient := NewMockOpenRouterClient()

	// Create analyzer and processor
	analyzer := service.NewImageAnalyzer(minioClient, mockClient, logger)
	processor := service.NewMessageProcessor(analyzer, publisher, logger, 3, 1*time.Second)

	const numMessages = 20
	const numWorkers = 5

	// Upload test images
	imageNames := make([]string, numMessages)
	for i := 0; i < numMessages; i++ {
		objectName := fmt.Sprintf("concurrent-test-%d-%s.jpg", i, time.Now().Format("20060102150405"))
		imageNames[i] = objectName
		testImageData := []byte(fmt.Sprintf("fake image data %d", i))
		err = minioClient.UploadFile(ctx, "original", objectName, strings.NewReader(string(testImageData)), int64(len(testImageData)))
		require.NoError(t, err, "Failed to upload test image")
	}

	// Process messages concurrently
	var wg sync.WaitGroup
	errors := make(chan error, numMessages)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := workerID; j < numMessages; j += numWorkers {
				message := model.UploadMessage{
					BucketName: "original",
					ObjectName: imageNames[j],
					TraceID:    fmt.Sprintf("trace-%d-%d", workerID, j),
				}
				messageBytes, err := json.Marshal(message)
				if err != nil {
					errors <- err
					continue
				}

				if err := processor.Process(ctx, messageBytes); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Processing error: %v", err)
		errorCount++
	}

	assert.Equal(t, 0, errorCount, "Should have no processing errors")
	assert.GreaterOrEqual(t, mockClient.GetCallCount(), numMessages, "Should process all messages")

	logger.Info("Concurrent message processing test passed")
}

// TestGracefulShutdown tests graceful shutdown of analyzer components
func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create consumer
	consumer, err := rabbitmq.NewConsumer(
		testRabbitMQURL,
		"test-shutdown-queue-"+time.Now().Format("20060102150405"),
		5,
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create consumer")

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-shutdown-output-"+time.Now().Format("20060102150405"),
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")

	// Start a goroutine that consumes messages
	shutdownCtx, shutdownCancel := context.WithCancel(ctx)
	consumeDone := make(chan struct{})

	go func() {
		defer close(consumeDone)
		handler := func(msg []byte) error {
			logger.Info("Received message during shutdown test")
			return nil
		}
		_ = consumer.Consume(shutdownCtx, handler)
	}()

	// Wait a bit to ensure consumer is running
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	shutdownCancel()

	// Wait for consumer to finish (with timeout)
	select {
	case <-consumeDone:
		logger.Info("Consumer shut down gracefully")
	case <-time.After(5 * time.Second):
		t.Error("Consumer did not shut down within timeout")
	}

	// Close connections
	err = consumer.Close()
	assert.NoError(t, err, "Consumer should close gracefully")

	err = publisher.Close()
	assert.NoError(t, err, "Publisher should close gracefully")

	logger.Info("Graceful shutdown test passed")
}

// TestEndToEndWorkflow tests complete workflow from upload to analysis
func TestEndToEndWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	queueName := "test-e2e-" + time.Now().Format("20060102150405")

	// Create MinIO client
	minioClient, err := minio.NewClient(
		testMinIOEndpoint,
		testMinIOAccessKey,
		testMinIOSecretKey,
		false,
		"original",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure bucket exists
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create bucket")

	// Create consumer
	consumer, err := rabbitmq.NewConsumer(
		testRabbitMQURL,
		queueName,
		1,
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create consumer")
	defer consumer.Close()

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-e2e-output-"+time.Now().Format("20060102150405"),
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")
	defer publisher.Close()

	// Create mock OpenRouter and analyzer
	mockClient := NewMockOpenRouterClient()
	analyzer := service.NewImageAnalyzer(minioClient, mockClient, logger)
	processor := service.NewMessageProcessor(analyzer, publisher, logger, 3, 1*time.Second)

	// Upload test image
	testImageData := []byte("end to end test image data")
	objectName := "e2e-test-" + time.Now().Format("20060102150405") + ".jpg"
	err = minioClient.UploadFile(ctx, "original", objectName, strings.NewReader(string(testImageData)), int64(len(testImageData)))
	require.NoError(t, err, "Failed to upload test image")

	// Publish message to queue
	message := model.UploadMessage{
		BucketName: "original",
		ObjectName: objectName,
		TraceID:    "e2e-trace-id",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err, "Failed to marshal message")

	// Manually publish to input queue (simulating gateway)
	conn, err := amqp.Dial(testRabbitMQURL)
	require.NoError(t, err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err, "Failed to create channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	require.NoError(t, err, "Failed to declare queue")

	err = ch.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        messageBytes,
	})
	require.NoError(t, err, "Failed to publish message")

	// Consume and process message
	messageProcessed := make(chan bool, 1)
	handler := func(msg []byte) error {
		err := processor.Process(ctx, msg)
		if err == nil {
			messageProcessed <- true
		}
		return err
	}

	go func() {
		_ = consumer.Consume(ctx, handler)
	}()

	// Wait for message to be processed
	select {
	case <-messageProcessed:
		logger.Info("Message processed successfully")
	case <-time.After(10 * time.Second):
		t.Error("Message was not processed within timeout")
	}

	// Verify mock was called
	assert.Equal(t, 1, mockClient.GetCallCount(), "Mock should be called once")

	logger.Info("End-to-end workflow test passed")
}
