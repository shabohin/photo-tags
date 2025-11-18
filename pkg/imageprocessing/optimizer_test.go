package imageprocessing

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestImage(width, height int, format ImageFormat) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			}
			img.Set(x, y, c)
		}
	}

	var buf bytes.Buffer
	switch format {
	case FormatJPEG:
		_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	case FormatPNG:
		_ = png.Encode(&buf, img)
	}

	return buf.Bytes()
}

func TestNewOptimizer(t *testing.T) {
	logger := logrus.New()
	optimizer := NewOptimizer(logger)

	assert.NotNil(t, optimizer)
	assert.Equal(t, logger, optimizer.logger)
}

func TestValidate(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs in tests
	optimizer := NewOptimizer(logger)

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid data",
			data:    []byte("not an image"),
			wantErr: true,
		},
		{
			name:    "valid JPEG",
			data:    createTestImage(100, 100, FormatJPEG),
			wantErr: false,
		},
		{
			name:    "valid PNG",
			data:    createTestImage(100, 100, FormatPNG),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := optimizer.Validate(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptimize_SmallImage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	// Create a small JPEG image (should not be optimized)
	data := createTestImage(500, 500, FormatJPEG)
	require.Less(t, len(data), MaxImageSize)

	result, err := optimizer.Optimize(data, "test-trace-id")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(data), result.OriginalSize)
	assert.Equal(t, len(data), result.OptimizedSize)
	assert.Equal(t, FormatJPEG, result.OriginalFormat)
	assert.False(t, result.WasResized)
	assert.False(t, result.WasCompressed)
	assert.False(t, result.WasConverted)
	assert.Equal(t, 1.0, result.CompressionRatio)
}

func TestOptimize_LargeImage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	// Create a large image that needs optimization
	data := createTestImage(3000, 3000, FormatJPEG)
	require.Greater(t, len(data), MaxImageSize)

	result, err := optimizer.Optimize(data, "test-trace-id")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(data), result.OriginalSize)
	assert.Less(t, result.OptimizedSize, result.OriginalSize)
	assert.Equal(t, FormatJPEG, result.OriginalFormat)
	assert.True(t, result.WasResized)
	assert.True(t, result.WasCompressed)
	assert.Less(t, result.CompressionRatio, 1.0)
}

func TestOptimize_PNGToJPEG(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	// Create a large PNG that should be converted to JPEG
	data := createTestImage(2000, 2000, FormatPNG)
	require.Greater(t, len(data), PNGToJPEGThreshold)

	result, err := optimizer.Optimize(data, "test-trace-id")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, FormatPNG, result.OriginalFormat)
	assert.Equal(t, FormatJPEG, result.OptimizedFormat)
	assert.True(t, result.WasConverted)
	assert.True(t, result.WasResized)
	assert.True(t, result.WasCompressed)
}

func TestOptimize_SmallPNG(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	// Create a small PNG that should not be converted
	data := createTestImage(200, 200, FormatPNG)
	require.Less(t, len(data), PNGToJPEGThreshold)

	result, err := optimizer.Optimize(data, "test-trace-id")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, FormatPNG, result.OriginalFormat)
	assert.Equal(t, FormatPNG, result.OptimizedFormat)
	assert.False(t, result.WasConverted)
}

func TestResize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	tests := []struct {
		name        string
		width       int
		height      int
		maxDim      int
		expectWidth int
		expectHeight int
	}{
		{
			name:        "landscape image",
			width:       3000,
			height:      2000,
			maxDim:      MaxImageDimension,
			expectWidth: MaxImageDimension,
			expectHeight: (2000 * MaxImageDimension) / 3000,
		},
		{
			name:        "portrait image",
			width:       2000,
			height:      3000,
			maxDim:      MaxImageDimension,
			expectWidth: (2000 * MaxImageDimension) / 3000,
			expectHeight: MaxImageDimension,
		},
		{
			name:        "square image",
			width:       3000,
			height:      3000,
			maxDim:      MaxImageDimension,
			expectWidth: MaxImageDimension,
			expectHeight: MaxImageDimension,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, tt.width, tt.height))
			resized := optimizer.resize(img, "test-trace-id")

			assert.Equal(t, tt.expectWidth, resized.Bounds().Dx())
			assert.Equal(t, tt.expectHeight, resized.Bounds().Dy())
		})
	}
}

func TestCompress(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	img := image.NewRGBA(image.Rect(0, 0, 500, 500))

	tests := []struct {
		name    string
		format  ImageFormat
		wantErr bool
	}{
		{
			name:    "compress JPEG",
			format:  FormatJPEG,
			wantErr: false,
		},
		{
			name:    "compress PNG",
			format:  FormatPNG,
			wantErr: false,
		},
		{
			name:    "unknown format",
			format:  FormatUnknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := optimizer.compress(img, tt.format, "test-trace-id")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, data)
				assert.Greater(t, len(data), 0)
			}
		})
	}
}

func TestShouldConvertToJPEG(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	tests := []struct {
		name   string
		format ImageFormat
		size   int
		want   bool
	}{
		{
			name:   "large PNG should convert",
			format: FormatPNG,
			size:   PNGToJPEGThreshold + 1,
			want:   true,
		},
		{
			name:   "small PNG should not convert",
			format: FormatPNG,
			size:   PNGToJPEGThreshold - 1,
			want:   false,
		},
		{
			name:   "JPEG should not convert",
			format: FormatJPEG,
			size:   PNGToJPEGThreshold + 1,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.shouldConvertToJPEG(tt.format, tt.size)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOptimizeReader(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	optimizer := NewOptimizer(logger)

	data := createTestImage(500, 500, FormatJPEG)
	reader := bytes.NewReader(data)

	result, err := optimizer.OptimizeReader(reader, "test-trace-id")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, len(data), result.OriginalSize)
}
