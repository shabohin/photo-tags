# Analyzer Service Documentation

## Table of Contents

-   [1. Service Overview](#1-service-overview)
-   [2. Architecture](#2-architecture)
-   [3. Service API](#3-service-api)
-   [4. Configuration](#4-configuration)
-   [5. Deployment](#5-deployment)
-   [6. Monitoring and Logging](#6-monitoring-and-logging)
-   [7. Error Handling](#7-error-handling)
-   [8. Testing](#8-testing)
-   [9. Limitations and Future Plans](#9-limitations-and-future-plans)

---

## 1. Service Overview

Analyzer Service is a core component of the Photo Tags Service platform. It performs automated image analysis using GPT-4o via OpenRouter API to generate metadata (title, description, keywords) for uploaded images.

**Main responsibilities:**

-   Consumes `image_upload` messages from RabbitMQ
-   Downloads images from MinIO storage
-   Sends images to OpenRouter API for analysis
-   Publishes generated metadata as `metadata_generated` messages to RabbitMQ

Analyzer Service enables automated, scalable enrichment of image data, facilitating downstream processing and search.

---

## 2. Architecture

### High-Level Design

The service is built with modular, layered architecture:

-   **Transport Layer:** RabbitMQ consumer and publisher
-   **Domain Layer:** Business logic, image analysis, message processing
-   **Infrastructure Layer:** MinIO client, OpenRouter API client
-   **Configuration Layer:** Environment-based configuration

### Component Diagram

```mermaid
graph TD
    A[main.go] --> B[app.App]
    B --> C[config.Config]
    B --> D[transport.RabbitMQConsumer]
    B --> E[transport.RabbitMQPublisher]
    B --> F[storage.MinIOClient]
    B --> G[api.OpenRouterClient]
    B --> H[service.AnalyzerService]
    B --> I[service.MessageProcessor]

    D --> I
    I --> H
    H --> F
    H --> G
    I --> E

    subgraph Domain
        H
        I
    end

    subgraph Infrastructure
        D
        E
        F
        G
    end

    subgraph Configuration
        C
    end
```

### Processing Flow

```mermaid
sequenceDiagram
    participant RabbitMQ
    participant Consumer
    participant Processor
    participant Analyzer
    participant MinIO
    participant OpenRouterAPI
    participant Publisher

    RabbitMQ->>Consumer: Receive message (image_upload)
    Consumer->>Processor: Pass message

    Note over Processor: Deserialize message

    Processor->>Analyzer: Request image analysis
    Analyzer->>MinIO: Download image
    MinIO-->>Analyzer: Image bytes

    Analyzer->>OpenRouterAPI: Send image for analysis
    OpenRouterAPI-->>Analyzer: Analysis result (metadata)

    Analyzer-->>Processor: Image metadata

    Note over Processor: Form metadata message

    Processor->>Publisher: Publish result
    Publisher->>RabbitMQ: Send message (metadata_generated)
```

### Package Structure

```
services/analyzer/
├── cmd/                    # Entry point
├── internal/
│   ├── api/openrouter/     # OpenRouter API client
│   ├── app/                # Application setup
│   ├── config/             # Configuration
│   ├── domain/
│   │   ├── model/          # Data models
│   │   └── service/        # Business logic
│   ├── storage/minio/      # MinIO client
│   └── transport/rabbitmq/ # RabbitMQ consumer/publisher
```

---

## 3. Service API

### RabbitMQ Message Formats

#### `image_upload` (Consumer)

```json
{
    "trace_id": "string",
    "group_id": "string",
    "telegram_id": 123456789,
    "telegram_username": "username",
    "original_filename": "photo.jpg",
    "original_path": "uploads/2024/04/photo.jpg",
    "timestamp": "2024-04-07T12:34:56Z"
}
```

#### `metadata_generated` (Publisher)

```json
{
    "trace_id": "string",
    "group_id": "string",
    "telegram_id": 123456789,
    "original_filename": "photo.jpg",
    "original_path": "uploads/2024/04/photo.jpg",
    "metadata": {
        "title": "string",
        "description": "string",
        "keywords": ["keyword1", "keyword2"]
    },
    "timestamp": "2024-04-07T12:35:56Z"
}
```

### OpenRouter API Integration

-   **Endpoint:** `https://openrouter.ai/api/v1/chat/completions`
-   **Authorization:** Bearer token (`OPENROUTER_API_KEY`)
-   **Request:**

```json
{
    "model": "openai/gpt-4o",
    "messages": [
        {
            "role": "user",
            "content": [
                {
                    "type": "text",
                    "text": "Generate title, description and keywords for this image. Return strictly in JSON format with fields 'title', 'description' and 'keywords'."
                },
                {
                    "type": "image_url",
                    "image_url": {
                        "url": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD..."
                    }
                }
            ]
        }
    ],
    "max_tokens": 500,
    "temperature": 0.7
}
```

-   **Response:**

```json
{
    "id": "...",
    "choices": [
        {
            "message": {
                "role": "assistant",
                "content": "{\"title\": \"...\", \"description\": \"...\", \"keywords\": [\"...\"]}"
            }
        }
    ]
}
```

The service parses the JSON string inside `content` to extract metadata.

---

## 4. Configuration

All configuration is environment-variable driven. Defaults are provided in code and `.env.example`.

| Variable                      | Description                   | Default                             |
| ----------------------------- | ----------------------------- | ----------------------------------- |
| `RABBITMQ_URL`                | RabbitMQ connection URL       | `amqp://guest:guest@rabbitmq:5672/` |
| `RABBITMQ_CONSUMER_QUEUE`     | Queue to consume from         | `image_upload`                      |
| `RABBITMQ_PUBLISHER_QUEUE`    | Queue to publish to           | `metadata_generated`                |
| `RABBITMQ_PREFETCH_COUNT`     | Prefetch count                | `1`                                 |
| `RABBITMQ_RECONNECT_ATTEMPTS` | Retry attempts                | `5`                                 |
| `RABBITMQ_RECONNECT_DELAY`    | Retry delay                   | `5s`                                |
| `MINIO_ENDPOINT`              | MinIO endpoint                | `minio:9000`                        |
| `MINIO_ACCESS_KEY`            | MinIO access key              | `minioadmin`                        |
| `MINIO_SECRET_KEY`            | MinIO secret key              | `minioadmin`                        |
| `MINIO_USE_SSL`               | Use SSL for MinIO             | `false`                             |
| `MINIO_ORIGINAL_BUCKET`       | Bucket name                   | `original`                          |
| `MINIO_DOWNLOAD_TIMEOUT`      | Download timeout              | `30s`                               |
| `OPENROUTER_API_KEY`          | OpenRouter API key            | (none)                              |
| `OPENROUTER_MODEL`            | Model name                    | `openai/gpt-4o`                     |
| `OPENROUTER_MAX_TOKENS`       | Max tokens                    | `500`                               |
| `OPENROUTER_TEMPERATURE`      | Temperature                   | `0.7`                               |
| `OPENROUTER_PROMPT`           | Prompt text                   | See `.env.example`                  |
| `LOG_LEVEL`                   | Log level                     | `info`                              |
| `LOG_FORMAT`                  | Log format (`json` or `text`) | `json`                              |
| `WORKER_CONCURRENCY`          | Number of workers             | `3`                                 |
| `WORKER_MAX_RETRIES`          | Max retries for analysis      | `3`                                 |
| `WORKER_RETRY_DELAY`          | Delay between retries         | `5s`                                |

---

## 5. Deployment

### Building

Build the service binary:

```bash
go build -o analyzer ./cmd/
```

### Docker

Multi-stage Dockerfile is used (`docker/Dockerfile.service`):

```dockerfile
FROM golang:1.24-alpine AS builder
ARG SERVICE
WORKDIR /app
COPY ./services/$SERVICE/go.mod .
COPY ./pkg/go.mod /pkg/go.mod
RUN go mod download
COPY ./services/$SERVICE/ .
COPY ./pkg/ /pkg/
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /bin/service /app/service
CMD ["/app/service"]
```

### Docker Compose

`docker-compose.yml` defines RabbitMQ, MinIO, Analyzer, Gateway, Processor services.

Start all services:

```bash
docker compose up --build
```

Analyzer depends on RabbitMQ and MinIO, and requires `OPENROUTER_API_KEY` to be set.

### Dependencies

-   **RabbitMQ:** message broker
-   **MinIO:** object storage
-   **OpenRouter API:** GPT-4o model

---

## 6. Monitoring and Logging

-   Uses `logrus` for structured logging
-   Log level and format configurable (`LOG_LEVEL`, `LOG_FORMAT`)
-   Logs include `trace_id`, filenames, error details
-   Logs are output to stdout (container logs)
-   Recommended to aggregate logs via ELK, Loki, or similar

**Metrics (suggested to add):**

| Metric                   | Description                 |
| ------------------------ | --------------------------- |
| Processed messages count | Total images processed      |
| Processing duration      | Time per image              |
| Error count              | Number of failed analyses   |
| Retry count              | Number of retries performed |

---

## 7. Error Handling

-   Errors classified as transient or permanent
-   Retries with exponential backoff for transient errors (configurable)
-   Max retries per message (`WORKER_MAX_RETRIES`)
-   Errors logged with context (`trace_id`, attempt)
-   After max retries, message is dropped and error logged
-   Context cancellation used for graceful shutdown
-   Trace IDs propagate through logs for debugging

---

## 8. Testing

-   Unit tests for core components (`*_test.go`)
-   Integration tests with RabbitMQ and MinIO (via docker-compose)
-   Run tests:

```bash
go test ./...
```

-   Test coverage includes:
    -   Message parsing
    -   MinIO interactions
    -   OpenRouter API client
    -   Retry logic
    -   Error handling

---

## 9. Limitations and Future Plans

### Current Limitations

-   No built-in metrics export (Prometheus, etc.)
-   No dead-letter queue for failed messages
-   No authentication/authorization on API level
-   Limited validation of OpenRouter API responses

### Future Enhancements

-   Add Prometheus metrics
-   Implement dead-letter queue support
-   Improve error classification
-   Support additional AI models/providers
-   Enhance metadata schema (e.g., categories, tags)
-   Add caching for repeated images

---

# End of Document
