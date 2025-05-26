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

type Client struct {
	logger         *logrus.Logger
	client         *minio.Client
	originalBucket string
}

func NewClient(endpoint, accessKey, secretKey string, useSSL bool, originalBucket string, logger *logrus.Logger, attempts int, delay time.Duration) (*Client, error) {
	var client *minio.Client
	var err error

	for i := 1; i <= attempts; i++ {
		client, err = minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
			Secure: useSSL,
		})
		if err != nil {
			logger.WithError(err).Errorf("Attempt %d: Failed to create MinIO client", i)
			time.Sleep(delay)
			continue
		}

		exists, errBucket := client.BucketExists(context.Background(), originalBucket)
		if errBucket != nil {
			logger.WithError(errBucket).Errorf("Attempt %d: Failed to check if bucket exists", i)
			time.Sleep(delay)
			continue
		}

		if !exists {
			errBucket = client.MakeBucket(context.Background(), originalBucket, minio.MakeBucketOptions{})
			if errBucket != nil {
				logger.WithError(errBucket).Errorf("Attempt %d: Failed to create bucket", i)
				time.Sleep(delay)
				continue
			}
			logger.WithField("bucket", originalBucket).Info("Bucket created successfully")
		}

		return &Client{
			client:         client,
			originalBucket: originalBucket,
			logger:         logger,
		}, nil
	}

	logger.WithError(err).Errorf("All %d attempts to connect to MinIO failed", attempts)
	return nil, err
}

func (c *Client) DownloadImage(ctx context.Context, path string) ([]byte, error) {
	object, err := c.client.GetObject(ctx, c.originalBucket, path, minio.GetObjectOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.originalBucket,
			"path":   path,
			"error":  err.Error(),
		}).Error("Failed to get object from MinIO")
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer func() {
		if closeErr := object.Close(); closeErr != nil {
			c.logger.WithFields(logrus.Fields{
				"bucket": c.originalBucket,
				"path":   path,
				"error":  closeErr.Error(),
			}).Error("Failed to close MinIO object")
		}
	}()

	info, err := object.Stat()
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.originalBucket,
			"path":   path,
			"error":  err.Error(),
		}).Error("Failed to get object info")
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	buffer := bytes.NewBuffer(make([]byte, 0, info.Size))
	if _, err := io.Copy(buffer, object); err != nil {
		c.logger.WithFields(logrus.Fields{
			"bucket": c.originalBucket,
			"path":   path,
			"error":  err.Error(),
		}).Error("Failed to read object data")
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"bucket":    c.originalBucket,
		"path":      path,
		"size":      info.Size,
		"etag":      info.ETag,
		"mime_type": info.ContentType,
	}).Debug("Successfully downloaded image")

	return buffer.Bytes(), nil
}
