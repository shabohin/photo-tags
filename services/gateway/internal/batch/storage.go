package batch

import (
	"fmt"
	"sync"
	"time"

	"github.com/shabohin/photo-tags/pkg/models"
)

// Storage manages batch jobs in memory
type Storage struct {
	jobs map[string]*models.BatchJob
	mu   sync.RWMutex
}

// NewStorage creates a new batch job storage
func NewStorage() *Storage {
	storage := &Storage{
		jobs: make(map[string]*models.BatchJob),
	}

	// Start cleanup goroutine to remove old completed jobs
	go storage.cleanupOldJobs()

	return storage
}

// CreateJob creates a new batch job
func (s *Storage) CreateJob(jobID string, totalImages int) *models.BatchJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	job := &models.BatchJob{
		JobID:       jobID,
		Status:      models.BatchJobStatusPending,
		TotalImages: totalImages,
		Completed:   0,
		Failed:      0,
		Images:      make([]models.BatchImageStatus, 0, totalImages),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.jobs[jobID] = job
	return job
}

// AddImage adds an image to the batch job
func (s *Storage) AddImage(jobID string, image models.BatchImageStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Images = append(job.Images, image)
	return nil
}

// GetJob retrieves a batch job by ID
func (s *Storage) GetJob(jobID string) (*models.BatchJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// UpdateImageStatus updates the status of a specific image in a batch
func (s *Storage) UpdateImageStatus(jobID string, traceID string, status string, processedPath string, errorMsg string) error {
	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.UpdateImageStatus(traceID, status, processedPath, errorMsg)
	return nil
}

// ListJobs returns all batch jobs
func (s *Storage) ListJobs() []*models.BatchJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*models.BatchJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs
}

// DeleteJob deletes a batch job
func (s *Storage) DeleteJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[jobID]; !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	delete(s.jobs, jobID)
	return nil
}

// cleanupOldJobs removes completed jobs older than 24 hours
func (s *Storage) cleanupOldJobs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for jobID, job := range s.jobs {
			if job.IsComplete() && job.CompletedAt != nil {
				if now.Sub(*job.CompletedAt) > 24*time.Hour {
					delete(s.jobs, jobID)
				}
			}
		}
		s.mu.Unlock()
	}
}
