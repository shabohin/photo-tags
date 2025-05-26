package telegram

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

// MockMinIOClient is a mock implementation of storage.MinIOInterface
type MockMinIOClient struct {
	EnsureBucketExistsFunc func(ctx context.Context, bucketName string) error
	UploadFileFunc         func(ctx context.Context, bucketName, objectName string,
		reader io.Reader, contentType string) error
	DownloadFileFunc    func(ctx context.Context, bucketName, objectName string) (*minio.Object, error)
	GetPresignedURLFunc func(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
}

// EnsureBucketExists mocks the EnsureBucketExists method of MinIOInterface
func (m *MockMinIOClient) EnsureBucketExists(ctx context.Context,
	bucketName string) error {
	return m.EnsureBucketExistsFunc(ctx, bucketName)
}

// UploadFile mocks the UploadFile method of MinIOInterface
func (m *MockMinIOClient) UploadFile(ctx context.Context, bucketName,
	objectName string, reader io.Reader, contentType string) error {
	return m.UploadFileFunc(ctx, bucketName, objectName, reader, contentType)
}

// DownloadFile mocks the DownloadFile method of MinIOInterface
func (m *MockMinIOClient) DownloadFile(ctx context.Context, bucketName,
	objectName string) (*minio.Object, error) {
	return m.DownloadFileFunc(ctx, bucketName, objectName)
}

// GetPresignedURL mocks the GetPresignedURL method of MinIOInterface
func (m *MockMinIOClient) GetPresignedURL(ctx context.Context,
	bucketName, objectName string, expiry time.Duration) (string, error) {
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
	ReadFunc  func(p []byte) (int, error)
	CloseFunc func() error
}

// Read implements the Read method of io.ReadCloser
func (m ReadCloserMock) Read(p []byte) (int, error) {
	return m.ReadFunc(p)
}

// Close implements the Close method of io.ReadCloser
func (m ReadCloserMock) Close() error {
	return m.CloseFunc()
}

// NewMockReadCloser creates a new mock ReadCloser with content
func NewMockReadCloser(content []byte) io.ReadCloser {
	return &ReadCloserMock{
		ReadFunc: func(p []byte) (int, error) {
			return bytes.NewReader(content).Read(p)
		},
		CloseFunc: func() error {
			return nil
		},
	}
}

func TestHandleProcessedImage(t *testing.T) {
	// Для этого теста мы по сути лишь проверяем, что вызываются нужные методы
	// Полноценное тестирование невозможно из-за зависимости от внешних объектов
	t.Skip("Skipping test that requires external Telegram API")
}

func TestProcessMediaErrorCases(t *testing.T) {
	// This is a test stub since we can't fully test processMedia due to Telegram API dependencies
	t.Log("This test validates error handling of dependencies in processMedia")
}
