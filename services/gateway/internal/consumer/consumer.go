package consumer

import (
	"context"
	"encoding/json"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/services/gateway/internal/storage"
)

// ImageProcessedConsumer handles messages from the image_processed queue
type ImageProcessedConsumer struct {
	logger       *logging.Logger
	rabbitMQ     messaging.RabbitMQInterface
	imageStorage *storage.ImageStorage
}

// NewImageProcessedConsumer creates a new ImageProcessedConsumer
func NewImageProcessedConsumer(
	logger *logging.Logger,
	rabbitMQ messaging.RabbitMQInterface,
	imageStorage *storage.ImageStorage,
) *ImageProcessedConsumer {
	return &ImageProcessedConsumer{
		logger:       logger,
		rabbitMQ:     rabbitMQ,
		imageStorage: imageStorage,
	}
}

// Start starts consuming messages from the image_processed queue
func (c *ImageProcessedConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting image_processed consumer", nil)

	messages, err := c.rabbitMQ.ConsumeMessages(messaging.QueueImageProcessed)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.logger.Info("Stopping image_processed consumer", nil)
				return
			case msg := <-messages:
				c.handleMessage(msg)
			}
		}
	}()

	return nil
}

// handleMessage processes a single message from the queue
func (c *ImageProcessedConsumer) handleMessage(msg []byte) {
	var processed models.ImageProcessed
	if err := json.Unmarshal(msg, &processed); err != nil {
		c.logger.Error("Failed to unmarshal image_processed message", err)
		return
	}

	c.logger.Info("Processing image_processed message", map[string]interface{}{
		"trace_id": processed.TraceID,
		"status":   processed.Status,
	})

	// Update image record based on status
	if processed.Status == "success" {
		// Get metadata from the message
		// Note: The ImageProcessed model doesn't include metadata, so we need to update it
		// For now, we'll mark it as completed. The metadata should be included in the message.
		// This is a simplification - in production, you might want to fetch metadata separately
		c.imageStorage.UpdateStatus(processed.TraceID, "completed")

		// Update processed path
		if record, exists := c.imageStorage.GetByTraceID(processed.TraceID); exists {
			record.ProcessedPath = processed.ProcessedPath
		}

		c.logger.Info("Image processing completed successfully", map[string]interface{}{
			"trace_id": processed.TraceID,
		})
	} else {
		// Mark as failed
		errorMsg := processed.Error
		if errorMsg == "" {
			errorMsg = "Unknown error occurred during processing"
		}

		c.imageStorage.MarkFailed(processed.TraceID, errorMsg)

		c.logger.Error("Image processing failed", nil)
	}
}
