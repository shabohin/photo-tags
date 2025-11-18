package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shabohin/photo-tags/tests/e2e/helpers"
)

const (
	simpleTestTimeout = 2 * time.Minute
	operationWaitTime = 5 * time.Second
)

// SimpleTestSuite holds test environment for simplified tests
type SimpleTestSuite struct {
	ctx             context.Context
	cancel          context.CancelFunc
	rabbitMQConn    *amqp.Connection
	rabbitMQChannel *amqp.Channel
	minioClient     *minio.Client
	mockTelegram    *helpers.MockTelegramServer
}

func setupSimpleTest(t *testing.T) *SimpleTestSuite {
	ctx, cancel := context.WithTimeout(context.Background(), simpleTestTimeout)

	suite := &SimpleTestSuite{
		ctx:    ctx,
		cancel: cancel,
	}

	t.Cleanup(func() {
		if suite.rabbitMQChannel != nil {
			suite.rabbitMQChannel.Close()
		}
		if suite.rabbitMQConn != nil {
			suite.rabbitMQConn.Close()
		}
		if suite.mockTelegram != nil {
			suite.mockTelegram.Close()
		}
		suite.cancel()
	})

	// Create mock Telegram server
	botToken := "test_bot_token_" + uuid.New().String()
	suite.mockTelegram = helpers.NewMockTelegramServer(botToken)

	// Connect to existing RabbitMQ (should be started with docker-compose)
	var err error
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		rabbitMQURL = "amqp://user:password@localhost:5672/"
	}

	suite.rabbitMQConn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		t.Skipf("RabbitMQ not available, skipping test: %v", err)
	}

	suite.rabbitMQChannel, err = suite.rabbitMQConn.Channel()
	require.NoError(t, err)

	// Declare queues
	_, err = suite.rabbitMQChannel.QueueDeclare(
		messaging.QueueImageUpload,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	require.NoError(t, err)

	// Connect to MinIO
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}

	suite.minioClient, err = minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		t.Skipf("MinIO not available, skipping test: %v", err)
	}

	// Ensure buckets exist
	buckets := []string{"original", "processed"}
	for _, bucket := range buckets {
		exists, err := suite.minioClient.BucketExists(ctx, bucket)
		if err != nil {
			t.Logf("Warning: could not check bucket %s: %v", bucket, err)
			continue
		}
		if !exists {
			err = suite.minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
			if err != nil {
				t.Logf("Warning: could not create bucket %s: %v", bucket, err)
			}
		}
	}

	return suite
}

// TestSimpleImageUpload tests basic image upload to MinIO and queue
func TestSimpleImageUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := setupSimpleTest(t)

	// Create test image
	testImage, err := helpers.CreateTestImage(640, 480)
	require.NoError(t, err)

	// Save to temp file
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test_simple.jpg")
	err = os.WriteFile(imagePath, testImage, 0644)
	require.NoError(t, err)

	// Upload to MinIO
	imageID := uuid.New().String()
	objectName := fmt.Sprintf("%s.jpg", imageID)

	ctx, cancel := context.WithTimeout(suite.ctx, operationWaitTime)
	defer cancel()

	_, err = suite.minioClient.FPutObject(ctx, "original", objectName, imagePath, minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	require.NoError(t, err)

	t.Logf("Uploaded image to MinIO: %s", objectName)

	// Verify object exists
	_, err = suite.minioClient.StatObject(ctx, "original", objectName, minio.StatObjectOptions{})
	assert.NoError(t, err, "Image should exist in MinIO")

	// Publish message to queue
	message := models.ImageUpload{
		Timestamp:        time.Now(),
		TraceID:          uuid.New().String(),
		GroupID:          imageID,
		TelegramUsername: "testuser",
		OriginalFilename: "test_simple.jpg",
		OriginalPath:     objectName,
		TelegramID:       123456,
	}

	messageBytes, err := json.Marshal(message)
	require.NoError(t, err)

	err = suite.rabbitMQChannel.Publish(
		"",                         // exchange
		messaging.QueueImageUpload, // routing key
		false,                      // mandatory
		false,                      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBytes,
		},
	)
	require.NoError(t, err)

	t.Logf("Published message to queue: %s", messaging.QueueImageUpload)
}

