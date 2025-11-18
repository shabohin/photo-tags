package openrouter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

type OpenRouterClient interface {
	AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error)
	GetAvailableModels(ctx context.Context) ([]Model, error)
	SelectBestFreeVisionModel(models []Model) (*Model, error)
}

const (
	defaultTimeout     = 60 * time.Second
	apiURL             = "https://openrouter.ai/api/v1/chat/completions"
	modelsURL          = "https://openrouter.ai/api/v1/models"
	maxRetries         = 3
	initialRetryDelay  = 2 * time.Second
	rateLimitResetWait = 5 * time.Second
)

type Client struct {
	httpClient  *http.Client
	logger      *logrus.Logger
	apiKey      string
	model       string
	prompt      string
	temperature float64
	maxTokens   int
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
	ImageURL *ImageURL `json:"image_url,omitempty"`
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
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

// Model represents an OpenRouter model
type Model struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Pricing      Pricing      `json:"pricing"`
	ContextLen   int          `json:"context_length"`
	Architecture Architecture `json:"architecture"`
	TopProvider  TopProvider  `json:"top_provider,omitempty"`
}

// Pricing represents model pricing information
type Pricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	Image      string `json:"image,omitempty"`
	Request    string `json:"request,omitempty"`
}

// Architecture contains model architecture details
type Architecture struct {
	Modality     string `json:"modality,omitempty"`
	Tokenizer    string `json:"tokenizer,omitempty"`
	InstructType string `json:"instruct_type,omitempty"`
}

// TopProvider contains provider information
type TopProvider struct {
	MaxCompletionTokens int  `json:"max_completion_tokens,omitempty"`
	IsModerated         bool `json:"is_moderated,omitempty"`
}

// ModelsResponse represents the response from /api/v1/models endpoint
type ModelsResponse struct {
	Data []Model `json:"data"`
}

// RateLimit holds rate limit information from response headers
type RateLimit struct {
	Remaining int
	Reset     time.Time
}

// RateLimitError represents a rate limit error
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s, retry after %v", e.Message, e.RetryAfter)
}

func NewClient(
	apiKey string,
	modelName string,
	maxTokens int,
	temperature float64,
	prompt string,
	logger *logrus.Logger,
) *Client {
	return &Client{
		apiKey:      apiKey,
		model:       modelName,
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

	var resp *http.Response
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// nolint:gosec // G115: safe conversion, attempt is bounded by maxRetries
			delay := initialRetryDelay * time.Duration(1<<uint(attempt-1))
			c.logger.WithFields(logrus.Fields{
				"trace_id": traceID,
				"attempt":  attempt,
				"delay":    delay,
			}).Info("Retrying AnalyzeImage request")

			select {
			case <-ctx.Done():
				return model.Metadata{}, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.WithFields(logrus.Fields{
				"trace_id": traceID,
				"attempt":  attempt,
				"error":    err.Error(),
			}).Warn("Failed to send request to OpenRouter API")
			continue
		}

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			resetTime := c.parseRateLimitReset(resp.Header)
			_ = resp.Body.Close()

			retryAfter := time.Until(resetTime)
			if retryAfter < 0 {
				retryAfter = rateLimitResetWait
			}

			c.logger.WithFields(logrus.Fields{
				"trace_id":    traceID,
				"retry_after": retryAfter,
				"reset_time":  resetTime,
			}).Warn("Rate limit exceeded for AnalyzeImage, waiting before retry")

			select {
			case <-ctx.Done():
				return model.Metadata{}, ctx.Err()
			case <-time.After(retryAfter):
			}
			continue
		}

		// Check for server errors (5xx) - retry these
		if resp.StatusCode >= 500 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			c.logger.WithFields(logrus.Fields{
				"trace_id":    traceID,
				"attempt":     attempt,
				"status_code": resp.StatusCode,
				"response":    string(body),
			}).Warn("OpenRouter API returned server error, will retry")

			lastErr = fmt.Errorf("server error: status %d, response: %s", resp.StatusCode, string(body))
			continue
		}

		// Success or client error (don't retry 4xx except 429)
		break
	}

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"error":    lastErr.Error(),
		}).Error("Failed to send request after retries")
		return model.Metadata{}, fmt.Errorf("failed to send request after %d retries: %w", maxRetries, lastErr)
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

