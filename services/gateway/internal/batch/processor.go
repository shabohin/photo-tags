package batch

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/shabohin/photo-tags/pkg/storage"
)

// Processor handles batch processing operations
type Processor struct {
	storage       *Storage
	minioClient   storage.MinIOInterface
	rabbitmqClient messaging.RabbitMQInterface
	wsHub         *Hub
	logger        *logging.Logger
	httpClient    *http.Client
}

// NewProcessor creates a new batch processor
func NewProcessor(
	storage *Storage,
	minioClient storage.MinIOInterface,
	rabbitmqClient messaging.RabbitMQInterface,
	wsHub *Hub,
	logger *logging.Logger,
) *Processor {
	return &Processor{
		storage:       storage,
		minioClient:   minioClient,
		rabbitmqClient: rabbitmqClient,
		wsHub:         wsHub,
		logger:        logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateBatchJob creates a new batch processing job
func (p *Processor) CreateBatchJob(ctx context.Context, images []models.ImageSource) (*models.BatchJob, error) {
	// Generate job ID
	jobID := uuid.New().String()

	// Create job in storage
	job := p.storage.CreateJob(jobID, len(images))

	p.logger.Info("Created batch job", map[string]interface{}{
		"job_id":       jobID,
		"total_images": len(images),
	})

	// Process images asynchronously
	go p.processBatchImages(ctx, job, images)

	return job, nil
}

// processBatchImages processes all images in a batch
func (p *Processor) processBatchImages(ctx context.Context, job *models.BatchJob, images []models.ImageSource) {
	for i, imageSource := range images {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			p.logger.Info("Batch processing canceled", map[string]interface{}{
				"job_id": job.JobID,
			})
			return
		default:
		}

		// Generate trace ID for this image
		traceID := uuid.New().String()
		groupID := uuid.New().String()

		// Determine filename
		filename := imageSource.Name
		if filename == "" {
			if imageSource.URL != "" {
				filename = fmt.Sprintf("url_image_%d_%s.jpg", i, traceID[:8])
			} else {
				filename = fmt.Sprintf("base64_image_%d_%s.jpg", i, traceID[:8])
			}
		}

		// Add image to job
		imageStatus := models.BatchImageStatus{
			Index:            i,
			OriginalFilename: filename,
			Status:           "pending",
			TraceID:          traceID,
		}
		p.storage.AddImage(job.JobID, imageStatus)

		// Process the image
		p.processImage(ctx, job.JobID, traceID, groupID, imageSource, filename)

		// Send progress update
		p.sendProgressUpdate(job.JobID, "progress", nil)
	}
}

// processImage processes a single image in the batch
func (p *Processor) processImage(
	ctx context.Context,
	jobID string,
	traceID string,
	groupID string,
	imageSource models.ImageSource,
	filename string,
) {
	// Update status to processing
	p.storage.UpdateImageStatus(jobID, traceID, "processing", "", "")
	p.sendProgressUpdate(jobID, "progress", nil)

	// Download or decode image data
	imageData, err := p.getImageData(imageSource)
	if err != nil {
		p.handleImageError(jobID, traceID, fmt.Sprintf("Failed to get image data: %v", err))
		return
	}

	// Upload to MinIO
	objectPath := fmt.Sprintf("%s/%s", traceID, filename)
	err = p.minioClient.UploadFile(ctx, storage.BucketOriginal, objectPath, bytes.NewReader(imageData), "application/octet-stream")
	if err != nil {
		p.handleImageError(jobID, traceID, fmt.Sprintf("Failed to upload to MinIO: %v", err))
		return
	}

	p.logger.Info("Uploaded image to MinIO", map[string]interface{}{
		"job_id":   jobID,
		"trace_id": traceID,
		"path":     objectPath,
	})

	// Publish to RabbitMQ
	message := models.ImageUpload{
		Timestamp:        time.Now(),
		TraceID:          traceID,
		GroupID:          groupID,
		TelegramUsername: fmt.Sprintf("batch_%s", jobID),
		OriginalFilename: filename,
		OriginalPath:     objectPath,
		TelegramID:       0, // Special value for batch processing
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		p.handleImageError(jobID, traceID, fmt.Sprintf("Failed to marshal message: %v", err))
		return
	}

	err = p.rabbitmqClient.PublishMessage(messaging.QueueImageUpload, messageData)
	if err != nil {
		p.handleImageError(jobID, traceID, fmt.Sprintf("Failed to publish to queue: %v", err))
		return
	}

	p.logger.Info("Published image to queue", map[string]interface{}{
		"job_id":   jobID,
		"trace_id": traceID,
		"queue":    messaging.QueueImageUpload,
	})
}

// getImageData retrieves image data from URL or base64
func (p *Processor) getImageData(source models.ImageSource) ([]byte, error) {
	if source.URL != "" {
		return p.downloadImageFromURL(source.URL)
	} else if source.Base64 != "" {
		return p.decodeBase64Image(source.Base64)
	}
	return nil, fmt.Errorf("no image source provided")
}

// downloadImageFromURL downloads an image from a URL
func (p *Processor) downloadImageFromURL(url string) ([]byte, error) {
	resp, err := p.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	// Read image data (limit to 10MB)
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return data, nil
}

// decodeBase64Image decodes a base64 encoded image
func (p *Processor) decodeBase64Image(base64Data string) ([]byte, error) {
	// Remove data URI prefix if present
	if strings.Contains(base64Data, ",") {
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) == 2 {
			base64Data = parts[1]
		}
	}

	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return data, nil
}

