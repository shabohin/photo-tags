package storage

import (
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOInterface defines the interface for MinIO operations
type MinIOInterface interface {
	EnsureBucketExists(ctx context.Context, bucketName string) error
	UploadFile(ctx context.Context, bucketName, objectName string, reader io.Reader, contentType string) error
	DownloadFile(ctx context.Context, bucketName, objectName string) (*minio.Object, error)
	GetPresignedURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
}

// MinIOClient handles object storage operations
type MinIOClient struct {
	client *minio.Client
}

// Bucket names
const (
	BucketOriginal  = "original"
	BucketProcessed = "processed"
)

// NewMinIOClient creates a new MinIO client
func NewMinIOClient(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOClient, error) {
	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	return &MinIOClient{
		client: client,
	}, nil
}

// UploadFile uploads a file to MinIO
func (c *MinIOClient) UploadFile(
	ctx context.Context,
	bucketName, objectName string,
	reader io.Reader,
	contentType string,
) error {
	// Upload file to bucket
	_, err := c.client.PutObject(ctx, bucketName, objectName, reader, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// DownloadFile downloads a file from MinIO
func (c *MinIOClient) DownloadFile(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	// Get object from bucket
	return c.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
}

// GetPresignedURL generates a presigned URL for an object
func (c *MinIOClient) GetPresignedURL(
	ctx context.Context,
	bucketName, objectName string,
	expiry time.Duration,
) (string, error) {
	// Generate presigned URL
	url, err := c.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// EnsureBucketExists creates a bucket if it doesn't exist
func (c *MinIOClient) EnsureBucketExists(ctx context.Context, bucketName string) error {
	// Check if bucket exists
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}

	// Create bucket if it doesn't exist
	if !exists {
		return c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	}

	return nil
}
