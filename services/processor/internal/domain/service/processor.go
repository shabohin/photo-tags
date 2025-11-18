package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/services/processor/internal/exiftool"
)

// ImageProcessorService handles image processing workflow
type ImageProcessorService struct {
	minioClient MinioClientInterface
	exifTool    ExifToolInterface
	tempDir     string
	logger      *logrus.Logger
}

// NewImageProcessor creates a new image processor service
func NewImageProcessor(
	minioClient MinioClientInterface,
	exifTool ExifToolInterface,
	tempDir string,
	logger *logrus.Logger,
) *ImageProcessorService {
	return &ImageProcessorService{
		minioClient: minioClient,
		exifTool:    exifTool,
		tempDir:     tempDir,
		logger:      logger,
	}
}

// ProcessImage processes an image: downloads, writes metadata, and uploads
func (s *ImageProcessorService) ProcessImage(
	ctx context.Context,
	originalPath string,
	processedPath string,
	metadata models.Metadata,
	traceID string,
) error {
	s.logger.WithFields(logrus.Fields{
		"trace_id":       traceID,
		"original_path":  originalPath,
		"processed_path": processedPath,
	}).Info("Starting image processing")

	// Step 1: Download image from MinIO (original bucket)
	imageBytes, err := s.minioClient.DownloadImage(ctx, originalPath)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to download image from MinIO")
		return fmt.Errorf("download failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":      traceID,
		"image_size_kb": len(imageBytes) / 1024,
	}).Debug("Image downloaded successfully")

	// Step 2: Save to temporary directory
	tempFilePath, err := s.saveTempFile(imageBytes, traceID)
	if err != nil {
		return fmt.Errorf("failed to save temp file: %w", err)
	}
	defer s.cleanupTempFile(tempFilePath, traceID)

	// Step 3: Convert metadata format
	exifMetadata := exiftool.Metadata{
		Title:       metadata.Title,
		Description: metadata.Description,
		Keywords:    metadata.Keywords,
	}

	// Step 4: Write metadata with ExifTool
	if err := s.exifTool.WriteMetadata(ctx, tempFilePath, exifMetadata, traceID); err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to write metadata")
		return fmt.Errorf("metadata write failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":       traceID,
		"title":          metadata.Title,
		"keywords_count": len(metadata.Keywords),
	}).Info("Metadata written successfully")

	// Step 5: Verify metadata (optional, log warning if failed)
	verified, err := s.exifTool.VerifyMetadata(ctx, tempFilePath, traceID)
	if err != nil || !verified {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"verified": verified,
			"error":    err,
		}).Warn("Metadata verification failed, proceeding anyway")
	}

	// Step 6: Read processed image
	processedImageBytes, err := os.ReadFile(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to read processed file: %w", err)
	}

	// Step 7: Upload to MinIO (processed bucket)
	if err := s.minioClient.UploadImage(ctx, processedPath, processedImageBytes); err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to upload processed image to MinIO")
		return fmt.Errorf("upload failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":       traceID,
		"processed_path": processedPath,
		"size_kb":        len(processedImageBytes) / 1024,
	}).Info("Image processing completed successfully")

	return nil
}

// saveTempFile saves image bytes to a temporary file
func (s *ImageProcessorService) saveTempFile(imageBytes []byte, traceID string) (string, error) {
	// Ensure temp directory exists
	if err := os.MkdirAll(s.tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create temp file with trace ID in name for debugging
	tempFilePath := filepath.Join(s.tempDir, fmt.Sprintf("%s_temp.jpg", traceID))

	if err := os.WriteFile(tempFilePath, imageBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
		"path":     tempFilePath,
	}).Debug("Temporary file created")

	return tempFilePath, nil
}

// cleanupTempFile removes temporary file and logs any errors
func (s *ImageProcessorService) cleanupTempFile(filePath string, traceID string) {
	if err := os.Remove(filePath); err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"path":     filePath,
			"error":    err.Error(),
		}).Warn("Failed to cleanup temporary file")
	} else {
		s.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"path":     filePath,
		}).Debug("Temporary file cleaned up")
	}
}
