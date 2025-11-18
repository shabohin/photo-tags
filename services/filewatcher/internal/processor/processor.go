package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/config"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/statistics"
)

// Processor handles file processing
type Processor struct {
	cfg      *config.Config
	logger   *logging.Logger
	minio    storage.MinIOInterface
	rabbitmq messaging.RabbitMQInterface
	stats    *statistics.Statistics
}

// NewProcessor creates a new Processor instance
func NewProcessor(
	cfg *config.Config,
	logger *logging.Logger,
	minio storage.MinIOInterface,
	rabbitmq messaging.RabbitMQInterface,
	stats *statistics.Statistics,
) *Processor {
	return &Processor{
		cfg:      cfg,
		logger:   logger,
		minio:    minio,
		rabbitmq: rabbitmq,
		stats:    stats,
	}
}

// ProcessFile processes a single file
func (p *Processor) ProcessFile(ctx context.Context, filePath string) error {
	traceID := uuid.New().String()

	p.logger.Info("Processing file", map[string]interface{}{
		"trace_id": traceID,
		"file":     filePath,
	})

	// Validate file
	if err := p.validateFile(filePath); err != nil {
		p.stats.AddError(fmt.Sprintf("Validation failed: %v", err), traceID)
		p.stats.IncrementFailed()
		return fmt.Errorf("validation failed: %w", err)
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		p.stats.AddError(fmt.Sprintf("Failed to read file: %v", err), traceID)
		p.stats.IncrementFailed()
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Upload to MinIO
	filename := filepath.Base(filePath)
	objectName := fmt.Sprintf("filewatcher/%s/%s", time.Now().Format("2006-01-02"), filename)

	if err := p.minio.UploadFile(ctx, storage.BucketOriginal, objectName, fileData); err != nil {
		p.stats.AddError(fmt.Sprintf("Failed to upload to MinIO: %v", err), traceID)
		p.stats.IncrementFailed()
		return fmt.Errorf("failed to upload to MinIO: %w", err)
	}

	p.logger.Info("Uploaded to MinIO", map[string]interface{}{
		"trace_id":    traceID,
		"bucket":      storage.BucketOriginal,
		"object_name": objectName,
	})

	// Create ImageUpload message
	message := models.ImageUpload{
		Timestamp:        time.Now(),
		TraceID:          traceID,
		GroupID:          "filewatcher",
		TelegramUsername: "filewatcher",
		OriginalFilename: filename,
		OriginalPath:     objectName,
		TelegramID:       0, // Not from Telegram
	}

	// Publish to RabbitMQ
	if err := p.rabbitmq.PublishMessage(messaging.QueueImageUpload, message); err != nil {
		p.stats.AddError(fmt.Sprintf("Failed to publish to RabbitMQ: %v", err), traceID)
		p.stats.IncrementFailed()
		return fmt.Errorf("failed to publish to RabbitMQ: %w", err)
	}

	p.logger.Info("Published to RabbitMQ", map[string]interface{}{
		"trace_id": traceID,
		"queue":    messaging.QueueImageUpload,
	})

	// Mark as processed
	if err := p.markAsProcessed(filePath); err != nil {
		p.logger.Error("Failed to mark file as processed", err)
		// Don't fail the entire operation if we can't move the file
	}

	p.stats.IncrementProcessed()
	p.stats.IncrementSuccessful()

	return nil
}

// ProcessBatch processes all files in the input directory
func (p *Processor) ProcessBatch(ctx context.Context) error {
	p.logger.Info("Starting batch processing", map[string]interface{}{
		"input_dir": p.cfg.InputDir,
	})

	files, err := os.ReadDir(p.cfg.InputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %w", err)
	}

	processedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(p.cfg.InputDir, file.Name())
		if !p.isValidExtension(file.Name()) {
			continue
		}

		if err := p.ProcessFile(ctx, filePath); err != nil {
			p.logger.Error("Failed to process file", err)
			continue
		}
		processedCount++
	}

	p.logger.Info("Batch processing completed", map[string]interface{}{
		"files_processed": processedCount,
	})

	return nil
}

// validateFile validates the file format and size
func (p *Processor) validateFile(filePath string) error {
	// Check extension
	if !p.isValidExtension(filePath) {
		return fmt.Errorf("invalid file extension")
	}

	// Check file size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	maxSize := p.cfg.MaxFileSizeMB * 1024 * 1024
	if info.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max: %d bytes)", info.Size(), maxSize)
	}

	return nil
}

// isValidExtension checks if the file has a valid extension
func (p *Processor) isValidExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range p.cfg.AllowedExtensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}
	return false
}

// markAsProcessed moves the file to the processed directory or deletes it
func (p *Processor) markAsProcessed(filePath string) error {
	// Create processed directory if it doesn't exist
	if err := os.MkdirAll(p.cfg.ProcessedDir, 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Move file to processed directory
	filename := filepath.Base(filePath)
	processedPath := filepath.Join(p.cfg.ProcessedDir, filename)

	// If file already exists in processed dir, append timestamp
	if _, err := os.Stat(processedPath); err == nil {
		timestamp := time.Now().Format("20060102-150405")
		ext := filepath.Ext(filename)
		nameWithoutExt := strings.TrimSuffix(filename, ext)
		processedPath = filepath.Join(p.cfg.ProcessedDir, fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext))
	}

	if err := os.Rename(filePath, processedPath); err != nil {
		return fmt.Errorf("failed to move file to processed directory: %w", err)
	}

	p.logger.Info("Marked file as processed", map[string]interface{}{
		"original_path":  filePath,
		"processed_path": processedPath,
	})

	return nil
}
