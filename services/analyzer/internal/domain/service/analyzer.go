package service

import (
	"context"
	"fmt"

	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
	"github.com/sirupsen/logrus"
)

type ImageAnalyzerService struct {
	minioClient      MinioClientInterface
	openRouterClient OpenRouterClientInterface
	logger           *logrus.Logger
}

func NewImageAnalyzer(minioClient MinioClientInterface, openRouterClient OpenRouterClientInterface, logger *logrus.Logger) *ImageAnalyzerService {
	return &ImageAnalyzerService{
		minioClient:      minioClient,
		openRouterClient: openRouterClient,
		logger:           logger,
	}
}

func (s *ImageAnalyzerService) AnalyzeImage(ctx context.Context, msg model.ImageUploadMessage) (model.Metadata, error) {
	s.logger.WithFields(logrus.Fields{
		"trace_id":          msg.TraceID,
		"original_filename": msg.OriginalFilename,
		"original_path":     msg.OriginalPath,
	}).Info("Starting image analysis")

	// Download image from MinIO
	imageBytes, err := s.minioClient.DownloadImage(ctx, msg.OriginalPath)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": msg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to download image")
		return model.Metadata{}, fmt.Errorf("failed to download image: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":      msg.TraceID,
		"image_size_kb": len(imageBytes) / 1024,
	}).Debug("Image downloaded successfully")

	// Analyze image with OpenRouter
	metadata, err := s.openRouterClient.AnalyzeImage(ctx, imageBytes, msg.TraceID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": msg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to analyze image")
		return model.Metadata{}, fmt.Errorf("failed to analyze image: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":       msg.TraceID,
		"title":          metadata.Title,
		"keywords_count": len(metadata.Keywords),
	}).Info("Image analyzed successfully")

	return metadata, nil
}
