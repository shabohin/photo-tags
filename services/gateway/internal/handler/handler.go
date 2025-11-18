package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shabohin/photo-tags/pkg/database"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/services/gateway/internal/batch"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
	"github.com/shabohin/photo-tags/services/gateway/internal/stats"
)

// Handler handles HTTP requests
type Handler struct {
	logger       *logging.Logger
	cfg          *config.Config
	batchHandler *batch.Handler
	adminHandler *AdminHandler
	statsHandler *stats.Handler
}

// NewHandler creates a new Handler
func NewHandler(logger *logging.Logger, cfg *config.Config, batchHandler *batch.Handler, rabbitmqClient messaging.RabbitMQInterface, repo database.RepositoryInterface) *Handler {
	var statsHandler *stats.Handler
	if repo != nil {
		statsHandler = stats.NewHandler(logger, repo)
	}

	return &Handler{
		logger:       logger,
		cfg:          cfg,
		batchHandler: batchHandler,
		adminHandler: NewAdminHandler(logger, rabbitmqClient),
		statsHandler: statsHandler,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "gateway",
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Write response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to write response", err)
	}
}

// SetupRoutes sets up HTTP routes
func (h *Handler) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Health check route
	mux.HandleFunc("/health", h.HealthCheck)

	// Add batch API routes if batch handler is configured
	if h.batchHandler != nil {
		h.batchHandler.SetupRoutes(mux)
	}

	// Admin routes
	mux.HandleFunc("/admin/failed-jobs", h.adminHandler.FailedJobsUI)
	mux.HandleFunc("/admin/failed-jobs/api", h.adminHandler.GetFailedJobs)
	mux.HandleFunc("/admin/failed-jobs/requeue", h.adminHandler.RequeueFailedJob)

	// Statistics API routes
	if h.statsHandler != nil {
		mux.HandleFunc("/api/v1/stats/user/images", h.statsHandler.GetUserImages)
		mux.HandleFunc("/api/v1/stats/user/summary", h.statsHandler.GetUserStats)
		mux.HandleFunc("/api/v1/stats/daily", h.statsHandler.GetDailyStats)
		mux.HandleFunc("/api/v1/stats/errors", h.statsHandler.GetRecentErrors)
		mux.HandleFunc("/api/v1/stats/errors/summary", h.statsHandler.GetErrorStats)
		mux.HandleFunc("/api/v1/images/trace", h.statsHandler.GetImageByTraceID)
	}

	// Log middleware
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.logger.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path), nil)

		mux.ServeHTTP(w, r)
		h.logger.Info(fmt.Sprintf("Completed in %v", time.Since(start)), nil)
	})
}

// StartServer starts the HTTP server
func (h *Handler) StartServer(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", h.cfg.ServerPort),
		Handler: h.SetupRoutes(),
	}

	h.logger.Info(fmt.Sprintf("Starting HTTP server on port %d", h.cfg.ServerPort), nil)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.logger.Error("Server error", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
