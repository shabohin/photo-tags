package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/eduardolat/openroutergo"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

type OpenRouterGoAdapter struct {
	apiKey      string
	model       string
	prompt      string
	temperature float64
	maxTokens   int
}

func NewOpenRouterGoAdapter(apiKey, model, prompt string, temperature float64, maxTokens int) OpenRouterClient {
	return &OpenRouterGoAdapter{
		apiKey:      apiKey,
		model:       model,
		prompt:      prompt,
		temperature: temperature,
		maxTokens:   maxTokens,
	}
}

func (a *OpenRouterGoAdapter) AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error) {
	imageBase64 := base64.StdEncoding.EncodeToString(imageBytes)
	dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", imageBase64)

	client, err := openroutergo.NewClient().
		WithAPIKey(a.apiKey).
		Create()
	if err != nil {
		return model.Metadata{}, err
	}

	_, resp, err := client.
		NewChatCompletion().
		WithModel(a.model).
		WithSystemMessage(a.prompt).
		WithUserMessage(fmt.Sprintf("Please analyze this image: %s", dataURL)).
		Execute()
	if err != nil {
		return model.Metadata{}, err
	}

	if len(resp.Choices) == 0 {
		return model.Metadata{}, fmt.Errorf("empty choices in response")
	}

	content := resp.Choices[0].Message.Content

	var metadata model.Metadata
	err = json.Unmarshal([]byte(content), &metadata)
	if err != nil {
		return model.Metadata{}, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return metadata, nil
}
