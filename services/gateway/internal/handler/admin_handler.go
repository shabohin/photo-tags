package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shabohin/photo-tags/pkg/dlq"
	"github.com/shabohin/photo-tags/pkg/logging"
	"github.com/shabohin/photo-tags/pkg/messaging"
	"github.com/shabohin/photo-tags/pkg/models"
)

// AdminHandler handles admin endpoints
type AdminHandler struct {
	logger          *logging.Logger
	rabbitmqClient  messaging.RabbitMQInterface
	dlqManager      *dlq.Manager
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(logger *logging.Logger, rabbitmqClient messaging.RabbitMQInterface) *AdminHandler {
	return &AdminHandler{
		logger:         logger,
		rabbitmqClient: rabbitmqClient,
		dlqManager:     dlq.NewManager(messaging.QueueDeadLetter),
	}
}

// GetFailedJobs returns all failed jobs from the dead letter queue
func (h *AdminHandler) GetFailedJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get messages from dead letter queue
	messages, err := h.rabbitmqClient.GetMessages(messaging.QueueDeadLetter, 100)
	if err != nil {
		h.logger.Error("Failed to get messages from DLQ", err)
		http.Error(w, fmt.Sprintf("Failed to get failed jobs: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert messages to FailedJob models
	failedJobs, err := h.dlqManager.ConvertFailedJobsFromMessages(messages)
	if err != nil {
		h.logger.Error("Failed to convert messages to FailedJob models", err)
		http.Error(w, fmt.Sprintf("Failed to process failed jobs: %v", err), http.StatusInternalServerError)
		return
	}

	// Acknowledge the messages we just read (they will still be in the queue)
	for _, msg := range messages {
		if err := msg.Nack(false, true); err != nil {
			h.logger.Error("Failed to nack message", err)
		}
	}

	response := map[string]interface{}{
		"jobs":      failedJobs,
		"count":     len(failedJobs),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// RequeueFailedJob requeues a failed job back to its original queue
func (h *AdminHandler) RequeueFailedJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		JobID string `json:"job_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.JobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	// Get all messages from DLQ to find the one to requeue
	messages, err := h.rabbitmqClient.GetMessages(messaging.QueueDeadLetter, 100)
	if err != nil {
		h.logger.Error("Failed to get messages from DLQ", err)
		http.Error(w, fmt.Sprintf("Failed to get failed jobs: %v", err), http.StatusInternalServerError)
		return
	}

	var targetMessage *models.FailedJob
	var targetMessageIdx int = -1

	for idx, msg := range messages {
		failedJob, err := h.dlqManager.ConvertToFailedJob(msg)
		if err != nil {
			continue
		}
		if failedJob.ID == request.JobID {
			targetMessage = failedJob
			targetMessageIdx = idx
			break
		}
	}

	if targetMessage == nil {
		// Nack all messages back to the queue
		for _, msg := range messages {
			if err := msg.Nack(false, true); err != nil {
				h.logger.Error("Failed to nack message", err)
			}
		}
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Determine original queue (default to image_upload if not specified)
	originalQueue := targetMessage.OriginalQueue
	if originalQueue == "" {
		originalQueue = messaging.QueueImageUpload
	}

	// Requeue the message to its original queue
	if err := h.rabbitmqClient.RequeueMessage(originalQueue, []byte(targetMessage.MessageBody)); err != nil {
		h.logger.Error("Failed to requeue message", err)
		// Nack all messages back to the queue
		for _, msg := range messages {
			if err := msg.Nack(false, true); err != nil {
				h.logger.Error("Failed to nack message", err)
			}
		}
		http.Error(w, fmt.Sprintf("Failed to requeue job: %v", err), http.StatusInternalServerError)
		return
	}

	// Acknowledge all messages except the one we requeued
	for idx, msg := range messages {
		if idx == targetMessageIdx {
			// Acknowledge the requeued message (remove from DLQ)
			if err := msg.Ack(false); err != nil {
				h.logger.Error("Failed to ack message", err)
			}
		} else {
			// Nack others back to the queue
			if err := msg.Nack(false, true); err != nil {
				h.logger.Error("Failed to nack message", err)
			}
		}
	}

	h.logger.Info(fmt.Sprintf("Requeued job %s to queue %s", request.JobID, originalQueue), nil)

	response := map[string]interface{}{
		"status":         "success",
		"message":        "Job requeued successfully",
		"job_id":         request.JobID,
		"original_queue": originalQueue,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", err)
	}
}

// FailedJobsUI serves the HTML UI for managing failed jobs
func (h *AdminHandler) FailedJobsUI(w http.ResponseWriter, _ *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Failed Jobs - Dead Letter Queue</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background-color: #f5f5f5;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            padding: 30px;
        }

        h1 {
            color: #333;
            margin-bottom: 10px;
        }

        .subtitle {
            color: #666;
            margin-bottom: 30px;
        }

        .stats {
            display: flex;
            gap: 20px;
            margin-bottom: 30px;
        }

        .stat-card {
            flex: 1;
            padding: 20px;
            background-color: #f8f9fa;
            border-radius: 6px;
            border-left: 4px solid #007bff;
        }

        .stat-label {
            font-size: 14px;
            color: #666;
            margin-bottom: 5px;
        }

        .stat-value {
            font-size: 32px;
            font-weight: bold;
            color: #333;
        }

        .controls {
            margin-bottom: 20px;
            display: flex;
            gap: 10px;
        }

        button {
            background-color: #007bff;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
            transition: background-color 0.2s;
        }

        button:hover {
            background-color: #0056b3;
        }

        button:disabled {
            background-color: #ccc;
            cursor: not-allowed;
        }

        .job-list {
            border: 1px solid #ddd;
            border-radius: 6px;
            overflow: hidden;
        }

        .job-item {
            border-bottom: 1px solid #ddd;
            padding: 20px;
            transition: background-color 0.2s;
        }

        .job-item:last-child {
            border-bottom: none;
        }

        .job-item:hover {
            background-color: #f8f9fa;
        }

        .job-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }

        .job-id {
            font-family: monospace;
            font-size: 14px;
            color: #666;
        }

        .job-queue {
            display: inline-block;
            background-color: #e7f3ff;
            color: #0066cc;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
        }

        .job-info {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin-bottom: 15px;
        }

        .info-item {
            font-size: 14px;
        }

        .info-label {
            color: #666;
            margin-bottom: 3px;
        }

        .info-value {
            color: #333;
            font-weight: 500;
        }

        .error-reason {
            background-color: #fff3cd;
            border-left: 4px solid #ffc107;
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 15px;
        }

        .error-label {
            font-size: 12px;
            color: #856404;
            font-weight: 600;
            margin-bottom: 5px;
        }

        .error-text {
            font-size: 14px;
            color: #856404;
            word-break: break-word;
        }

        .message-body {
            background-color: #f8f9fa;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 12px;
            margin-bottom: 15px;
            max-height: 200px;
            overflow-y: auto;
        }

        .message-body pre {
            font-size: 12px;
            color: #333;
            white-space: pre-wrap;
            word-break: break-word;
        }

        .job-actions {
            display: flex;
            gap: 10px;
        }

        .btn-retry {
            background-color: #28a745;
        }

        .btn-retry:hover {
            background-color: #218838;
        }

        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: #666;
        }

        .empty-state-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #666;
        }

        .error {
            background-color: #f8d7da;
            border: 1px solid #f5c6cb;
            color: #721c24;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }

        .success {
            background-color: #d4edda;
            border: 1px solid #c3e6cb;
            color: #155724;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Dead Letter Queue</h1>
        <p class="subtitle">Monitor and manage failed jobs</p>

        <div class="stats">
            <div class="stat-card">
                <div class="stat-label">Total Failed Jobs</div>
                <div class="stat-value" id="totalJobs">-</div>
            </div>
        </div>

        <div class="controls">
            <button onclick="loadFailedJobs()">Refresh</button>
        </div>

        <div id="alert"></div>
        <div id="jobList"></div>
    </div>

    <script>
        let jobs = [];

        async function loadFailedJobs() {
            const jobList = document.getElementById('jobList');
            const alert = document.getElementById('alert');
            const totalJobs = document.getElementById('totalJobs');

            jobList.innerHTML = '<div class="loading">Loading...</div>';
            alert.innerHTML = '';

            try {
                const response = await fetch('/admin/failed-jobs/api');
                const data = await response.json();

                jobs = data.jobs || [];
                totalJobs.textContent = data.count || 0;

                if (jobs.length === 0) {
                    jobList.innerHTML = `
                        <div class="empty-state">
                            <div class="empty-state-icon">✓</div>
                            <h2>No Failed Jobs</h2>
                            <p>All jobs are processing successfully</p>
                        </div>
                    `;
                } else {
                    renderJobs();
                }
            } catch (error) {
                jobList.innerHTML = '';
                alert.innerHTML = '<div class="error">Failed to load jobs: ' + error.message + '</div>';
            }
        }

        function renderJobs() {
            const jobList = document.getElementById('jobList');

            jobList.innerHTML = '<div class="job-list">' + jobs.map(job => `
                <div class="job-item">
                    <div class="job-header">
                        <div class="job-id">ID: ${job.id}</div>
                        <div class="job-queue">${job.original_queue || 'unknown'}</div>
                    </div>

                    <div class="error-reason">
                        <div class="error-label">ERROR REASON</div>
                        <div class="error-text">${job.error_reason}</div>
                    </div>

                    <div class="job-info">
                        <div class="info-item">
                            <div class="info-label">Failed At</div>
                            <div class="info-value">${new Date(job.failed_at).toLocaleString()}</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">Retry Count</div>
                            <div class="info-value">${job.retry_count}</div>
                        </div>
                        ${job.last_retry_at ? `
                        <div class="info-item">
                            <div class="info-label">Last Retry</div>
                            <div class="info-value">${new Date(job.last_retry_at).toLocaleString()}</div>
                        </div>
                        ` : ''}
                    </div>

                    <div class="message-body">
                        <pre>${JSON.stringify(JSON.parse(job.message_body), null, 2)}</pre>
                    </div>

                    <div class="job-actions">
                        <button class="btn-retry" onclick="retryJob('${job.id}')">Retry Job</button>
                    </div>
                </div>
            `).join('') + '</div>';
        }

        async function retryJob(jobId) {
            const alert = document.getElementById('alert');
            alert.innerHTML = '';

            try {
                const response = await fetch('/admin/failed-jobs/requeue', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ job_id: jobId })
                });

                const data = await response.json();

                if (response.ok) {
                    alert.innerHTML = '<div class="success">Job ' + jobId + ' requeued successfully to ' + data.original_queue + '</div>';
                    setTimeout(() => {
                        loadFailedJobs();
                    }, 1500);
                } else {
                    throw new Error(data.message || 'Failed to requeue job');
                }
            } catch (error) {
                alert.innerHTML = '<div class="error">Failed to retry job: ' + error.message + '</div>';
            }
        }

        // Load jobs on page load
        loadFailedJobs();

        // Auto-refresh every 30 seconds
        setInterval(loadFailedJobs, 30000);
    </script>
</body>
</html>
`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(html)); err != nil {
		h.logger.Error("Failed to write HTML response", err)
	}
}
