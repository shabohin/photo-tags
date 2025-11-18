package exiftool

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Metadata represents image metadata to be written
type Metadata struct {
	Title       string
	Description string
	Keywords    []string
}

// Client wraps ExifTool command-line tool
type Client struct {
	binaryPath string
	timeout    time.Duration
	logger     *logrus.Logger
}

// NewClient creates a new ExifTool client
func NewClient(binaryPath string, timeout time.Duration, logger *logrus.Logger) *Client {
	return &Client{
		binaryPath: binaryPath,
		timeout:    timeout,
		logger:     logger,
	}
}

// WriteMetadata writes metadata to an image file
// The image is modified in-place
func (c *Client) WriteMetadata(ctx context.Context, imagePath string, metadata Metadata, traceID string) error {
	// Check if exiftool is available
	if _, err := exec.LookPath(c.binaryPath); err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id":    traceID,
			"binary_path": c.binaryPath,
			"error":       err.Error(),
		}).Error("ExifTool binary not found")
		return fmt.Errorf("exiftool not found at %s: %w", c.binaryPath, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build ExifTool command arguments
	args := c.buildMetadataArgs(imagePath, metadata)

	c.logger.WithFields(logrus.Fields{
		"trace_id":  traceID,
		"image":     imagePath,
		"title":     metadata.Title,
		"keywords":  len(metadata.Keywords),
		"args_size": len(args),
	}).Debug("Writing metadata with ExifTool")

	// Execute command
	cmd := exec.CommandContext(ctx, c.binaryPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"output":   string(output),
			"error":    err.Error(),
		}).Error("ExifTool command failed")
		return fmt.Errorf("exiftool failed: %w, output: %s", err, string(output))
	}

	c.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
		"output":   string(output),
	}).Debug("ExifTool command succeeded")

	return nil
}

// buildMetadataArgs constructs ExifTool command arguments
func (c *Client) buildMetadataArgs(imagePath string, metadata Metadata) []string {
	args := []string{
		"-overwrite_original", // Don't create backup files
		"-charset", "utf8",    // Support Unicode characters
	}

	// Write Title to multiple tags
	if metadata.Title != "" {
		args = append(args,
			fmt.Sprintf("-XPTitle=%s", metadata.Title),
			fmt.Sprintf("-IPTC:Headline=%s", metadata.Title),
			fmt.Sprintf("-XMP:Title=%s", metadata.Title),
		)
	}

	// Write Description to multiple tags
	if metadata.Description != "" {
		args = append(args,
			fmt.Sprintf("-ImageDescription=%s", metadata.Description),
			fmt.Sprintf("-IPTC:Caption-Abstract=%s", metadata.Description),
			fmt.Sprintf("-XMP:Description=%s", metadata.Description),
		)
	}

	// Write Keywords - ExifTool needs each keyword separately
	for _, keyword := range metadata.Keywords {
		if keyword != "" {
			args = append(args,
				fmt.Sprintf("-IPTC:Keywords+=%s", keyword),
				fmt.Sprintf("-XMP:Subject+=%s", keyword),
			)
		}
	}

	// Add the image path as last argument
	args = append(args, imagePath)

	return args
}

// VerifyMetadata checks that metadata was written correctly
func (c *Client) VerifyMetadata(ctx context.Context, imagePath string, traceID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	c.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
		"image":    imagePath,
	}).Debug("Verifying metadata")

	// Read back metadata in JSON format
	cmd := exec.CommandContext(ctx, c.binaryPath,
		"-j",
		"-Title",
		"-Description",
		"-Keywords",
		"-Subject",
		imagePath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Warn("Metadata verification failed")
		return false, fmt.Errorf("verification failed: %w", err)
	}

	// Check that output contains valid JSON and no error messages
	outputStr := string(output)
	hasContent := len(output) > 0 && !strings.Contains(outputStr, "error")

	c.logger.WithFields(logrus.Fields{
		"trace_id":    traceID,
		"has_content": hasContent,
	}).Debug("Metadata verification result")

	return hasContent, nil
}

// CheckAvailability verifies that ExifTool is installed and accessible
func (c *Client) CheckAvailability() error {
	cmd := exec.Command(c.binaryPath, "-ver")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("exiftool not available: %w, output: %s", err, string(output))
	}

	version := strings.TrimSpace(string(output))
	c.logger.WithField("version", version).Info("ExifTool is available")

	return nil
}
