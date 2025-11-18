package helpers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ExifData represents EXIF metadata
type ExifData struct {
	ImageDescription string `json:"ImageDescription"`
	XPKeywords       string `json:"XPKeywords"`
	Keywords         string `json:"Keywords"`
	Subject          string `json:"Subject"`
	UserComment      string `json:"UserComment"`
}

// ExtractExifData extracts EXIF data from an image file using exiftool
func ExtractExifData(imagePath string) (*ExifData, error) {
	// Check if exiftool is available
	if _, err := exec.LookPath("exiftool"); err != nil {
		return nil, fmt.Errorf("exiftool not found in PATH: %w", err)
	}

	// Run exiftool with JSON output
	cmd := exec.Command("exiftool", "-j", "-ImageDescription", "-XPKeywords",
		"-Keywords", "-Subject", "-UserComment", imagePath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run exiftool: %w", err)
	}

	// Parse JSON output
	var results []map[string]interface{}
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse exiftool output: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no exif data found")
	}

	data := &ExifData{}
	result := results[0]

	if desc, ok := result["ImageDescription"].(string); ok {
		data.ImageDescription = desc
	}
	if keywords, ok := result["XPKeywords"].(string); ok {
		data.XPKeywords = keywords
	}
	if keywords, ok := result["Keywords"].(string); ok {
		data.Keywords = keywords
	}
	if subject, ok := result["Subject"].(string); ok {
		data.Subject = subject
	}
	if comment, ok := result["UserComment"].(string); ok {
		data.UserComment = comment
	}

	return data, nil
}

// ValidateMetadata validates that metadata was properly embedded
func ValidateMetadata(exifData *ExifData, expectedTags []string) error {
	if exifData == nil {
		return fmt.Errorf("exif data is nil")
	}

	// Check if any metadata field contains the expected tags
	allMetadata := strings.Join([]string{
		exifData.ImageDescription,
		exifData.XPKeywords,
		exifData.Keywords,
		exifData.Subject,
		exifData.UserComment,
	}, " ")

	for _, tag := range expectedTags {
		if !strings.Contains(allMetadata, tag) {
			return fmt.Errorf("expected tag %q not found in metadata: %s", tag, allMetadata)
		}
	}

	return nil
}

// HasMetadata checks if the image has any metadata
func HasMetadata(exifData *ExifData) bool {
	if exifData == nil {
		return false
	}

	return exifData.ImageDescription != "" ||
		exifData.XPKeywords != "" ||
		exifData.Keywords != "" ||
		exifData.Subject != "" ||
		exifData.UserComment != ""
}
