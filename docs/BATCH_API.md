# Batch Processing API Documentation

The Batch Processing API allows you to process multiple images at once through HTTP endpoints, with real-time progress tracking via WebSocket.

## Table of Contents

- [Overview](#overview)
- [Endpoints](#endpoints)
- [WebSocket Updates](#websocket-updates)
- [Usage Examples](#usage-examples)
- [Error Handling](#error-handling)

## Overview

The Batch Processing API provides the following capabilities:

- Submit multiple images (up to 100) for processing in a single request
- Support for both URL-based and base64-encoded images
- Track processing progress in real-time via WebSocket
- Query batch job status at any time
- List all batch jobs

## Endpoints

### 1. Create Batch Job

Creates a new batch processing job with multiple images.

**Endpoint:** `POST /api/v1/batch`

**Request Body:**
```json
{
  "images": [
    {
      "url": "https://example.com/image1.jpg",
      "name": "optional-filename.jpg"
    },
    {
      "base64": "data:image/jpeg;base64,/9j/4AAQSkZJRg...",
      "name": "another-image.jpg"
    }
  ]
}
```

**Request Fields:**
- `images` (array, required): Array of image sources (max 100)
  - `url` (string, optional): URL to download the image from
  - `base64` (string, optional): Base64-encoded image data
  - `name` (string, optional): Filename for the image
  - **Note:** Each image must have either `url` or `base64`, but not both

**Response:** `201 Created`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2025-11-18T10:30:00Z",
  "message": "Batch job created with 2 images"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "No images provided",
  "status": 400
}
```

### 2. Get Batch Job Status

Retrieves the current status of a batch job.

**Endpoint:** `GET /api/v1/batch/{job_id}`

**Response:** `200 OK`
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
      "original_filename": "image1.jpg",
      "status": "completed",
      "trace_id": "abc-123-def",
      "processed_path": "abc-123-def/image1.jpg",
      "start_time": "2025-11-18T10:30:01Z",
      "end_time": "2025-11-18T10:30:15Z"
    },
    {
      "index": 1,
      "original_filename": "image2.jpg",
      "status": "processing",
      "trace_id": "xyz-456-ghi",
      "start_time": "2025-11-18T10:30:02Z"
    }
  ],
  "created_at": "2025-11-18T10:30:00Z",
  "updated_at": "2025-11-18T10:30:15Z"
}
```

**Status Values:**
- `pending`: Job is waiting to start
- `processing`: Job is currently processing images
- `completed`: All images have been processed
- `failed`: All images failed to process
- `cancelled`: Job was cancelled

**Image Status Values:**
- `pending`: Image is waiting to be processed
- `processing`: Image is currently being processed
- `completed`: Image has been successfully processed
- `failed`: Image processing failed

**Error Response:** `404 Not Found`
```json
{
  "error": "Job not found: 550e8400-e29b-41d4-a716-446655440000",
  "status": 404
}
```

### 3. List All Batch Jobs

Lists all batch jobs currently stored in the system.

**Endpoint:** `GET /api/v1/batch`

**Response:** `200 OK`
```json
{
  "jobs": [
    {
      "job_id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "completed",
      "total_images": 2,
      "completed": 2,
      "failed": 0,
      "progress": 100.0,
      "images": [...],
      "created_at": "2025-11-18T10:30:00Z",
      "updated_at": "2025-11-18T10:31:00Z",
      "completed_at": "2025-11-18T10:31:00Z"
    }
  ],
  "total": 1
}
```

### 4. WebSocket Connection

Connect to receive real-time progress updates for a batch job.

**Endpoint:** `WS /api/v1/batch/{job_id}/ws`

**WebSocket Messages:**

The server sends JSON messages with the following structure:

```json
{
  "type": "progress",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "progress": 50.0,
  "completed": 1,
  "failed": 0,
  "total_images": 2,
  "timestamp": "2025-11-18T10:30:15Z"
}
```

**Message Types:**
- `progress`: General progress update
- `image_complete`: An individual image has finished processing
- `job_complete`: The entire batch job is complete

**Example `image_complete` message:**
```json
{
  "type": "image_complete",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "processing",
  "progress": 50.0,
  "completed": 1,
  "failed": 0,
  "total_images": 2,
  "image": {
    "index": 0,
    "original_filename": "image1.jpg",
    "status": "completed",
    "trace_id": "abc-123-def",
    "processed_path": "abc-123-def/image1.jpg"
  },
  "timestamp": "2025-11-18T10:30:15Z"
}
```

## Usage Examples

### Example 1: Submit Batch with URLs

```bash
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "images": [
      {
        "url": "https://example.com/photo1.jpg",
        "name": "vacation-photo-1.jpg"
      },
      {
        "url": "https://example.com/photo2.jpg",
        "name": "vacation-photo-2.jpg"
      },
      {
        "url": "https://example.com/photo3.jpg",
        "name": "vacation-photo-3.jpg"
      }
    ]
  }'
```

**Response:**
```json
{
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "pending",
  "created_at": "2025-11-18T10:30:00Z",
  "message": "Batch job created with 3 images"
}
```

### Example 2: Submit Batch with Base64 Images

```bash
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "images": [
      {
        "base64": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD...",
        "name": "screenshot-1.jpg"
      },
      {
        "base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
        "name": "diagram.png"
      }
    ]
  }'
```

### Example 3: Check Batch Status

```bash
curl http://localhost:8080/api/v1/batch/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

**Response:**
```json
{
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "processing",
  "total_images": 3,
  "completed": 2,
  "failed": 0,
  "progress": 66.67,
  "images": [
    {
      "index": 0,
      "original_filename": "vacation-photo-1.jpg",
      "status": "completed",
      "trace_id": "trace-001",
      "processed_path": "trace-001/vacation-photo-1.jpg",
      "start_time": "2025-11-18T10:30:01Z",
      "end_time": "2025-11-18T10:30:45Z"
    },
    {
      "index": 1,
      "original_filename": "vacation-photo-2.jpg",
      "status": "completed",
      "trace_id": "trace-002",
      "processed_path": "trace-002/vacation-photo-2.jpg",
      "start_time": "2025-11-18T10:30:02Z",
      "end_time": "2025-11-18T10:31:10Z"
    },
    {
      "index": 2,
      "original_filename": "vacation-photo-3.jpg",
      "status": "processing",
      "trace_id": "trace-003",
      "start_time": "2025-11-18T10:30:03Z"
    }
  ],
  "created_at": "2025-11-18T10:30:00Z",
  "updated_at": "2025-11-18T10:31:10Z"
}
```

### Example 4: WebSocket Client (JavaScript)

```javascript
const jobId = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890';
const ws = new WebSocket(`ws://localhost:8080/api/v1/batch/${jobId}/ws`);

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);

  console.log(`Progress: ${update.progress.toFixed(2)}%`);
  console.log(`Completed: ${update.completed}/${update.total_images}`);
  console.log(`Failed: ${update.failed}`);

  if (update.type === 'image_complete') {
    console.log(`Image ${update.image.original_filename} completed`);
    if (update.image.status === 'failed') {
      console.error(`Error: ${update.image.error}`);
    }
  }

  if (update.type === 'job_complete') {
    console.log('Batch job completed!');
    ws.close();
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
};
```

### Example 5: WebSocket Client (Python)

```python
import asyncio
import websockets
import json

async def watch_batch_progress(job_id):
    uri = f"ws://localhost:8080/api/v1/batch/{job_id}/ws"

    async with websockets.connect(uri) as websocket:
        print("WebSocket connected")

        async for message in websocket:
            update = json.loads(message)

            print(f"Progress: {update['progress']:.2f}%")
            print(f"Completed: {update['completed']}/{update['total_images']}")
            print(f"Failed: {update['failed']}")

            if update['type'] == 'image_complete':
                image = update['image']
                print(f"Image {image['original_filename']} completed")

                if image['status'] == 'failed':
                    print(f"Error: {image.get('error', 'Unknown error')}")

            if update['type'] == 'job_complete':
                print("Batch job completed!")
                break

# Usage
asyncio.run(watch_batch_progress('a1b2c3d4-e5f6-7890-abcd-ef1234567890'))
```

### Example 6: Complete Workflow (Python)

```python
import requests
import json
import time

# 1. Create batch job
response = requests.post('http://localhost:8080/api/v1/batch', json={
    'images': [
        {'url': 'https://example.com/image1.jpg'},
        {'url': 'https://example.com/image2.jpg'},
        {'url': 'https://example.com/image3.jpg'},
    ]
})

job_data = response.json()
job_id = job_data['job_id']
print(f"Created batch job: {job_id}")

# 2. Poll for status
while True:
    response = requests.get(f'http://localhost:8080/api/v1/batch/{job_id}')
    status = response.json()

    print(f"Progress: {status['progress']:.2f}%")
    print(f"Status: {status['status']}")

    if status['status'] in ['completed', 'failed']:
        break

    time.sleep(5)

# 3. Display results
print("\nFinal Results:")
for image in status['images']:
    print(f"- {image['original_filename']}: {image['status']}")
    if image['status'] == 'completed':
        print(f"  Processed path: {image['processed_path']}")
    elif image['status'] == 'failed':
        print(f"  Error: {image.get('error', 'Unknown error')}")
```

## Error Handling

### Common Error Codes

- `400 Bad Request`: Invalid request format or parameters
  - No images provided
  - Too many images (>100)
  - Invalid image source (missing both URL and base64)
  - Both URL and base64 provided

- `404 Not Found`: Job ID not found

- `405 Method Not Allowed`: Wrong HTTP method for endpoint

- `500 Internal Server Error`: Server-side processing error

### Error Response Format

All errors follow this format:

```json
{
  "error": "Error message describing what went wrong",
  "status": 400
}
```

### Image Processing Errors

Individual images can fail while the batch continues processing. Failed images will have:

```json
{
  "index": 0,
  "original_filename": "image.jpg",
  "status": "failed",
  "error": "Failed to download image: status code 404"
}
```

## Limits and Constraints

- **Maximum images per batch:** 100
- **Maximum image size:** 10 MB per image
- **Supported formats:** JPEG, PNG, JPG
- **Job retention:** Completed jobs are automatically deleted after 24 hours
- **WebSocket timeout:** 60 seconds of inactivity

## Architecture

The batch processing system works as follows:

1. **Job Creation**: Client submits batch request → Gateway creates job and stores in memory
2. **Image Upload**: Each image is downloaded/decoded → uploaded to MinIO
3. **Queue Publishing**: Image metadata is published to RabbitMQ `image_upload` queue
4. **Analysis**: Analyzer service processes images → generates metadata
5. **Processing**: Processor service writes metadata → uploads to MinIO
6. **Completion**: Gateway receives completion messages → updates job status → broadcasts via WebSocket

## Best Practices

1. **Use WebSocket for real-time updates** instead of polling the status endpoint
2. **Handle failed images gracefully** - partial success is common
3. **Set appropriate timeouts** for HTTP clients
4. **Monitor job progress** before assuming completion
5. **Clean up completed jobs** if you store job IDs locally
6. **Validate image URLs** before submission to avoid unnecessary failures
7. **Use descriptive filenames** to easily identify processed images

## Support

For issues or questions about the Batch Processing API, please check:
- System logs in Gateway service
- RabbitMQ message queue status
- MinIO storage availability
