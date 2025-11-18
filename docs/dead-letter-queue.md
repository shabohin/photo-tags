# Dead Letter Queue (DLQ) Implementation

## Overview

The Photo Tags service now includes a Dead Letter Queue (DLQ) implementation for handling failed message processing. This feature provides visibility into failed jobs and allows manual retry of failed messages.

## Features

- **Automatic DLQ Routing**: Failed messages are automatically sent to a dead letter queue instead of being lost or infinitely retried
- **Error Tracking**: Each failed job captures the error reason and retry count
- **Web UI**: Simple admin interface for viewing and managing failed jobs
- **Manual Retry**: Ability to manually requeue failed jobs back to their original queue
- **Queue Persistence**: All failed jobs are stored in RabbitMQ with full metadata

## Architecture

### Queue Configuration

The implementation uses RabbitMQ's built-in Dead Letter Exchange (DLX) feature:

1. **Main Queues**: Configured with `x-dead-letter-exchange` and `x-dead-letter-routing-key` parameters
2. **Dead Letter Queue**: A dedicated queue (`dead_letter_queue`) that receives failed messages
3. **Automatic Routing**: When a message is rejected with `requeue=false`, it's automatically sent to the DLQ

### Components

#### 1. Failed Job Model (`pkg/models/messages.go`)

```go
type FailedJob struct {
    ID            string    `json:"id"`
    OriginalQueue string    `json:"original_queue"`
    MessageBody   string    `json:"message_body"`
    ErrorReason   string    `json:"error_reason"`
    FailedAt      time.Time `json:"failed_at"`
    RetryCount    int       `json:"retry_count"`
    LastRetryAt   time.Time `json:"last_retry_at,omitempty"`
}
```

#### 2. DLQ Manager (`pkg/dlq/manager.go`)

Provides utilities for:
- Converting AMQP messages to FailedJob models
- Extracting metadata from message headers
- Managing DLQ operations

#### 3. RabbitMQ Client Extensions (`pkg/messaging/rabbitmq.go`)

New methods:
- `DeclareQueueWithDLQ()`: Declares a queue with DLQ configuration
- `PublishMessageWithHeaders()`: Publishes messages with custom headers
- `GetMessages()`: Retrieves messages from a queue for inspection
- `RequeueMessage()`: Republishes a message to its original queue

#### 4. Consumer Updates (`services/analyzer/internal/transport/rabbitmq/consumer.go`)

- Automatically declares DLQ when connecting
- Configures main queue with DLX parameters
- Sends failed messages to DLQ instead of requeuing

#### 5. Admin Endpoints (`services/gateway/internal/handler/admin_handler.go`)

Three endpoints:
- `GET /admin/failed-jobs` - Web UI for viewing failed jobs
- `GET /admin/failed-jobs/api` - JSON API for retrieving failed jobs
- `POST /admin/failed-jobs/requeue` - Requeue a specific failed job

## Usage

### Accessing the Admin UI

1. Start the services:
   ```bash
   ./scripts/start.sh
   ```

2. Open the DLQ admin interface:
   ```
   http://localhost:8080/admin/failed-jobs
   ```

### Web UI Features

The admin interface provides:

- **Job List**: View all failed jobs with details
- **Job Details**:
  - Job ID (unique identifier)
  - Original queue name
  - Error reason
  - Retry count
  - Failed timestamp
  - Full message body (formatted JSON)
- **Retry Button**: Manually requeue jobs back to their original queue
- **Auto-refresh**: Updates every 30 seconds
- **Manual Refresh**: Click the "Refresh" button

### API Endpoints

#### Get Failed Jobs

```bash
curl http://localhost:8080/admin/failed-jobs/api
```

