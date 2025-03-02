package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
)

// Handler handles HTTP requests
type Handler struct {
	logger *logging.Logger
	cfg    *config.Config
}

// NewHandler creates a new Handler
func NewHandler(logger *logging.Logger, cfg *config.Config) *Handler {
	return &Handler{
		logger: logger,
		cfg:    cfg,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
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

	// Add routes
	mux.HandleFunc("/health", h.HealthCheck)

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
