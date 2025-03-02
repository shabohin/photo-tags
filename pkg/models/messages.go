package models

import "time"

// ImageUpload represents a message for the image_upload queue
type ImageUpload struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Timestamp        time.Time `json:"timestamp"`
}

// MetadataGenerated represents a message for the metadata_generated queue
type MetadataGenerated struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Metadata         Metadata  `json:"metadata"`
	Timestamp        time.Time `json:"timestamp"`
}

// ImageProcess represents a message for the image_process queue
type ImageProcess struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	ProcessedPath    string    `json:"processed_path"`
	Metadata         Metadata  `json:"metadata"`
	Timestamp        time.Time `json:"timestamp"`
}

// ImageProcessed represents a message for the image_processed queue
type ImageProcessed struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	ProcessedPath    string    `json:"processed_path"`
	Status           string    `json:"status"`
	Error            string    `json:"error,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// Metadata represents image metadata
type Metadata struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}
