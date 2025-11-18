package models

import "time"

// ImageUpload represents a message for the image_upload queue
type ImageUpload struct {
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	TelegramID       int64     `json:"telegram_id"`
}

// MetadataGenerated represents a message for the metadata_generated queue
type MetadataGenerated struct {
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Metadata         Metadata  `json:"metadata"`
	TelegramID       int64     `json:"telegram_id"`
}

// ImageProcess represents a message for the image_process queue
type ImageProcess struct {
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	ProcessedPath    string    `json:"processed_path"`
	Metadata         Metadata  `json:"metadata"`
	TelegramID       int64     `json:"telegram_id"`
}

// ImageProcessed represents a message for the image_processed queue
type ImageProcessed struct {
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	ProcessedPath    string    `json:"processed_path"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	TelegramID       int64     `json:"telegram_id"`
}

// Metadata represents image metadata
type Metadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

// FailedJob represents a failed message in the dead letter queue
type FailedJob struct {
	ID            string    `json:"id"`
	OriginalQueue string    `json:"original_queue"`
	MessageBody   string    `json:"message_body"`
	ErrorReason   string    `json:"error_reason"`
	FailedAt      time.Time `json:"failed_at"`
	RetryCount    int       `json:"retry_count"`
	LastRetryAt   time.Time `json:"last_retry_at,omitempty"`
}
