package dlq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shabohin/photo-tags/pkg/models"
	"github.com/streadway/amqp"
)

// Manager handles dead letter queue operations
type Manager struct {
	queueName string
}

// NewManager creates a new DLQ manager
func NewManager(queueName string) *Manager {
	return &Manager{
		queueName: queueName,
	}
}

// ConvertToFailedJob converts an AMQP delivery to a FailedJob model
func (m *Manager) ConvertToFailedJob(msg amqp.Delivery) (*models.FailedJob, error) {
	// Get original queue from death header
	originalQueue := ""
	retryCount := 0
	errorReason := "Unknown error"

	if msg.Headers != nil {
		// Get x-death header which contains information about the original queue
		if deaths, ok := msg.Headers["x-death"].([]interface{}); ok && len(deaths) > 0 {
			if death, ok := deaths[0].(amqp.Table); ok {
				if queue, ok := death["queue"].(string); ok {
					originalQueue = queue
				}
				if count, ok := death["count"].(int64); ok {
					retryCount = int(count)
				}
			}
		}

		// Get custom error reason if provided
		if reason, ok := msg.Headers["x-error-reason"].(string); ok {
			errorReason = reason
		}
	}

	// Generate ID if not present
	id := uuid.New().String()
	if msg.MessageId != "" {
		id = msg.MessageId
	}

	failedJob := &models.FailedJob{
		ID:            id,
		OriginalQueue: originalQueue,
		MessageBody:   string(msg.Body),
		ErrorReason:   errorReason,
		FailedAt:      time.Now(),
		RetryCount:    retryCount,
	}

	return failedJob, nil
}

// ConvertFailedJobsFromMessages converts multiple AMQP deliveries to FailedJob models
func (m *Manager) ConvertFailedJobsFromMessages(msgs []amqp.Delivery) ([]*models.FailedJob, error) {
	failedJobs := make([]*models.FailedJob, 0, len(msgs))

	for _, msg := range msgs {
		failedJob, err := m.ConvertToFailedJob(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		failedJobs = append(failedJobs, failedJob)
	}

	return failedJobs, nil
}

// CreateFailedJobMessage creates a failed job message with metadata
func CreateFailedJobMessage(originalQueue string, messageBody []byte, errorReason string, retryCount int) ([]byte, map[string]interface{}, error) {
	// Parse the message to ensure it's valid JSON
	var msg map[string]interface{}
	if err := json.Unmarshal(messageBody, &msg); err != nil {
		return nil, nil, fmt.Errorf("invalid message body: %w", err)
	}

	headers := map[string]interface{}{
		"x-original-queue": originalQueue,
		"x-error-reason":   errorReason,
		"x-retry-count":    retryCount,
		"x-failed-at":      time.Now().Format(time.RFC3339),
	}

	return messageBody, headers, nil
}
