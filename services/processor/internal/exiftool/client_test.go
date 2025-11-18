package exiftool

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewClient(t *testing.T) {
	logger := logrus.New()
	client := NewClient("/usr/bin/exiftool", 10*time.Second, logger)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.binaryPath != "/usr/bin/exiftool" {
		t.Errorf("Expected binary path '/usr/bin/exiftool', got %s", client.binaryPath)
	}

	if client.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.timeout)
	}
}

func TestBuildMetadataArgs(t *testing.T) {
	logger := logrus.New()
	client := NewClient("/usr/bin/exiftool", 10*time.Second, logger)

	metadata := Metadata{
		Title:       "Test Title",
		Description: "Test Description",
		Keywords:    []string{"keyword1", "keyword2", "keyword3"},
	}

	args := client.buildMetadataArgs("/tmp/test.jpg", metadata)

	// Check that essential arguments are present
	if args[0] != "-overwrite_original" {
		t.Errorf("Expected first arg '-overwrite_original', got %s", args[0])
	}

	// Check for UTF-8 charset
	hasCharset := false
	for i, arg := range args {
		if arg == "-charset" && i+1 < len(args) && args[i+1] == "utf8" {
			hasCharset = true
			break
		}
	}
	if !hasCharset {
		t.Error("Expected charset utf8 argument")
	}

	// Check that title is set
	hasTitle := false
	for _, arg := range args {
		if arg == "-XPTitle=Test Title" {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		t.Error("Expected title argument")
	}

	// Check that keywords are added
	keywordCount := 0
	for _, arg := range args {
		if arg == "-IPTC:Keywords+=keyword1" || arg == "-IPTC:Keywords+=keyword2" {
			keywordCount++
		}
	}
	if keywordCount < 2 {
		t.Errorf("Expected at least 2 keyword arguments, got %d", keywordCount)
	}

	// Last argument should be the image path
	if args[len(args)-1] != "/tmp/test.jpg" {
		t.Errorf("Expected last arg to be image path, got %s", args[len(args)-1])
	}
}

func TestBuildMetadataArgs_EmptyFields(t *testing.T) {
	logger := logrus.New()
	client := NewClient("/usr/bin/exiftool", 10*time.Second, logger)

	metadata := Metadata{
		Title:       "",
		Description: "",
		Keywords:    []string{},
	}

	args := client.buildMetadataArgs("/tmp/test.jpg", metadata)

	// Should still have basic args and image path
	if len(args) < 3 {
		t.Errorf("Expected at least 3 args, got %d", len(args))
	}

	// Should not have title or description args
	for _, arg := range args {
		if arg == "-XPTitle=" || arg == "-ImageDescription=" {
			t.Errorf("Unexpected empty metadata arg: %s", arg)
		}
	}
}

func TestBuildMetadataArgs_UnicodeContent(t *testing.T) {
	logger := logrus.New()
	client := NewClient("/usr/bin/exiftool", 10*time.Second, logger)

	metadata := Metadata{
		Title:       "Тестовое изображение",
		Description: "Описание на русском языке",
		Keywords:    []string{"ключевое слово", "тест", "фото"},
	}

	args := client.buildMetadataArgs("/tmp/test.jpg", metadata)

	// Check that UTF-8 is specified
	hasCharset := false
	for i, arg := range args {
		if arg == "-charset" && i+1 < len(args) && args[i+1] == "utf8" {
			hasCharset = true
			break
		}
	}
	if !hasCharset {
		t.Error("Expected UTF-8 charset for Unicode content")
	}

	// Check that Russian title is present
	hasRussianTitle := false
	for _, arg := range args {
		if arg == "-XPTitle=Тестовое изображение" {
			hasRussianTitle = true
			break
		}
	}
	if !hasRussianTitle {
		t.Error("Expected Russian title in arguments")
	}
}

// Integration test - only runs if exiftool is available
func TestWriteMetadata_Integration(t *testing.T) {
	// Check if exiftool is available
	if _, err := exec.LookPath("exiftool"); err != nil {
		t.Skip("ExifTool not available, skipping integration test")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	client := NewClient("exiftool", 10*time.Second, logger)

	// Create a temporary test image (1x1 pixel JPEG)
	tempDir := t.TempDir()
	testImagePath := filepath.Join(tempDir, "test.jpg")

	// Create minimal valid JPEG
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
	}

	if err := os.WriteFile(testImagePath, jpegData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Test metadata
	metadata := Metadata{
		Title:       "Test Image Title",
		Description: "Test image description with detailed info",
		Keywords:    []string{"test", "image", "metadata", "exiftool"},
	}

	ctx := context.Background()
	err := client.WriteMetadata(ctx, testImagePath, metadata, "test-trace-id")

	if err != nil {
		t.Fatalf("WriteMetadata failed: %v", err)
	}

	// Verify metadata was written
	verified, err := client.VerifyMetadata(ctx, testImagePath, "test-trace-id")
	if err != nil {
		t.Errorf("VerifyMetadata failed: %v", err)
	}

	if !verified {
		t.Error("Metadata verification returned false")
	}
}

func TestCheckAvailability(t *testing.T) {
	logger := logrus.New()

	// Test with likely invalid path
	client := NewClient("/nonexistent/exiftool", 10*time.Second, logger)
	err := client.CheckAvailability()

	if err == nil {
		t.Error("Expected error for nonexistent exiftool, got nil")
	}
}
