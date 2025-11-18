package selector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/shabohin/photo-tags/services/analyzer/internal/api/openrouter"
	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

// MockOpenRouterClient is a mock implementation of OpenRouterClient
type MockOpenRouterClient struct {
	mock.Mock
}

func (m *MockOpenRouterClient) AnalyzeImage(
	ctx context.Context, imageBytes []byte, traceID string,
) (model.Metadata, error) {
	args := m.Called(ctx, imageBytes, traceID)
	return args.Get(0).(model.Metadata), args.Error(1)
}

func (m *MockOpenRouterClient) GetAvailableModels(ctx context.Context) ([]openrouter.Model, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]openrouter.Model), args.Error(1)
}

func (m *MockOpenRouterClient) SelectBestFreeVisionModel(models []openrouter.Model) (*openrouter.Model, error) {
	args := m.Called(models)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*openrouter.Model), args.Error(1)
}

func TestNewModelSelector(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	checkInterval := 1 * time.Hour
	fallbackModel := "test-model"

	selector := NewModelSelector(mockClient, logger, checkInterval, fallbackModel)

	assert.NotNil(t, selector)
	assert.Equal(t, checkInterval, selector.checkInterval)
	assert.Equal(t, fallbackModel, selector.fallbackModel)
}

func TestModelSelector_UpdateModels_Success(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	testModels := []openrouter.Model{
		{
			ID:         "test-model-1",
			Name:       "Test Model 1",
			ContextLen: 8192,
		},
	}

	selectedModel := &openrouter.Model{
		ID:         "test-model-1",
		Name:       "Test Model 1",
		ContextLen: 8192,
	}

	mockClient.On("GetAvailableModels", mock.Anything).Return(testModels, nil)
	mockClient.On("SelectBestFreeVisionModel", testModels).Return(selectedModel, nil)

	selector := NewModelSelector(mockClient, logger, 1*time.Hour, "fallback-model")
	selector.updateModels(context.Background())

	currentModel, err := selector.GetCurrentModel()
	assert.NoError(t, err)
	assert.Equal(t, "test-model-1", currentModel)

	mockClient.AssertExpectations(t)
}

func TestModelSelector_UpdateModels_FetchError(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockClient.On("GetAvailableModels", mock.Anything).Return(nil, errors.New("fetch error"))

	selector := NewModelSelector(mockClient, logger, 1*time.Hour, "fallback-model")
	selector.updateModels(context.Background())

	// Should use fallback model when fetch fails
	currentModel, err := selector.GetCurrentModel()
	assert.NoError(t, err)
	assert.Equal(t, "fallback-model", currentModel)

	mockClient.AssertExpectations(t)
}

func TestModelSelector_UpdateModels_SelectionError(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testModels := []openrouter.Model{
		{
			ID:         "test-model-1",
			Name:       "Test Model 1",
			ContextLen: 8192,
		},
	}

	mockClient.On("GetAvailableModels", mock.Anything).Return(testModels, nil)
	mockClient.On("SelectBestFreeVisionModel", testModels).Return(nil, errors.New("selection error"))

	selector := NewModelSelector(mockClient, logger, 1*time.Hour, "fallback-model")
	selector.updateModels(context.Background())

	// Should use fallback model when selection fails
	currentModel, err := selector.GetCurrentModel()
	assert.NoError(t, err)
	assert.Equal(t, "fallback-model", currentModel)

	mockClient.AssertExpectations(t)
}

func TestModelSelector_GetCurrentModel_NoModel(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()

	selector := NewModelSelector(mockClient, logger, 1*time.Hour, "fallback-model")

	_, err := selector.GetCurrentModel()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no model selected yet")
}

func TestModelSelector_StartStop(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testModels := []openrouter.Model{
		{
			ID:         "test-model-1",
			Name:       "Test Model 1",
			ContextLen: 8192,
		},
	}

	selectedModel := &openrouter.Model{
		ID:         "test-model-1",
		Name:       "Test Model 1",
		ContextLen: 8192,
	}

	mockClient.On("GetAvailableModels", mock.Anything).Return(testModels, nil).Maybe()
	mockClient.On("SelectBestFreeVisionModel", testModels).Return(selectedModel, nil).Maybe()

	selector := NewModelSelector(mockClient, logger, 100*time.Millisecond, "fallback-model")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the selector
	selector.Start(ctx)

	// Wait a bit to ensure it starts
	time.Sleep(50 * time.Millisecond)

	// Get the current model
	currentModel, err := selector.GetCurrentModel()
	assert.NoError(t, err)
	assert.Equal(t, "test-model-1", currentModel)

	// Stop the selector
	selector.Stop()

	// Verify it stopped gracefully
	mockClient.AssertExpectations(t)
}

func TestModelSelector_ThreadSafety(t *testing.T) {
	mockClient := new(MockOpenRouterClient)
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	testModels := []openrouter.Model{
		{
			ID:         "test-model-1",
			Name:       "Test Model 1",
			ContextLen: 8192,
		},
	}

	selectedModel := &openrouter.Model{
		ID:         "test-model-1",
		Name:       "Test Model 1",
		ContextLen: 8192,
	}

	mockClient.On("GetAvailableModels", mock.Anything).Return(testModels, nil)
	mockClient.On("SelectBestFreeVisionModel", testModels).Return(selectedModel, nil)

	selector := NewModelSelector(mockClient, logger, 1*time.Hour, "fallback-model")
	selector.updateModels(context.Background())

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			currentModel, err := selector.GetCurrentModel()
			assert.NoError(t, err)
			assert.Equal(t, "test-model-1", currentModel)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	mockClient.AssertExpectations(t)
}
