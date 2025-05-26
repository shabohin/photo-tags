package service

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
	"github.com/shabohin/photo-tags/services/analyzer/internal/mocks"
)

func TestNewImageAnalyzer(t *testing.T) {
	minioClient := &mocks.MockMinioClient{}
	openRouterClient := &mocks.MockOpenRouterClient{}
	logger := logrus.New()

	analyzer := NewImageAnalyzer(minioClient, openRouterClient, logger)

	assert.NotNil(t, analyzer)
	assert.Equal(t, minioClient, analyzer.minioClient)
	assert.Equal(t, openRouterClient, analyzer.openRouterClient)
	assert.Equal(t, logger, analyzer.logger)
}

func TestAnalyzeImage_Success(t *testing.T) {
	// Create mocks
	minioClient := &mocks.MockMinioClient{}
	openRouterClient := &mocks.MockOpenRouterClient{}
	logger := logrus.New()

	// Setup mock for MinIO client
	imageBytes := []byte("test-image-data")
	minioClient.On("DownloadImage", mock.Anything, "test-image.jpg").Return(imageBytes, nil)

	// Setup mock for OpenRouter client
	expectedMetadata := model.Metadata{
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"test", "image", "analysis"},
	}
	openRouterClient.On("AnalyzeImage", mock.Anything, imageBytes, "test-trace-id").Return(expectedMetadata, nil)

	// Create service with mocks
	analyzer := NewImageAnalyzer(minioClient, openRouterClient, logger)

	// Prepare test message
	message := model.ImageUploadMessage{
		TraceID:          "test-trace-id",
		GroupID:          "test-group-id",
		TelegramID:       123456,
		OriginalFilename: "test-image.jpg",
		OriginalPath:     "test-image.jpg",
	}

	// Test AnalyzeImage function
	metadata, err := analyzer.AnalyzeImage(context.Background(), message)

	// Check results
	assert.NoError(t, err)
	assert.Equal(t, expectedMetadata, metadata)

	// Check that mocks were called with correct parameters
	minioClient.AssertExpectations(t)
	openRouterClient.AssertExpectations(t)
}

func TestAnalyzeImage_MinioError(t *testing.T) {
	// Create mocks
	minioClient := &mocks.MockMinioClient{}
	openRouterClient := &mocks.MockOpenRouterClient{}
	logger := logrus.New()

	// Setup mock for MinIO client with error
	minioClient.On("DownloadImage", mock.Anything, "test-image.jpg").Return([]byte{}, errors.New("minio error"))

	// Create service with mocks
	analyzer := NewImageAnalyzer(minioClient, openRouterClient, logger)

	// Prepare test message
	message := model.ImageUploadMessage{
		TraceID:          "test-trace-id",
		GroupID:          "test-group-id",
		TelegramID:       123456,
		OriginalFilename: "test-image.jpg",
		OriginalPath:     "test-image.jpg",
	}

	// Test AnalyzeImage function with MinIO error
	_, err := analyzer.AnalyzeImage(context.Background(), message)

	// Check results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download image")
	assert.Contains(t, err.Error(), "minio error")

	// Check that mocks were called with correct parameters
	minioClient.AssertExpectations(t)
	openRouterClient.AssertNotCalled(t, "AnalyzeImage")
}

func TestAnalyzeImage_OpenRouterError(t *testing.T) {
	// Create mocks
	minioClient := &mocks.MockMinioClient{}
	openRouterClient := &mocks.MockOpenRouterClient{}
	logger := logrus.New()

	// Setup mock for MinIO client
	imageBytes := []byte("test-image-data")
	minioClient.On("DownloadImage", mock.Anything, "test-image.jpg").Return(imageBytes, nil)

	// Setup mock for OpenRouter client with error
	openRouterClient.
		On(
			"AnalyzeImage",
			mock.Anything,
			imageBytes,
			"test-trace-id",
		).
		Return(
			model.Metadata{},
			errors.New("openrouter error"),
		)

	// Create service with mocks
	analyzer := NewImageAnalyzer(minioClient, openRouterClient, logger)

	// Prepare test message
	message := model.ImageUploadMessage{
		TraceID:          "test-trace-id",
		GroupID:          "test-group-id",
		TelegramID:       123456,
		OriginalFilename: "test-image.jpg",
		OriginalPath:     "test-image.jpg",
	}

	// Test AnalyzeImage function with OpenRouter error
	_, err := analyzer.AnalyzeImage(context.Background(), message)

	// Check results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to analyze image")
	assert.Contains(t, err.Error(), "openrouter error")

	// Check that mocks were called with correct parameters
	minioClient.AssertExpectations(t)
	openRouterClient.AssertExpectations(t)
}