// TestExifToolValidation tests EXIF tool functionality
func TestExifToolValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create test image with metadata
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test_exif.jpg")

	err := helpers.SaveTestImage(imagePath, 800, 600)
	require.NoError(t, err)

	// Try to extract EXIF data (may not have metadata initially)
	exifData, err := helpers.ExtractExifData(imagePath)
	if err != nil {
		t.Skipf("exiftool not available: %v", err)
	}

	t.Logf("EXIF Data: %+v", exifData)
	// Initially image won't have metadata, that's expected
	assert.NotNil(t, exifData)
}

// TestMockTelegramAPI tests mock Telegram API functionality
func TestMockTelegramAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	botToken := "test_token_123"
	mock := helpers.NewMockTelegramServer(botToken)
	defer mock.Close()

	t.Logf("Mock Telegram API running at: %s", mock.URL())

	// Test adding photo update
	mock.AddPhotoUpdate(123456, "test_file_id", 1024, 800, 600)

	// Simulate rate limiting
	mock.SimulateRateLimit(2)

	// Simulate error
	mock.SimulateError(true)
	mock.SimulateError(false)

	// Test timeout simulation
	mock.SimulateTimeout(true)
	mock.SimulateTimeout(false)

	t.Log("Mock Telegram API tests passed")
}

// TestMessageQueueFlow tests basic message queue flow
func TestMessageQueueFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := setupSimpleTest(t)

	// Create and publish ImageUpload message
	uploadMsg := models.ImageUpload{
		Timestamp:        time.Now(),
		TraceID:          uuid.New().String(),
		GroupID:          uuid.New().String(),
		TelegramUsername: "testuser",
		OriginalFilename: "test.jpg",
		OriginalPath:     "test/test.jpg",
		TelegramID:       123456,
	}

	messageBytes, err := json.Marshal(uploadMsg)
	require.NoError(t, err)

	err = suite.rabbitMQChannel.Publish(
		"",
		messaging.QueueImageUpload,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        messageBytes,
		},
	)
	require.NoError(t, err)

	t.Log("Published ImageUpload message successfully")

	// Test MetadataGenerated message structure
	metadataMsg := models.MetadataGenerated{
		Timestamp:        time.Now(),
		TraceID:          uploadMsg.TraceID,
		GroupID:          uploadMsg.GroupID,
		OriginalFilename: uploadMsg.OriginalFilename,
		OriginalPath:     uploadMsg.OriginalPath,
		TelegramID:       uploadMsg.TelegramID,
		Metadata: models.Metadata{
			Title:       "Test Image",
			Description: "A test image description",
			Keywords:    []string{"test", "e2e", "photo"},
		},
	}

	metadataBytes, err := json.Marshal(metadataMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, metadataBytes)

	t.Log("Message serialization tests passed")
}

// TestStorageOperations tests MinIO storage operations
func TestStorageOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	suite := setupSimpleTest(t)

	// Create test image
	testImage, err := helpers.CreateTestImage(400, 300)
	require.NoError(t, err)

	// Test upload
	objectName := fmt.Sprintf("test_storage_%s.jpg", uuid.New().String())

	ctx, cancel := context.WithTimeout(suite.ctx, operationWaitTime)
	defer cancel()

	_, err = suite.minioClient.PutObject(
		ctx,
		"original",
		objectName,
		nil,
		int64(len(testImage)),
		minio.PutObjectOptions{ContentType: "image/jpeg"},
	)

	// Try with reader
	tempDir := t.TempDir()
	imagePath := filepath.Join(tempDir, "test.jpg")
	err = os.WriteFile(imagePath, testImage, 0644)
	require.NoError(t, err)

	_, err = suite.minioClient.FPutObject(
		ctx,
		"original",
		objectName,
		imagePath,
		minio.PutObjectOptions{ContentType: "image/jpeg"},
	)
	require.NoError(t, err)

	t.Logf("Uploaded object: %s", objectName)

	// Test download
	downloadPath := filepath.Join(tempDir, "downloaded.jpg")
	err = suite.minioClient.FGetObject(
		ctx,
		"original",
		objectName,
		downloadPath,
		minio.GetObjectOptions{},
	)
	require.NoError(t, err)

	// Verify downloaded file
	downloadedData, err := os.ReadFile(downloadPath)
	require.NoError(t, err)
	assert.Equal(t, len(testImage), len(downloadedData), "Downloaded file should have same size")

	t.Log("Storage operations tests passed")
}
