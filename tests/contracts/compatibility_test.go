package contracts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackwardsCompatibility_ImageUpload tests that old versions of ImageUpload messages
// can still be deserialized by the current code
func TestBackwardsCompatibility_ImageUpload(t *testing.T) {
	t.Run("V1 message without telegram_id can be read", func(t *testing.T) {
		// Simulate an old message format that doesn't have telegram_id
		oldMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "old-trace-123",
			"group_id": "old-group-456",
			"telegram_username": "olduser",
			"original_filename": "old_photo.jpg",
			"original_path": "/uploads/old_photo.jpg"
		}`

		var msg models.ImageUpload
		err := json.Unmarshal([]byte(oldMessageJSON), &msg)

		// Should not fail to unmarshal
		require.NoError(t, err, "Old message format should be parseable")

		// Verify the fields that exist
		assert.Equal(t, "old-trace-123", msg.TraceID)
		assert.Equal(t, "old-group-456", msg.GroupID)
		assert.Equal(t, "olduser", msg.TelegramUsername)
		assert.Equal(t, "old_photo.jpg", msg.OriginalFilename)
		assert.Equal(t, "/uploads/old_photo.jpg", msg.OriginalPath)

		// telegram_id should be zero value
		assert.Equal(t, int64(0), msg.TelegramID)
	})

	t.Run("Extra fields in old message are ignored", func(t *testing.T) {
		// Simulate an old message with deprecated fields
		oldMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "old-trace-123",
			"group_id": "old-group-456",
			"telegram_username": "olduser",
			"original_filename": "old_photo.jpg",
			"original_path": "/uploads/old_photo.jpg",
			"telegram_id": 12345,
			"deprecated_field": "this should be ignored",
			"another_old_field": 999
		}`

		var msg models.ImageUpload
		err := json.Unmarshal([]byte(oldMessageJSON), &msg)

		// Should not fail despite extra fields
		require.NoError(t, err, "Message with extra fields should be parseable")

		// Verify known fields
		assert.Equal(t, "old-trace-123", msg.TraceID)
		assert.Equal(t, int64(12345), msg.TelegramID)
	})
}

// TestBackwardsCompatibility_MetadataGenerated tests backwards compatibility for MetadataGenerated
func TestBackwardsCompatibility_MetadataGenerated(t *testing.T) {
	t.Run("Old message without telegram_id", func(t *testing.T) {
		oldMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "old-trace-456",
			"group_id": "old-group-789",
			"original_filename": "old_sunset.jpg",
			"original_path": "/uploads/old_sunset.jpg",
			"metadata": {
				"title": "Old Sunset",
				"description": "A vintage photo",
				"keywords": ["old", "sunset"]
			}
		}`

		var msg models.MetadataGenerated
		err := json.Unmarshal([]byte(oldMessageJSON), &msg)

		require.NoError(t, err, "Old MetadataGenerated format should be parseable")

		assert.Equal(t, "old-trace-456", msg.TraceID)
		assert.Equal(t, "Old Sunset", msg.Metadata.Title)
		assert.Equal(t, int64(0), msg.TelegramID) // default value
	})

	t.Run("Metadata with null keywords becomes empty array", func(t *testing.T) {
		messageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "trace-789",
			"group_id": "group-123",
			"original_filename": "test.jpg",
			"original_path": "/test.jpg",
			"metadata": {
				"title": "Test",
				"description": "Test description",
				"keywords": null
			},
			"telegram_id": 123
		}`

		var msg models.MetadataGenerated
		err := json.Unmarshal([]byte(messageJSON), &msg)

		require.NoError(t, err)
		assert.Nil(t, msg.Metadata.Keywords) // null becomes nil in Go
	})

	t.Run("Old metadata without keywords field", func(t *testing.T) {
		// In early versions, keywords might not have existed
		messageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "trace-789",
			"group_id": "group-123",
			"original_filename": "test.jpg",
			"original_path": "/test.jpg",
			"metadata": {
				"title": "Test",
				"description": "Test description"
			},
			"telegram_id": 123
		}`

		var msg models.MetadataGenerated
		err := json.Unmarshal([]byte(messageJSON), &msg)

		require.NoError(t, err)
		assert.Equal(t, "Test", msg.Metadata.Title)
		assert.Nil(t, msg.Metadata.Keywords) // should be nil/empty
	})
}

// TestBackwardsCompatibility_ImageProcessed tests backwards compatibility for ImageProcessed
func TestBackwardsCompatibility_ImageProcessed(t *testing.T) {
	t.Run("Old message without status enum values", func(t *testing.T) {
		// In the past, status might have been boolean or different values
		oldMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "old-trace-999",
			"group_id": "old-group-111",
			"telegram_username": "olduser",
			"original_filename": "old_file.jpg",
			"processed_path": "/processed/old_file.jpg",
			"status": "completed",
			"telegram_id": 55555
		}`

		var msg models.ImageProcessed
		err := json.Unmarshal([]byte(oldMessageJSON), &msg)

		require.NoError(t, err)
		assert.Equal(t, "completed", msg.Status)
	})

	t.Run("Message without error field", func(t *testing.T) {
		// Error field might not have existed in early versions
		messageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "trace-888",
			"group_id": "group-222",
			"telegram_username": "user",
			"original_filename": "test.jpg",
			"processed_path": "/processed/test.jpg",
			"status": "completed",
			"telegram_id": 77777
		}`

		var msg models.ImageProcessed
		err := json.Unmarshal([]byte(messageJSON), &msg)

		require.NoError(t, err)
		assert.Equal(t, "completed", msg.Status)
		assert.Empty(t, msg.Error) // should be empty string
	})

	t.Run("Failed message with error field", func(t *testing.T) {
		messageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "trace-fail-123",
			"group_id": "group-333",
			"telegram_username": "user",
			"original_filename": "bad.jpg",
			"processed_path": "",
			"status": "failed",
			"error": "File corrupted",
			"telegram_id": 88888
		}`

		var msg models.ImageProcessed
		err := json.Unmarshal([]byte(messageJSON), &msg)

		require.NoError(t, err)
		assert.Equal(t, "failed", msg.Status)
		assert.Equal(t, "File corrupted", msg.Error)
	})
}

