# Statistics API Documentation

The Photo Tags service now includes PostgreSQL-backed statistics and history tracking. This document describes the available API endpoints for querying statistics.

## Database Schema

### Tables

#### images
Tracks all image processing history:
- `id`: Serial primary key
- `trace_id`: Unique trace ID for the image
- `telegram_id`: Telegram user ID
- `telegram_username`: Telegram username (optional)
- `filename`: Original filename
- `original_path`: Path in MinIO original bucket
- `processed_path`: Path in MinIO processed bucket
- `status`: Processing status (pending, processing, success, failed)
- `error_message`: Error message if failed
- `metadata`: JSONB field with image metadata
- `created_at`: Timestamp when record was created
- `updated_at`: Timestamp when record was last updated

#### processing_stats
Daily aggregated statistics:
- `id`: Serial primary key
- `date`: Date of statistics
- `total_images`: Total number of images processed
- `successful_images`: Number of successfully processed images
- `failed_images`: Number of failed images
- `pending_images`: Number of pending/processing images
- `total_users`: Number of unique users
- `avg_processing_time_ms`: Average processing time in milliseconds
- `created_at`: Timestamp when record was created
- `updated_at`: Timestamp when record was last updated

#### errors
Detailed error tracking:
- `id`: Serial primary key
- `trace_id`: Associated trace ID (optional)
- `service`: Service that generated the error
- `error_type`: Type of error
- `error_message`: Error message
- `stack_trace`: Stack trace (optional)
- `telegram_id`: Associated user ID (optional)
- `filename`: Associated filename (optional)
- `metadata`: JSONB field with additional error context
- `created_at`: Timestamp when error occurred

## API Endpoints

All endpoints are prefixed with `/api/v1` and return JSON responses.

### 1. Get User Images

Retrieves images for a specific user with pagination.

**Endpoint:** `GET /api/v1/stats/user/images`

**Query Parameters:**
- `telegram_id` (required): Telegram user ID
- `limit` (optional): Number of results per page (default: 50, max: 100)
- `offset` (optional): Pagination offset (default: 0)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/stats/user/images?telegram_id=123456789&limit=10&offset=0"
```

**Response:**
```json
{
  "images": [
    {
      "id": 1,
      "trace_id": "abc-123-def",
      "telegram_id": 123456789,
      "telegram_username": "john_doe",
      "filename": "photo.jpg",
      "original_path": "abc-123-def/photo.jpg",
      "processed_path": "abc-123-def/photo_processed.jpg",
      "status": "success",
      "metadata": {
        "title": "Sunset Beach",
        "description": "Beautiful sunset at the beach",
        "keywords": ["sunset", "beach", "nature"]
      },
      "created_at": "2025-11-18T12:00:00Z",
      "updated_at": "2025-11-18T12:01:00Z"
    }
  ],
  "count": 1,
  "limit": 10,
  "offset": 0
}
```

### 2. Get User Statistics Summary

Retrieves statistics summary for a specific user.

**Endpoint:** `GET /api/v1/stats/user/summary`

**Query Parameters:**
- `telegram_id` (required): Telegram user ID

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/stats/user/summary?telegram_id=123456789"
```

**Response:**
```json
{
  "telegram_id": 123456789,
  "stats": {
    "pending": 2,
    "processing": 0,
    "success": 145,
    "failed": 3
  }
}
```

### 3. Get Daily Statistics

Retrieves daily processing statistics for a date range.

**Endpoint:** `GET /api/v1/stats/daily`

