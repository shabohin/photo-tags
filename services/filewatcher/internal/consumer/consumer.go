package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/config"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/statistics"
)

// Consumer handles consuming messages from RabbitMQ
type Consumer struct {
	cfg      *config.Config
	logger   *logging.Logger
	minio    storage.MinIOInterface
	rabbitmq messaging.RabbitMQInterface
	stats    *statistics.Statistics
}

// NewConsumer creates a new Consumer instance
func NewConsumer(
	cfg *config.Config,
	logger *logging.Logger,
	minio storage.MinIOInterface,
	rabbitmq messaging.RabbitMQInterface,
	stats *statistics.Statistics,
) *Consumer {
	return &Consumer{
		cfg:      cfg,
		logger:   logger,
		minio:    minio,
		rabbitmq: rabbitmq,
		stats:    stats,
	}
}

// Start starts consuming messages from RabbitMQ
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("Starting consumer", map[string]interface{}{
		"queue": messaging.QueueImageProcessed,
	})

	// Consume messages from image_processed queue
	return c.rabbitmq.ConsumeMessages(messaging.QueueImageProcessed, func(body []byte) error {
		return c.handleMessage(ctx, body)
	})
}

// handleMessage handles a single message from RabbitMQ
func (c *Consumer) handleMessage(ctx context.Context, body []byte) error {
	var msg models.ImageProcessed
	if err := json.Unmarshal(body, &msg); err != nil {
		c.logger.Error("Failed to unmarshal message", err)
		return err
	}

	c.logger.Info("Received processed image", map[string]interface{}{
		"trace_id": msg.TraceID,
		"status":   msg.Status,
	})

	c.stats.IncrementReceived()

	// Only process images from filewatcher
	if msg.GroupID != "filewatcher" {
		c.logger.Info("Skipping message from different group", map[string]interface{}{
			"trace_id": msg.TraceID,
			"group_id": msg.GroupID,
		})
		return nil
	}

	// Check status
	if msg.Status != "success" {
		c.logger.Error("Image processing failed", fmt.Errorf("%s", msg.Error))
		c.stats.AddError(fmt.Sprintf("Processing failed: %s", msg.Error), msg.TraceID)
		return nil // Don't requeue, just log the error
	}

	// Download processed image from MinIO
	fileData, err := c.minio.DownloadFile(ctx, storage.BucketProcessed, msg.ProcessedPath)
	if err != nil {
		c.logger.Error("Failed to download from MinIO", err)
		c.stats.AddError(fmt.Sprintf("Failed to download: %v", err), msg.TraceID)
		return err
	}

	// Ensure output directory exists
	if err := os.MkdirAll(c.cfg.OutputDir, 0755); err != nil {
		c.logger.Error("Failed to create output directory", err)
		c.stats.AddError(fmt.Sprintf("Failed to create output dir: %v", err), msg.TraceID)
		return err
	}

	// Save to output directory
	outputPath := filepath.Join(c.cfg.OutputDir, msg.OriginalFilename)
	if err := os.WriteFile(outputPath, fileData, 0644); err != nil {
		c.logger.Error("Failed to write file", err)
		c.stats.AddError(fmt.Sprintf("Failed to write file: %v", err), msg.TraceID)
		return err
	}

	c.logger.Info("Saved processed image", map[string]interface{}{
		"trace_id":    msg.TraceID,
		"output_path": outputPath,
	})

	// Save metadata JSON if available
	if err := c.saveMetadata(msg, outputPath); err != nil {
		c.logger.Error("Failed to save metadata", err)
		// Don't fail the entire operation
	}

	return nil
}

// saveMetadata saves metadata to a JSON file next to the image
func (c *Consumer) saveMetadata(msg models.ImageProcessed, imagePath string) error {
	metadata := map[string]interface{}{
		"trace_id":          msg.TraceID,
		"original_filename": msg.OriginalFilename,
		"processed_path":    msg.ProcessedPath,
		"timestamp":         msg.Timestamp,
		"status":            msg.Status,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metadataPath := imagePath + ".json"
	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	c.logger.Info("Saved metadata file", map[string]interface{}{
		"trace_id":      msg.TraceID,
		"metadata_path": metadataPath,
	})

	return nil
}
