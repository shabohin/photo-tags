package contracts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImageUploadSerialization tests that ImageUpload can be serialized and deserialized correctly
func TestImageUploadSerialization(t *testing.T) {
	original := models.ImageUpload{
		Timestamp:        time.Date(2024, 1, 15, 12, 30, 0, 0, time.UTC),
		TraceID:          "trace-abc-123",
		GroupID:          "group-456",
		TelegramUsername: "testuser",
		OriginalFilename: "sunset.jpg",
		OriginalPath:     "/uploads/2024/01/sunset.jpg",
		TelegramID:       987654321,
	}

	t.Run("Marshal and Unmarshal", func(t *testing.T) {
		// Serialize
		jsonData, err := json.Marshal(original)
		require.NoError(t, err, "Failed to marshal ImageUpload")

		// Deserialize
		var deserialized models.ImageUpload
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err, "Failed to unmarshal ImageUpload")

		// Compare
		assert.Equal(t, original.TraceID, deserialized.TraceID)
		assert.Equal(t, original.GroupID, deserialized.GroupID)
		assert.Equal(t, original.TelegramUsername, deserialized.TelegramUsername)
		assert.Equal(t, original.OriginalFilename, deserialized.OriginalFilename)
		assert.Equal(t, original.OriginalPath, deserialized.OriginalPath)
		assert.Equal(t, original.TelegramID, deserialized.TelegramID)
		assert.True(t, original.Timestamp.Equal(deserialized.Timestamp), "Timestamps should be equal")
	})

	t.Run("JSON field names match schema", func(t *testing.T) {
		jsonData, err := json.Marshal(original)
		require.NoError(t, err)

		var rawJSON map[string]interface{}
		err = json.Unmarshal(jsonData, &rawJSON)
		require.NoError(t, err)

		// Verify JSON field names match the contract
		assert.Contains(t, rawJSON, "timestamp")
		assert.Contains(t, rawJSON, "trace_id")
		assert.Contains(t, rawJSON, "group_id")
		assert.Contains(t, rawJSON, "telegram_username")
		assert.Contains(t, rawJSON, "original_filename")
		assert.Contains(t, rawJSON, "original_path")
		assert.Contains(t, rawJSON, "telegram_id")

		// Verify no unexpected fields
		expectedFields := []string{
			"timestamp", "trace_id", "group_id", "telegram_username",
			"original_filename", "original_path", "telegram_id",
		}
		assert.Len(t, rawJSON, len(expectedFields))
	})
}

// TestMetadataGeneratedSerialization tests MetadataGenerated message serialization
func TestMetadataGeneratedSerialization(t *testing.T) {
	original := models.MetadataGenerated{
		Timestamp:        time.Date(2024, 1, 15, 12, 35, 0, 0, time.UTC),
		TraceID:          "trace-abc-123",
		GroupID:          "group-456",
		OriginalFilename: "sunset.jpg",
		OriginalPath:     "/uploads/2024/01/sunset.jpg",
		Metadata: models.Metadata{
			Title:       "Beautiful Sunset",
			Description: "A stunning sunset over the ocean",
			Keywords:    []string{"sunset", "ocean", "nature", "beautiful"},
		},
		TelegramID: 987654321,
	}

	t.Run("Marshal and Unmarshal", func(t *testing.T) {
		// Serialize
		jsonData, err := json.Marshal(original)
		require.NoError(t, err, "Failed to marshal MetadataGenerated")

		// Deserialize
		var deserialized models.MetadataGenerated
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err, "Failed to unmarshal MetadataGenerated")

		// Compare
		assert.Equal(t, original.TraceID, deserialized.TraceID)
		assert.Equal(t, original.GroupID, deserialized.GroupID)
		assert.Equal(t, original.OriginalFilename, deserialized.OriginalFilename)
		assert.Equal(t, original.OriginalPath, deserialized.OriginalPath)
		assert.Equal(t, original.TelegramID, deserialized.TelegramID)
		assert.True(t, original.Timestamp.Equal(deserialized.Timestamp))

		// Compare metadata
		assert.Equal(t, original.Metadata.Title, deserialized.Metadata.Title)
		assert.Equal(t, original.Metadata.Description, deserialized.Metadata.Description)
		assert.Equal(t, original.Metadata.Keywords, deserialized.Metadata.Keywords)
	})

	t.Run("JSON field names match schema", func(t *testing.T) {
		jsonData, err := json.Marshal(original)
		require.NoError(t, err)

		var rawJSON map[string]interface{}
		err = json.Unmarshal(jsonData, &rawJSON)
		require.NoError(t, err)

		// Verify top-level fields
		assert.Contains(t, rawJSON, "timestamp")
		assert.Contains(t, rawJSON, "trace_id")
		assert.Contains(t, rawJSON, "group_id")
		assert.Contains(t, rawJSON, "original_filename")
		assert.Contains(t, rawJSON, "original_path")
		assert.Contains(t, rawJSON, "metadata")
		assert.Contains(t, rawJSON, "telegram_id")

		// Verify metadata fields
		metadata, ok := rawJSON["metadata"].(map[string]interface{})
		require.True(t, ok, "metadata should be an object")
		assert.Contains(t, metadata, "title")
		assert.Contains(t, metadata, "description")
		assert.Contains(t, metadata, "keywords")
	})

	t.Run("Empty keywords array serializes correctly", func(t *testing.T) {
		msg := models.MetadataGenerated{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			OriginalFilename: "test.jpg",
			OriginalPath:     "/test.jpg",
			Metadata: models.Metadata{
				Title:       "Test",
				Description: "Test description",
				Keywords:    []string{}, // empty array
			},
			TelegramID: 123,
		}

		jsonData, err := json.Marshal(msg)
		require.NoError(t, err)

		var deserialized models.MetadataGenerated
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		assert.NotNil(t, deserialized.Metadata.Keywords)
		assert.Len(t, deserialized.Metadata.Keywords, 0)
	})
}

