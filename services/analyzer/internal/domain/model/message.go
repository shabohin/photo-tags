package model

import (
	"time"
)

type ImageUploadMessage struct {
	Timestamp        time.Time `json:"timestamp"`
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	TelegramID       int64     `json:"telegram_id"`
}

type MetadataGeneratedMessage struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Timestamp        time.Time `json:"timestamp"`
	Metadata         Metadata  `json:"metadata"`
	TelegramID       int64     `json:"telegram_id"`
}
