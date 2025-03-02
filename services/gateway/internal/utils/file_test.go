package utils

import (
	"bytes"
	"testing"

	"github.com/shabohin/photo-tags/services/gateway/internal/constants"
)

func TestValidateFileFormat(t *testing.T) {
	// Test valid formats
	validFiles := []string{
		"image.jpg",
		"photo.jpeg",
		"picture.png",
		"IMAGE.JPG",  // Test case insensitivity
		"Photo.JPEG", // Test case insensitivity
		"pic.Png",    // Test case insensitivity
	}

	for _, file := range validFiles {
		err := ValidateFileFormat(file)
		if err != nil {
			t.Errorf("Expected file '%s' to be valid, got error: %v", file, err)
		}
	}

	// Test invalid formats
	invalidFiles := []string{
		"image.gif",
		"document.pdf",
		"text.txt",
		"photo",      // No extension
		".jpg",       // Only extension
		"image.jpg.", // Extra dot
	}

	for _, file := range invalidFiles {
		err := ValidateFileFormat(file)
		if err != ErrUnsupportedFormat {
			t.Errorf("Expected file '%s' to be invalid with error ErrUnsupportedFormat, got %v", file, err)
		}
	}
}

func TestValidateFileSize(t *testing.T) {
	// Test empty file
	emptyFile := bytes.NewReader([]byte{})
	err := ValidateFileSize(emptyFile)
	if err != ErrEmptyFile {
		t.Errorf("Expected empty file to be invalid with error ErrEmptyFile, got %v", err)
	}

	// Test file within size limit
	smallFile := bytes.NewReader(make([]byte, 1024)) // 1 KB
	err = ValidateFileSize(smallFile)
	if err != nil {
		t.Errorf("Expected small file to be valid, got error: %v", err)
	}

	// Test file at size limit
	limitFile := bytes.NewReader(make([]byte, constants.MaxFileSize))
	err = ValidateFileSize(limitFile)
	if err != nil {
		t.Errorf("Expected file at size limit to be valid, got error: %v", err)
	}

	// Test file exceeding size limit
	largeFile := bytes.NewReader(make([]byte, constants.MaxFileSize+1))
	err = ValidateFileSize(largeFile)
	if err != ErrFileTooLarge {
		t.Errorf("Expected large file to be invalid with error ErrFileTooLarge, got %v", err)
	}
}

func TestGetMimeType(t *testing.T) {
	// Test known MIME types
	tests := map[string]string{
		"image.jpg":  "image/jpeg",
		"photo.jpeg": "image/jpeg",
		"pic.png":    "image/png",
		"IMAGE.JPG":  "image/jpeg", // Test case insensitivity
	}

	for file, expected := range tests {
		mimeType := GetMimeType(file)
		if mimeType != expected {
			t.Errorf("Expected MIME type for '%s' to be '%s', got '%s'", file, expected, mimeType)
		}
	}

	// Test unknown MIME type
	unknownFile := "document.xyz"
	mimeType := GetMimeType(unknownFile)
	if mimeType != "application/octet-stream" {
		t.Errorf("Expected MIME type for unknown file to be 'application/octet-stream', got '%s'", mimeType)
	}
}
