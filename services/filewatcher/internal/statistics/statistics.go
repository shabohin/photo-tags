package statistics

import (
	"sync"
	"time"
)

// Statistics tracks processing statistics
type Statistics struct {
	mu                sync.RWMutex
	FilesProcessed    int64     `json:"files_processed"`
	FilesSuccessful   int64     `json:"files_successful"`
	FilesFailed       int64     `json:"files_failed"`
	FilesReceived     int64     `json:"files_received"`
	LastProcessedTime time.Time `json:"last_processed_time"`
	StartTime         time.Time `json:"start_time"`
	Errors            []Error   `json:"recent_errors"`
	maxErrors         int
}

// Error represents an error that occurred during processing
type Error struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// NewStatistics creates a new Statistics instance
func NewStatistics() *Statistics {
	return &Statistics{
		StartTime: time.Now(),
		Errors:    make([]Error, 0, 10),
		maxErrors: 10,
	}
}

// IncrementProcessed increments the files processed counter
func (s *Statistics) IncrementProcessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesProcessed++
	s.LastProcessedTime = time.Now()
}

// IncrementSuccessful increments the successful files counter
func (s *Statistics) IncrementSuccessful() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesSuccessful++
}

// IncrementFailed increments the failed files counter
func (s *Statistics) IncrementFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesFailed++
}

// IncrementReceived increments the received files counter
func (s *Statistics) IncrementReceived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FilesReceived++
}

// AddError adds an error to the statistics
func (s *Statistics) AddError(message, traceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := Error{
		Timestamp: time.Now(),
		Message:   message,
		TraceID:   traceID,
	}

	// Keep only last N errors
	s.Errors = append(s.Errors, err)
	if len(s.Errors) > s.maxErrors {
		s.Errors = s.Errors[len(s.Errors)-s.maxErrors:]
	}
}

// GetSnapshot returns a snapshot of current statistics
func (s *Statistics) GetSnapshot() Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid race conditions
	errorsCopy := make([]Error, len(s.Errors))
	copy(errorsCopy, s.Errors)

	return Statistics{
		FilesProcessed:    s.FilesProcessed,
		FilesSuccessful:   s.FilesSuccessful,
		FilesFailed:       s.FilesFailed,
		FilesReceived:     s.FilesReceived,
		LastProcessedTime: s.LastProcessedTime,
		StartTime:         s.StartTime,
		Errors:            errorsCopy,
	}
}
