package batch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/models"
)

type mockMinIOClient struct{}

func (m *mockMinIOClient) EnsureBucketExists(ctx context.Context, bucketName string) error {
	return nil
}

func (m *mockMinIOClient) UploadFile(ctx context.Context, bucketName, objectName string, data []byte, size int64) error {
	return nil
}

func (m *mockMinIOClient) DownloadFile(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockMinIOClient) GetPresignedURL(ctx context.Context, bucketName, objectName string) (string, error) {
	return "http://example.com/test.jpg", nil
}

type mockRabbitMQClient struct{}

func (m *mockRabbitMQClient) Publish(queueName string, data []byte) error {
	return nil
}

func (m *mockRabbitMQClient) Consume(queueName string) (<-chan []byte, error) {
	ch := make(chan []byte)
	return ch, nil
}

func (m *mockRabbitMQClient) DeclareQueue(queueName string) (interface{}, error) {
	return nil, nil
}

func (m *mockRabbitMQClient) Close() error {
	return nil
}

func setupTestHandler() *Handler {
	logger := logging.NewLogger("test")
	storage := NewStorage()
	wsHub := NewHub(logger)
	processor := NewProcessor(
		storage,
		&mockMinIOClient{},
		&mockRabbitMQClient{},
		wsHub,
		logger,
	)
	handler := NewHandler(processor, storage, wsHub, logger)
	return handler
}

func TestCreateBatch_Success(t *testing.T) {
	handler := setupTestHandler()

	reqBody := models.BatchCreateRequest{
		Images: []models.ImageSource{
			{
				URL:  "http://example.com/image1.jpg",
				Name: "image1.jpg",
			},
			{
				URL:  "http://example.com/image2.jpg",
				Name: "image2.jpg",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateBatch(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.BatchCreateResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.JobID == "" {
		t.Error("Expected job_id to be set")
	}

	if response.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", response.Status)
	}
}

func TestCreateBatch_NoImages(t *testing.T) {
	handler := setupTestHandler()

	reqBody := models.BatchCreateRequest{
		Images: []models.ImageSource{},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateBatch_TooManyImages(t *testing.T) {
	handler := setupTestHandler()

	images := make([]models.ImageSource, 101)
	for i := range images {
		images[i] = models.ImageSource{
			URL: "http://example.com/image.jpg",
		}
	}

	reqBody := models.BatchCreateRequest{
		Images: images,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateBatch_InvalidImageSource(t *testing.T) {
	handler := setupTestHandler()

	reqBody := models.BatchCreateRequest{
		Images: []models.ImageSource{
			{
				// No URL or Base64 provided
				Name: "image.jpg",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batch", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CreateBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetBatchStatus_Success(t *testing.T) {
	handler := setupTestHandler()

	// Create a job first
	job := handler.storage.CreateJob("test-job-id", 2)
	handler.storage.AddImage("test-job-id", models.BatchImageStatus{
		Index:            0,
		OriginalFilename: "image1.jpg",
		Status:           "pending",
		TraceID:          "trace-1",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/batch/test-job-id", nil)
	w := httptest.NewRecorder()

	handler.GetBatchStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response models.BatchStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.JobID != job.JobID {
		t.Errorf("Expected job_id '%s', got '%s'", job.JobID, response.JobID)
	}

	if response.TotalImages != 2 {
		t.Errorf("Expected total_images 2, got %d", response.TotalImages)
	}
}

func TestGetBatchStatus_NotFound(t *testing.T) {
	handler := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/batch/non-existent-id", nil)
	w := httptest.NewRecorder()

	handler.GetBatchStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestListBatches(t *testing.T) {
	handler := setupTestHandler()

	// Create some jobs
	handler.storage.CreateJob("job-1", 1)
	handler.storage.CreateJob("job-2", 2)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/batch", nil)
	w := httptest.NewRecorder()

	handler.ListBatches(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	total, ok := response["total"].(float64)
	if !ok {
		t.Fatal("Expected 'total' field in response")
	}

	if int(total) != 2 {
		t.Errorf("Expected 2 jobs, got %d", int(total))
	}
}

func TestExtractJobID(t *testing.T) {
	handler := setupTestHandler()

	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/batch/test-job-id", "test-job-id"},
		{"/api/v1/batch/test-job-id/ws", "test-job-id"},
		{"/api/v1/batch/abc-123-def", "abc-123-def"},
		{"/api/v1/batch/", ""},
		{"/api/v1/batch", ""},
		{"/invalid/path", ""},
	}

	for _, tt := range tests {
		result := handler.extractJobID(tt.path)
		if result != tt.expected {
			t.Errorf("extractJobID(%s) = %s, expected %s", tt.path, result, tt.expected)
		}
	}
}