**Query Parameters:**
- `start_date` (optional): Start date in YYYY-MM-DD format (default: 7 days ago)
- `end_date` (optional): End date in YYYY-MM-DD format (default: today)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/stats/daily?start_date=2025-11-10&end_date=2025-11-18"
```

**Response:**
```json
{
  "stats": [
    {
      "id": 1,
      "date": "2025-11-18T00:00:00Z",
      "total_images": 50,
      "successful_images": 48,
      "failed_images": 2,
      "pending_images": 0,
      "total_users": 15,
      "avg_processing_time_ms": 2500,
      "created_at": "2025-11-18T23:59:59Z",
      "updated_at": "2025-11-18T23:59:59Z"
    }
  ],
  "start_date": "2025-11-10",
  "end_date": "2025-11-18"
}
```

### 4. Get Recent Errors

Retrieves recent errors with optional service filter.

**Endpoint:** `GET /api/v1/stats/errors`

**Query Parameters:**
- `service` (optional): Filter by service name (e.g., "gateway", "processor", "analyzer")
- `limit` (optional): Number of results (default: 50, max: 100)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/stats/errors?service=processor&limit=20"
```

**Response:**
```json
{
  "errors": [
    {
      "id": 1,
      "trace_id": "abc-123-def",
      "service": "processor",
      "error_type": "processing_error",
      "error_message": "Failed to extract EXIF data",
      "telegram_id": 123456789,
      "filename": "photo.jpg",
      "created_at": "2025-11-18T12:00:00Z"
    }
  ],
  "count": 1,
  "limit": 20
}
```

### 5. Get Error Statistics

Retrieves error statistics grouped by type for a date range.

**Endpoint:** `GET /api/v1/stats/errors/summary`

**Query Parameters:**
- `start_date` (optional): Start date in YYYY-MM-DD format (default: 7 days ago)
- `end_date` (optional): End date in YYYY-MM-DD format (default: today)

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/stats/errors/summary?start_date=2025-11-10&end_date=2025-11-18"
```

**Response:**
```json
{
  "stats": {
    "processing_error": 15,
    "network_error": 5,
    "timeout_error": 2
  },
  "start_date": "2025-11-10",
  "end_date": "2025-11-18"
}
```

### 6. Get Image by Trace ID

Retrieves image details by trace ID.

**Endpoint:** `GET /api/v1/images/trace`

**Query Parameters:**
- `trace_id` (required): Trace ID of the image

**Example Request:**
```bash
curl "http://localhost:8080/api/v1/images/trace?trace_id=abc-123-def"
```

**Response:**
```json
{
  "id": 1,
  "trace_id": "abc-123-def",
  "telegram_id": 123456789,
  "telegram_username": "john_doe",
  "filename": "photo.jpg",
  "original_path": "abc-123-def/photo.jpg",
  "processed_path": "abc-123-def/photo_processed.jpg",
  "status": "success",
  "metadata": {
    "title": "Sunset Beach",
    "description": "Beautiful sunset at the beach",
    "keywords": ["sunset", "beach", "nature"]
  },
  "created_at": "2025-11-18T12:00:00Z",
  "updated_at": "2025-11-18T12:01:00Z"
}
```

## Environment Variables

The following environment variables are required for PostgreSQL:

```bash
POSTGRES_HOST=localhost          # PostgreSQL host
POSTGRES_PORT=5432              # PostgreSQL port
POSTGRES_DB=photo_tags          # Database name
POSTGRES_USER=photo_tags_user   # Database user
POSTGRES_PASSWORD=photo_tags_password  # Database password
POSTGRES_SSL_MODE=disable       # SSL mode (disable, require, verify-ca, verify-full)
```

## Running with Docker Compose

The PostgreSQL service is automatically configured in `docker-compose.yml`. To start all services including PostgreSQL:

```bash
cd docker
docker-compose up -d
```

The database will be automatically initialized with the schema migrations on the first run.

## Testing the API

You can test the API endpoints using curl or any HTTP client:

```bash
# Get user images
curl "http://localhost:8080/api/v1/stats/user/images?telegram_id=123456789"

# Get daily stats
curl "http://localhost:8080/api/v1/stats/daily"

# Get recent errors
curl "http://localhost:8080/api/v1/stats/errors?limit=10"
```

## Error Handling

All endpoints return standard HTTP status codes:
- `200 OK`: Successful request
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

Error responses include a message:
```json
{
  "error": "Error message here"
}
```