// TestImageProcessedSerialization tests ImageProcessed message serialization
func TestImageProcessedSerialization(t *testing.T) {
	t.Run("Completed status", func(t *testing.T) {
		original := models.ImageProcessed{
			Timestamp:        time.Date(2024, 1, 15, 12, 40, 0, 0, time.UTC),
			TraceID:          "trace-abc-123",
			GroupID:          "group-456",
			TelegramUsername: "testuser",
			OriginalFilename: "sunset.jpg",
			ProcessedPath:    "/processed/2024/01/sunset_tagged.jpg",
			Status:           "completed",
			Error:            "",
			TelegramID:       987654321,
		}

		// Serialize
		jsonData, err := json.Marshal(original)
		require.NoError(t, err, "Failed to marshal ImageProcessed")

		// Deserialize
		var deserialized models.ImageProcessed
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err, "Failed to unmarshal ImageProcessed")

		// Compare
		assert.Equal(t, original.TraceID, deserialized.TraceID)
		assert.Equal(t, original.GroupID, deserialized.GroupID)
		assert.Equal(t, original.TelegramUsername, deserialized.TelegramUsername)
		assert.Equal(t, original.OriginalFilename, deserialized.OriginalFilename)
		assert.Equal(t, original.ProcessedPath, deserialized.ProcessedPath)
		assert.Equal(t, original.Status, deserialized.Status)
		assert.Equal(t, original.Error, deserialized.Error)
		assert.Equal(t, original.TelegramID, deserialized.TelegramID)
		assert.True(t, original.Timestamp.Equal(deserialized.Timestamp))
	})

	t.Run("Failed status with error", func(t *testing.T) {
		original := models.ImageProcessed{
			Timestamp:        time.Date(2024, 1, 15, 12, 40, 0, 0, time.UTC),
			TraceID:          "trace-abc-123",
			GroupID:          "group-456",
			TelegramUsername: "testuser",
			OriginalFilename: "corrupted.jpg",
			ProcessedPath:    "",
			Status:           "failed",
			Error:            "Image format not supported",
			TelegramID:       987654321,
		}

		jsonData, err := json.Marshal(original)
		require.NoError(t, err)

		var deserialized models.ImageProcessed
		err = json.Unmarshal(jsonData, &deserialized)
		require.NoError(t, err)

		assert.Equal(t, "failed", deserialized.Status)
		assert.Equal(t, "Image format not supported", deserialized.Error)
		assert.Empty(t, deserialized.ProcessedPath)
	})

	t.Run("JSON field names match schema", func(t *testing.T) {
		msg := models.ImageProcessed{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			TelegramUsername: "user",
			OriginalFilename: "test.jpg",
			ProcessedPath:    "/processed/test.jpg",
			Status:           "completed",
			TelegramID:       123,
		}

		jsonData, err := json.Marshal(msg)
		require.NoError(t, err)

		var rawJSON map[string]interface{}
		err = json.Unmarshal(jsonData, &rawJSON)
		require.NoError(t, err)

		assert.Contains(t, rawJSON, "timestamp")
		assert.Contains(t, rawJSON, "trace_id")
		assert.Contains(t, rawJSON, "group_id")
		assert.Contains(t, rawJSON, "telegram_username")
		assert.Contains(t, rawJSON, "original_filename")
		assert.Contains(t, rawJSON, "processed_path")
		assert.Contains(t, rawJSON, "status")
		assert.Contains(t, rawJSON, "telegram_id")
	})

	t.Run("Error field is omitted when empty", func(t *testing.T) {
		msg := models.ImageProcessed{
			Timestamp:        time.Now(),
			TraceID:          "trace-123",
			GroupID:          "group-456",
			TelegramUsername: "user",
			OriginalFilename: "test.jpg",
			ProcessedPath:    "/processed/test.jpg",
			Status:           "completed",
			Error:            "", // empty error
			TelegramID:       123,
		}

		jsonData, err := json.Marshal(msg)
		require.NoError(t, err)

		var rawJSON map[string]interface{}
		err = json.Unmarshal(jsonData, &rawJSON)
		require.NoError(t, err)

		// error field should not be present when empty due to omitempty tag
		_, hasError := rawJSON["error"]
		assert.False(t, hasError, "Empty error field should be omitted from JSON")
	})
}

