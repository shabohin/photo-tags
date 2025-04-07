package model

import (
	"time"
)

type ImageUploadMessage struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	TelegramUsername string    `json:"telegram_username"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Timestamp        time.Time `json:"timestamp"`
}

type MetadataGeneratedMessage struct {
	TraceID          string    `json:"trace_id"`
	GroupID          string    `json:"group_id"`
	TelegramID       int64     `json:"telegram_id"`
	OriginalFilename string    `json:"original_filename"`
	OriginalPath     string    `json:"original_path"`
	Metadata         Metadata  `json:"metadata"`
	Timestamp        time.Time `json:"timestamp"`
}
