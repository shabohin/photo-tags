package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ImageStatus represents the status of image processing
type ImageStatus string

const (
	StatusPending    ImageStatus = "pending"
	StatusProcessing ImageStatus = "processing"
	StatusSuccess    ImageStatus = "success"
	StatusFailed     ImageStatus = "failed"
)

// Image represents an image record in the database
type Image struct {
	ID              int64           `json:"id"`
	TraceID         string          `json:"trace_id"`
	TelegramID      int64           `json:"telegram_id"`
	TelegramUsername *string        `json:"telegram_username,omitempty"`
	Filename        string          `json:"filename"`
	OriginalPath    *string         `json:"original_path,omitempty"`
	ProcessedPath   *string         `json:"processed_path,omitempty"`
	Status          ImageStatus     `json:"status"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	Metadata        *ImageMetadata  `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ImageMetadata represents metadata stored as JSONB
type ImageMetadata struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
}

// Value implements driver.Valuer for ImageMetadata
func (m ImageMetadata) Value() (driver.Value, error) {
	if m.Title == "" && m.Description == "" && len(m.Keywords) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for ImageMetadata
func (m *ImageMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// ProcessingStats represents daily processing statistics
type ProcessingStats struct {
	ID                  int64     `json:"id"`
	Date                time.Time `json:"date"`
	TotalImages         int       `json:"total_images"`
	SuccessfulImages    int       `json:"successful_images"`
	FailedImages        int       `json:"failed_images"`
	PendingImages       int       `json:"pending_images"`
	TotalUsers          int       `json:"total_users"`
	AvgProcessingTimeMs int64     `json:"avg_processing_time_ms"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Error represents an error record in the database
type Error struct {
	ID           int64           `json:"id"`
	TraceID      *string         `json:"trace_id,omitempty"`
	Service      string          `json:"service"`
	ErrorType    string          `json:"error_type"`
	ErrorMessage string          `json:"error_message"`
	StackTrace   *string         `json:"stack_trace,omitempty"`
	TelegramID   *int64          `json:"telegram_id,omitempty"`
	Filename     *string         `json:"filename,omitempty"`
	Metadata     *ErrorMetadata  `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ErrorMetadata represents additional error context as JSONB
type ErrorMetadata map[string]interface{}

// Value implements driver.Valuer for ErrorMetadata
func (m ErrorMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for ErrorMetadata
func (m *ErrorMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// StatsFilter represents filters for statistics queries
type StatsFilter struct {
	StartDate *time.Time
	EndDate   *time.Time
	TelegramID *int64
	Status    *ImageStatus
	Limit     int
	Offset    int
}
