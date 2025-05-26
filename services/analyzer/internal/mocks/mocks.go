package mocks

import (
	"context"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
	"github.com/stretchr/testify/mock"
)

// MockMinioClient mock for MinIO client
type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) DownloadImage(ctx context.Context, path string) ([]byte, error) {
	args := m.Called(ctx, path)
	return args.Get(0).([]byte), args.Error(1)
}

// MockOpenRouterClient mock for OpenRouter client
type MockOpenRouterClient struct {
	mock.Mock
}

func (m *MockOpenRouterClient) AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error) {
	args := m.Called(ctx, imageBytes, traceID)
	return args.Get(0).(model.Metadata), args.Error(1)
}

// MockPublisher mock for RabbitMQ Publisher
type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) Publish(ctx context.Context, message []byte) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockImageAnalyzer mock for ImageAnalyzerService
type MockImageAnalyzer struct {
	mock.Mock
}

func (m *MockImageAnalyzer) AnalyzeImage(ctx context.Context, msg model.ImageUploadMessage) (model.Metadata, error) {
	args := m.Called(ctx, msg)
	return args.Get(0).(model.Metadata), args.Error(1)
}
