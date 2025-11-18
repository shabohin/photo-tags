# File Watcher Service

File Watcher Service monitors input directories for new images, processes them through the photo-tags pipeline, and saves the results to an output directory. This enables batch processing of images without using the Telegram bot interface.

## Features

- **Automated File Monitoring**: Uses fsnotify or polling to detect new images
- **Batch Processing**: Process all files in input directory at once
- **File Validation**: Validates image format (JPG, JPEG, PNG) and file size
- **MinIO Integration**: Uploads images to MinIO storage
- **RabbitMQ Integration**: Publishes messages to processing queue and consumes results
- **REST API**: HTTP endpoints for manual scanning and statistics
- **Statistics Tracking**: Tracks files processed, success/failure rates, and errors
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals properly
- **Metadata Export**: Saves processing metadata as JSON files alongside images

## Architecture

```
┌─────────────┐
│   Input     │
│  Directory  │
└──────┬──────┘
       │
       v
┌─────────────┐     ┌──────────┐     ┌──────────────┐
│   Watcher   │────>│  MinIO   │────>│   RabbitMQ   │
│  (fsnotify) │     │ (upload) │     │ (image_upload)│
└─────────────┘     └──────────┘     └──────────────┘
                                             │
                                             v
                                      ┌──────────────┐
                                      │   Analyzer   │
                                      │   Service    │
                                      └──────┬───────┘
                                             │
                                             v
                                      ┌──────────────┐
                                      │  Processor   │
                                      │   Service    │
                                      └──────┬───────┘
                                             │
       ┌─────────────────────────────────────┘
       │
       v
┌─────────────┐     ┌──────────┐     ┌──────────────┐
│  Consumer   │<────│  MinIO   │<────│   RabbitMQ   │
│             │     │(download)│     │(image_processed)│
└──────┬──────┘     └──────────┘     └──────────────┘
       │
       v
┌─────────────┐
│   Output    │
│  Directory  │
└─────────────┘
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `INPUT_DIR` | `/app/input` | Directory to watch for new images |
| `OUTPUT_DIR` | `/app/output` | Directory to save processed images |
| `PROCESSED_DIR` | `/app/input/processed` | Directory to move processed files |
| `SCAN_INTERVAL_SECONDS` | `5` | Polling interval in seconds (if fsnotify disabled) |
| `USE_FSNOTIFY` | `true` | Use fsnotify for file system events |
| `MAX_FILE_SIZE_MB` | `50` | Maximum file size in megabytes |
| `SERVER_PORT` | `8081` | HTTP API server port |
| `RABBITMQ_URL` | `amqp://user:password@localhost:5672/` | RabbitMQ connection URL |
| `MINIO_ENDPOINT` | `localhost:9000` | MinIO server endpoint |
| `MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key |
| `MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key |
| `MINIO_USE_SSL` | `false` | Use SSL for MinIO connection |

## API Endpoints

### Health Check
```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "service": "filewatcher",
  "time": "2025-01-15T10:30:00Z"
}
```

### Statistics
```bash
GET /stats
```

Response:
```json
{
  "files_processed": 42,
  "files_successful": 40,
  "files_failed": 2,
  "files_received": 38,
  "last_processed_time": "2025-01-15T10:29:45Z",
  "start_time": "2025-01-15T09:00:00Z",
  "recent_errors": [
    {
      "timestamp": "2025-01-15T10:15:30Z",
      "message": "File too large: 52428800 bytes (max: 52428800 bytes)",
      "trace_id": "abc123-def456"
    }
  ]
}
```

### Manual Scan
```bash
POST /scan
```

Response:
```json
{
  "status": "started",
  "message": "Manual scan initiated",
  "time": "2025-01-15T10:30:00Z"
}
```

## Usage

### Using Docker Compose

1. **Start the service**:
```bash
cd docker
docker-compose up -d filewatcher
```

2. **Add images to input directory**:
```bash
# Copy images to the input volume
docker cp image.jpg filewatcher:/app/input/
```

3. **Check statistics**:
```bash
curl http://localhost:8081/stats
```

4. **Trigger manual scan**:
```bash
curl -X POST http://localhost:8081/scan
```

5. **Get processed images**:
```bash
# Copy from output volume
docker cp filewatcher:/app/output/image.jpg ./
```

### Using Local Directories

You can also mount local directories as volumes:

```yaml
volumes:
  - ./data/input:/app/input
  - ./data/output:/app/output
```

Then simply drop images into `./data/input` and find processed results in `./data/output`.

## File Processing Flow

1. **Detection**: File watcher detects new image in input directory
2. **Validation**: Checks file extension and size
3. **Upload**: Uploads to MinIO `original` bucket
4. **Queue**: Publishes `ImageUpload` message to RabbitMQ
5. **Processing**: Analyzer generates metadata, Processor embeds it
6. **Download**: Consumer receives `ImageProcessed` message
7. **Save**: Downloads from MinIO `processed` bucket and saves to output directory
8. **Metadata**: Creates `.json` file with processing metadata
9. **Cleanup**: Moves original file to processed directory

## Logging

All operations are logged in JSON format with trace IDs for correlation:

```json
{
  "level": "info",
  "service": "filewatcher",
  "message": "Processing file",
  "trace_id": "abc123-def456",
  "file": "/app/input/image.jpg"
}
```

## Development

### Building Locally

```bash
cd services/filewatcher
go mod download
go build -o bin/filewatcher cmd/main.go
```

### Running Tests

```bash
go test ./...
```

### Running Locally

```bash
export INPUT_DIR=./data/input
export OUTPUT_DIR=./data/output
export PROCESSED_DIR=./data/processed
export RABBITMQ_URL=amqp://user:password@localhost:5672/
export MINIO_ENDPOINT=localhost:9000

./bin/filewatcher
```

## Troubleshooting

### Files not being processed

1. Check file permissions on input directory
2. Verify file extension is supported (jpg, jpeg, png)
3. Check file size is under MAX_FILE_SIZE_MB
4. Review logs: `docker logs filewatcher`

### Output directory empty

1. Check RabbitMQ connection
2. Verify Analyzer and Processor services are running
3. Check statistics for errors: `curl http://localhost:8081/stats`
4. Review consumer logs for download errors

### High memory usage

1. Reduce SCAN_INTERVAL_SECONDS for less frequent scanning
2. Process smaller batches of files
3. Increase MAX_FILE_SIZE_MB limit to filter out large files

## License

This service is part of the photo-tags project.