// TestForwardCompatibility tests that current messages can handle future additions
func TestForwardCompatibility(t *testing.T) {
	t.Run("ImageUpload with future fields", func(t *testing.T) {
		// Simulate a message from a newer version with additional fields
		futureMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "future-trace-123",
			"group_id": "future-group-456",
			"telegram_username": "futureuser",
			"original_filename": "future_photo.jpg",
			"original_path": "/uploads/future_photo.jpg",
			"telegram_id": 12345,
			"file_size": 1024000,
			"mime_type": "image/jpeg",
			"checksum": "abc123def456"
		}`

		var msg models.ImageUpload
		err := json.Unmarshal([]byte(futureMessageJSON), &msg)

		// Should successfully parse known fields and ignore unknown ones
		require.NoError(t, err, "Future message with extra fields should be parseable")

		assert.Equal(t, "future-trace-123", msg.TraceID)
		assert.Equal(t, "future-group-456", msg.GroupID)
		assert.Equal(t, int64(12345), msg.TelegramID)
	})

	t.Run("MetadataGenerated with future metadata fields", func(t *testing.T) {
		futureMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "future-trace-789",
			"group_id": "future-group-789",
			"original_filename": "future.jpg",
			"original_path": "/uploads/future.jpg",
			"metadata": {
				"title": "Future Photo",
				"description": "Advanced metadata",
				"keywords": ["future", "advanced"],
				"ai_confidence": 0.95,
				"detected_objects": ["person", "car"],
				"color_palette": ["#FF0000", "#00FF00"]
			},
			"telegram_id": 99999
		}`

		var msg models.MetadataGenerated
		err := json.Unmarshal([]byte(futureMessageJSON), &msg)

		// Should parse successfully, ignoring extra metadata fields
		require.NoError(t, err, "Future metadata fields should be ignored gracefully")

		assert.Equal(t, "Future Photo", msg.Metadata.Title)
		assert.Equal(t, "Advanced metadata", msg.Metadata.Description)
		assert.Contains(t, msg.Metadata.Keywords, "future")
	})

	t.Run("ImageProcessed with future status values should be handled", func(t *testing.T) {
		// Note: This will actually work in Go's JSON unmarshaler since string accepts any value
		futureMessageJSON := `{
			"timestamp": "2024-01-15T12:00:00Z",
			"trace_id": "future-trace-999",
			"group_id": "future-group-999",
			"telegram_username": "futureuser",
			"original_filename": "future.jpg",
			"processed_path": "/processed/future.jpg",
			"status": "processing",
			"telegram_id": 11111,
			"retry_count": 2,
			"processing_duration_ms": 1500
		}`

		var msg models.ImageProcessed
		err := json.Unmarshal([]byte(futureMessageJSON), &msg)

		require.NoError(t, err)
		// The status field will contain the new value, even if not in our current enum
		assert.Equal(t, "processing", msg.Status)
	})
}

