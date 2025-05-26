package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

type OpenRouterClient interface {
	AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error)
}

const (
	defaultTimeout = 60 * time.Second
	apiURL         = "https://openrouter.ai/api/v1/chat/completions"
)

type Client struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	prompt      string
	httpClient  *http.Client
	logger      *logrus.Logger
}

type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type OpenRouterResponse struct {
	Id      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
}

type MetadataResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Keywords    []string `json:"keywords"`
}

func NewClient(apiKey, model string, maxTokens int, temperature float64, prompt string, logger *logrus.Logger) *Client {
	return &Client{
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		prompt:      prompt,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		logger: logger,
	}
}

func (c *Client) AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error) {
	imageBase64 := base64.StdEncoding.EncodeToString(imageBytes)
	dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", imageBase64)

	messages := []Message{
		{
			Role: "user",
			Content: []ContentItem{
				{
					Type: "text",
					Text: c.prompt,
				},
				{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: dataURL,
					},
				},
			},
		},
	}

	requestBody := OpenRouterRequest{
		Model:       c.model,
		Messages:    messages,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to marshal request body")
		return model.Metadata{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to create HTTP request")
		return model.Metadata{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("HTTP-Referer", "https://github.com/shabohin/photo-tags")

	c.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
	}).Info("Sending request to OpenRouter API")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to send request to OpenRouter API")
		return model.Metadata{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.WithFields(logrus.Fields{
				"trace_id": traceID,
				"error":    closeErr.Error(),
			}).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"trace_id": traceID,
				"error":    err.Error(),
			}).Error("Failed to read error response body")
			return model.Metadata{}, fmt.Errorf("API error, status code: %d, failed to read response: %w", resp.StatusCode, err)
		}
		c.logger.WithFields(logrus.Fields{
			"trace_id":    traceID,
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("OpenRouter API returned error")
		return model.Metadata{}, fmt.Errorf("API error: %s, status code: %d", string(body), resp.StatusCode)
	}

	var openRouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    err.Error(),
		}).Error("Failed to decode API response")
		return model.Metadata{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
		}).Error("Empty choices in API response")
		return model.Metadata{}, fmt.Errorf("empty choices in API response")
	}

	content := openRouterResp.Choices[0].Message.Content
	var metadataResp MetadataResponse

	c.logger.WithFields(logrus.Fields{
		"trace_id": traceID,
		"content":  content,
	}).Debug("Received content from OpenRouter")

	if err := json.Unmarshal([]byte(content), &metadataResp); err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"content":  content,
			"error":    err.Error(),
		}).Error("Failed to parse metadata from response")
		return model.Metadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return model.Metadata{
		Title:       metadataResp.Title,
		Description: metadataResp.Description,
		Keywords:    metadataResp.Keywords,
	}, nil
}
