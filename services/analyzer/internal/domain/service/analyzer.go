package service

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/shabohin/photo-tags/pkg/imageprocessing"
	"github.com/shabohin/photo-tags/services/analyzer/internal/domain/model"
)

type ImageAnalyzerService struct {
	minioClient      MinioClientInterface
	openRouterClient OpenRouterClientInterface
	imageOptimizer   *imageprocessing.Optimizer
	logger           *logrus.Logger
}

func NewImageAnalyzer(minioClient MinioClientInterface,
	openRouterClient OpenRouterClientInterface,
	logger *logrus.Logger) *ImageAnalyzerService {
	return &ImageAnalyzerService{
		minioClient:      minioClient,
		openRouterClient: openRouterClient,
		imageOptimizer:   imageprocessing.NewOptimizer(logger),
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

	// Optimize image before sending to OpenRouter
	optimizationResult, err := s.imageOptimizer.Optimize(imageBytes, msg.TraceID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"trace_id": msg.TraceID,
			"error":    err.Error(),
		}).Error("Failed to optimize image")
		return model.Metadata{}, fmt.Errorf("failed to optimize image: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"trace_id":           msg.TraceID,
		"original_size_kb":   optimizationResult.OriginalSize / 1024,
		"optimized_size_kb":  optimizationResult.OptimizedSize / 1024,
		"compression_ratio":  optimizationResult.CompressionRatio,
		"was_resized":        optimizationResult.WasResized,
		"was_converted":      optimizationResult.WasConverted,
		"original_format":    optimizationResult.OriginalFormat,
		"optimized_format":   optimizationResult.OptimizedFormat,
	}).Info("Image optimized successfully")

	// Analyze image with OpenRouter using optimized data
	metadata, err := s.openRouterClient.AnalyzeImage(ctx, optimizationResult.Data, msg.TraceID)
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
