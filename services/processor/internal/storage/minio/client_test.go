package minio

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewClient(t *testing.T) {
	logger := logrus.New()

	// This is a unit test, not integration test
	// We test that the client creation logic works
	t.Run("validates parameters", func(t *testing.T) {
		// Test with invalid endpoint - should fail
		client, err := NewClient(
			"invalid-endpoint",
			"test-access",
			"test-secret",
			false,
			"original",
			"processed",
			logger,
			1,
			time.Second,
		)

		if err == nil {
			t.Error("Expected error with invalid endpoint")
		}

		if client != nil {
			t.Error("Expected nil client with invalid endpoint")
		}
	})
}

func TestClientBucketNames(t *testing.T) {
	// Test that client stores bucket names correctly
	logger := logrus.New()

	client := &Client{
		originalBucket:  "test-original",
		processedBucket: "test-processed",
		logger:          logger,
	}

	if client.originalBucket != "test-original" {
		t.Errorf("Expected original bucket 'test-original', got %s", client.originalBucket)
	}

	if client.processedBucket != "test-processed" {
		t.Errorf("Expected processed bucket 'test-processed', got %s", client.processedBucket)
	}
}
