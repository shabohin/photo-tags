package constants

import "testing"

func TestSupportedFormats(t *testing.T) {
	// Test supported formats
	if !SupportedFormats[".jpg"] {
		t.Error("Expected .jpg to be supported")
	}
	if !SupportedFormats[".jpeg"] {
		t.Error("Expected .jpeg to be supported")
	}
	if !SupportedFormats[".png"] {
		t.Error("Expected .png to be supported")
	}

	// Test unsupported formats
	if SupportedFormats[".gif"] {
		t.Error("Expected .gif to be unsupported")
	}
	if SupportedFormats[".bmp"] {
		t.Error("Expected .bmp to be unsupported")
	}
}

func TestMimeTypes(t *testing.T) {
	// Test MIME types
	if MimeTypes[".jpg"] != "image/jpeg" {
		t.Errorf("Expected MIME type for .jpg to be 'image/jpeg', got '%s'", MimeTypes[".jpg"])
	}
	if MimeTypes[".jpeg"] != "image/jpeg" {
		t.Errorf("Expected MIME type for .jpeg to be 'image/jpeg', got '%s'", MimeTypes[".jpeg"])
	}
	if MimeTypes[".png"] != "image/png" {
		t.Errorf("Expected MIME type for .png to be 'image/png', got '%s'", MimeTypes[".png"])
	}
}

func TestConstants(t *testing.T) {
	// Test max file size
	if MaxFileSize != 10*1024*1024 {
		t.Errorf("Expected MaxFileSize to be %d, got %d", 10*1024*1024, MaxFileSize)
	}

	// Test max concurrent uploads
	if MaxConcurrentUploads != 5 {
		t.Errorf("Expected MaxConcurrentUploads to be %d, got %d", 5, MaxConcurrentUploads)
	}

	// Test status constants
	if StatusPending != "pending" {
		t.Errorf("Expected StatusPending to be 'pending', got '%s'", StatusPending)
	}
	if StatusProcessing != "processing" {
		t.Errorf("Expected StatusProcessing to be 'processing', got '%s'", StatusProcessing)
	}
	if StatusCompleted != "completed" {
		t.Errorf("Expected StatusCompleted to be 'completed', got '%s'", StatusCompleted)
	}
	if StatusFailed != "failed" {
		t.Errorf("Expected StatusFailed to be 'failed', got '%s'", StatusFailed)
	}
}
