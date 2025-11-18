package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

	"github.com/shabohin/photo-tags/services/processor/internal/domain/model"
	"github.com/shabohin/photo-tags/services/processor/internal/domain/service"
	"github.com/shabohin/photo-tags/services/processor/internal/exiftool"
	"github.com/shabohin/photo-tags/services/processor/internal/storage/minio"
	"github.com/shabohin/photo-tags/services/processor/internal/transport/rabbitmq"
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

// TestMain sets up and tears down test infrastructure
func TestMain(m *testing.M) {
	// Check if test infrastructure is running
	if !checkInfrastructure() {
		fmt.Println("Test infrastructure not ready. Run: docker-compose -f docker-compose.test.yml up -d")
		os.Exit(1)
	}

	// Check if ExifTool is installed
	if !checkExifTool() {
		fmt.Println("ExifTool not found. Please install it: apt-get install libimage-exiftool-perl")
		os.Exit(1)
	}

	// Create temp directory for tests
	tempDir := "/tmp/processor-integration-tests"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

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

// checkExifTool verifies ExifTool is installed
func checkExifTool() bool {
	_, err := exec.LookPath("exiftool")
	return err == nil
}

// createTestImage creates a minimal JPEG image for testing
func createTestImage(t *testing.T, path string) {
	// Minimal JPEG header (1x1 pixel red square)
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00,
		0x3F, 0x00, 0xD2, 0xCF, 0xFF, 0xD9,
	}

	err := os.WriteFile(path, jpegData, 0644)
	require.NoError(t, err, "Failed to create test image")
}

// TestExifToolBasicOperation tests basic ExifTool functionality
func TestExifToolBasicOperation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create ExifTool client
	exifClient := exiftool.NewClient("exiftool", 30*time.Second, logger)

	// Check availability
	err := exifClient.CheckAvailability()
	require.NoError(t, err, "ExifTool should be available")

	// Create temp test image
	tempDir := "/tmp/processor-integration-tests"
	testImagePath := filepath.Join(tempDir, "test-basic.jpg")
	createTestImage(t, testImagePath)
	defer os.Remove(testImagePath)

	// Write metadata
	metadata := exiftool.Metadata{
		Title:       "Test Image Title",
		Description: "Test image description",
		Keywords:    []string{"test", "image", "exiftool"},
	}

	err = exifClient.WriteMetadata(ctx, testImagePath, metadata, "test-trace-id")
	require.NoError(t, err, "Failed to write metadata")

	// Verify metadata was written
	cmd := exec.Command("exiftool", "-Title", "-Description", "-Keywords", testImagePath)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to read metadata")

	outputStr := string(output)
	assert.Contains(t, outputStr, "Test Image Title", "Title should be written")

	logger.Info("ExifTool basic operation test passed")
}

// TestProcessorWithRealStorage tests processor with real MinIO storage
func TestProcessorWithRealStorage(t *testing.T) {
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
		"processed",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure buckets exist
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create original bucket")
	err = minioClient.EnsureBucketExists(ctx, "processed")
	require.NoError(t, err, "Failed to create processed bucket")

	// Create ExifTool client
	tempDir := "/tmp/processor-integration-tests"
	exifClient := exiftool.NewClient("exiftool", 30*time.Second, logger)

	// Create test image
	testImagePath := filepath.Join(tempDir, "test-storage.jpg")
	createTestImage(t, testImagePath)
	defer os.Remove(testImagePath)

	// Upload image to original bucket
	imageFile, err := os.Open(testImagePath)
	require.NoError(t, err, "Failed to open test image")
	defer imageFile.Close()

	stat, err := imageFile.Stat()
	require.NoError(t, err, "Failed to stat test image")

	objectName := "test-storage-" + time.Now().Format("20060102150405") + ".jpg"
	err = minioClient.UploadFile(ctx, "original", objectName, imageFile, stat.Size())
	require.NoError(t, err, "Failed to upload test image")

	// Create image processor
	imageProcessor := service.NewImageProcessor(minioClient, exifClient, tempDir, logger)

	// Process image
	metadata := model.Metadata{
		Title:       "Processed Image",
		Description: "This image was processed",
		Keywords:    []string{"processed", "test"},
	}

	processedObjectName, err := imageProcessor.ProcessImage(ctx, "original", objectName, metadata, "test-trace-storage")
	require.NoError(t, err, "Failed to process image")
	assert.NotEmpty(t, processedObjectName, "Processed object name should not be empty")

	// Verify processed image exists in processed bucket
	reader, err := minioClient.DownloadFile(ctx, "processed", processedObjectName)
	require.NoError(t, err, "Failed to download processed image")
	defer reader.Close()

	downloadedData, err := io.ReadAll(reader)
	require.NoError(t, err, "Failed to read downloaded image")
	assert.NotEmpty(t, downloadedData, "Downloaded image should not be empty")

	logger.Info("Processor with real storage test passed")
}

