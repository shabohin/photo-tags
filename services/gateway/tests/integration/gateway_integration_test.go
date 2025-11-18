package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
)

const (
	testRabbitMQURL     = "amqp://testuser:testpass@localhost:5673/"
	testMinIOEndpoint   = "localhost:9002"
	testMinIOAccessKey  = "testuser"
	testMinIOSecretKey  = "testpass123"
	testBucket          = "test-bucket"
	testTimeout         = 30 * time.Second
	retryAttempts       = 5
	retryDelay          = 2 * time.Second
)

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

// TestRabbitMQConnection tests basic RabbitMQ connectivity
func TestRabbitMQConnection(t *testing.T) {
	logger := logging.NewLogger("test")

	client, err := messaging.NewRabbitMQClient(testRabbitMQURL)
	require.NoError(t, err, "Failed to create RabbitMQ client")
	defer client.Close()

	// Declare a test queue
	queueName := "test-queue-" + time.Now().Format("20060102150405")
	queue, err := client.DeclareQueue(queueName)
	require.NoError(t, err, "Failed to declare queue")
	assert.Equal(t, queueName, queue.Name)

	logger.Info("RabbitMQ connection test passed", nil)
}

// TestRabbitMQRetryLogic tests retry logic for RabbitMQ operations
func TestRabbitMQRetryLogic(t *testing.T) {
	logger := logging.NewLogger("test")

	// Test with invalid URL first (should fail after retries)
	invalidURL := "amqp://invalid:invalid@localhost:9999/"

	var client messaging.RabbitMQInterface
	err := retry(3, 1*time.Second, logger, "RabbitMQ connection", func() error {
		var retryErr error
		client, retryErr = messaging.NewRabbitMQClient(invalidURL)
		return retryErr
	})
	assert.Error(t, err, "Should fail with invalid URL")

	// Test with valid URL (should succeed)
	err = retry(retryAttempts, retryDelay, logger, "RabbitMQ connection", func() error {
		var retryErr error
		client, retryErr = messaging.NewRabbitMQClient(testRabbitMQURL)
		return retryErr
	})
	require.NoError(t, err, "Should succeed with valid URL after retries")
	if client != nil {
		defer client.Close()
	}

	logger.Info("RabbitMQ retry logic test passed", nil)
}

// TestMinIOConnection tests basic MinIO connectivity
func TestMinIOConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logging.NewLogger("test")

	client, err := storage.NewMinIOClient(testMinIOEndpoint, testMinIOAccessKey, testMinIOSecretKey, false)
	require.NoError(t, err, "Failed to create MinIO client")

	// Ensure test bucket exists
	bucketName := "test-bucket-" + time.Now().Format("20060102150405")
	err = client.EnsureBucketExists(ctx, bucketName)
	require.NoError(t, err, "Failed to create test bucket")

	// Upload a test file
	testData := []byte("test file content")
	objectName := "test-object.txt"
	err = client.UploadFile(ctx, bucketName, objectName, strings.NewReader(string(testData)), int64(len(testData)), "text/plain")
	require.NoError(t, err, "Failed to upload test file")

	// Download the file
	reader, err := client.DownloadFile(ctx, bucketName, objectName)
	require.NoError(t, err, "Failed to download test file")
	defer reader.Close()

	downloadedData, err := io.ReadAll(reader)
	require.NoError(t, err, "Failed to read downloaded file")
	assert.Equal(t, testData, downloadedData, "Downloaded data should match uploaded data")

	// Cleanup
	err = client.DeleteFile(ctx, bucketName, objectName)
	assert.NoError(t, err, "Failed to delete test file")

	logger.Info("MinIO connection test passed", nil)
}

// TestMinIORetryLogic tests retry logic for MinIO operations
func TestMinIORetryLogic(t *testing.T) {
	logger := logging.NewLogger("test")

	// Test with invalid endpoint (should fail)
	var client storage.MinIOInterface
	err := retry(3, 1*time.Second, logger, "MinIO connection", func() error {
		var retryErr error
		client, retryErr = storage.NewMinIOClient("invalid:9999", "invalid", "invalid", false)
		return retryErr
	})
	assert.Error(t, err, "Should fail with invalid endpoint")

	// Test with valid endpoint (should succeed)
	err = retry(retryAttempts, retryDelay, logger, "MinIO connection", func() error {
		var retryErr error
		client, retryErr = storage.NewMinIOClient(testMinIOEndpoint, testMinIOAccessKey, testMinIOSecretKey, false)
		return retryErr
	})
	require.NoError(t, err, "Should succeed with valid endpoint after retries")

	logger.Info("MinIO retry logic test passed", nil)
}

