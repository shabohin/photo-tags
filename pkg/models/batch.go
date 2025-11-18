package models

import (
	"sync"
	"time"
)

// BatchJobStatus represents the status of a batch job
type BatchJobStatus string

const (
	BatchJobStatusPending    BatchJobStatus = "pending"
	BatchJobStatusProcessing BatchJobStatus = "processing"
	BatchJobStatusCompleted  BatchJobStatus = "completed"
	BatchJobStatusFailed     BatchJobStatus = "failed"
	BatchJobStatusCancelled  BatchJobStatus = "cancelled"
)

// ImageSource represents the source of an image in a batch
type ImageSource struct {
	URL    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	Name   string `json:"name,omitempty"`
}

// BatchImageStatus represents the status of a single image in a batch
type BatchImageStatus struct {
	Index            int        `json:"index"`
	OriginalFilename string     `json:"original_filename"`
	Status           string     `json:"status"` // pending, processing, completed, failed
	TraceID          string     `json:"trace_id"`
	ProcessedPath    string     `json:"processed_path,omitempty"`
	Error            string     `json:"error,omitempty"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
}

// BatchJob represents a batch processing job
type BatchJob struct {
	JobID        string             `json:"job_id"`
	Status       BatchJobStatus     `json:"status"`
	TotalImages  int                `json:"total_images"`
	Completed    int                `json:"completed"`
	Failed       int                `json:"failed"`
	Images       []BatchImageStatus `json:"images"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	ErrorMessage string             `json:"error_message,omitempty"`
	mu           sync.RWMutex       `json:"-"`
}

// UpdateImageStatus updates the status of a specific image in the batch
func (b *BatchJob) UpdateImageStatus(traceID string, status string, processedPath string, errorMsg string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.Images {
		if b.Images[i].TraceID == traceID {
			b.Images[i].Status = status
			b.Images[i].ProcessedPath = processedPath

			if errorMsg != "" {
				b.Images[i].Error = errorMsg
			}

			now := time.Now()
			if status == "processing" && b.Images[i].StartTime == nil {
				b.Images[i].StartTime = &now
			}
			if status == "completed" || status == "failed" {
				b.Images[i].EndTime = &now
			}

			// Update job counters
			if status == "completed" {
				b.Completed++
			} else if status == "failed" {
				b.Failed++
			}

			// Update job status
			b.UpdatedAt = now
			if b.Completed+b.Failed == b.TotalImages {
				if b.Failed == b.TotalImages {
					b.Status = BatchJobStatusFailed
				} else {
					b.Status = BatchJobStatusCompleted
				}
				b.CompletedAt = &now
			} else if status == "processing" && b.Status == BatchJobStatusPending {
				b.Status = BatchJobStatusProcessing
			}
			break
		}
	}
}

// GetProgress returns the current progress as a percentage
func (b *BatchJob) GetProgress() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.TotalImages == 0 {
		return 0
	}
	return float64(b.Completed+b.Failed) / float64(b.TotalImages) * 100
}

// IsComplete returns true if the batch job is complete
func (b *BatchJob) IsComplete() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.Status == BatchJobStatusCompleted ||
		b.Status == BatchJobStatusFailed ||
		b.Status == BatchJobStatusCancelled
}

// BatchCreateRequest represents a request to create a batch job
type BatchCreateRequest struct {
	Images []ImageSource `json:"images"`
}

// BatchCreateResponse represents the response after creating a batch job
type BatchCreateResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message"`
}

// BatchStatusResponse represents the response for batch status queries
type BatchStatusResponse struct {
	JobID       string             `json:"job_id"`
	Status      string             `json:"status"`
	TotalImages int                `json:"total_images"`
	Completed   int                `json:"completed"`
	Failed      int                `json:"failed"`
	Progress    float64            `json:"progress"`
	Images      []BatchImageStatus `json:"images"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
}

// WSProgressUpdate represents a WebSocket progress update message
type WSProgressUpdate struct {
	Type        string             `json:"type"` // "progress", "image_complete", "job_complete"
	JobID       string             `json:"job_id"`
	Status      string             `json:"status"`
	Progress    float64            `json:"progress"`
	Completed   int                `json:"completed"`
	Failed      int                `json:"failed"`
	TotalImages int                `json:"total_images"`
	Image       *BatchImageStatus  `json:"image,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}
