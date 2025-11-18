package imageprocessing

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

const (
	// MaxImageSize is the maximum file size before optimization (2MB)
	MaxImageSize = 2 * 1024 * 1024

	// MaxImageDimension is the maximum width or height for resized images
	MaxImageDimension = 2048

	// JPEGQuality is the quality for JPEG compression (85 is good balance)
	JPEGQuality = 85

	// PNGToJPEGThreshold is the size threshold for PNG to JPEG conversion (500KB)
	PNGToJPEGThreshold = 500 * 1024
)

// ImageFormat represents the image format
type ImageFormat string

const (
	FormatJPEG ImageFormat = "jpeg"
	FormatPNG  ImageFormat = "png"
	FormatUnknown ImageFormat = "unknown"
)

// OptimizationResult contains the results of image optimization
type OptimizationResult struct {
	Data              []byte
	OriginalSize      int
	OptimizedSize     int
	OriginalFormat    ImageFormat
	OptimizedFormat   ImageFormat
	WasResized        bool
	WasCompressed     bool
	WasConverted      bool
	CompressionRatio  float64
}

// Optimizer handles image optimization
type Optimizer struct {
	logger *logrus.Logger
}

// NewOptimizer creates a new image optimizer
func NewOptimizer(logger *logrus.Logger) *Optimizer {
	return &Optimizer{
		logger: logger,
	}
}

// Validate checks if the image data is valid
func (o *Optimizer) Validate(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty image data")
	}

	// Try to decode the image to validate format
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	// Check if format is supported
	if format != "jpeg" && format != "png" {
		return fmt.Errorf("unsupported image format: %s (only jpeg and png are supported)", format)
	}

	return nil
}

// Optimize optimizes the image by resizing, compressing, and optionally converting format
func (o *Optimizer) Optimize(data []byte, traceID string) (*OptimizationResult, error) {
	result := &OptimizationResult{
		OriginalSize: len(data),
	}

	// Validate the image
	if err := o.Validate(data); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	result.OriginalFormat = ImageFormat(format)
	result.OptimizedFormat = result.OriginalFormat

	o.logger.WithFields(logrus.Fields{
		"trace_id":        traceID,
		"original_size":   result.OriginalSize,
		"original_format": result.OriginalFormat,
		"width":           img.Bounds().Dx(),
		"height":          img.Bounds().Dy(),
	}).Info("Starting image optimization")

	// Check if image needs optimization
	needsOptimization := result.OriginalSize > MaxImageSize
	needsResize := img.Bounds().Dx() > MaxImageDimension || img.Bounds().Dy() > MaxImageDimension

	// If image is small enough and doesn't need resize, return original
	if !needsOptimization && !needsResize {
		o.logger.WithFields(logrus.Fields{
			"trace_id": traceID,
			"size":     result.OriginalSize,
		}).Debug("Image doesn't need optimization, returning original")

		result.Data = data
		result.OptimizedSize = len(data)
		result.CompressionRatio = 1.0
		return result, nil
	}

	// Resize if needed
	if needsResize {
		img = o.resize(img, traceID)
		result.WasResized = true
	}

	// Determine target format
	targetFormat := result.OriginalFormat
	if o.shouldConvertToJPEG(result.OriginalFormat, result.OriginalSize) {
		targetFormat = FormatJPEG
		result.WasConverted = true
		result.OptimizedFormat = FormatJPEG
	}

	// Compress the image
	optimizedData, err := o.compress(img, targetFormat, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to compress image: %w", err)
	}

	result.Data = optimizedData
	result.OptimizedSize = len(optimizedData)
	result.WasCompressed = true
	result.CompressionRatio = float64(result.OptimizedSize) / float64(result.OriginalSize)

	o.logger.WithFields(logrus.Fields{
		"trace_id":           traceID,
		"original_size":      result.OriginalSize,
		"optimized_size":     result.OptimizedSize,
		"compression_ratio":  result.CompressionRatio,
		"was_resized":        result.WasResized,
		"was_converted":      result.WasConverted,
		"original_format":    result.OriginalFormat,
		"optimized_format":   result.OptimizedFormat,
	}).Info("Image optimization completed")

	return result, nil
}

// resize resizes the image to fit within MaxImageDimension while maintaining aspect ratio
func (o *Optimizer) resize(img image.Image, traceID string) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate new dimensions while maintaining aspect ratio
	var newWidth, newHeight int
	if width > height {
		newWidth = MaxImageDimension
		newHeight = (height * MaxImageDimension) / width
	} else {
		newHeight = MaxImageDimension
		newWidth = (width * MaxImageDimension) / height
	}

	o.logger.WithFields(logrus.Fields{
		"trace_id":   traceID,
		"old_width":  width,
		"old_height": height,
		"new_width":  newWidth,
		"new_height": newHeight,
	}).Debug("Resizing image")

	// Use Lanczos resampling for high quality resize
	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
}

// compress compresses the image to the target format
func (o *Optimizer) compress(img image.Image, format ImageFormat, traceID string) ([]byte, error) {
	var buf bytes.Buffer
	var err error

	switch format {
	case FormatJPEG:
		// JPEG compression with quality setting
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality})
	case FormatPNG:
		// PNG compression (lossless)
		encoder := png.Encoder{
			CompressionLevel: png.BestCompression,
		}
		err = encoder.Encode(&buf, img)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	o.logger.WithFields(logrus.Fields{
		"trace_id":        traceID,
		"format":          format,
		"compressed_size": buf.Len(),
	}).Debug("Image compressed")

	return buf.Bytes(), nil
}

// shouldConvertToJPEG determines if a PNG should be converted to JPEG
// PNG to JPEG conversion is beneficial for:
// - Large PNG files (> 500KB)
// - Photos (vs graphics with transparency)
func (o *Optimizer) shouldConvertToJPEG(format ImageFormat, size int) bool {
	// Only convert PNG to JPEG
	if format != FormatPNG {
		return false
	}

	// Convert if PNG is larger than threshold
	return size > PNGToJPEGThreshold
}

// OptimizeReader is a convenience method that reads from io.Reader
func (o *Optimizer) OptimizeReader(r io.Reader, traceID string) (*OptimizationResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return o.Optimize(data, traceID)
}
