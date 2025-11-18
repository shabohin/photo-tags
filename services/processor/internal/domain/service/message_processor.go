package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/pkg/models"
)

// MessageProcessorService processes messages from RabbitMQ
type MessageProcessorService struct {
	imageProcessor ImageProcessorInterface
	publisher      PublisherInterface
	logger         *logrus.Logger
	maxRetries     int
	retryDelay     time.Duration
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(
	imageProcessor ImageProcessorInterface,
	publisher PublisherInterface,
	logger *logrus.Logger,
	maxRetries int,
	retryDelay time.Duration,
) *MessageProcessorService {
	return &MessageProcessorService{
		imageProcessor: imageProcessor,
		publisher:      publisher,
		logger:         logger,
		maxRetries:     maxRetries,
		retryDelay:     retryDelay,
	}
}

// Process handles a single message from metadata_generated queue
func (s *MessageProcessorService) Process(ctx context.Context, messageBody []byte) error {
	// Parse message
	var msg models.MetadataGenerated
	if err := json.Unmarshal(messageBody, &msg); err != nil {
		s.logger.WithError(err).Error("Failed to unmarshal message")
		// Don't retry for malformed messages
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":    msg.TraceID,
		"group_id":    msg.GroupID,
		"telegram_id": msg.TelegramID,
		"filename":    msg.OriginalFilename,
	}).Info("Processing metadata_generated message")

	// Generate processed path
	processedPath := fmt.Sprintf("processed/%s/%s", msg.TraceID, msg.OriginalFilename)

	// Process with retry logic
	var lastErr error
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		if attempt > 0 {
			s.logger.WithFields(logrus.Fields{
				"trace_id": msg.TraceID,
				"attempt":  attempt + 1,
				"max":      s.maxRetries,
			}).Info("Retrying image processing")

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.retryDelay):
			}
		}

		err := s.imageProcessor.ProcessImage(
			ctx,
			msg.OriginalPath,
			processedPath,
			msg.Metadata,
			msg.TraceID,
		)

		if err == nil {
			// Success - publish completed message
			return s.publishResult(ctx, msg, processedPath, "completed", "")
		}

		lastErr = err
		s.logger.WithFields(logrus.Fields{
			"trace_id": msg.TraceID,
			"attempt":  attempt + 1,
			"error":    err.Error(),
		}).Warn("Image processing attempt failed")
	}

	// All retries exhausted - publish failed message
	s.logger.WithFields(logrus.Fields{
		"trace_id": msg.TraceID,
		"error":    lastErr.Error(),
	}).Error("Image processing failed after all retries")

	return s.publishResult(ctx, msg, "", "failed", lastErr.Error())
}

// publishResult sends result to image_processed queue
func (s *MessageProcessorService) publishResult(
	ctx context.Context,
	originalMsg models.MetadataGenerated,
	processedPath string,
	status string,
	errorMsg string,
) error {
	result := models.ImageProcessed{
		TraceID:          originalMsg.TraceID,
		GroupID:          originalMsg.GroupID,
		TelegramID:       originalMsg.TelegramID,
		TelegramUsername: "", // Will be filled by Gateway
		OriginalFilename: originalMsg.OriginalFilename,
		ProcessedPath:    processedPath,
		Status:           status,
		Error:            errorMsg,
		Timestamp:        time.Now(),
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": originalMsg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to marshal result message")
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := s.publisher.Publish(ctx, resultBytes); err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": originalMsg.TraceID,
			"status":   status,
			"error":    err.Error(),
		}).Error("Failed to publish result message")
		return fmt.Errorf("publish failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id": originalMsg.TraceID,
		"status":   status,
	}).Info("Result published successfully")

	return nil
}
