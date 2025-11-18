package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
	"github.com/shabohin/photo-tags/services/analyzer/internal/monitoring"
	"github.com/shabohin/photo-tags/services/analyzer/internal/transport/rabbitmq"
)

type MessageProcessorService struct {
	imageAnalyzer *ImageAnalyzerService
	publisher     *rabbitmq.Publisher
	logger        *logrus.Logger
	maxRetries    int
	retryDelay    time.Duration
	metrics       *monitoring.Metrics
}

func NewMessageProcessor(
	imageAnalyzer *ImageAnalyzerService,
	publisher *rabbitmq.Publisher,
	logger *logrus.Logger,
	maxRetries int,
	retryDelay time.Duration,
) *MessageProcessorService {
	return &MessageProcessorService{
		imageAnalyzer: imageAnalyzer,
		publisher:     publisher,
		logger:        logger,
		maxRetries:    maxRetries,
		retryDelay:    retryDelay,
		metrics:       monitoring.NewMetrics(),
	}
}

func (s *MessageProcessorService) Process(ctx context.Context, message []byte) error {
	startTime := time.Now()
	s.metrics.Incr("rabbitmq.messages.consumed", []string{"queue:image_upload"})

	var uploadMsg model.ImageUploadMessage
	if err := json.Unmarshal(message, &uploadMsg); err != nil {
		s.metrics.Incr("message_processor.errors", []string{"error:unmarshal_failed"})
		s.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to unmarshal message")
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":          uploadMsg.TraceID,
		"group_id":          uploadMsg.GroupID,
		"telegram_id":       uploadMsg.TelegramID,
		"original_filename": uploadMsg.OriginalFilename,
	}).Info("Processing image upload message")
	s.metrics.Incr("image.processing.started", []string{})

	var metadata model.Metadata
	var err error

	// Try to analyze the image with retries
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip delay on first attempt
		if attempt > 0 {
			s.logger.WithFields(logrus.Fields{
				"trace_id": uploadMsg.TraceID,
				"attempt":  attempt,
			}).Info("Retrying image analysis")
			time.Sleep(s.retryDelay)
		}

		metadata, err = s.imageAnalyzer.AnalyzeImage(ctx, uploadMsg)
		if err == nil {
			break
		}

		s.logger.WithFields(logrus.Fields{
			"trace_id": uploadMsg.TraceID,
			"attempt":  attempt,
			"error":    err.Error(),
		}).Warn("Image analysis attempt failed")

		if attempt == s.maxRetries {
			s.metrics.Incr("image.processing.failed", []string{"error:max_retries"})
			s.metrics.Gauge("image.processing.retries", float64(attempt), []string{})
			s.logger.WithFields(logrus.Fields{
				"trace_id": uploadMsg.TraceID,
				"error":    err.Error(),
			}).Error("Image analysis failed after max retries")
			return fmt.Errorf("image analysis failed after %d retries: %w", s.maxRetries, err)
		}
	}

	// Record retry count if any
	if err == nil && metadata.Title != "" {
		s.metrics.Gauge("image.processing.retries", float64(attempt), []string{})
	}

	// Create metadata generated message
	generatedMsg := model.MetadataGeneratedMessage{
		TraceID:          uploadMsg.TraceID,
		GroupID:          uploadMsg.GroupID,
		TelegramID:       uploadMsg.TelegramID,
		OriginalFilename: uploadMsg.OriginalFilename,
		OriginalPath:     uploadMsg.OriginalPath,
		Metadata:         metadata,
		Timestamp:        time.Now(),
	}

	// Marshal the message to JSON
	jsonMsg, err := json.Marshal(generatedMsg)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": uploadMsg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to marshal metadata message")
		return fmt.Errorf("failed to marshal metadata message: %w", err)
	}

	// Publish the message
	if err := s.publisher.Publish(ctx, jsonMsg); err != nil {
		s.metrics.Incr("rabbitmq.messages.publish.errors", []string{"queue:metadata_generated", "error:publish_failed"})
		s.logger.WithFields(logrus.Fields{
			"trace_id": uploadMsg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to publish metadata message")
		return fmt.Errorf("failed to publish metadata message: %w", err)
	}

	// Record successful processing
	duration := time.Since(startTime).Milliseconds()
	s.metrics.Timing("image.processing.duration", duration, []string{"status:success"})
	s.metrics.Incr("image.processing.success", []string{})
	s.metrics.Incr("rabbitmq.messages.published", []string{"queue:metadata_generated"})

	s.logger.WithFields(logrus.Fields{
		"trace_id": uploadMsg.TraceID,
	}).Info("Metadata message published successfully")

	return nil
}
