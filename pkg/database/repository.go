package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository provides methods for database operations
type Repository struct {
	client *Client
}

// NewRepository creates a new repository instance
func NewRepository(client *Client) *Repository {
	return &Repository{client: client}
}

// CreateImage inserts a new image record
func (r *Repository) CreateImage(ctx context.Context, img *Image) error {
	query := `
		INSERT INTO images (trace_id, telegram_id, telegram_username, filename, original_path, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	err := r.client.db.QueryRowContext(
		ctx, query,
		img.TraceID, img.TelegramID, img.TelegramUsername, img.Filename, img.OriginalPath, img.Status, img.Metadata,
	).Scan(&img.ID, &img.CreatedAt, &img.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	return nil
}

// UpdateImageStatus updates the status of an image
func (r *Repository) UpdateImageStatus(ctx context.Context, traceID string, status ImageStatus, errorMsg *string) error {
	query := `
		UPDATE images
		SET status = $1, error_message = $2, updated_at = CURRENT_TIMESTAMP
		WHERE trace_id = $3
	`

	result, err := r.client.db.ExecContext(ctx, query, status, errorMsg, traceID)
	if err != nil {
		return fmt.Errorf("failed to update image status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("image with trace_id %s not found", traceID)
	}

	return nil
}

// UpdateImageProcessed updates image with processed information
func (r *Repository) UpdateImageProcessed(ctx context.Context, traceID string, processedPath string, metadata *ImageMetadata, status ImageStatus) error {
	query := `
		UPDATE images
		SET processed_path = $1, metadata = $2, status = $3, updated_at = CURRENT_TIMESTAMP
		WHERE trace_id = $4
	`

	result, err := r.client.db.ExecContext(ctx, query, processedPath, metadata, status, traceID)
	if err != nil {
		return fmt.Errorf("failed to update processed image: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("image with trace_id %s not found", traceID)
	}

	return nil
}

// GetImageByTraceID retrieves an image by trace ID
func (r *Repository) GetImageByTraceID(ctx context.Context, traceID string) (*Image, error) {
	query := `
		SELECT id, trace_id, telegram_id, telegram_username, filename, original_path,
		       processed_path, status, error_message, metadata, created_at, updated_at
		FROM images
		WHERE trace_id = $1
	`

	img := &Image{}
	err := r.client.db.QueryRowContext(ctx, query, traceID).Scan(
		&img.ID, &img.TraceID, &img.TelegramID, &img.TelegramUsername, &img.Filename,
		&img.OriginalPath, &img.ProcessedPath, &img.Status, &img.ErrorMessage,
		&img.Metadata, &img.CreatedAt, &img.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("image not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return img, nil
}

// GetImagesByUser retrieves images for a specific user with pagination
func (r *Repository) GetImagesByUser(ctx context.Context, telegramID int64, limit, offset int) ([]*Image, error) {
	query := `
		SELECT id, trace_id, telegram_id, telegram_username, filename, original_path,
		       processed_path, status, error_message, metadata, created_at, updated_at
		FROM images
		WHERE telegram_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.client.db.QueryContext(ctx, query, telegramID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query images: %w", err)
	}
	defer rows.Close()

	var images []*Image
	for rows.Next() {
		img := &Image{}
		err := rows.Scan(
			&img.ID, &img.TraceID, &img.TelegramID, &img.TelegramUsername, &img.Filename,
			&img.OriginalPath, &img.ProcessedPath, &img.Status, &img.ErrorMessage,
			&img.Metadata, &img.CreatedAt, &img.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image: %w", err)
		}
		images = append(images, img)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return images, nil
}

// GetUserStats retrieves statistics for a specific user
func (r *Repository) GetUserStats(ctx context.Context, telegramID int64) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM images
		WHERE telegram_id = $1
		GROUP BY status
	`

	rows, err := r.client.db.QueryContext(ctx, query, telegramID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats: %w", err)
		}
		stats[status] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return stats, nil
}

// CreateOrUpdateDailyStats creates or updates daily processing statistics
func (r *Repository) CreateOrUpdateDailyStats(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO processing_stats (
			date, total_images, successful_images, failed_images, pending_images, total_users
		)
		SELECT
			$1::date,
			COUNT(*),
			COUNT(*) FILTER (WHERE status = 'success'),
			COUNT(*) FILTER (WHERE status = 'failed'),
			COUNT(*) FILTER (WHERE status = 'pending' OR status = 'processing'),
			COUNT(DISTINCT telegram_id)
		FROM images
		WHERE DATE(created_at) = $1::date
		ON CONFLICT (date) DO UPDATE SET
			total_images = EXCLUDED.total_images,
			successful_images = EXCLUDED.successful_images,
			failed_images = EXCLUDED.failed_images,
			pending_images = EXCLUDED.pending_images,
			total_users = EXCLUDED.total_users,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.client.db.ExecContext(ctx, query, date)
	if err != nil {
		return fmt.Errorf("failed to create/update daily stats: %w", err)
	}

	return nil
}

// GetDailyStats retrieves daily statistics for a date range
func (r *Repository) GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]*ProcessingStats, error) {
	query := `
		SELECT id, date, total_images, successful_images, failed_images, pending_images,
		       total_users, avg_processing_time_ms, created_at, updated_at
		FROM processing_stats
		WHERE date BETWEEN $1 AND $2
		ORDER BY date DESC
	`

	rows, err := r.client.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*ProcessingStats
	for rows.Next() {
		stat := &ProcessingStats{}
		err := rows.Scan(
			&stat.ID, &stat.Date, &stat.TotalImages, &stat.SuccessfulImages,
			&stat.FailedImages, &stat.PendingImages, &stat.TotalUsers,
			&stat.AvgProcessingTimeMs, &stat.CreatedAt, &stat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stat: %w", err)
		}
		stats = append(stats, stat)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return stats, nil
}

// LogError inserts an error record
func (r *Repository) LogError(ctx context.Context, err *Error) error {
	query := `
		INSERT INTO errors (trace_id, service, error_type, error_message, stack_trace, telegram_id, filename, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	queryErr := r.client.db.QueryRowContext(
		ctx, query,
		err.TraceID, err.Service, err.ErrorType, err.ErrorMessage,
		err.StackTrace, err.TelegramID, err.Filename, err.Metadata,
	).Scan(&err.ID, &err.CreatedAt)

	if queryErr != nil {
		return fmt.Errorf("failed to log error: %w", queryErr)
	}

	return nil
}

// GetRecentErrors retrieves recent errors with optional filters
func (r *Repository) GetRecentErrors(ctx context.Context, service *string, limit int) ([]*Error, error) {
	query := `
		SELECT id, trace_id, service, error_type, error_message, stack_trace,
		       telegram_id, filename, metadata, created_at
		FROM errors
		WHERE ($1::text IS NULL OR service = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.client.db.QueryContext(ctx, query, service, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query errors: %w", err)
	}
	defer rows.Close()

	var errors []*Error
	for rows.Next() {
		e := &Error{}
		err := rows.Scan(
			&e.ID, &e.TraceID, &e.Service, &e.ErrorType, &e.ErrorMessage,
			&e.StackTrace, &e.TelegramID, &e.Filename, &e.Metadata, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan error: %w", err)
		}
		errors = append(errors, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return errors, nil
}

// GetErrorStats retrieves error statistics grouped by type
func (r *Repository) GetErrorStats(ctx context.Context, startDate, endDate time.Time) (map[string]int, error) {
	query := `
		SELECT error_type, COUNT(*) as count
		FROM errors
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY error_type
		ORDER BY count DESC
	`

	rows, err := r.client.db.QueryContext(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query error stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var errorType string
		var count int
		if err := rows.Scan(&errorType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan error stat: %w", err)
		}
		stats[errorType] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return stats, nil
}
