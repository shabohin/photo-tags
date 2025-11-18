package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

// loadSchema loads a JSON schema from the schemas directory
func loadSchema(t *testing.T, schemaFile string) *gojsonschema.Schema {
	t.Helper()

	schemaPath := filepath.Join("schemas", schemaFile)
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

	schema, err := gojsonschema.NewSchema(schemaLoader)
	require.NoError(t, err, "Failed to load schema: %s", schemaFile)

	return schema
}

// validateAgainstSchema validates a Go struct against a JSON schema
func validateAgainstSchema(t *testing.T, schema *gojsonschema.Schema, data interface{}) *gojsonschema.Result {
	t.Helper()

	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "Failed to marshal data to JSON")

	documentLoader := gojsonschema.NewBytesLoader(jsonData)
	result, err := schema.Validate(documentLoader)
	require.NoError(t, err, "Schema validation error")

	return result
}

func TestImageUploadSchema(t *testing.T) {
	schema := loadSchema(t, "image_upload.json")

	t.Run("Valid ImageUpload message", func(t *testing.T) {
		msg := models.ImageUpload{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			TelegramUsername: "testuser",
			OriginalFilename: "photo.jpg",
			OriginalPath:     "/uploads/photo.jpg",
			TelegramID:       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		if !result.Valid() {
			for _, err := range result.Errors() {
				t.Errorf("Validation error: %s", err)
			}
		}
		assert.True(t, result.Valid(), "Valid ImageUpload should pass schema validation")
	})

	t.Run("Missing required field - trace_id", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"group_id":          "group-456",
			"telegram_username": "testuser",
			"original_filename": "photo.jpg",
			"original_path":     "/uploads/photo.jpg",
			"telegram_id":       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Missing trace_id should fail validation")
	})

	t.Run("Invalid telegram_id type", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"telegram_username": "testuser",
			"original_filename": "photo.jpg",
			"original_path":     "/uploads/photo.jpg",
			"telegram_id":       "not-a-number",
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Invalid telegram_id type should fail validation")
	})

	t.Run("Additional properties not allowed", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"telegram_username": "testuser",
			"original_filename": "photo.jpg",
			"original_path":     "/uploads/photo.jpg",
			"telegram_id":       12345,
			"extra_field":       "should not be here",
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Additional properties should fail validation")
	})
}

func TestMetadataGeneratedSchema(t *testing.T) {
	// First check if schemas directory exists
	if _, err := os.Stat("schemas"); os.IsNotExist(err) {
		t.Skip("Schemas directory not found")
	}

	schema := loadSchema(t, "metadata_generated.json")

	t.Run("Valid MetadataGenerated message", func(t *testing.T) {
		msg := models.MetadataGenerated{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			OriginalFilename: "photo.jpg",
			OriginalPath:     "/uploads/photo.jpg",
			Metadata: models.Metadata{
				Title:       "Test Photo",
				Description: "A test photo description",
				Keywords:    []string{"test", "photo", "sample"},
			},
			TelegramID: 12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		if !result.Valid() {
			for _, err := range result.Errors() {
				t.Errorf("Validation error: %s", err)
			}
		}
		assert.True(t, result.Valid(), "Valid MetadataGenerated should pass schema validation")
	})

	t.Run("Missing metadata field", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"original_filename": "photo.jpg",
			"original_path":     "/uploads/photo.jpg",
			"telegram_id":       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Missing metadata should fail validation")
	})

	t.Run("Invalid metadata structure", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"original_filename": "photo.jpg",
			"original_path":     "/uploads/photo.jpg",
			"metadata": map[string]interface{}{
				"title": "Test",
				// missing description
				"keywords": []string{"test"},
			},
			"telegram_id": 12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Invalid metadata structure should fail validation")
	})
}

func TestImageProcessedSchema(t *testing.T) {
	schema := loadSchema(t, "image_processed.json")

	t.Run("Valid ImageProcessed message - completed", func(t *testing.T) {
		msg := models.ImageProcessed{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			TelegramUsername: "testuser",
			OriginalFilename: "photo.jpg",
			ProcessedPath:    "/processed/photo.jpg",
			Status:           "completed",
			Error:            "",
			TelegramID:       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		if !result.Valid() {
			for _, err := range result.Errors() {
				t.Errorf("Validation error: %s", err)
			}
		}
		assert.True(t, result.Valid(), "Valid ImageProcessed (completed) should pass schema validation")
	})

	t.Run("Valid ImageProcessed message - failed", func(t *testing.T) {
		msg := models.ImageProcessed{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			TelegramUsername: "testuser",
			OriginalFilename: "photo.jpg",
			ProcessedPath:    "",
			Status:           "failed",
			Error:            "Processing failed due to invalid format",
			TelegramID:       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		if !result.Valid() {
			for _, err := range result.Errors() {
				t.Errorf("Validation error: %s", err)
			}
		}
		assert.True(t, result.Valid(), "Valid ImageProcessed (failed) should pass schema validation")
	})

	t.Run("Invalid status value", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"telegram_username": "testuser",
			"original_filename": "photo.jpg",
			"processed_path":    "/processed/photo.jpg",
			"status":            "pending", // not in enum
			"telegram_id":       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Invalid status value should fail validation")
	})

	t.Run("Missing required status field", func(t *testing.T) {
		msg := map[string]interface{}{
			"timestamp":         time.Now().Format(time.RFC3339),
			"trace_id":          "trace-123",
			"group_id":          "group-456",
			"telegram_username": "testuser",
			"original_filename": "photo.jpg",
			"processed_path":    "/processed/photo.jpg",
			"telegram_id":       12345,
		}

		result := validateAgainstSchema(t, schema, msg)
		assert.False(t, result.Valid(), "Missing status field should fail validation")
	})
}

func TestMetadataSchema(t *testing.T) {
	schema := loadSchema(t, "metadata.json")

	t.Run("Valid Metadata", func(t *testing.T) {
		metadata := models.Metadata{
			Title:       "Test Photo",
			Description: "A detailed description",
			Keywords:    []string{"test", "photo"},
		}

		result := validateAgainstSchema(t, schema, metadata)
		if !result.Valid() {
			for _, err := range result.Errors() {
				t.Errorf("Validation error: %s", err)
			}
		}
		assert.True(t, result.Valid(), "Valid Metadata should pass schema validation")
	})

	t.Run("Empty keywords array", func(t *testing.T) {
		metadata := models.Metadata{
			Title:       "Test Photo",
			Description: "A detailed description",
			Keywords:    []string{},
		}

		result := validateAgainstSchema(t, schema, metadata)
		assert.True(t, result.Valid(), "Empty keywords array should be valid")
	})

	t.Run("Missing required field", func(t *testing.T) {
		metadata := map[string]interface{}{
			"title":    "Test Photo",
			"keywords": []string{"test"},
			// missing description
		}

		result := validateAgainstSchema(t, schema, metadata)
		assert.False(t, result.Valid(), "Missing description should fail validation")
	})
}
