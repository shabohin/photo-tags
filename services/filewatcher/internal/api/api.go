package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/config"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/statistics"
	"github.com/shabohin/photo-tags/services/filewatcher/internal/watcher"
)

// Server represents the HTTP API server
type Server struct {
	cfg     *config.Config
	logger  *logging.Logger
	watcher *watcher.Watcher
	stats   *statistics.Statistics
	server  *http.Server
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	logger *logging.Logger,
	watcher *watcher.Watcher,
	stats *statistics.Statistics,
) *Server {
	return &Server{
		cfg:     cfg,
		logger:  logger,
		watcher: watcher,
		stats:   stats,
	}
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/scan", s.handleScan)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.ServerPort),
		Handler: s.loggingMiddleware(mux),
	}

	s.logger.Info("Starting HTTP server", map[string]interface{}{
		"port": s.cfg.ServerPort,
	})

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server", nil)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"service": "filewatcher",
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStats handles statistics requests
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.stats.GetSnapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleScan handles manual scan trigger requests
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Info("Manual scan triggered via API", nil)

	// Trigger scan in background
	go func() {
		ctx := context.Background()
		if err := s.watcher.TriggerManualScan(ctx); err != nil {
			s.logger.Error("Manual scan failed", err)
		}
	}()

	response := map[string]interface{}{
		"status":  "started",
		"message": "Manual scan initiated",
		"time":    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request
		s.logger.Info("HTTP request", map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		})

		// Call next handler
		next.ServeHTTP(w, r)

		// Log response
		s.logger.Info("HTTP response", map[string]interface{}{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start).String(),
		})
	})
}
