package batch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/models"
)

// Handler handles batch API requests
type Handler struct {
	processor *Processor
	storage   *Storage
	wsHub     *Hub
	logger    *logging.Logger
}

// NewHandler creates a new batch handler
func NewHandler(processor *Processor, storage *Storage, wsHub *Hub, logger *logging.Logger) *Handler {
	return &Handler{
		processor: processor,
		storage:   storage,
		wsHub:     wsHub,
		logger:    logger,
	}
}

// CreateBatch handles POST /api/v1/batch
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request body
	var req models.BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	// Validate request
	if len(req.Images) == 0 {
		h.sendError(w, http.StatusBadRequest, "No images provided")
		return
	}

	if len(req.Images) > 100 {
		h.sendError(w, http.StatusBadRequest, "Maximum 100 images per batch")
		return
	}

	// Validate each image source
	for i, img := range req.Images {
		if img.URL == "" && img.Base64 == "" {
			h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Image %d: either url or base64 must be provided", i))
			return
		}
		if img.URL != "" && img.Base64 != "" {
			h.sendError(w, http.StatusBadRequest, fmt.Sprintf("Image %d: only one of url or base64 should be provided", i))
			return
		}
	}

	// Create batch job
	job, err := h.processor.CreateBatchJob(r.Context(), req.Images)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create batch job: %v", err))
		return
	}

	// Send response
	response := models.BatchCreateResponse{
		JobID:     job.JobID,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt,
		Message:   fmt.Sprintf("Batch job created with %d images", job.TotalImages),
	}

	h.sendJSON(w, http.StatusCreated, response)

	h.logger.Info("Batch job created", map[string]interface{}{
		"job_id":       job.JobID,
		"total_images": job.TotalImages,
	})
}

// GetBatchStatus handles GET /api/v1/batch/{job_id}
func (h *Handler) GetBatchStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract job ID from URL
	jobID := h.extractJobID(r.URL.Path)
	if jobID == "" {
		h.sendError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	// Get job
	job, err := h.storage.GetJob(jobID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Job not found: %v", err))
		return
	}

	// Build response
	response := models.BatchStatusResponse{
		JobID:       job.JobID,
		Status:      string(job.Status),
		TotalImages: job.TotalImages,
		Completed:   job.Completed,
		Failed:      job.Failed,
		Progress:    job.GetProgress(),
		Images:      job.Images,
		CreatedAt:   job.CreatedAt,
		UpdatedAt:   job.UpdatedAt,
		CompletedAt: job.CompletedAt,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetBatchWebSocket handles WebSocket connections for batch updates
func (h *Handler) GetBatchWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract job ID from URL
	jobID := h.extractJobID(r.URL.Path)
	if jobID == "" {
		h.sendError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	// Check if job exists
	_, err := h.storage.GetJob(jobID)
	if err != nil {
		h.sendError(w, http.StatusNotFound, fmt.Sprintf("Job not found: %v", err))
		return
	}

	// Upgrade to WebSocket
	h.wsHub.ServeWS(w, r, jobID)
}

// ListBatches handles GET /api/v1/batch
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jobs := h.storage.ListJobs()

	// Build response
	responses := make([]models.BatchStatusResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, models.BatchStatusResponse{
			JobID:       job.JobID,
			Status:      string(job.Status),
			TotalImages: job.TotalImages,
			Completed:   job.Completed,
			Failed:      job.Failed,
			Progress:    job.GetProgress(),
			Images:      job.Images,
			CreatedAt:   job.CreatedAt,
			UpdatedAt:   job.UpdatedAt,
			CompletedAt: job.CompletedAt,
		})
	}

	h.sendJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  responses,
		"total": len(responses),
	})
}

// SetupRoutes sets up batch API routes
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateBatch(w, r)
		} else if r.Method == http.MethodGet && r.URL.Path == "/api/v1/batch" {
			h.ListBatches(w, r)
		} else {
			h.sendError(w, http.StatusNotFound, "Not found")
		}
	})

	mux.HandleFunc("/api/v1/batch/", func(w http.ResponseWriter, r *http.Request) {
		// Check for WebSocket upgrade
		if strings.HasSuffix(r.URL.Path, "/ws") {
			h.GetBatchWebSocket(w, r)
		} else {
			h.GetBatchStatus(w, r)
		}
	})
}

// extractJobID extracts job ID from URL path
func (h *Handler) extractJobID(path string) string {
	// Remove /ws suffix if present
	path = strings.TrimSuffix(path, "/ws")

	// Extract job ID from path like /api/v1/batch/{job_id}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "batch" {
		return parts[3]
	}
	return ""
}

// sendJSON sends a JSON response
func (h *Handler) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", err)
	}
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, message string) {
	h.sendJSON(w, statusCode, map[string]interface{}{
		"error":   message,
		"status":  statusCode,
	})
}
