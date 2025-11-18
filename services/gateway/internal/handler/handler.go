package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shabohin/photo-tags/pkg/database"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/gateway/internal/batch"
	"github.com/shabohin/photo-tags/services/gateway/internal/config"
	"github.com/shabohin/photo-tags/services/gateway/internal/stats"
	imagestorage "github.com/shabohin/photo-tags/services/gateway/internal/storage"
)

// Handler handles HTTP requests
type Handler struct {
	logger       *logging.Logger
	cfg          *config.Config
	templates    *template.Template
	imageStorage *imagestorage.ImageStorage
	minioClient  storage.MinIOInterface
	rabbitMQ     messaging.RabbitMQInterface
	batchHandler *batch.Handler
	adminHandler *AdminHandler
	statsHandler *stats.Handler
}

// NewHandler creates a new Handler
func NewHandler(
	logger *logging.Logger,
	cfg *config.Config,
	minioClient storage.MinIOInterface,
	rabbitmqClient messaging.RabbitMQInterface,
	batchHandler *batch.Handler,
	repo database.RepositoryInterface,
) *Handler {
	// Load templates
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"subtract": func(a, b int) int {
			return a - b
		},
	}).ParseGlob("web/templates/*.html"))

	var statsHandler *stats.Handler
	if repo != nil {
		statsHandler = stats.NewHandler(logger, repo)
	}

	return &Handler{
		logger:       logger,
		cfg:          cfg,
		templates:    tmpl,
		imageStorage: imagestorage.NewImageStorage(),
		minioClient:  minioClient,
		rabbitMQ:     rabbitmqClient,
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

	// Health check
	mux.HandleFunc("/health", h.HealthCheck)

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Web pages
	mux.HandleFunc("/", h.IndexPage)
	mux.HandleFunc("/gallery", h.GalleryPage)
	mux.HandleFunc("/image/", h.ImageDetailsPage)

	// API endpoints for web UI
	mux.HandleFunc("/api/upload", h.UploadImage)
	mux.HandleFunc("/api/status/", h.GetStatus)
	mux.HandleFunc("/api/images", h.GetImages)
	mux.HandleFunc("/api/image/", h.HandleImageAPI)

	// Add batch API routes if batch handler is configured
	if h.batchHandler != nil {
		h.batchHandler.SetupRoutes(mux)
	}

	// Admin routes
	if h.adminHandler != nil {
		mux.HandleFunc("/admin/failed-jobs", h.adminHandler.FailedJobsUI)
		mux.HandleFunc("/admin/failed-jobs/api", h.adminHandler.GetFailedJobs)
		mux.HandleFunc("/admin/failed-jobs/requeue", h.adminHandler.RequeueFailedJob)
	}

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

// IndexPage renders the main upload page
func (h *Handler) IndexPage(w http.ResponseWriter, _ *http.Request) {
	data := map[string]interface{}{
		"Title":     "Upload",
		"ActiveTab": "upload",
	}

	if err := h.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		h.logger.Error("Failed to render template", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// GalleryPage renders the gallery page
func (h *Handler) GalleryPage(w http.ResponseWriter, _ *http.Request) {
	images := h.imageStorage.GetAll()

	data := map[string]interface{}{
		"Title":       "Gallery",
		"ActiveTab":   "gallery",
		"Images":      h.formatImagesForTemplate(images),
		"TotalImages": len(images),
	}

	if err := h.templates.ExecuteTemplate(w, "layout.html", data); err != nil {
		h.logger.Error("Failed to render template", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ImageDetailsPage renders the image details page
func (h *Handler) ImageDetailsPage(w http.ResponseWriter, r *http.Request) {
	imageID := strings.TrimPrefix(r.URL.Path, "/image/")

	record, exists := h.imageStorage.GetByID(imageID)
	if !exists {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title": "Image Details",
		"Image": h.formatImageForTemplate(record),
	}

	if err := h.templates.ExecuteTemplate(w, "details.html", data); err != nil {
		h.logger.Error("Failed to render template", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// UploadImage handles image upload
func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 50MB)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		h.logger.Error("Failed to parse multipart form", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		h.logger.Error("Failed to get file from form", err)
		http.Error(w, "No image provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		http.Error(w, "Only JPEG and PNG images are supported", http.StatusBadRequest)
		return
	}

	// Generate IDs
	traceID := uuid.New().String()
	groupID := uuid.New().String()
	imageID := uuid.New().String()
	objectName := fmt.Sprintf("%s/%s%s", groupID, imageID, ext)

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		h.logger.Error("Failed to read file", err)
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Upload to MinIO
	ctx := context.Background()
	if err := h.minioClient.UploadFile(ctx, storage.BucketOriginal, objectName, bytes.NewReader(fileContent), int64(len(fileContent))); err != nil {
		h.logger.Error("Failed to upload to MinIO", err)
		http.Error(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}

	// Create image record
	record := &imagestorage.ImageRecord{
		ID:               imageID,
		TraceID:          traceID,
		GroupID:          groupID,
		OriginalFilename: header.Filename,
		OriginalPath:     objectName,
		Status:           "analyzing",
		UploadedAt:       time.Now(),
	}
	h.imageStorage.Add(record)

	// Send message to RabbitMQ
	message := models.ImageUpload{
		Timestamp:        time.Now(),
		TraceID:          traceID,
		GroupID:          groupID,
		OriginalFilename: header.Filename,
		OriginalPath:     objectName,
		TelegramID:       0, // Web upload, no Telegram ID
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal message", err)
		http.Error(w, "Failed to process upload", http.StatusInternalServerError)
		return
	}

	if err := h.rabbitMQ.PublishMessage(messaging.QueueImageUpload, messageBytes); err != nil {
		h.logger.Error("Failed to publish message", err)
		http.Error(w, "Failed to process upload", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Image uploaded successfully", map[string]interface{}{
		"trace_id": traceID,
		"image_id": imageID,
		"filename": header.Filename,
	})

	// Return status template
	h.renderStatusTemplate(w, record)
}

// GetStatus returns the current status of an image
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	traceID := strings.TrimPrefix(r.URL.Path, "/api/status/")

	record, exists := h.imageStorage.GetByTraceID(traceID)
	if !exists {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	h.renderStatusTemplate(w, record)
}

// GetImages returns the list of images (for gallery)
func (h *Handler) GetImages(w http.ResponseWriter, _ *http.Request) {
	images := h.imageStorage.GetAll()

	data := map[string]interface{}{
		"Images":      h.formatImagesForTemplate(images),
		"TotalImages": len(images),
	}

	if err := h.templates.ExecuteTemplate(w, "gallery.html", data); err != nil {
		h.logger.Error("Failed to render template", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleImageAPI handles all image-related API endpoints
func (h *Handler) HandleImageAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/image/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	imageID := parts[0]
	action := parts[1]

	record, exists := h.imageStorage.GetByID(imageID)
	if !exists {
		http.NotFound(w, r)
		return
	}

	switch action {
	case "thumbnail", "view":
		h.serveImage(w, r, record, action)
	case "download":
		h.downloadProcessed(w, r, record)
	case "download-original":
		h.downloadOriginal(w, r, record)
	default:
		http.NotFound(w, r)
	}
}

// serveImage serves an image file
func (h *Handler) serveImage(w http.ResponseWriter, _ *http.Request, record *imagestorage.ImageRecord, imageType string) {
	ctx := context.Background()

	// Determine which path to use
	path := record.OriginalPath
	if imageType == "view" && record.ProcessedPath != "" {
		path = record.ProcessedPath
	}

	// Determine bucket
	bucket := storage.BucketOriginal
	if record.ProcessedPath != "" && imageType == "view" {
		bucket = storage.BucketProcessed
	}

	// Get file from MinIO
	reader, err := h.minioClient.GetFile(ctx, bucket, path)
	if err != nil {
		h.logger.Error("Failed to get file from MinIO", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer reader.Close()

	// Set content type
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Copy file to response
	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("Failed to write image response", err)
	}
}

// downloadProcessed downloads the processed image
func (h *Handler) downloadProcessed(w http.ResponseWriter, r *http.Request, record *imagestorage.ImageRecord) {
	if record.ProcessedPath == "" {
		http.Error(w, "Processed image not available", http.StatusNotFound)
		return
	}

	ctx := context.Background()
	reader, err := h.minioClient.GetFile(ctx, storage.BucketProcessed, record.ProcessedPath)
	if err != nil {
		h.logger.Error("Failed to get processed file from MinIO", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer reader.Close()

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"processed_%s\"", record.OriginalFilename))
	w.Header().Set("Content-Type", "application/octet-stream")

	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("Failed to write download response", err)
	}
}

// downloadOriginal downloads the original image
func (h *Handler) downloadOriginal(w http.ResponseWriter, r *http.Request, record *imagestorage.ImageRecord) {
	ctx := context.Background()
	reader, err := h.minioClient.GetFile(ctx, storage.BucketOriginal, record.OriginalPath)
	if err != nil {
		h.logger.Error("Failed to get original file from MinIO", err)
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer reader.Close()

	// Set headers for download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", record.OriginalFilename))
	w.Header().Set("Content-Type", "application/octet-stream")

	if _, err := io.Copy(w, reader); err != nil {
		h.logger.Error("Failed to write download response", err)
	}
}

// renderStatusTemplate renders the status template
func (h *Handler) renderStatusTemplate(w http.ResponseWriter, record *imagestorage.ImageRecord) {
	data := map[string]interface{}{
		"TraceID":    record.TraceID,
		"ImageID":    record.ID,
		"Status":     record.Status,
		"Error":      record.Error,
		"Metadata":   record.Metadata,
		"IsComplete": record.Status == "completed" || record.Status == "failed",
	}

	if err := h.templates.ExecuteTemplate(w, "status.html", data); err != nil {
		h.logger.Error("Failed to render status template", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// formatImagesForTemplate formats images for template rendering
func (h *Handler) formatImagesForTemplate(records []*imagestorage.ImageRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(records))
	for i, record := range records {
		result[i] = h.formatImageForTemplate(record)
	}
	return result
}

// formatImageForTemplate formats a single image for template rendering
func (h *Handler) formatImageForTemplate(record *imagestorage.ImageRecord) map[string]interface{} {
	formatted := map[string]interface{}{
		"ID":               record.ID,
		"TraceID":          record.TraceID,
		"OriginalFilename": record.OriginalFilename,
		"Status":           record.Status,
		"UploadedAt":       record.UploadedAt.Format("2006-01-02 15:04:05"),
		"Error":            record.Error,
	}

	if record.CompletedAt != nil {
		formatted["CompletedAt"] = record.CompletedAt.Format("2006-01-02 15:04:05")
	}

	if record.Metadata != nil {
		formatted["Metadata"] = map[string]interface{}{
			"Title":       record.Metadata.Title,
			"Description": record.Metadata.Description,
			"Keywords":    record.Metadata.Keywords,
		}
	}

	return formatted
}

// GetImageStorage returns the image storage (for consumer)
func (h *Handler) GetImageStorage() *imagestorage.ImageStorage {
	return h.imageStorage
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
