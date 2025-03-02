package utils

import (
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/shabohin/photo-tags/services/gateway/internal/constants"
)

// Common errors
var (
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrFileTooLarge      = errors.New("file too large")
	ErrEmptyFile         = errors.New("empty file")
)

// ValidateFileFormat validates file format based on filename
func ValidateFileFormat(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	// Check if format is supported
	if !constants.SupportedFormats[ext] {
		return ErrUnsupportedFormat
	}

	return nil
}

// ValidateFileSize validates file size
func ValidateFileSize(file io.ReadSeeker) error {
	// Get file size
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Reset file position
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	// Check if file is empty
	if size == 0 {
		return ErrEmptyFile
	}

	// Check if file is too large
	if size > constants.MaxFileSize {
		return ErrFileTooLarge
	}

	return nil
}

// GetMimeType returns MIME type based on filename
func GetMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Get MIME type
	if mimeType, ok := constants.MimeTypes[ext]; ok {
		return mimeType
	}

	// Default MIME type
	return "application/octet-stream"
}
