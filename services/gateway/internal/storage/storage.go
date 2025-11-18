package storage

import (
	"sync"
	"time"

	"github.com/shabohin/photo-tags/pkg/models"
)

// ImageRecord represents an image processing record
type ImageRecord struct {
	ID               string
	TraceID          string
	GroupID          string
	OriginalFilename string
	OriginalPath     string
	ProcessedPath    string
	Status           string // uploading, analyzing, processing, completed, failed
	Metadata         *models.Metadata
	UploadedAt       time.Time
	CompletedAt      *time.Time
	Error            string
}

// ImageStorage provides in-memory storage for image records
type ImageStorage struct {
	mu      sync.RWMutex
	byID    map[string]*ImageRecord
	byTrace map[string]*ImageRecord
	list    []*ImageRecord
}

// NewImageStorage creates a new ImageStorage
func NewImageStorage() *ImageStorage {
	return &ImageStorage{
		byID:    make(map[string]*ImageRecord),
		byTrace: make(map[string]*ImageRecord),
		list:    make([]*ImageRecord, 0),
	}
}

// Add adds a new image record
func (s *ImageStorage) Add(record *ImageRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byID[record.ID] = record
	s.byTrace[record.TraceID] = record
	s.list = append([]*ImageRecord{record}, s.list...) // Prepend to show newest first
}

// GetByID retrieves an image record by ID
func (s *ImageStorage) GetByID(id string) (*ImageRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.byID[id]
	return record, exists
}

// GetByTraceID retrieves an image record by trace ID
func (s *ImageStorage) GetByTraceID(traceID string) (*ImageRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.byTrace[traceID]
	return record, exists
}

// UpdateStatus updates the status of an image record
func (s *ImageStorage) UpdateStatus(traceID, status string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.byTrace[traceID]
	if !exists {
		return false
	}

	record.Status = status
	return true
}

// UpdateMetadata updates the metadata and marks as completed
func (s *ImageStorage) UpdateMetadata(traceID string, metadata *models.Metadata, processedPath string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.byTrace[traceID]
	if !exists {
		return false
	}

	record.Metadata = metadata
	record.ProcessedPath = processedPath
	record.Status = "completed"
	now := time.Now()
	record.CompletedAt = &now

	return true
}

// MarkFailed marks an image as failed with an error message
func (s *ImageStorage) MarkFailed(traceID, errorMsg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.byTrace[traceID]
	if !exists {
		return false
	}

	record.Status = "failed"
	record.Error = errorMsg
	now := time.Now()
	record.CompletedAt = &now

	return true
}

// GetAll returns all image records
func (s *ImageStorage) GetAll() []*ImageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]*ImageRecord, len(s.list))
	copy(result, s.list)
	return result
}

// Count returns the total number of records
func (s *ImageStorage) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.list)
}
