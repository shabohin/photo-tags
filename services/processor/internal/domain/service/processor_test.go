package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/services/processor/internal/exiftool"
)

// Mock MinIO client
type mockMinioClient struct {
	downloadFunc func(ctx context.Context, path string) ([]byte, error)
	uploadFunc   func(ctx context.Context, path string, data []byte) error
}

func (m *mockMinioClient) DownloadImage(ctx context.Context, path string) ([]byte, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, path)
	}
	return []byte("test image data"), nil
}

func (m *mockMinioClient) UploadImage(ctx context.Context, path string, data []byte) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, path, data)
	}
	return nil
}

// Mock ExifTool client
type mockExifTool struct {
	writeFunc  func(ctx context.Context, path string, metadata exiftool.Metadata, traceID string) error
	verifyFunc func(ctx context.Context, path string, traceID string) (bool, error)
}

func (m *mockExifTool) WriteMetadata(ctx context.Context, path string, metadata exiftool.Metadata, traceID string) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, path, metadata, traceID)
	}
	return nil
}

func (m *mockExifTool) VerifyMetadata(ctx context.Context, path string, traceID string) (bool, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, path, traceID)
	}
	return true, nil
}

func (m *mockExifTool) CheckAvailability() error {
	return nil
}

func TestNewImageProcessor(t *testing.T) {
	logger := logrus.New()
	minioClient := &mockMinioClient{}
	exifTool := &mockExifTool{}

	processor := NewImageProcessor(minioClient, exifTool, "/tmp/test", logger)

	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	if processor.tempDir != "/tmp/test" {
		t.Errorf("Expected temp dir '/tmp/test', got %s", processor.tempDir)
	}
}

func TestProcessImage_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	tempDir := t.TempDir()

	minioClient := &mockMinioClient{
		downloadFunc: func(ctx context.Context, path string) ([]byte, error) {
			// Return minimal JPEG data
			return []byte{
				0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
				0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
			}, nil
		},
		uploadFunc: func(ctx context.Context, path string, data []byte) error {
			if len(data) == 0 {
				t.Error("Expected non-empty data in upload")
			}
			return nil
		},
	}

	exifTool := &mockExifTool{
		writeFunc: func(ctx context.Context, path string, metadata exiftool.Metadata, traceID string) error {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("File does not exist: %s", path)
			}
			return nil
		},
	}

	processor := NewImageProcessor(minioClient, exifTool, tempDir, logger)

	metadata := models.Metadata{
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"test", "image", "metadata"},
	}

	ctx := context.Background()
	err := processor.ProcessImage(ctx, "original/test.jpg", "processed/test.jpg", metadata, "test-trace-id")

	if err != nil {
		t.Errorf("ProcessImage failed: %v", err)
	}
}

func TestProcessImage_DownloadFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	minioClient := &mockMinioClient{
		downloadFunc: func(ctx context.Context, path string) ([]byte, error) {
			return nil, errors.New("download failed")
		},
	}

	exifTool := &mockExifTool{}

	processor := NewImageProcessor(minioClient, exifTool, "/tmp/test", logger)

	metadata := models.Metadata{
		Title: "Test",
	}

	ctx := context.Background()
	err := processor.ProcessImage(ctx, "original/test.jpg", "processed/test.jpg", metadata, "test-trace-id")

	if err == nil {
		t.Error("Expected error when download fails")
	}

	if !errors.Is(err, errors.New("download failed")) && err.Error() != "download failed: download failed" {
		t.Errorf("Expected download error, got: %v", err)
	}
}

func TestProcessImage_MetadataWriteFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tempDir := t.TempDir()

	minioClient := &mockMinioClient{
		downloadFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte{0xFF, 0xD8, 0xFF, 0xD9}, nil
		},
	}

	exifTool := &mockExifTool{
		writeFunc: func(ctx context.Context, path string, metadata exiftool.Metadata, traceID string) error {
			return errors.New("exiftool failed")
		},
	}

	processor := NewImageProcessor(minioClient, exifTool, tempDir, logger)

	metadata := models.Metadata{
		Title: "Test",
	}

	ctx := context.Background()
	err := processor.ProcessImage(ctx, "original/test.jpg", "processed/test.jpg", metadata, "test-trace-id")

	if err == nil {
		t.Error("Expected error when metadata write fails")
	}

	if err.Error() != "metadata write failed: exiftool failed" {
		t.Errorf("Expected metadata write error, got: %v", err)
	}
}

func TestProcessImage_UploadFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tempDir := t.TempDir()

	minioClient := &mockMinioClient{
		downloadFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte{0xFF, 0xD8, 0xFF, 0xD9}, nil
		},
		uploadFunc: func(ctx context.Context, path string, data []byte) error {
			return errors.New("upload failed")
		},
	}

	exifTool := &mockExifTool{}

	processor := NewImageProcessor(minioClient, exifTool, tempDir, logger)

	metadata := models.Metadata{
		Title: "Test",
	}

	ctx := context.Background()
	err := processor.ProcessImage(ctx, "original/test.jpg", "processed/test.jpg", metadata, "test-trace-id")

	if err == nil {
		t.Error("Expected error when upload fails")
	}

	if err.Error() != "upload failed: upload failed" {
		t.Errorf("Expected upload error, got: %v", err)
	}
}

func TestProcessImage_VerificationWarning(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	tempDir := t.TempDir()

	minioClient := &mockMinioClient{
		downloadFunc: func(ctx context.Context, path string) ([]byte, error) {
			return []byte{0xFF, 0xD8, 0xFF, 0xD9}, nil
		},
	}

	// Verification fails but processing continues
	exifTool := &mockExifTool{
		verifyFunc: func(ctx context.Context, path string, traceID string) (bool, error) {
			return false, errors.New("verification failed")
		},
	}

	processor := NewImageProcessor(minioClient, exifTool, tempDir, logger)

	metadata := models.Metadata{
		Title: "Test",
	}

	ctx := context.Background()
	err := processor.ProcessImage(ctx, "original/test.jpg", "processed/test.jpg", metadata, "test-trace-id")

	// Should still succeed even if verification fails
	if err != nil {
		t.Errorf("ProcessImage should succeed even with verification failure, got: %v", err)
	}
}
