package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/pkg/models"
)

// Mock publisher
type mockPublisher struct {
	publishFunc func(ctx context.Context, message []byte) error
	messages    [][]byte
}

func (m *mockPublisher) Publish(ctx context.Context, message []byte) error {
	m.messages = append(m.messages, message)
	if m.publishFunc != nil {
		return m.publishFunc(ctx, message)
	}
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

// Mock image processor
type mockImageProcessor struct {
	processFunc func(ctx context.Context, originalPath string, processedPath string, metadata models.Metadata, traceID string) error
	callCount   int
}

func (m *mockImageProcessor) ProcessImage(ctx context.Context, originalPath string, processedPath string, metadata models.Metadata, traceID string) error {
	m.callCount++
	if m.processFunc != nil {
		return m.processFunc(ctx, originalPath, processedPath, metadata, traceID)
	}
	return nil
}

func TestNewMessageProcessor(t *testing.T) {
	logger := logrus.New()
	imageProcessor := &mockImageProcessor{}
	publisher := &mockPublisher{}

	processor := NewMessageProcessor(imageProcessor, publisher, logger, 3, 5*time.Second)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	if processor.maxRetries != 3 {
		t.Errorf("Expected maxRetries 3, got %d", processor.maxRetries)
	}
}

func TestProcess_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	imageProcessor := &mockImageProcessor{}
	publisher := &mockPublisher{}

	processor := NewMessageProcessor(imageProcessor, publisher, logger, 3, 100*time.Millisecond)

	// Create test message
	msg := models.MetadataGenerated{
		TraceID:          "test-trace-id",
		GroupID:          "test-group-id",
		TelegramID:       123456789,
		OriginalFilename: "test.jpg",
		OriginalPath:     "original/test-trace-id/test.jpg",
		Metadata: models.Metadata{
			Title:       "Test Title",
			Description: "Test Description",
			Keywords:    []string{"test", "image"},
		},
		Timestamp: time.Now(),
	}

	msgBytes, _ := json.Marshal(msg)

	ctx := context.Background()
	err := processor.Process(ctx, msgBytes)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	// Check that result was published
	if len(publisher.messages) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(publisher.messages))
	}

	// Parse published message
	var result models.ImageProcessed
	json.Unmarshal(publisher.messages[0], &result)

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}

	if result.TraceID != "test-trace-id" {
		t.Errorf("Expected trace_id 'test-trace-id', got %s", result.TraceID)
	}

	// Check that image processor was called once
	if imageProcessor.callCount != 1 {
		t.Errorf("Expected imageProcessor to be called once, got %d", imageProcessor.callCount)
	}
}

func TestProcess_RetryAndFail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Image processor always fails
	imageProcessor := &mockImageProcessor{
		processFunc: func(ctx context.Context, originalPath string, processedPath string, metadata models.Metadata, traceID string) error {
			return errors.New("processing failed")
		},
	}

	publisher := &mockPublisher{}

	maxRetries := 3
	processor := NewMessageProcessor(imageProcessor, publisher, logger, maxRetries, 10*time.Millisecond)

	msg := models.MetadataGenerated{
		TraceID:          "test-trace-id",
		OriginalFilename: "test.jpg",
		OriginalPath:     "original/test.jpg",
		Metadata:         models.Metadata{Title: "Test"},
		Timestamp:        time.Now(),
	}

	msgBytes, _ := json.Marshal(msg)

	ctx := context.Background()
	err := processor.Process(ctx, msgBytes)

	if err != nil {
		t.Errorf("Process should not return error on retry exhaustion: %v", err)
	}

	// Check that image processor was called maxRetries times
	if imageProcessor.callCount != maxRetries {
		t.Errorf("Expected imageProcessor to be called %d times, got %d", maxRetries, imageProcessor.callCount)
	}

	// Check that failed result was published
	if len(publisher.messages) != 1 {
		t.Errorf("Expected 1 published message, got %d", len(publisher.messages))
	}

	var result models.ImageProcessed
	json.Unmarshal(publisher.messages[0], &result)

	if result.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", result.Status)
	}

	if result.Error == "" {
		t.Error("Expected error message in failed result")
	}
}

func TestProcess_InvalidJSON(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	imageProcessor := &mockImageProcessor{}
	publisher := &mockPublisher{}

	processor := NewMessageProcessor(imageProcessor, publisher, logger, 3, 100*time.Millisecond)

	// Invalid JSON
	invalidJSON := []byte("{invalid json")

	ctx := context.Background()
	err := processor.Process(ctx, invalidJSON)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Should not publish anything
	if len(publisher.messages) != 0 {
		t.Errorf("Expected 0 published messages for invalid JSON, got %d", len(publisher.messages))
	}

	// Should not call image processor
	if imageProcessor.callCount != 0 {
		t.Errorf("Expected imageProcessor not to be called for invalid JSON, got %d calls", imageProcessor.callCount)
	}
}

func TestProcess_PublishFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	imageProcessor := &mockImageProcessor{}

	// Publisher fails
	publisher := &mockPublisher{
		publishFunc: func(ctx context.Context, message []byte) error {
			return errors.New("publish failed")
		},
	}

	processor := NewMessageProcessor(imageProcessor, publisher, logger, 3, 100*time.Millisecond)

	msg := models.MetadataGenerated{
		TraceID:          "test-trace-id",
		OriginalFilename: "test.jpg",
		OriginalPath:     "original/test.jpg",
		Metadata:         models.Metadata{Title: "Test"},
		Timestamp:        time.Now(),
	}

	msgBytes, _ := json.Marshal(msg)

	ctx := context.Background()
	err := processor.Process(ctx, msgBytes)

	if err == nil {
		t.Error("Expected error when publish fails")
	}

	// Should still call image processor
	if imageProcessor.callCount != 1 {
		t.Errorf("Expected imageProcessor to be called once, got %d", imageProcessor.callCount)
	}
}

func TestProcess_ContextCancellation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Image processor always fails to force retry
	imageProcessor := &mockImageProcessor{
		processFunc: func(ctx context.Context, originalPath string, processedPath string, metadata models.Metadata, traceID string) error {
			return errors.New("processing failed")
		},
	}

	publisher := &mockPublisher{}

	processor := NewMessageProcessor(imageProcessor, publisher, logger, 10, 100*time.Millisecond)

	msg := models.MetadataGenerated{
		TraceID:          "test-trace-id",
		OriginalFilename: "test.jpg",
		OriginalPath:     "original/test.jpg",
		Metadata:         models.Metadata{Title: "Test"},
		Timestamp:        time.Now(),
	}

	msgBytes, _ := json.Marshal(msg)

	// Cancel context after short delay
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := processor.Process(ctx, msgBytes)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	// Should have attempted processing at least once
	if imageProcessor.callCount == 0 {
		t.Error("Expected at least one processing attempt")
	}
}
