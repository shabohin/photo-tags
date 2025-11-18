package benchmarks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/shabohin/photo-tags/pkg/storage"
	"github.com/shabohin/photo-tags/services/analyzer/internal/storage/minio"
	"github.com/sirupsen/logrus"
)

const (
	testBucket = "benchmark-test"
)

// Skip benchmarks if MinIO is not available
var skipMinIOTests = os.Getenv("SKIP_MINIO_BENCHMARKS") == "true"

// setupMinIOClient creates a MinIO client for testing
func setupMinIOClient(t testing.TB) *storage.MinIOClient {
	t.Helper()

	if skipMinIOTests {
		t.Skip("Skipping MinIO benchmarks (SKIP_MINIO_BENCHMARKS=true)")
	}

	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	client, err := storage.NewMinIOClient(endpoint, accessKey, secretKey, false)
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	if err := client.EnsureBucketExists(ctx, testBucket); err != nil {
		t.Fatalf("Failed to create test bucket: %v", err)
	}

	return client
}

// setupAnalyzerMinIOClient creates an Analyzer MinIO client for testing
func setupAnalyzerMinIOClient(t testing.TB) *minio.Client {
	t.Helper()

	if skipMinIOTests {
		t.Skip("Skipping MinIO benchmarks (SKIP_MINIO_BENCHMARKS=true)")
	}

	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	client, err := minio.NewClient(endpoint, accessKey, secretKey, false, testBucket, logger, 3, time.Second)
	if err != nil {
		t.Fatalf("Failed to create Analyzer MinIO client: %v", err)
	}

	return client
}

// generateTestData creates test data of specified size
func generateTestData(size int) []byte {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}
	return data
}

// BenchmarkMinIOUpload benchmarks file upload with different sizes
func BenchmarkMinIOUpload(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()

	// Test with different file sizes (in bytes)
	sizes := []int{
		1024,        // 1 KB
		102400,      // 100 KB
		1048576,     // 1 MB
		5242880,     // 5 MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			data := generateTestData(size)
			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				objectName := fmt.Sprintf("bench-upload-%d-%d", size, i)
				reader := bytes.NewReader(data)

				err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream")
				if err != nil {
					b.Fatalf("Upload failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkMinIOUploadParallel benchmarks parallel uploads
func BenchmarkMinIOUploadParallel(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()
	data := generateTestData(102400) // 100 KB

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			objectName := fmt.Sprintf("bench-upload-parallel-%d", i)
			reader := bytes.NewReader(data)

			err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream")
			if err != nil {
				b.Fatalf("Upload failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkMinIODownload benchmarks file download with different sizes
func BenchmarkMinIODownload(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()

	sizes := []int{
		1024,        // 1 KB
		102400,      // 100 KB
		1048576,     // 1 MB
		5242880,     // 5 MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			// Setup: upload test file
			data := generateTestData(size)
			objectName := fmt.Sprintf("bench-download-%d", size)
			reader := bytes.NewReader(data)

			if err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream"); err != nil {
				b.Fatalf("Setup failed: %v", err)
			}

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				obj, err := client.DownloadFile(ctx, testBucket, objectName)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				// Read all data to measure actual download performance
				downloaded, err := io.ReadAll(obj)
				if err != nil {
					b.Fatalf("Read failed: %v", err)
				}

				if len(downloaded) != size {
					b.Fatalf("Size mismatch: expected %d, got %d", size, len(downloaded))
				}

				obj.Close()
			}
		})
	}
}

// BenchmarkMinIODownloadParallel benchmarks parallel downloads
func BenchmarkMinIODownloadParallel(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()

	// Setup: upload test file
	data := generateTestData(102400) // 100 KB
	objectName := "bench-download-parallel"
	reader := bytes.NewReader(data)

	if err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream"); err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj, err := client.DownloadFile(ctx, testBucket, objectName)
			if err != nil {
				b.Fatalf("Download failed: %v", err)
			}

			downloaded, err := io.ReadAll(obj)
			if err != nil {
				b.Fatalf("Read failed: %v", err)
			}

			if len(downloaded) != len(data) {
				b.Fatalf("Size mismatch: expected %d, got %d", len(data), len(downloaded))
			}

			obj.Close()
		}
	})
}

// BenchmarkMinIOAnalyzerDownload benchmarks Analyzer's DownloadImage method
func BenchmarkMinIOAnalyzerDownload(b *testing.B) {
	client := setupAnalyzerMinIOClient(b)
	ctx := context.Background()

	sizes := []int{
		102400,      // 100 KB
		1048576,     // 1 MB
		5242880,     // 5 MB
	}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			// Setup: upload test image
			data := generateTestData(size)
			objectName := fmt.Sprintf("bench-analyzer-download-%d", size)

			// Use standard client for upload
			stdClient := setupMinIOClient(b)
			reader := bytes.NewReader(data)
			if err := stdClient.UploadFile(ctx, testBucket, objectName, reader, "image/jpeg"); err != nil {
				b.Fatalf("Setup failed: %v", err)
			}

			b.SetBytes(int64(size))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				downloaded, err := client.DownloadImage(ctx, objectName)
				if err != nil {
					b.Fatalf("DownloadImage failed: %v", err)
				}

				if len(downloaded) != size {
					b.Fatalf("Size mismatch: expected %d, got %d", size, len(downloaded))
				}
			}
		})
	}
}

// BenchmarkMinIOCompleteWorkflow benchmarks upload + download workflow
func BenchmarkMinIOCompleteWorkflow(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()
	data := generateTestData(102400) // 100 KB

	b.SetBytes(int64(len(data)) * 2) // Upload + Download
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		objectName := fmt.Sprintf("bench-workflow-%d", i)

		// Upload
		reader := bytes.NewReader(data)
		if err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream"); err != nil {
			b.Fatalf("Upload failed: %v", err)
		}

		// Download
		obj, err := client.DownloadFile(ctx, testBucket, objectName)
		if err != nil {
			b.Fatalf("Download failed: %v", err)
		}

		downloaded, err := io.ReadAll(obj)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}

		if len(downloaded) != len(data) {
			b.Fatalf("Size mismatch: expected %d, got %d", len(data), len(downloaded))
		}

		obj.Close()
	}
}

// BenchmarkMinIOGetPresignedURL benchmarks presigned URL generation
func BenchmarkMinIOGetPresignedURL(b *testing.B) {
	client := setupMinIOClient(b)
	ctx := context.Background()

	// Setup: upload test file
	data := generateTestData(1024)
	objectName := "bench-presigned-url"
	reader := bytes.NewReader(data)

	if err := client.UploadFile(ctx, testBucket, objectName, reader, "application/octet-stream"); err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		url, err := client.GetPresignedURL(ctx, testBucket, objectName, 15*time.Minute)
		if err != nil {
			b.Fatalf("GetPresignedURL failed: %v", err)
		}

		if url == "" {
			b.Fatal("Empty URL returned")
		}
	}
}