// TestConcurrentRabbitMQOperations tests concurrent message publishing and consuming
func TestConcurrentRabbitMQOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logging.NewLogger("test")

	client, err := messaging.NewRabbitMQClient(testRabbitMQURL)
	require.NoError(t, err, "Failed to create RabbitMQ client")
	defer client.Close()

	queueName := "test-concurrent-queue-" + time.Now().Format("20060102150405")
	_, err = client.DeclareQueue(queueName)
	require.NoError(t, err, "Failed to declare queue")

	const numMessages = 100
	const numWorkers = 10

	// Publish messages concurrently
	var publishWg sync.WaitGroup
	publishWg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer publishWg.Done()
			for j := 0; j < numMessages/numWorkers; j++ {
				message := []byte(fmt.Sprintf("message-%d-%d", workerID, j))
				err := client.PublishMessage(ctx, queueName, message)
				if err != nil {
					t.Logf("Failed to publish message: %v", err)
				}
			}
		}(i)
	}
	publishWg.Wait()

	// Consume messages concurrently
	receivedMessages := make(chan string, numMessages)
	var consumeWg sync.WaitGroup
	consumeWg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer consumeWg.Done()
			for j := 0; j < numMessages/numWorkers; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					msg, err := client.ConsumeMessage(ctx, queueName)
					if err != nil {
						t.Logf("Worker %d failed to consume message: %v", workerID, err)
						continue
					}
					receivedMessages <- string(msg)
					// Acknowledge message by not returning error
				}
			}
		}(i)
	}

	// Wait for all consumers to finish
	consumeWg.Wait()
	close(receivedMessages)

	// Count received messages
	count := 0
	for range receivedMessages {
		count++
	}

	assert.GreaterOrEqual(t, count, numMessages-10, "Should receive most messages (allowing some loss)")
	logger.Info(fmt.Sprintf("Concurrent RabbitMQ operations test passed: received %d/%d messages", count, numMessages), nil)
}

// TestConcurrentMinIOOperations tests concurrent file uploads and downloads
func TestConcurrentMinIOOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logging.NewLogger("test")

	client, err := storage.NewMinIOClient(testMinIOEndpoint, testMinIOAccessKey, testMinIOSecretKey, false)
	require.NoError(t, err, "Failed to create MinIO client")

	bucketName := "test-concurrent-bucket-" + time.Now().Format("20060102150405")
	err = client.EnsureBucketExists(ctx, bucketName)
	require.NoError(t, err, "Failed to create test bucket")

	const numFiles = 50
	const numWorkers = 5

	// Upload files concurrently
	var uploadWg sync.WaitGroup
	uploadWg.Add(numWorkers)
	uploadErrors := make(chan error, numFiles)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer uploadWg.Done()
			for j := 0; j < numFiles/numWorkers; j++ {
				objectName := fmt.Sprintf("file-%d-%d.txt", workerID, j)
				content := fmt.Sprintf("content-%d-%d", workerID, j)
				err := client.UploadFile(ctx, bucketName, objectName, strings.NewReader(content), int64(len(content)), "text/plain")
				if err != nil {
					uploadErrors <- err
				}
			}
		}(i)
	}
	uploadWg.Wait()
	close(uploadErrors)

	errorCount := 0
	for range uploadErrors {
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "Should have no upload errors")

	// Download files concurrently
	var downloadWg sync.WaitGroup
	downloadWg.Add(numWorkers)
	downloadErrors := make(chan error, numFiles)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer downloadWg.Done()
			for j := 0; j < numFiles/numWorkers; j++ {
				objectName := fmt.Sprintf("file-%d-%d.txt", workerID, j)
				reader, err := client.DownloadFile(ctx, bucketName, objectName)
				if err != nil {
					downloadErrors <- err
					continue
				}
				reader.Close()
			}
		}(i)
	}
	downloadWg.Wait()
	close(downloadErrors)

	errorCount = 0
	for range downloadErrors {
		errorCount++
	}
	assert.Equal(t, 0, errorCount, "Should have no download errors")

	logger.Info(fmt.Sprintf("Concurrent MinIO operations test passed: uploaded and downloaded %d files", numFiles), nil)
}

// TestGracefulShutdown tests graceful shutdown of connections
func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	logger := logging.NewLogger("test")

	// Create RabbitMQ client
	rabbitClient, err := messaging.NewRabbitMQClient(testRabbitMQURL)
	require.NoError(t, err, "Failed to create RabbitMQ client")

	queueName := "test-shutdown-queue-" + time.Now().Format("20060102150405")
	_, err = rabbitClient.DeclareQueue(queueName)
	require.NoError(t, err, "Failed to declare queue")

	// Publish a message
	err = rabbitClient.PublishMessage(ctx, queueName, []byte("test message"))
	require.NoError(t, err, "Failed to publish message")

	// Close gracefully
	err = rabbitClient.Close()
	assert.NoError(t, err, "Should close RabbitMQ client gracefully")

	// Create MinIO client
	minioClient, err := storage.NewMinIOClient(testMinIOEndpoint, testMinIOAccessKey, testMinIOSecretKey, false)
	require.NoError(t, err, "Failed to create MinIO client")

	bucketName := "test-shutdown-bucket-" + time.Now().Format("20060102150405")
	err = minioClient.EnsureBucketExists(ctx, bucketName)
	require.NoError(t, err, "Failed to create test bucket")

	// MinIO client doesn't require explicit close, but we can still test it's working
	err = minioClient.UploadFile(ctx, bucketName, "test.txt", strings.NewReader("test"), 4, "text/plain")
	assert.NoError(t, err, "Should be able to use MinIO client before shutdown")

	logger.Info("Graceful shutdown test passed", nil)
}

// retry helper function
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