// TestProcessorRetryLogic tests retry logic for image processing
func TestProcessorRetryLogic(t *testing.T) {
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
		"processed",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure buckets exist
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create original bucket")
	err = minioClient.EnsureBucketExists(ctx, "processed")
	require.NoError(t, err, "Failed to create processed bucket")

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-retry-output-"+time.Now().Format("20060102150405"),
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")
	defer publisher.Close()

	tempDir := "/tmp/processor-integration-tests"
	exifClient := exiftool.NewClient("exiftool", 30*time.Second, logger)
	imageProcessor := service.NewImageProcessor(minioClient, exifClient, tempDir, logger)

	// Create message processor with retry logic
	messageProcessor := service.NewMessageProcessor(imageProcessor, publisher, logger, 3, 1*time.Second)

	// Test with non-existent image (should retry and fail)
	message := model.AnalysisMessage{
		BucketName: "original",
		ObjectName: "non-existent-image.jpg",
		Metadata: model.Metadata{
			Title:       "Test",
			Description: "Test",
			Keywords:    []string{"test"},
		},
		TraceID: "test-retry-nonexistent",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err, "Failed to marshal message")

	err = messageProcessor.Process(ctx, messageBytes)
	assert.Error(t, err, "Should fail with non-existent image after retries")

	logger.Info("Processor retry logic test passed")
}

