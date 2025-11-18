package stats

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/shabohin/photo-tags/pkg/database"
	"github.com/shabohin/photo-tags/pkg/logging"
)

// Handler handles statistics HTTP requests
type Handler struct {
	logger *logging.Logger
	repo   database.RepositoryInterface
}

// NewHandler creates a new statistics handler
func NewHandler(logger *logging.Logger, repo database.RepositoryInterface) *Handler {
	return &Handler{
		logger: logger,
		repo:   repo,
	}
}

// GetUserImages returns images for a specific user
func (h *Handler) GetUserImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse telegram_id from query params
	telegramIDStr := r.URL.Query().Get("telegram_id")
	if telegramIDStr == "" {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid telegram_id", http.StatusBadRequest)
		return
	}

	// Parse pagination params
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get images from database
	images, err := h.repo.GetImagesByUser(r.Context(), telegramID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get user images", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"images": images,
		"count":  len(images),
		"limit":  limit,
		"offset": offset,
	}); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// GetUserStats returns statistics for a specific user
func (h *Handler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse telegram_id from query params
	telegramIDStr := r.URL.Query().Get("telegram_id")
	if telegramIDStr == "" {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid telegram_id", http.StatusBadRequest)
		return
	}

	// Get stats from database
	stats, err := h.repo.GetUserStats(r.Context(), telegramID)
	if err != nil {
		h.logger.Error("Failed to get user stats", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"telegram_id": telegramID,
		"stats":       stats,
	}); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// GetDailyStats returns daily processing statistics
func (h *Handler) GetDailyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse date range from query params
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		// Default to last 7 days
		startDate = time.Now().AddDate(0, 0, -7)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	}

	// Get stats from database
	stats, err := h.repo.GetDailyStats(r.Context(), startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get daily stats", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":      stats,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// GetRecentErrors returns recent errors with optional service filter
func (h *Handler) GetRecentErrors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse service filter
	var service *string
	if s := r.URL.Query().Get("service"); s != "" {
		service = &s
	}

	// Parse limit
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Get errors from database
	errors, err := h.repo.GetRecentErrors(r.Context(), service, limit)
	if err != nil {
		h.logger.Error("Failed to get recent errors", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": errors,
		"count":  len(errors),
		"limit":  limit,
	}); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// GetErrorStats returns error statistics grouped by type
func (h *Handler) GetErrorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse date range from query params
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		// Default to last 7 days
		startDate = time.Now().AddDate(0, 0, -7)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	}

	if endDateStr == "" {
		endDate = time.Now()
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end_date format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
	}

	// Get stats from database
	stats, err := h.repo.GetErrorStats(r.Context(), startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get error stats", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":      stats,
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
	}); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// GetImageByTraceID returns image details by trace ID
func (h *Handler) GetImageByTraceID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse trace_id from query params
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		http.Error(w, "trace_id is required", http.StatusBadRequest)
		return
	}

	// Get image from database
	image, err := h.repo.GetImageByTraceID(r.Context(), traceID)
	if err != nil {
		h.logger.Error("Failed to get image by trace ID", err)
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(image); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}
