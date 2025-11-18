# Batch Processing API - Feature Summary

## Overview

This feature adds a REST API endpoint for batch processing of images through HTTP, with real-time progress tracking via WebSocket. Previously, images could only be processed one at a time through the Telegram bot interface or by placing files in folders.

## What's New

### 1. REST API Endpoints

- **POST /api/v1/batch** - Submit a batch of images for processing
- **GET /api/v1/batch/{job_id}** - Get the status of a batch job
- **GET /api/v1/batch** - List all batch jobs
- **WS /api/v1/batch/{job_id}/ws** - WebSocket connection for real-time progress updates

### 2. Image Input Support

The API accepts images from two sources:
- **URLs**: Direct image URLs that will be downloaded
- **Base64**: Base64-encoded image data (with or without data URI prefix)

### 3. Real-time Progress Tracking

WebSocket connections provide real-time updates about:
- Overall batch progress percentage
- Individual image processing status
- Completed and failed image counts
- Error messages for failed images

## Technical Implementation

### New Components

#### 1. Batch Storage (`services/gateway/internal/batch/storage.go`)
- In-memory storage for batch jobs
- Thread-safe with mutex protection
- Auto-cleanup of completed jobs after 24 hours
- Tracks job status, progress, and individual image states

#### 2. WebSocket Hub (`services/gateway/internal/batch/websocket.go`)
- Manages WebSocket connections for multiple clients
- Broadcasts progress updates to all connected clients
- Supports multiple concurrent batch jobs
- Handles client registration/unregistration

#### 3. Batch Processor (`services/gateway/internal/batch/processor.go`)
- Downloads images from URLs or decodes base64 data
- Uploads images to MinIO storage
- Publishes messages to RabbitMQ for processing pipeline
- Tracks progress and updates job status
- Integrates with existing Analyzer and Processor services

#### 4. HTTP Handler (`services/gateway/internal/batch/handler.go`)
- REST API endpoints for batch operations
- Request validation and error handling
- JSON request/response handling
- WebSocket upgrade handling

#### 5. Data Models (`pkg/models/batch.go`)
- `BatchJob` - Represents a batch processing job
- `BatchImageStatus` - Tracks individual image status within a batch
- `ImageSource` - Defines image input (URL or base64)
- `WSProgressUpdate` - WebSocket message format

### Integration

The batch processing system integrates seamlessly with the existing architecture:

1. **Gateway Service** - Extended to include batch API endpoints
2. **MinIO Storage** - Uses existing storage buckets (`original` and `processed`)
3. **RabbitMQ Queues** - Uses existing queues (`image_upload`, `image_processed`)
4. **Analyzer Service** - No changes required, processes batch images like regular ones
5. **Processor Service** - No changes required, handles batch images automatically

### Architecture Flow

```
Client → POST /api/v1/batch → Gateway (Batch Handler)
                                  ↓
                        Create Batch Job in Storage
                                  ↓
                        For each image in batch:
                                  ↓
                    Download from URL or Decode Base64
                                  ↓
                        Upload to MinIO (original bucket)
                                  ↓
                    Publish to RabbitMQ (image_upload queue)
                                  ↓
                        Analyzer Service (AI analysis)
                                  ↓
                    Publish to RabbitMQ (metadata_generated queue)
                                  ↓
                        Processor Service (write metadata)
                                  ↓
                    Upload to MinIO (processed bucket)
                                  ↓
                    Publish to RabbitMQ (image_processed queue)
                                  ↓
            Gateway consumes message → Update batch job status
                                  ↓
            Broadcast progress update via WebSocket
                                  ↓
                            Client receives update
```

## Files Changed/Added

### New Files
- `pkg/models/batch.go` - Batch processing data models
- `services/gateway/internal/batch/storage.go` - Batch job storage
- `services/gateway/internal/batch/websocket.go` - WebSocket hub implementation
- `services/gateway/internal/batch/processor.go` - Batch processing logic
- `services/gateway/internal/batch/handler.go` - HTTP handlers for batch API
- `services/gateway/internal/batch/handler_test.go` - Unit tests
- `docs/BATCH_API.md` - Complete API documentation with examples

### Modified Files
- `services/gateway/cmd/main.go` - Initialize batch components
- `services/gateway/internal/handler/handler.go` - Integrate batch routes
- `services/gateway/go.mod` - Added `github.com/gorilla/websocket` dependency

## Configuration

No additional configuration is required. The batch processing system uses existing configuration:

