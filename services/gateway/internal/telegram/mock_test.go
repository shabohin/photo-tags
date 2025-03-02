package telegram

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
)

// MockHTTPClient is a mock HTTP client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do mocks the Do method of http.Client
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// Get mocks the Get method of http.Client
func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return m.Do(req)
}

// MockMinIOClient is a mock MinIO client for testing
type MockMinIOClient struct {
	EnsureBucketExistsFunc func(ctx context.Context, bucketName string) error
	UploadFileFunc         func(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) error
	DownloadFileFunc       func(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
}

// EnsureBucketExists mocks the EnsureBucketExists method of MinIOClient
func (m *MockMinIOClient) EnsureBucketExists(ctx context.Context, bucketName string) error {
	return m.EnsureBucketExistsFunc(ctx, bucketName)
}

// UploadFile mocks the UploadFile method of MinIOClient
func (m *MockMinIOClient) UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) error {
	return m.UploadFileFunc(ctx, bucketName, objectName, reader, contentType)
}

// DownloadFile mocks the DownloadFile method of MinIOClient
func (m *MockMinIOClient) DownloadFile(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error) {
	return m.DownloadFileFunc(ctx, bucketName, objectName)
}

// MockRabbitMQClient is a mock RabbitMQ client for testing
type MockRabbitMQClient struct {
	DeclareQueueFunc    func(name string) (interface{}, error)
	PublishMessageFunc  func(queueName string, message interface{}) error
	ConsumeMessagesFunc func(queueName string, handler func([]byte) error) error
}

// DeclareQueue mocks the DeclareQueue method of RabbitMQClient
func (m *MockRabbitMQClient) DeclareQueue(name string) (interface{}, error) {
	return m.DeclareQueueFunc(name)
}

// PublishMessage mocks the PublishMessage method of RabbitMQClient
func (m *MockRabbitMQClient) PublishMessage(queueName string, message interface{}) error {
	return m.PublishMessageFunc(queueName, message)
}

// ConsumeMessages mocks the ConsumeMessages method of RabbitMQClient
func (m *MockRabbitMQClient) ConsumeMessages(queueName string, handler func([]byte) error) error {
	return m.ConsumeMessagesFunc(queueName, handler)
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

// MockResponse creates a mock HTTP response
func MockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// MockServer creates a test server that returns a predefined response
func MockServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	}))
}
