package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// Client wraps MinIO client for processor service
type Client struct {
	client          *minio.Client
	originalBucket  string
	processedBucket string
	logger          *logrus.Logger
}

// NewClient creates a new MinIO client and ensures buckets exist
func NewClient(
	endpoint string,
	accessKey string,
	secretKey string,
	useSSL bool,
	originalBucket string,
	processedBucket string,
	logger *logrus.Logger,
	connectAttempts int,
	connectDelay time.Duration,
) (*Client, error) {
	var minioClient *minio.Client
	var err error

	// Retry connection
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		minioClient, err = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})

		if err == nil {
			break
		}

		logger.WithFields(logrus.Fields{
			"attempt": attempt,
			"error":   err.Error(),
		}).Warn("Failed to create MinIO client, retrying...")

		if attempt < connectAttempts {
			time.Sleep(connectDelay)
		}
	}

	if err != nil {
		logger.WithError(err).Error("Failed to create MinIO client after all attempts")
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	client := &Client{
		client:          minioClient,
		originalBucket:  originalBucket,
		processedBucket: processedBucket,
		logger:          logger,
	}

	// Check if buckets exist
	ctx := context.Background()

	if err := client.ensureBucketExists(ctx, originalBucket); err != nil {
		return nil, fmt.Errorf("failed to ensure original bucket exists: %w", err)
	}

	if err := client.ensureBucketExists(ctx, processedBucket); err != nil {
		return nil, fmt.Errorf("failed to ensure processed bucket exists: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"endpoint":         endpoint,
		"original_bucket":  originalBucket,
		"processed_bucket": processedBucket,
	}).Info("MinIO client initialized successfully")

	return client, nil
}

// ensureBucketExists checks if a bucket exists, creates it if not
func (c *Client) ensureBucketExists(ctx context.Context, bucketName string) error {
	exists, err := c.client.BucketExists(ctx, bucketName)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": bucketName,
			"error":  err.Error(),
		}).Error("Failed to check bucket existence")
		return fmt.Errorf("failed to check bucket %s: %w", bucketName, err)
	}

	if !exists {
		c.logger.WithField("bucket", bucketName).Info("Bucket does not exist, creating...")

		err = c.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			c.logger.WithFields(logrus.Fields{
				"bucket": bucketName,
				"error":  err.Error(),
			}).Error("Failed to create bucket")
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}

		c.logger.WithField("bucket", bucketName).Info("Bucket created successfully")
	}

	return nil
}

// DownloadImage downloads an image from the original bucket
func (c *Client) DownloadImage(ctx context.Context, objectPath string) ([]byte, error) {
	c.logger.WithFields(logrus.Fields{
		"bucket": c.originalBucket,
		"object": objectPath,
	}).Debug("Downloading image from MinIO")

	object, err := c.client.GetObject(ctx, c.originalBucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.originalBucket,
			"object": objectPath,
			"error":  err.Error(),
		}).Error("Failed to get object from MinIO")
		return nil, fmt.Errorf("failed to get object %s: %w", objectPath, err)
	}
	defer func() {
		if closeErr := object.Close(); closeErr != nil {
			c.logger.WithError(closeErr).Warn("Failed to close MinIO object")
		}
	}()

	// Read all data
	data, err := io.ReadAll(object)
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.originalBucket,
			"object": objectPath,
			"error":  err.Error(),
		}).Error("Failed to read object data")
		return nil, fmt.Errorf("failed to read object %s: %w", objectPath, err)
	}

	c.logger.WithFields(logrus.Fields{
		"bucket":    c.originalBucket,
		"object":    objectPath,
		"size_bytes": len(data),
	}).Debug("Image downloaded successfully")

	return data, nil
}

// UploadImage uploads a processed image to the processed bucket
func (c *Client) UploadImage(ctx context.Context, objectPath string, data []byte) error {
	c.logger.WithFields(logrus.Fields{
		"bucket":     c.processedBucket,
		"object":     objectPath,
		"size_bytes": len(data),
	}).Debug("Uploading image to MinIO")

	reader := bytes.NewReader(data)

	// Determine content type
	contentType := "image/jpeg"
	if len(objectPath) > 4 && objectPath[len(objectPath)-4:] == ".png" {
		contentType = "image/png"
	}

	_, err := c.client.PutObject(
		ctx,
		c.processedBucket,
		objectPath,
		reader,
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.processedBucket,
			"object": objectPath,
			"error":  err.Error(),
		}).Error("Failed to upload object to MinIO")
		return fmt.Errorf("failed to upload object %s: %w", objectPath, err)
	}

	c.logger.WithFields(logrus.Fields{
		"bucket": c.processedBucket,
		"object": objectPath,
	}).Debug("Image uploaded successfully")

	return nil
}

// GetClient returns the underlying MinIO client (for testing)
func (c *Client) GetClient() *minio.Client {
	return c.client
}
