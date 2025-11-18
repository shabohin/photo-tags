package service

import (
	"context"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/services/processor/internal/exiftool"
)

// MinioClientInterface defines methods for MinIO operations
type MinioClientInterface interface {
	DownloadImage(ctx context.Context, objectPath string) ([]byte, error)
	UploadImage(ctx context.Context, objectPath string, data []byte) error
}

// ExifToolInterface defines methods for ExifTool operations
type ExifToolInterface interface {
	WriteMetadata(ctx context.Context, imagePath string, metadata exiftool.Metadata, traceID string) error
	VerifyMetadata(ctx context.Context, imagePath string, traceID string) (bool, error)
	CheckAvailability() error
}

// PublisherInterface defines methods for publishing messages
type PublisherInterface interface {
	Publish(ctx context.Context, message []byte) error
	Close() error
}

// ImageProcessorInterface defines methods for image processing
type ImageProcessorInterface interface {
	ProcessImage(ctx context.Context, originalPath string, processedPath string, metadata models.Metadata, traceID string) error
}
