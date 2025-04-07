package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

type Client struct {
	client         *minio.Client
	originalBucket string
	logger         *logrus.Logger
}

func NewClient(endpoint, accessKey, secretKey string, useSSL bool, originalBucket string, logger *logrus.Logger) (*Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create MinIO client")
		return nil, err
	}

	// Check if the bucket exists, if not create it
	exists, err := client.BucketExists(context.Background(), originalBucket)
	if err != nil {
		logger.WithError(err).Error("Failed to check if bucket exists")
		return nil, err
	}

	if !exists {
		err = client.MakeBucket(context.Background(), originalBucket, minio.MakeBucketOptions{})
		if err != nil {
			logger.WithError(err).Error("Failed to create bucket")
			return nil, err
		}
		logger.WithField("bucket", originalBucket).Info("Bucket created successfully")
	}

	return &Client{
		client:         client,
		originalBucket: originalBucket,
		logger:         logger,
	}, nil
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
	defer object.Close()

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
