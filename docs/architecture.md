# Architecture Documentation

This document provides a detailed overview of the Photo Tags Service architecture, describing its components, data structures, and communication flow.

## Documentation Links

-   [Main README](../README.md)
-   [Development Guide](development.md)
-   [Testing Strategy](testing.md)
-   [Deployment Guide](deployment.md)

## System Overview

Photo Tags Service uses a microservices architecture to process images and add metadata. The services communicate asynchronously through RabbitMQ message queues with Dead Letter Queue support, images are stored in MinIO object storage, and statistics are tracked in PostgreSQL database. The system includes Datadog integration for comprehensive monitoring and observability.

## Component Diagram

```
                         +---------------------+
                         |                     |
                         |    Gateway Service  |
                         | (receiving and sending) |
                         |                     |
                         +----------+----------+
                                    |
                                    | Publishes to queue
                                    | image_upload
                                    v
+------------------------------------------------+
|                                                |
|                     RabbitMQ                   |
|                                                |
|  +--------+    +----------+    +-----------+   |    +-----------+
|  |image_  |    |metadata_ |    |image_     |   |    |image_     |
|  |upload  |    |generated |    |process    |   |    |processed  |
|  +----+---+    +-----+----+    +-----+------+  |    +-----+-----+
|       |              ^                |         |          ^
|       |              |                |         |          |
+-------|--------------|-----------------|---------          |
        |              |                |                    |
        | Consumes from| Publishes to   | Consumes from      | Consumes from
        | image_upload | queue          | queue              | queue
        | queue        | metadata_      | image_process      | image_processed
        v              | generated      v                    |
  +-------------+      |          +-------------+            |
  |             |      |          |             |            |
  | Analyzer    +------+          | Processor   |            |
  | Service     |                 | Service     +------------+
  +-------------+                 |             |
                                  +-------------+

Storage:
+-------------+
|             |
|    MinIO    |
| (2 buckets) |
+-------------+
```

## Components

### 1. Gateway Service

The Gateway Service acts as the entry point for user interactions through a Telegram bot interface and provides REST APIs for statistics and administration.

**Responsibilities:**

-   Receive images from users via Telegram API
-   Validate image formats (JPG/PNG)
-   Upload original images to MinIO
-   Publish image processing tasks to the `image_upload` queue
-   Receive processed images from the `image_processed` queue
-   Send processed images back to users
-   Provide Statistics API (PostgreSQL-backed)
-   Provide Dead Letter Queue admin interface
-   Track processing history and errors

**Technologies:**

-   Go
-   Telegram Bot API
-   RabbitMQ client
-   PostgreSQL
-   MinIO SDK

### 2. Analyzer Service

The Analyzer Service processes images with AI to generate appropriate metadata using automatically selected free vision models.

**Responsibilities:**

-   Consume tasks from the `image_upload` queue
-   Download images from MinIO
-   Automatically select best available free vision model from OpenRouter
-   Interact with OpenRouter API using selected model to analyze images
-   Handle rate limits with intelligent retry scheduling
-   Generate metadata (title, description, keywords)
-   Publish results to the `metadata_generated` queue

**Technologies:**

-   Go
-   OpenRouter API with dynamic model selection
-   MinIO SDK
-   RabbitMQ client
-   Model Selector service (periodic updates)

### 3. Processor Service

The Processor Service writes metadata into image files.

**Responsibilities:**

-   Consume metadata from the `metadata_generated` queue
-   Download original images from MinIO
-   Write metadata into image EXIF/IPTC/XMP tags
-   Upload processed images to MinIO
-   Publish results to the `image_processed` queue

**Technologies:**

-   Go
-   ExifTool integration
-   MinIO SDK
-   RabbitMQ client

### 4. Filewatcher Service

The Filewatcher Service monitors directories for batch image processing without Telegram interface.

**Responsibilities:**

-   Monitor input directory for new images
-   Process images in bulk
-   Publish to the same `image_upload` queue
-   Provide REST API for statistics and manual triggers
-   Move processed images to output directory

**Technologies:**

-   Go
-   File system monitoring
-   RabbitMQ client
-   MinIO SDK

### 5. Dashboard Service

The Dashboard Service provides a web-based interface for monitoring and statistics.

**Responsibilities:**

-   Display real-time processing statistics
-   Show system health and status
-   Provide visual analytics
-   User-friendly interface for non-technical users

**Technologies:**

-   Go
-   HTTP server
-   Static file serving
-   PostgreSQL for data retrieval

### 6. RabbitMQ

Message broker that enables asynchronous communication between services.