// handleImageError handles errors during image processing
func (p *Processor) handleImageError(jobID string, traceID string, errorMsg string) {
	p.logger.Error("Image processing error", fmt.Errorf("%s", errorMsg))
	p.storage.UpdateImageStatus(jobID, traceID, "failed", "", errorMsg)

	// Get updated job
	job, err := p.storage.GetJob(jobID)
	if err != nil {
		p.logger.Error("Failed to get job", err)
		return
	}

	// Find the image
	for _, img := range job.Images {
		if img.TraceID == traceID {
			p.sendProgressUpdate(jobID, "image_complete", &img)
			break
		}
	}

	// Check if job is complete
	if job.IsComplete() {
		p.sendProgressUpdate(jobID, "job_complete", nil)
	}
}

// sendProgressUpdate sends a progress update via WebSocket
func (p *Processor) sendProgressUpdate(jobID string, updateType string, image *models.BatchImageStatus) {
	job, err := p.storage.GetJob(jobID)
	if err != nil {
		p.logger.Error("Failed to get job for progress update", err)
		return
	}

	update := &models.WSProgressUpdate{
		Type:        updateType,
		JobID:       jobID,
		Status:      string(job.Status),
		Progress:    job.GetProgress(),
		Completed:   job.Completed,
		Failed:      job.Failed,
		TotalImages: job.TotalImages,
		Image:       image,
		Timestamp:   time.Now(),
	}

	p.wsHub.BroadcastProgress(update)
}

// StartProcessedImageConsumer starts consuming processed image messages
func (p *Processor) StartProcessedImageConsumer(ctx context.Context) error {
	handler := func(msg []byte) error {
		var processed models.ImageProcessed
		if err := json.Unmarshal(msg, &processed); err != nil {
			p.logger.Error("Failed to unmarshal processed message", err)
			return err
		}

		// Check if this is a batch job (TelegramID == 0)
		if processed.TelegramID == 0 {
			p.handleProcessedImage(processed)
		}
		return nil
	}

	go func() {
		if err := p.rabbitmqClient.ConsumeMessages(messaging.QueueImageProcessed, handler); err != nil {
			p.logger.Error("Failed to consume messages", err)
		}
	}()

	return nil
}

// handleProcessedImage handles a processed image from the queue
func (p *Processor) handleProcessedImage(processed models.ImageProcessed) {
	// Find which job this image belongs to
	jobs := p.storage.ListJobs()
	for _, job := range jobs {
		for _, img := range job.Images {
			if img.TraceID == processed.TraceID {
				// Update image status
				status := "completed"
				if processed.Status == "failed" {
					status = "failed"
				}
				p.storage.UpdateImageStatus(job.JobID, processed.TraceID, status, processed.ProcessedPath, processed.Error)

				// Send progress update
				updatedJob, _ := p.storage.GetJob(job.JobID)
				for _, updatedImg := range updatedJob.Images {
					if updatedImg.TraceID == processed.TraceID {
						p.sendProgressUpdate(job.JobID, "image_complete", &updatedImg)
						break
					}
				}

				// Check if job is complete
				if updatedJob.IsComplete() {
					p.sendProgressUpdate(job.JobID, "job_complete", nil)
				}

				return
			}
		}
	}
}