// TestCrossServiceCompatibility ensures messages can be correctly sent between services
func TestCrossServiceCompatibility(t *testing.T) {
	t.Run("Gateway -> Analyzer: ImageUpload", func(t *testing.T) {
		// Simulate Gateway creating and sending ImageUpload
		gatewayMsg := models.ImageUpload{
			Timestamp:        time.Now(),
			TraceID:          "trace-gateway-123",
			GroupID:          "group-789",
			TelegramUsername: "alice",
			OriginalFilename: "vacation.jpg",
			OriginalPath:     "/uploads/vacation.jpg",
			TelegramID:       111222333,
		}

		// Serialize (as Gateway would do)
		msgBytes, err := json.Marshal(gatewayMsg)
		require.NoError(t, err)

		// Deserialize (as Analyzer would do)
		var analyzerMsg models.ImageUpload
		err = json.Unmarshal(msgBytes, &analyzerMsg)
		require.NoError(t, err)

		// Verify all fields are preserved
		assert.Equal(t, gatewayMsg.TraceID, analyzerMsg.TraceID)
		assert.Equal(t, gatewayMsg.GroupID, analyzerMsg.GroupID)
		assert.Equal(t, gatewayMsg.TelegramUsername, analyzerMsg.TelegramUsername)
		assert.Equal(t, gatewayMsg.OriginalFilename, analyzerMsg.OriginalFilename)
		assert.Equal(t, gatewayMsg.OriginalPath, analyzerMsg.OriginalPath)
		assert.Equal(t, gatewayMsg.TelegramID, analyzerMsg.TelegramID)
	})

	t.Run("Analyzer -> Processor: MetadataGenerated", func(t *testing.T) {
		// Simulate Analyzer creating and sending MetadataGenerated
		analyzerMsg := models.MetadataGenerated{
			Timestamp:        time.Now(),
			TraceID:          "trace-analyzer-456",
			GroupID:          "group-789",
			OriginalFilename: "vacation.jpg",
			OriginalPath:     "/uploads/vacation.jpg",
			Metadata: models.Metadata{
				Title:       "Summer Vacation",
				Description: "Beach vacation memories",
				Keywords:    []string{"beach", "summer", "vacation"},
			},
			TelegramID: 111222333,
		}

		// Serialize
		msgBytes, err := json.Marshal(analyzerMsg)
		require.NoError(t, err)

		// Deserialize (as Processor would do)
		var processorMsg models.MetadataGenerated
		err = json.Unmarshal(msgBytes, &processorMsg)
		require.NoError(t, err)

		// Verify all fields are preserved
		assert.Equal(t, analyzerMsg.TraceID, processorMsg.TraceID)
		assert.Equal(t, analyzerMsg.Metadata.Title, processorMsg.Metadata.Title)
		assert.Equal(t, analyzerMsg.Metadata.Description, processorMsg.Metadata.Description)
		assert.Equal(t, analyzerMsg.Metadata.Keywords, processorMsg.Metadata.Keywords)
	})

	t.Run("Processor -> Gateway: ImageProcessed", func(t *testing.T) {
		// Simulate Processor creating and sending ImageProcessed
		processorMsg := models.ImageProcessed{
			Timestamp:        time.Now(),
			TraceID:          "trace-processor-789",
			GroupID:          "group-789",
			TelegramUsername: "", // Will be filled by Gateway
			OriginalFilename: "vacation.jpg",
			ProcessedPath:    "/processed/vacation_tagged.jpg",
			Status:           "completed",
			Error:            "",
			TelegramID:       111222333,
		}

		// Serialize
		msgBytes, err := json.Marshal(processorMsg)
		require.NoError(t, err)

		// Deserialize (as Gateway would do)
		var gatewayMsg models.ImageProcessed
		err = json.Unmarshal(msgBytes, &gatewayMsg)
		require.NoError(t, err)

		// Verify all fields are preserved
		assert.Equal(t, processorMsg.TraceID, gatewayMsg.TraceID)
		assert.Equal(t, processorMsg.Status, gatewayMsg.Status)
		assert.Equal(t, processorMsg.ProcessedPath, gatewayMsg.ProcessedPath)
		assert.Equal(t, processorMsg.TelegramID, gatewayMsg.TelegramID)
	})
}