**Queues:**

-   `image_upload`: Messages about new images to process
-   `metadata_generated`: Messages with generated metadata
-   `image_processed`: Messages about completed image processing
-   `dead_letter_queue`: Failed messages for manual inspection and retry

**Features:**

-   Dead Letter Exchange (DLX) for failed messages
-   Message persistence
-   Retry mechanisms
-   Manual retry capability via Gateway admin interface

### 7. MinIO

Object storage for all images.

**Buckets:**

-   `original`: Original user-uploaded images
-   `processed`: Images with embedded metadata

### 8. PostgreSQL

Relational database for statistics and history tracking.

**Tables:**

-   `images`: Image processing history
-   `processing_stats`: Daily aggregated statistics
-   `errors`: Detailed error tracking

**Features:**

-   JSONB support for flexible metadata storage
-   Indexes for query optimization
-   Statistics API backend

## Data Structures

### 1. RabbitMQ: ImageUpload Message

```json
{
    "trace_id": "uuid-string",
    "group_id": "uuid-string",
    "telegram_id": 123456789,
    "telegram_username": "username",
    "original_filename": "image.jpg",
    "original_path": "original/trace_id/image.jpg",
    "timestamp": "2023-01-01T00:00:00Z"
}
```

### 2. RabbitMQ: MetadataGenerated Message

```json
{
    "trace_id": "uuid-string",
    "group_id": "uuid-string",
    "telegram_id": 123456789,
    "original_filename": "image.jpg",
    "original_path": "original/trace_id/image.jpg",
    "metadata": {
        "title": "Sample Image Title",
        "description": "This is a sample description",
        "keywords": ["keyword1", "keyword2", "..."]
    },
    "timestamp": "2023-01-01T00:00:00Z"
}
```

### 3. RabbitMQ: ImageProcess Message

```json
{
    "trace_id": "uuid-string",
    "group_id": "uuid-string",
    "telegram_id": 123456789,
    "original_filename": "image.jpg",
    "original_path": "original/trace_id/image.jpg",
    "processed_path": "processed/trace_id/image.jpg",
    "metadata": {
        "title": "Sample Image Title",
        "description": "This is a sample description",
        "keywords": ["keyword1", "keyword2", "..."]
    },
    "timestamp": "2023-01-01T00:00:00Z"
}
```

### 4. RabbitMQ: ImageProcessed Message

```json
{
    "trace_id": "uuid-string",
    "group_id": "uuid-string",
    "telegram_id": 123456789,
    "telegram_username": "username",
    "original_filename": "image.jpg",
    "processed_path": "processed/trace_id/image.jpg",
    "status": "completed|failed",
    "error": "Error message if applicable",
    "timestamp": "2023-01-01T00:00:00Z"
}
```

## Process Flow

1. **Image Reception**:

    - User sends image(s) to the Telegram bot
    - Gateway Service validates image format
    - Gateway uploads image to MinIO 'original' bucket
    - Gateway publishes message to 'image_upload' queue

2. **Metadata Generation**:

    - Analyzer Service consumes message from 'image_upload' queue
    - Analyzer downloads image from MinIO
    - Analyzer uses automatically selected free vision model from OpenRouter for analysis
    - Analyzer handles rate limits with intelligent retry scheduling
    - Analyzer receives metadata (title, description, keywords)
    - Analyzer publishes metadata to 'metadata_generated' queue

3. **Metadata Application**:

    - Processor Service consumes message from 'metadata_generated' queue
    - Processor downloads original image from MinIO
    - Processor embeds metadata into image using ExifTool
    - Processor uploads processed image to MinIO 'processed' bucket
    - Processor publishes message to 'image_processed' queue

4. **Result Delivery**:
    - Gateway Service consumes message from 'image_processed' queue
    - Gateway downloads processed image from MinIO
    - Gateway sends image back to user through Telegram

## Error Handling

Each service implements comprehensive error handling:

-   Failed messages are automatically sent to Dead Letter Queue (DLQ)
-   Failed operations result in error messages in the 'image_processed' queue
-   Gateway Service informs users about processing failures
-   All errors are logged with the related trace_id for debugging
-   Errors are tracked in PostgreSQL for analysis
-   Services implement retry mechanisms with exponential backoff for transient failures
-   Manual retry capability via DLQ admin interface
-   Datadog integration for error monitoring and alerting

## Logging and Tracing

-   Each message includes a unique trace_id for end-to-end tracing
-   Group_id connects related images sent in a single batch
-   All services log operations with the trace_id and group_id
-   Logs are structured in JSON format for easier analysis
