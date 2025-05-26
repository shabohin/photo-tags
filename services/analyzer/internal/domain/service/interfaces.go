package service

import (
	"context"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

type MinioClientInterface interface {
	DownloadImage(ctx context.Context, path string) ([]byte, error)
}

type OpenRouterClientInterface interface {
	AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error)
}