// GetAvailableModels fetches the list of available models from OpenRouter
func (c *Client) GetAvailableModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, http.NoBody)
	if err != nil {
		c.logger.WithError(err).Error("Failed to create models request")
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("HTTP-Referer", "https://github.com/shabohin/photo-tags")
	req.Header.Set("X-Title", "Photo Tags Service")

	var resp *http.Response
	var lastErr error

	// Retry logic with exponential backoff
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// nolint:gosec // G115: safe conversion, attempt is bounded by maxRetries
			delay := initialRetryDelay * time.Duration(1<<uint(attempt-1))
			c.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   delay,
			}).Info("Retrying GetAvailableModels request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"error":   err.Error(),
			}).Warn("Failed to fetch models")
			continue
		}

		// Check for rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			resetTime := c.parseRateLimitReset(resp.Header)
			_ = resp.Body.Close()

			retryAfter := time.Until(resetTime)
			if retryAfter < 0 {
				retryAfter = rateLimitResetWait
			}

			c.logger.WithFields(logrus.Fields{
				"retry_after": retryAfter,
				"reset_time":  resetTime,
			}).Warn("Rate limit exceeded, waiting before retry")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}
			continue
		}

		// Success or non-retryable error
		break
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch models after %d retries: %w", maxRetries, lastErr)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Error("Failed to close models response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("Models API returned error")
		return nil, fmt.Errorf("models API error: status %d, response: %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		c.logger.WithError(err).Error("Failed to decode models response")
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	c.logger.WithField("models_count", len(modelsResp.Data)).Info("Successfully fetched models")
	return modelsResp.Data, nil
}

// SelectBestFreeVisionModel selects the best free vision model from the list
func (c *Client) SelectBestFreeVisionModel(models []Model) (*Model, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models available")
	}

	var freeVisionModels []Model

	// Filter for free vision models
	for _, m := range models {
		// Check if the model is free (pricing.prompt == "0")
		if m.Pricing.Prompt == "0" || m.Pricing.Prompt == "" {
			// Check if it supports vision (multimodal or image in modality)
			if strings.Contains(strings.ToLower(m.Architecture.Modality), "multimodal") ||
				strings.Contains(strings.ToLower(m.Architecture.Modality), "image") ||
				strings.Contains(strings.ToLower(m.ID), "vision") ||
				strings.Contains(strings.ToLower(m.Name), "vision") {
				freeVisionModels = append(freeVisionModels, m)
			}
		}
	}

	if len(freeVisionModels) == 0 {
		c.logger.Warn("No free vision models found, looking for any free multimodal models")

		// Fallback: look for any free model with decent context length
		for _, m := range models {
			if m.Pricing.Prompt == "0" || m.Pricing.Prompt == "" {
				if m.ContextLen > 0 {
					freeVisionModels = append(freeVisionModels, m)
				}
			}
		}
	}

	if len(freeVisionModels) == 0 {
		return nil, fmt.Errorf("no free vision models available")
	}

	// Sort by context length (higher is better) and return the best one
	sort.Slice(freeVisionModels, func(i, j int) bool {
		return freeVisionModels[i].ContextLen > freeVisionModels[j].ContextLen
	})

	selected := &freeVisionModels[0]
	c.logger.WithFields(logrus.Fields{
		"model_id":    selected.ID,
		"model_name":  selected.Name,
		"context_len": selected.ContextLen,
		"modality":    selected.Architecture.Modality,
	}).Info("Selected best free vision model")

	return selected, nil
}

// parseRateLimitReset parses the X-RateLimit-Reset header
func (c *Client) parseRateLimitReset(header http.Header) time.Time {
	resetStr := header.Get("X-RateLimit-Reset")
	if resetStr == "" {
		return time.Now().Add(rateLimitResetWait)
	}

	// Try parsing as Unix timestamp
	if resetUnix, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
		return time.Unix(resetUnix, 0)
	}

	// Try parsing as RFC3339
	if resetTime, err := time.Parse(time.RFC3339, resetStr); err == nil {
		return resetTime
	}

	return time.Now().Add(rateLimitResetWait)
}
