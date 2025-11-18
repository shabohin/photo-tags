package openrouter

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "test prompt", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "test-model", client.model)
	assert.Equal(t, 100, client.maxTokens)
	assert.Equal(t, 0.5, client.temperature)
	assert.Equal(t, "test prompt", client.prompt)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.logger)
}

// MockTransport implements http.RoundTripper for mocking HTTP requests
type MockTransport struct {
	Response *http.Response
	Err      error
}

func (t *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Response, t.Err
}

func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestAnalyzeImage_Success(t *testing.T) {
	// Prepare mock data
	responseBody := `{
  "id": "test-id",
  "choices": [
    {
      "message": {
        "content": "` +
		`{\"title\": \"Test Title\", ` +
		`\"description\": \"Test Description\", ` +
		`\"keywords\": [\"test\", \"image\", \"analysis\"]}` + `",
        "role": "assistant"
      }
    }
  ]
}`

	// Create mocked transport
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusOK, responseBody),
	}

	// Create client and set mocked transport
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	// Test image analysis
	imageBytes := []byte("fake-image-data")
	metadata, err := client.AnalyzeImage(context.Background(), imageBytes, "test-trace-id")

	// Check results
	assert.NoError(t, err)
	assert.Equal(t, "Test Title", metadata.Title)
	assert.Equal(t, "Test Description", metadata.Description)
	assert.Equal(t, 3, len(metadata.Keywords))
	if len(metadata.Keywords) > 0 {
		assert.Equal(t, "test", metadata.Keywords[0])
	}
	if len(metadata.Keywords) > 1 {
		assert.Equal(t, "image", metadata.Keywords[1])
	}
	if len(metadata.Keywords) > 2 {
		assert.Equal(t, "analysis", metadata.Keywords[2])
	}
}

func TestAnalyzeImage_ErrorResponse(t *testing.T) {
	// Create mocked transport with error
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusInternalServerError, `{"error": "Internal Server Error"}`),
	}

	// Create client and set mocked transport
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	// Test image analysis with error response
	imageBytes := []byte("fake-image-data")
	_, err := client.AnalyzeImage(context.Background(), imageBytes, "test-trace-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestAnalyzeImage_InvalidJSON(t *testing.T) {
	// Create mocked transport with invalid JSON
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusOK, `{"invalid json`),
	}

	// Create client and set mocked transport
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	// Test image analysis with invalid JSON
	imageBytes := []byte("fake-image-data")
	_, err := client.AnalyzeImage(context.Background(), imageBytes, "test-trace-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestAnalyzeImage_EmptyChoices(t *testing.T) {
	// Prepare mock data with empty choices array
	responseBody := `{
  "id": "test-id",
  "choices": []
}`

	// Create mocked transport
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusOK, responseBody),
	}

	// Create client and set mocked transport
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	// Test image analysis with empty choices array
	imageBytes := []byte("fake-image-data")
	_, err := client.AnalyzeImage(context.Background(), imageBytes, "test-trace-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty choices in API response")
}

func TestAnalyzeImage_InvalidMetadataJSON(t *testing.T) {
	// Prepare mock data with invalid JSON in metadata
	responseBody := `{
  "id": "test-id",
  "choices": [
    {
      "message": {
        "content": "{\"title\": \"Test Title\", \"description\": \"Test Description\", \"keywords\": \"not-an-array\"}",
        "role": "assistant"
      }
    }
  ]
}`

	// Create mocked transport
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusOK, responseBody),
	}

	// Create client and set mocked transport
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	// Test image analysis with invalid JSON in metadata
	imageBytes := []byte("fake-image-data")
	_, err := client.AnalyzeImage(context.Background(), imageBytes, "test-trace-id")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse metadata")
}

func TestGetAvailableModels_Success(t *testing.T) {
	responseBody := `{
		"data": [
			{
				"id": "google/gemini-2.0-flash-exp:free",
				"name": "Gemini 2.0 Flash (free)",
				"pricing": {
					"prompt": "0",
					"completion": "0"
				},
				"context_length": 32768,
				"architecture": {
					"modality": "multimodal"
				}
			},
			{
				"id": "openai/gpt-4o",
				"name": "GPT-4o",
				"pricing": {
					"prompt": "0.005",
					"completion": "0.015"
				},
				"context_length": 128000,
				"architecture": {
					"modality": "text"
				}
			}
		]
	}`

	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusOK, responseBody),
	}

	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	models, err := client.GetAvailableModels(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 2, len(models))
	assert.Equal(t, "google/gemini-2.0-flash-exp:free", models[0].ID)
	assert.Equal(t, "0", models[0].Pricing.Prompt)
	assert.Equal(t, 32768, models[0].ContextLen)
}

func TestGetAvailableModels_ErrorResponse(t *testing.T) {
	mockTransport := &MockTransport{
		Response: newMockResponse(http.StatusInternalServerError, `{"error": "Server Error"}`),
	}

	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)
	client.httpClient = &http.Client{Transport: mockTransport}

	_, err := client.GetAvailableModels(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "models API error")
}

func TestSelectBestFreeVisionModel_Success(t *testing.T) {
	models := []Model{
		{
			ID:         "google/gemini-2.0-flash-exp:free",
			Name:       "Gemini 2.0 Flash (free)",
			Pricing:    Pricing{Prompt: "0", Completion: "0"},
			ContextLen: 32768,
			Architecture: Architecture{
				Modality: "multimodal",
			},
		},
		{
			ID:         "meta-llama/llama-3.2-11b-vision-instruct:free",
			Name:       "Llama 3.2 Vision (free)",
			Pricing:    Pricing{Prompt: "0", Completion: "0"},
			ContextLen: 8192,
			Architecture: Architecture{
				Modality: "multimodal",
			},
		},
		{
			ID:         "openai/gpt-4o",
			Name:       "GPT-4o",
			Pricing:    Pricing{Prompt: "0.005", Completion: "0.015"},
			ContextLen: 128000,
			Architecture: Architecture{
				Modality: "text+image",
			},
		},
	}

	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)

	selected, err := client.SelectBestFreeVisionModel(models)

	assert.NoError(t, err)
	assert.NotNil(t, selected)
	// Should select the free model with highest context length
	assert.Equal(t, "google/gemini-2.0-flash-exp:free", selected.ID)
	assert.Equal(t, 32768, selected.ContextLen)
}

func TestSelectBestFreeVisionModel_NoModels(t *testing.T) {
	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)

	_, err := client.SelectBestFreeVisionModel([]Model{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no models available")
}

func TestSelectBestFreeVisionModel_NoFreeModels(t *testing.T) {
	models := []Model{
		{
			ID:         "openai/gpt-4o",
			Name:       "GPT-4o",
			Pricing:    Pricing{Prompt: "0.005", Completion: "0.015"},
			ContextLen: 128000,
			Architecture: Architecture{
				Modality: "text+image",
			},
		},
	}

	logger := logrus.New()
	client := NewClient("test-api-key", "test-model", 100, 0.5, "Test prompt", logger)

	_, err := client.SelectBestFreeVisionModel(models)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no free vision models available")
}
