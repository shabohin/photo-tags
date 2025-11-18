package database

import (
	"context"
	"time"
)

// RepositoryInterface defines the interface for database operations
type RepositoryInterface interface {
	// Image operations
	CreateImage(ctx context.Context, img *Image) error
	UpdateImageStatus(ctx context.Context, traceID string, status ImageStatus, errorMsg *string) error
	UpdateImageProcessed(ctx context.Context, traceID string, processedPath string, metadata *ImageMetadata, status ImageStatus) error
	GetImageByTraceID(ctx context.Context, traceID string) (*Image, error)
	GetImagesByUser(ctx context.Context, telegramID int64, limit, offset int) ([]*Image, error)
	GetUserStats(ctx context.Context, telegramID int64) (map[string]int, error)

	// Statistics operations
	CreateOrUpdateDailyStats(ctx context.Context, date time.Time) error
	GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]*ProcessingStats, error)

	// Error operations
	LogError(ctx context.Context, err *Error) error
	GetRecentErrors(ctx context.Context, service *string, limit int) ([]*Error, error)
	GetErrorStats(ctx context.Context, startDate, endDate time.Time) (map[string]int, error)
}
