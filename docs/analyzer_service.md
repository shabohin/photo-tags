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

Analyzer Service is a core component of the Photo Tags Service platform. It performs automated image analysis using vision models via OpenRouter API to generate metadata (title, description, keywords) for uploaded images.

**Main responsibilities:**

-   Consumes `image_upload` messages from RabbitMQ
-   Downloads images from MinIO storage
-   **Automatically selects best free vision models** from OpenRouter
-   Sends images to OpenRouter API for analysis
-   Publishes generated metadata as `metadata_generated` messages to RabbitMQ
-   **Handles rate limits** with automatic retry and backoff
-   **Periodically updates** available models (configurable interval)

**Advanced Features:**

-   **Model Selector Service**: Automatically discovers and selects the best free vision models every 24 hours
-   **Rate Limit Handling**: Intelligent retry with exponential backoff and reset time parsing
-   **Thread-Safe Caching**: Concurrent-safe access to selected models
-   **Graceful Degradation**: Fallback to configured model if selection fails
-   **High Availability**: Thread-safe operations with context-based cancellation

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
    B --> J[selector.ModelSelector]

    J --> G
    D --> I
    I --> H
    H --> F
    H --> G
    I --> E

    subgraph Domain
        H
        I
        J
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
│   ├── selector/           # Model Selector service (NEW)
│   ├── storage/minio/      # MinIO client
│   └── transport/rabbitmq/ # RabbitMQ consumer/publisher
```

**New Components:**

-   **selector/**: Automatic model selection and periodic updates
    -   `selector.go`: Model selection logic with caching
    -   `selector_test.go`: Comprehensive tests

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

| Variable                               | Description                        | Default                             |
| -------------------------------------- | ---------------------------------- | ----------------------------------- |
| `RABBITMQ_URL`                         | RabbitMQ connection URL            | `amqp://guest:guest@rabbitmq:5672/` |
| `RABBITMQ_CONSUMER_QUEUE`              | Queue to consume from              | `image_upload`                      |
| `RABBITMQ_PUBLISHER_QUEUE`             | Queue to publish to                | `metadata_generated`                |
| `RABBITMQ_PREFETCH_COUNT`              | Prefetch count                     | `1`                                 |
| `RABBITMQ_RECONNECT_ATTEMPTS`          | Retry attempts                     | `5`                                 |
| `RABBITMQ_RECONNECT_DELAY`             | Retry delay                        | `5s`                                |
| `MINIO_ENDPOINT`                       | MinIO endpoint                     | `minio:9000`                        |
| `MINIO_ACCESS_KEY`                     | MinIO access key                   | `minioadmin`                        |
| `MINIO_SECRET_KEY`                     | MinIO secret key                   | `minioadmin`                        |
| `MINIO_USE_SSL`                        | Use SSL for MinIO                  | `false`                             |
| `MINIO_ORIGINAL_BUCKET`                | Bucket name                        | `original`                          |
| `MINIO_DOWNLOAD_TIMEOUT`               | Download timeout                   | `30s`                               |
| `OPENROUTER_API_KEY`                   | OpenRouter API key                 | (none)                              |
| `OPENROUTER_MODEL`                     | Fallback model name                | `openai/gpt-4o`                     |
| `OPENROUTER_MAX_TOKENS`                | Max tokens                         | `500`                               |
| `OPENROUTER_TEMPERATURE`               | Temperature                        | `0.7`                               |
| `OPENROUTER_PROMPT`                    | Prompt text                        | See `.env.example`                  |
| `OPENROUTER_MODEL_CHECK_INTERVAL` 🆕  | Model update interval              | `24h`                               |
| `USE_OPENROUTERGO_ADAPTER`             | Use OpenRouterGo library           | `false`                             |
| `LOG_LEVEL`                            | Log level                          | `info`                              |
| `LOG_FORMAT`                           | Log format (`json` or `text`)      | `json`                              |
| `WORKER_CONCURRENCY`                   | Number of workers                  | `3`                                 |
| `WORKER_MAX_RETRIES`                   | Max retries for analysis           | `3`                                 |
| `WORKER_RETRY_DELAY`                   | Delay between retries              | `5s`                                |

**🆕 New Configuration:**

-   `OPENROUTER_MODEL_CHECK_INTERVAL`: Controls how often the Model Selector checks for available free models (default: 24h)
-   Model selection happens automatically on startup and periodically thereafter

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

### ✅ Recently Implemented

-   ✅ **Automatic Model Selection**: Best free vision models auto-selected
-   ✅ **Rate Limit Handling**: Smart retry with exponential backoff
-   ✅ **Model Selector Service**: Periodic updates every 24h (configurable)
-   ✅ **Thread-Safe Operations**: Concurrent-safe model caching
-   ✅ **Comprehensive Error Handling**: Retry logic for 5xx and network errors
-   ✅ **High Test Coverage**: >70% coverage with unit and integration tests

### Future Enhancements

-   Add Prometheus metrics
-   Implement dead-letter queue support
-   Support additional AI models/providers (beyond OpenRouter)
-   Enhance metadata schema (e.g., categories, tags, EXIF data)
-   Add caching for repeated images (hash-based)
-   Implement model performance tracking
-   Add support for batch processing

---

## 10. OpenRouter Model Selection

### How Model Selection Works

1. **On Startup**: Model Selector fetches all available models from OpenRouter
2. **Filtering**: Only free models (`pricing.prompt == "0"`) with vision/multimodal capabilities
3. **Sorting**: Models sorted by context length (higher is better)
4. **Selection**: Best model cached in memory
5. **Periodic Updates**: Every 24h (configurable), process repeats
6. **Fallback**: If no free models found, uses configured `OPENROUTER_MODEL`

### Supported Model Types

-   Models with `modality: multimodal`
-   Models with `modality: image` or containing "vision" in name/ID
-   Free pricing (`pricing.prompt == "0"`)

### Rate Limit Handling

-   Parses `X-RateLimit-Remaining` and `X-RateLimit-Reset` headers
-   Automatic retry after reset time
-   Exponential backoff for transient errors (2s, 4s, 8s)
-   Up to 3 retries for 5xx server errors

---

# End of Document