// TestMessageEvolution tests schema changes over time
func TestMessageEvolution(t *testing.T) {
	t.Run("All message versions through time", func(t *testing.T) {
		// Version 1: Basic message
		v1JSON := `{"timestamp": "2024-01-01T10:00:00Z", "trace_id": "v1", "group_id": "g1", "telegram_username": "u1", "original_filename": "f1.jpg", "original_path": "/f1.jpg"}`

		var v1Msg models.ImageUpload
		err := json.Unmarshal([]byte(v1JSON), &v1Msg)
		require.NoError(t, err, "V1 message should parse")
		assert.Equal(t, "v1", v1Msg.TraceID)

		// Version 2: Added telegram_id
		v2JSON := `{"timestamp": "2024-02-01T10:00:00Z", "trace_id": "v2", "group_id": "g2", "telegram_username": "u2", "original_filename": "f2.jpg", "original_path": "/f2.jpg", "telegram_id": 123}`

		var v2Msg models.ImageUpload
		err = json.Unmarshal([]byte(v2JSON), &v2Msg)
		require.NoError(t, err, "V2 message should parse")
		assert.Equal(t, "v2", v2Msg.TraceID)
		assert.Equal(t, int64(123), v2Msg.TelegramID)

		// Current version can read both
		assert.NotEqual(t, v1Msg.TraceID, v2Msg.TraceID)
	})

	t.Run("Metadata evolution", func(t *testing.T) {
		// Early version with minimal metadata
		earlyJSON := `{
			"timestamp": "2024-01-01T10:00:00Z",
			"trace_id": "early",
			"group_id": "g1",
			"original_filename": "early.jpg",
			"original_path": "/early.jpg",
			"metadata": {
				"title": "Early",
				"description": "Early photo"
			},
			"telegram_id": 100
		}`

		var earlyMsg models.MetadataGenerated
		err := json.Unmarshal([]byte(earlyJSON), &earlyMsg)
		require.NoError(t, err)
		assert.Equal(t, "Early", earlyMsg.Metadata.Title)

		// Later version with keywords
		laterJSON := `{
			"timestamp": "2024-02-01T10:00:00Z",
			"trace_id": "later",
			"group_id": "g2",
			"original_filename": "later.jpg",
			"original_path": "/later.jpg",
			"metadata": {
				"title": "Later",
				"description": "Later photo",
				"keywords": ["later", "test"]
			},
			"telegram_id": 200
		}`

		var laterMsg models.MetadataGenerated
		err = json.Unmarshal([]byte(laterJSON), &laterMsg)
		require.NoError(t, err)
		assert.Equal(t, "Later", laterMsg.Metadata.Title)
		assert.Len(t, laterMsg.Metadata.Keywords, 2)
	})
}

// TestRealWorldScenarios tests real-world compatibility scenarios
func TestRealWorldScenarios(t *testing.T) {
	t.Run("Gradual rollout - old producer, new consumer", func(t *testing.T) {
		// Old Gateway service sends message without new field
		oldGatewayMsg := models.ImageUpload{
			Timestamp:        time.Now(),
			TraceID:          "old-gateway",
			GroupID:          "group",
			TelegramUsername: "user",
			OriginalFilename: "test.jpg",
			OriginalPath:     "/test.jpg",
			// telegram_id might be 0 or not set in old version
		}

		msgBytes, _ := json.Marshal(oldGatewayMsg)

		// New Analyzer service receives it
		var newAnalyzerMsg models.ImageUpload
		err := json.Unmarshal(msgBytes, &newAnalyzerMsg)

		require.NoError(t, err, "New service should handle old messages")
		assert.Equal(t, "old-gateway", newAnalyzerMsg.TraceID)
	})

	t.Run("Gradual rollout - new producer, old consumer", func(t *testing.T) {
		// New Gateway sends message with all fields
		newGatewayMsg := models.ImageUpload{
			Timestamp:        time.Now(),
			TraceID:          "new-gateway",
			GroupID:          "group",
			TelegramUsername: "user",
			OriginalFilename: "test.jpg",
			OriginalPath:     "/test.jpg",
			TelegramID:       12345,
		}

		msgBytes, _ := json.Marshal(newGatewayMsg)

		// Old Analyzer (simulated by ignoring telegram_id) receives it
		var partialMsg struct {
			TraceID          string    `json:"trace_id"`
			GroupID          string    `json:"group_id"`
			TelegramUsername string    `json:"telegram_username"`
			OriginalFilename string    `json:"original_filename"`
			OriginalPath     string    `json:"original_path"`
			Timestamp        time.Time `json:"timestamp"`
			// telegram_id field doesn't exist in old version
		}
		err := json.Unmarshal(msgBytes, &partialMsg)

		require.NoError(t, err, "Old service should ignore new fields")
		assert.Equal(t, "new-gateway", partialMsg.TraceID)
	})

	t.Run("Mixed versions in production", func(t *testing.T) {
		// Test that both old and new message formats work simultaneously
		messages := []string{
			`{"timestamp": "2024-01-15T12:00:00Z", "trace_id": "msg1", "group_id": "g1", "telegram_username": "u1", "original_filename": "f1.jpg", "original_path": "/f1.jpg"}`,
			`{"timestamp": "2024-01-15T12:00:00Z", "trace_id": "msg2", "group_id": "g2", "telegram_username": "u2", "original_filename": "f2.jpg", "original_path": "/f2.jpg", "telegram_id": 200}`,
			`{"timestamp": "2024-01-15T12:00:00Z", "trace_id": "msg3", "group_id": "g3", "telegram_username": "u3", "original_filename": "f3.jpg", "original_path": "/f3.jpg", "telegram_id": 300, "extra_field": "ignored"}`,
		}

		for i, msgJSON := range messages {
			var msg models.ImageUpload
			err := json.Unmarshal([]byte(msgJSON), &msg)
			require.NoError(t, err, "Message %d should parse correctly", i+1)
			assert.NotEmpty(t, msg.TraceID, "Message %d should have trace_id", i+1)
		}
	})
}
