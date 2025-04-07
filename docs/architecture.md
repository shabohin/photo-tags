# Architecture Documentation

This document provides a detailed overview of the Photo Tags Service architecture, describing its components, data structures, and communication flow.

## Documentation Links

-   [Main README](../README.md)
-   [Development Guide](development.md)
-   [Testing Strategy](testing.md)
-   [Deployment Guide](deployment.md)

## System Overview

Photo Tags Service uses a microservices architecture to process images and add metadata. The services communicate asynchronously through RabbitMQ message queues, and images are stored in MinIO object storage.

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

The Gateway Service acts as the entry point for user interactions through a Telegram bot interface.

**Responsibilities:**

-   Receive images from users via Telegram API
-   Validate image formats (JPG/PNG)
-   Upload original images to MinIO
-   Publish image processing tasks to the `image_upload` queue
-   Receive processed images from the `image_processed` queue
-   Send processed images back to users

**Technologies:**

-   Go
-   Telegram Bot API
-   RabbitMQ client

### 2. Analyzer Service

The Analyzer Service processes images with AI to generate appropriate metadata.

**Responsibilities:**

-   Consume tasks from the `image_upload` queue
-   Download images from MinIO
-   Interact with OpenRouter's GPT-4o to analyze images
-   Generate metadata (title, description, keywords)
-   Publish results to the `metadata_generated` queue

**Technologies:**

-   Go
-   OpenRouter API
-   MinIO SDK
-   RabbitMQ client

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

### 4. RabbitMQ

Message broker that enables asynchronous communication between services.

**Queues:**

-   `image_upload`: Messages about new images to process
-   `metadata_generated`: Messages with generated metadata
-   `image_processed`: Messages about completed image processing

### 5. MinIO

Object storage for all images.

**Buckets:**

-   `original`: Original user-uploaded images
-   `processed`: Images with embedded metadata

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
      | - Analyzer sends image to GPT-4o via OpenRouter for analysis
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

Each service implements basic error handling:

-   Failed operations result in error messages in the 'image_processed' queue
-   Gateway Service informs users about processing failures
-   All errors are logged with the related trace_id for debugging
-   Services implement retry mechanisms for transient failures

## Logging and Tracing

-   Each message includes a unique trace_id for end-to-end tracing
-   Group_id connects related images sent in a single batch
-   All services log operations with the trace_id and group_id
-   Logs are structured in JSON format for easier analysis