// TestConcurrentImageProcessing tests concurrent image processing
func TestConcurrentImageProcessing(t *testing.T) {
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
		"processed",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure buckets exist
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create original bucket")
	err = minioClient.EnsureBucketExists(ctx, "processed")
	require.NoError(t, err, "Failed to create processed bucket")

	tempDir := "/tmp/processor-integration-tests"
	exifClient := exiftool.NewClient("exiftool", 30*time.Second, logger)
	imageProcessor := service.NewImageProcessor(minioClient, exifClient, tempDir, logger)

	const numImages = 10
	const numWorkers = 3

	// Create and upload test images
	imageNames := make([]string, numImages)
	for i := 0; i < numImages; i++ {
		testImagePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-test-%d.jpg", i))
		createTestImage(t, testImagePath)
		defer os.Remove(testImagePath)

		imageFile, err := os.Open(testImagePath)
		require.NoError(t, err, "Failed to open test image")

		stat, err := imageFile.Stat()
		require.NoError(t, err, "Failed to stat test image")

		objectName := fmt.Sprintf("concurrent-%d-%s.jpg", i, time.Now().Format("20060102150405"))
		imageNames[i] = objectName

		err = minioClient.UploadFile(ctx, "original", objectName, imageFile, stat.Size())
		imageFile.Close()
		require.NoError(t, err, "Failed to upload test image")
	}

	// Process images concurrently
	var wg sync.WaitGroup
	results := make(chan error, numImages)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := workerID; j < numImages; j += numWorkers {
				metadata := model.Metadata{
					Title:       fmt.Sprintf("Concurrent Image %d", j),
					Description: fmt.Sprintf("Processed by worker %d", workerID),
					Keywords:    []string{"concurrent", "test", fmt.Sprintf("worker-%d", workerID)},
				}

				_, err := imageProcessor.ProcessImage(ctx, "original", imageNames[j], metadata, fmt.Sprintf("concurrent-%d-%d", workerID, j))
				results <- err
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Check results
	successCount := 0
	errorCount := 0
	for err := range results {
		if err == nil {
			successCount++
		} else {
			errorCount++
			t.Logf("Processing error: %v", err)
		}
	}

	assert.Equal(t, numImages, successCount, "All images should be processed successfully")
	assert.Equal(t, 0, errorCount, "Should have no processing errors")

	logger.Info(fmt.Sprintf("Concurrent image processing test passed: %d/%d images processed", successCount, numImages))
}

// TestGracefulShutdown tests graceful shutdown of processor components
func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create consumer
	consumer, err := rabbitmq.NewConsumer(
		testRabbitMQURL,
		"test-processor-shutdown-"+time.Now().Format("20060102150405"),
		5,
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create consumer")

	// Create publisher
	publisher, err := rabbitmq.NewPublisher(
		testRabbitMQURL,
		"test-processor-shutdown-output-"+time.Now().Format("20060102150405"),
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

// TestEndToEndProcessing tests complete processing workflow
func TestEndToEndProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	queueName := "test-processor-e2e-" + time.Now().Format("20060102150405")

	// Create MinIO client
	minioClient, err := minio.NewClient(
		testMinIOEndpoint,
		testMinIOAccessKey,
		testMinIOSecretKey,
		false,
		"original",
		"processed",
		logger,
		retryAttempts,
		retryDelay,
	)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure buckets exist
	err = minioClient.EnsureBucketExists(ctx, "original")
	require.NoError(t, err, "Failed to create original bucket")
	err = minioClient.EnsureBucketExists(ctx, "processed")
	require.NoError(t, err, "Failed to create processed bucket")

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
		"test-processor-e2e-output-"+time.Now().Format("20060102150405"),
		retryAttempts,
		retryDelay,
		logger,
	)
	require.NoError(t, err, "Failed to create publisher")
	defer publisher.Close()

	// Create processor components
	tempDir := "/tmp/processor-integration-tests"
	exifClient := exiftool.NewClient("exiftool", 30*time.Second, logger)
	imageProcessor := service.NewImageProcessor(minioClient, exifClient, tempDir, logger)
	messageProcessor := service.NewMessageProcessor(imageProcessor, publisher, logger, 3, 1*time.Second)

	// Create and upload test image
	testImagePath := filepath.Join(tempDir, "e2e-test.jpg")
	createTestImage(t, testImagePath)
	defer os.Remove(testImagePath)

	imageFile, err := os.Open(testImagePath)
	require.NoError(t, err, "Failed to open test image")

	stat, err := imageFile.Stat()
	require.NoError(t, err, "Failed to stat test image")

	objectName := "e2e-test-" + time.Now().Format("20060102150405") + ".jpg"
	err = minioClient.UploadFile(ctx, "original", objectName, imageFile, stat.Size())
	imageFile.Close()
	require.NoError(t, err, "Failed to upload test image")

	// Publish message to queue
	message := model.AnalysisMessage{
		BucketName: "original",
		ObjectName: objectName,
		Metadata: model.Metadata{
			Title:       "End-to-End Test Image",
			Description: "This is an end-to-end test",
			Keywords:    []string{"e2e", "test", "processor"},
		},
		TraceID: "e2e-trace-id",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err, "Failed to marshal message")

	// Manually publish to input queue
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
		err := messageProcessor.Process(ctx, msg)
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
	case <-time.After(15 * time.Second):
		t.Error("Message was not processed within timeout")
	}

	logger.Info("End-to-end processing test passed")
}
