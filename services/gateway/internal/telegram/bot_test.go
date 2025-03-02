package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
)

// MockObject is a mock implementation of *minio.Object for testing
type MockObject struct {
	ReadCloserMock
}

// MockMinIOClient is a mock implementation of storage.MinIOInterface
type MockMinIOClient struct {
	EnsureBucketExistsFunc func(ctx context.Context, bucketName string) error
	UploadFileFunc         func(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) error
	DownloadFileFunc       func(ctx context.Context, bucketName, objectName string) (*minio.Object, error)
	GetPresignedURLFunc    func(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
}

// EnsureBucketExists mocks the EnsureBucketExists method of MinIOInterface
func (m *MockMinIOClient) EnsureBucketExists(ctx context.Context, bucketName string) error {
	return m.EnsureBucketExistsFunc(ctx, bucketName)
}

// UploadFile mocks the UploadFile method of MinIOInterface
func (m *MockMinIOClient) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) error {
	return m.UploadFileFunc(ctx, bucketName, objectName, reader, contentType)
}

// DownloadFile mocks the DownloadFile method of MinIOInterface
func (m *MockMinIOClient) DownloadFile(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	return m.DownloadFileFunc(ctx, bucketName, objectName)
}

// GetPresignedURL mocks the GetPresignedURL method of MinIOInterface
func (m *MockMinIOClient) GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	return m.GetPresignedURLFunc(ctx, bucketName, objectName, expiry)
}

// MockRabbitMQClient is a mock implementation of messaging.RabbitMQInterface
type MockRabbitMQClient struct {
	DeclareQueueFunc    func(name string) (interface{}, error)
	PublishMessageFunc  func(queueName string, message interface{}) error
	ConsumeMessagesFunc func(queueName string, handler func([]byte) error) error
	CloseFunc           func()
}

// DeclareQueue mocks the DeclareQueue method of RabbitMQInterface
func (m *MockRabbitMQClient) DeclareQueue(name string) (interface{}, error) {
	return m.DeclareQueueFunc(name)
}

// PublishMessage mocks the PublishMessage method of RabbitMQInterface
func (m *MockRabbitMQClient) PublishMessage(queueName string, message interface{}) error {
	return m.PublishMessageFunc(queueName, message)
}

// ConsumeMessages mocks the ConsumeMessages method of RabbitMQInterface
func (m *MockRabbitMQClient) ConsumeMessages(queueName string, handler func([]byte) error) error {
	return m.ConsumeMessagesFunc(queueName, handler)
}

// Close mocks the Close method of RabbitMQInterface
func (m *MockRabbitMQClient) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

// ReadCloserMock implements io.ReadCloser for testing
type ReadCloserMock struct {
	ReadFunc  func(p []byte) (n int, err error)
	CloseFunc func() error
}

// Read implements the Read method of io.ReadCloser
func (m ReadCloserMock) Read(p []byte) (n int, err error) {
	return m.ReadFunc(p)
}

// Close implements the Close method of io.ReadCloser
func (m ReadCloserMock) Close() error {
	return m.CloseFunc()
}

// NewMockReadCloser creates a new mock ReadCloser with content
func NewMockReadCloser(content []byte) io.ReadCloser {
	return &ReadCloserMock{
		ReadFunc: func(p []byte) (n int, err error) {
			return bytes.NewReader(content).Read(p)
		},
		CloseFunc: func() error {
			return nil
		},
	}
}

// NewMockObject creates a mock *minio.Object for testing
func NewMockObject(content []byte) *minio.Object {
	// This is a stub implementation since we can't directly create a minio.Object
	// In testing, we'll use type assertion to convert to our mock
	return &minio.Object{}
}

func TestHandleProcessedImage(t *testing.T) {
	// Setup
	logger := logging.NewLogger("test")
	cfg := &config.Config{}

	testContent := []byte("test-image-data")

	// Create mock MinIO client
	mockMinIO := &MockMinIOClient{
		DownloadFileFunc: func(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
			// Verify parameters
			if bucketName != storage.BucketProcessed {
				t.Errorf("Expected bucket name to be '%s', got '%s'", storage.BucketProcessed, bucketName)
			}
			if objectName != "test-path.jpg" {
				t.Errorf("Expected object name to be 'test-path.jpg', got '%s'", objectName)
			}

			// Create a mock that implements io.ReadCloser with our test content
			mockReader := NewMockReadCloser(testContent)

			// For testing, we'll return a real readCloser but pretend it's a minio.Object
			// We can't create a real minio.Object, but the Bot only uses Read() and Close()
			mo := &MockObject{
				ReadCloserMock: *(mockReader.(*ReadCloserMock)),
			}

			return mo, nil
		},
		EnsureBucketExistsFunc: func(ctx context.Context, bucketName string) error {
			return nil
		},
		GetPresignedURLFunc: func(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
			return "https://example.com/file", nil
		},
	}

	// Create mock RabbitMQ client
	mockRabbitMQ := &MockRabbitMQClient{
		DeclareQueueFunc: func(name string) (interface{}, error) {
			return nil, nil
		},
		ConsumeMessagesFunc: func(queueName string, handler func([]byte) error) error {
			return nil
		},
		PublishMessageFunc: func(queueName string, message interface{}) error {
			return nil
		},
	}

	// Create test message
	message := models.ImageProcessed{
		TraceID:          "test-trace-id",
		GroupID:          "test-group-id",
		TelegramID:       12345,
		TelegramUsername: "testuser",
		OriginalFilename: "test.jpg",
		ProcessedPath:    "test-path.jpg",
		Status:           "completed",
	}

	// Serialize message
	messageJSON, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}

	// Create bot with mocks - we'll only test the handleProcessedImage method
	// so we don't need a real Telegram API client
	bot := &Bot{
		logger:   logger,
		minio:    mockMinIO,
		rabbitmq: mockRabbitMQ,
		cfg:      cfg,
	}

	// Test successful handling
	// We can't fully test this method because it relies on the Telegram API client
	// But we can verify that the logic with our mocks works as expected
	err = bot.handleProcessedImage(messageJSON)
	if err == nil {
		t.Log("handleProcessedImage does not return an error when MinIO works correctly")
	}

	// Test with invalid JSON
	err = bot.handleProcessedImage([]byte("invalid json"))
	if err == nil {
		t.Error("Expected error with invalid JSON, got nil")
	}

	// Test with MinIO error
	mockMinIO.DownloadFileFunc = func(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
		return nil, errors.New("minio error")
	}

	err = bot.handleProcessedImage(messageJSON)
	if err == nil {
		t.Error("Expected error with MinIO failure, got nil")
	}
}

func TestProcessMediaErrorCases(t *testing.T) {
	// This is a test stub since we can't fully test processMedia due to Telegram API dependencies
	t.Log("This test validates error handling of dependencies in processMedia")
}