- MinIO endpoint, credentials, and buckets
- RabbitMQ connection URL and queues
- Gateway HTTP server port (default: 8080)

## Usage Examples

### 1. Submit a Batch Job (cURL)

```bash
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "images": [
      {"url": "https://example.com/photo1.jpg", "name": "photo1.jpg"},
      {"url": "https://example.com/photo2.jpg", "name": "photo2.jpg"}
    ]
  }'
```

Response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2025-11-18T10:30:00Z",
  "message": "Batch job created with 2 images"
}
```

### 2. Check Job Status

```bash
curl http://localhost:8080/api/v1/batch/550e8400-e29b-41d4-a716-446655440000
```

Response:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "total_images": 2,
  "completed": 1,
  "failed": 0,
  "progress": 50.0,
  "images": [
    {
      "index": 0,
      "original_filename": "photo1.jpg",
      "status": "completed",
      "trace_id": "abc-123",
      "processed_path": "abc-123/photo1.jpg"
    },
    {
      "index": 1,
      "original_filename": "photo2.jpg",
      "status": "processing",
      "trace_id": "def-456"
    }
  ],
  "created_at": "2025-11-18T10:30:00Z",
  "updated_at": "2025-11-18T10:30:45Z"
}
```

### 3. WebSocket Connection (JavaScript)

```javascript
const jobId = '550e8400-e29b-41d4-a716-446655440000';
const ws = new WebSocket(`ws://localhost:8080/api/v1/batch/${jobId}/ws`);

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(`Progress: ${update.progress}% (${update.completed}/${update.total_images})`);

  if (update.type === 'job_complete') {
    console.log('Batch job completed!');
    ws.close();
  }
};
```

For more detailed examples, see [docs/BATCH_API.md](docs/BATCH_API.md).

## Testing

Unit tests are included in `services/gateway/internal/batch/handler_test.go`.

Run tests:
```bash
cd services/gateway
go test ./internal/batch/... -v
```

## Limitations

- Maximum 100 images per batch
- Maximum 10 MB per image
- Completed jobs are automatically deleted after 24 hours
- WebSocket connection timeout: 60 seconds of inactivity
- Supported image formats: JPEG, PNG, JPG

## Future Enhancements

Potential improvements for future versions:

1. **Persistent Storage**: Move from in-memory to Redis or MongoDB for job persistence
2. **Job Cancellation**: Add ability to cancel in-progress batch jobs
3. **Priority Queue**: Allow high-priority batch jobs to skip the queue
4. **Batch Templates**: Save and reuse batch configurations
5. **Webhooks**: Send HTTP callbacks when batch jobs complete
6. **Progress Estimation**: Predict completion time based on historical data
7. **Batch Download**: Download all processed images as a single ZIP file
8. **Authentication**: Add API key or OAuth authentication
9. **Rate Limiting**: Prevent abuse with per-user rate limits
10. **Metrics Dashboard**: Web UI for monitoring batch jobs

## Monitoring

The batch processing system integrates with existing Datadog monitoring:

- Batch job creation events
- Processing progress metrics
- Error rates and failed images
- WebSocket connection metrics

## Troubleshooting

### Common Issues

1. **Job not progressing**
   - Check RabbitMQ connection and queues
   - Verify Analyzer and Processor services are running
   - Check MinIO storage availability

2. **WebSocket connection drops**
   - Ensure client sends ping/pong messages
   - Check network stability
   - Verify firewall allows WebSocket connections

3. **Images failing to download**
   - Verify URLs are accessible
   - Check image size (max 10 MB)
   - Ensure content-type is image/*

4. **Out of memory**
   - Reduce concurrent batch jobs
   - Implement Redis/MongoDB for persistent storage
   - Increase server memory allocation

## Security Considerations

1. **Input Validation**: All image sources are validated before processing
2. **Size Limits**: Maximum image size enforced (10 MB)
3. **URL Validation**: Only HTTP/HTTPS URLs accepted
4. **Content-Type Check**: Verifies image content-type before processing
5. **CORS**: WebSocket connections allow all origins (configure in production)

For production deployment, consider:
- Adding authentication (API keys, OAuth)
- Implementing rate limiting
- Restricting allowed image domains
- Adding request signing
- Enabling HTTPS/WSS only

## Documentation

Full API documentation with examples: [docs/BATCH_API.md](docs/BATCH_API.md)

## Support

For issues or questions:
1. Check system logs in Gateway service
2. Verify RabbitMQ message queue status
3. Confirm MinIO storage availability
4. Review batch job status via REST API