Response:
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "original_queue": "image_upload",
      "message_body": "{\"trace_id\":\"abc123\",...}",
      "error_reason": "Failed to process message",
      "failed_at": "2025-11-18T19:30:00Z",
      "retry_count": 1
    }
  ],
  "count": 1,
  "timestamp": "2025-11-18T19:45:00Z"
}
```

#### Requeue Failed Job

```bash
curl -X POST http://localhost:8080/admin/failed-jobs/requeue \
  -H "Content-Type: application/json" \
  -d '{"job_id": "550e8400-e29b-41d4-a716-446655440000"}'
```

Response:
```json
{
  "status": "success",
  "message": "Job requeued successfully",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "original_queue": "image_upload",
  "timestamp": "2025-11-18T19:46:00Z"
}
```

## How It Works

### Message Processing Flow

1. **Normal Processing**:
   ```
   Message → Queue → Consumer → Process → ACK → Success
   ```

2. **Failed Processing**:
   ```
   Message → Queue → Consumer → Process Error → NACK (requeue=false) → DLQ
   ```

3. **Manual Retry**:
   ```
   DLQ → Admin UI → Requeue → Original Queue → Consumer → Process
   ```

### Error Information

When a message fails:

1. The consumer calls `msg.Nack(false, false)` (no requeue)
2. RabbitMQ automatically routes the message to the DLQ
3. RabbitMQ adds `x-death` header with metadata:
   - Original queue name
   - Failure count
   - Failure reason
   - Failure timestamp

### Retry Mechanism

When you retry a job:

1. The job is fetched from the DLQ
2. The original message body is extracted
3. The message is published to the original queue
4. The DLQ entry is acknowledged (removed from DLQ)
5. The message is processed normally by the consumer

## Monitoring

### RabbitMQ Management UI

You can also view the DLQ through RabbitMQ's management interface:

1. Open http://localhost:15672
2. Login with `user` / `password`
3. Navigate to "Queues" tab
4. Look for the `dead_letter_queue`

### Metrics

Key metrics to monitor:
- **DLQ Message Count**: Number of messages in the dead letter queue
- **Retry Success Rate**: Percentage of retried jobs that succeed
- **Error Patterns**: Common error reasons

## Configuration

### Queue Declaration

Queues are automatically configured with DLQ support in the Analyzer service. The configuration includes:

```go
amqp.Table{
    "x-dead-letter-exchange":    "",
    "x-dead-letter-routing-key": "dead_letter_queue",
}
```

### Consumer Behavior

The consumer now sends failed messages directly to DLQ instead of requeuing:

```go
// Old behavior
msg.Nack(false, true)  // requeue=true

// New behavior
msg.Nack(false, false) // requeue=false, sends to DLQ
```

## Best Practices

1. **Monitor DLQ Regularly**: Check the admin UI daily for failed jobs
2. **Investigate Error Patterns**: Look for common error reasons
3. **Retry After Fixes**: After fixing issues, retry failed jobs
4. **Clean Up Old Failures**: Remove or archive jobs that can't be retried
5. **Set Alerts**: Configure alerts when DLQ reaches a threshold

## Troubleshooting

### DLQ Not Receiving Messages

1. Verify queue is configured with DLX parameters
2. Check that consumer is using `Nack(false, false)`
3. Ensure DLQ is declared before main queue

### Retry Not Working

1. Check original queue name is correctly stored
2. Verify message body is valid JSON
3. Ensure the original queue still exists

### UI Not Loading

1. Verify Gateway service is running
2. Check that RabbitMQ client is initialized
3. Review Gateway logs for errors

## Future Enhancements

Potential improvements:
- Automatic retry with exponential backoff
- Batch retry operations
- DLQ message filtering and search
- Export failed jobs to CSV/JSON
- Alerting integration (email, Slack)
- Retention policies for old failed jobs
- Error pattern analysis and reporting

## Related Documentation

- [RabbitMQ Dead Letter Exchanges](https://www.rabbitmq.com/dlx.html)
- [Monitoring Guide](monitoring.md)
- Main [README](../README.md)
