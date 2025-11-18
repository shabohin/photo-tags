# Analyzer Service Architecture

This document describes the architecture for the Analyzer Service in the Photo Tags Service project.

## Service Overview

The Analyzer Service is responsible for:

-   Consuming tasks from RabbitMQ queue (`image_upload`)
-   Downloading images from MinIO storage
-   Interacting with OpenRouter API (GPT-4o) for image analysis
-   Generating metadata (title, description, keywords)
-   Publishing results to RabbitMQ queue (`metadata_generated`)

## Package Structure

```
services/analyzer/
├── cmd/
│   └── main.go                    # Application entry point
├── internal/
│   ├── config/                    # Service configuration
│   │   ├── config.go              # Configuration structures and loading
│   │   └── config_test.go         # Configuration tests
│   ├── domain/                    # Domain models and business logic
│   │   ├── model/                 # Data models
│   │   │   ├── message.go         # Message structures
│   │   │   └── metadata.go        # Metadata model
│   │   └── service/               # Service layers
│   │       ├── analyzer.go        # Image analysis service
│   │       ├── analyzer_test.go   # Analyzer tests
│   │       ├── processor.go       # Message processor
│   │       └── interfaces.go      # Service interfaces
│   ├── selector/                  # Model selection service (NEW) 🆕
│   │   ├── selector.go            # Automatic model selection
│   │   └── selector_test.go       # Selector tests
│   ├── transport/                 # Transport layer
│   │   └── rabbitmq/              # RabbitMQ client
│   │       ├── consumer.go        # Message consumer
│   │       └── publisher.go       # Message publisher
│   ├── storage/                   # Storage layer
│   │   └── minio/                 # MinIO client
│   │       └── client.go          # Storage operations
│   ├── api/                       # External API interactions
│   │   └── openrouter/            # OpenRouter API client
│   │       ├── client.go          # Vision models interface
│   │       ├── client_test.go     # Client tests
│   │       └── openroutergo_adapter.go  # Adapter for library
│   └── app/                       # Application initialization
│       └── app.go                 # Application assembly and startup
└── go.mod                         # Application module
```

**🆕 New Package: `selector/`**

The Model Selector service automatically discovers and selects the best free vision models from OpenRouter:

-   Fetches available models from OpenRouter API
-   Filters for free models with vision/multimodal capabilities
-   Selects model with highest context length
-   Caches selected model in memory (thread-safe)
-   Periodically updates every 24h (configurable)
-   Provides fallback to configured model

## Component Diagram

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

    style J fill:#90EE90
```

## Processing Sequence Diagram

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

## Key Interfaces and Structures

### Data Models

```go
// domain/model/message.go
type ImageUploadMessage struct {
    TraceID          string    `json:"trace_id"`
    GroupID          string    `json:"group_id"`
    TelegramID       int64     `json:"telegram_id"`
    TelegramUsername string    `json:"telegram_username"`
    OriginalFilename string    `json:"original_filename"`
    OriginalPath     string    `json:"original_path"`
    Timestamp        time.Time `json:"timestamp"`
}

type MetadataGeneratedMessage struct {
    TraceID          string    `json:"trace_id"`
    GroupID          string    `json:"group_id"`
    TelegramID       int64     `json:"telegram_id"`
    OriginalFilename string    `json:"original_filename"`
    OriginalPath     string    `json:"original_path"`
    Metadata         Metadata  `json:"metadata"`
    Timestamp        time.Time `json:"timestamp"`
}

// domain/model/metadata.go
type Metadata struct {
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Keywords    []string `json:"keywords"`
}
```

### Configuration

```go
// config/config.go
type Config struct {
    RabbitMQ struct {
        URL               string
        ConsumerQueue     string
        PublisherQueue    string
        PrefetchCount     int
        ReconnectAttempts int
        ReconnectDelay    time.Duration
    }

    MinIO struct {
        Endpoint        string
        AccessKey       string
        SecretKey       string
        UseSSL          bool
        OriginalBucket  string
        DownloadTimeout time.Duration
    }

    OpenRouter struct {
        APIKey      string
        Model       string
        MaxTokens   int
        Temperature float64
        Prompt      string
    }

    Log struct {
        Level  string
        Format string
    }

    Worker struct {
        Concurrency int
        MaxRetries  int
        RetryDelay  time.Duration
    }
}
```

### Service Interfaces

```go
// api/openrouter/client.go
type OpenRouterClient interface {
    AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (Metadata, error)
}

// storage/minio/client.go
type MinIOClient interface {
    DownloadImage(ctx context.Context, path string) ([]byte, error)
}

// transport/rabbitmq/consumer.go
type MessageConsumer interface {
    Consume(ctx context.Context, handler func(message []byte) error) error
    Close() error
}

// transport/rabbitmq/publisher.go
type MessagePublisher interface {
    Publish(ctx context.Context, message []byte) error
    Close() error
}

// domain/service/analyzer.go
type ImageAnalyzer interface {
    AnalyzeImage(ctx context.Context, msg ImageUploadMessage) (Metadata, error)
}

// domain/service/processor.go
type MessageProcessor interface {
    Process(ctx context.Context, message []byte) error
}
```

## Key Architecture Aspects

### 1. Modularity and Separation of Concerns

-   Clear separation into layers: transport, domain, infrastructure
-   Isolated business logic in domain/service packages
-   Interface-based design for testability and flexibility

### 2. Horizontal Scaling

-   Stateless approach
-   Configurable number of workers
-   Support for multiple service instances
-   Workers can process messages concurrently

### 3. Reliability and Error Handling

-   Retry mechanism with exponential backoff
-   Structured logging with trace information
-   Context-based lifecycle management for operations
-   Error classification (transient vs. permanent failures)

### 4. OpenRouter API Integration

-   Adapter for GPT-4o
-   Processing API responses and metadata parsing
-   Flexible prompt configuration via environment variables

### 5. Environment-based Configuration

-   Parameterization of all components
-   Easy configuration in different environments
-   Use of environment variables for secrets and settings

## Application Startup Flow

1. Load configuration from environment variables
2. Initialize infrastructure components (RabbitMQ, MinIO, OpenRouter clients)
3. Set up service components (analyzer, processor)
4. Start configurable number of worker goroutines for message consumption
5. Handle system signals for graceful shutdown
6. Release resources on shutdown

## Error Handling and Retries

The service implements a robust error handling strategy:

-   Classification of errors as transient or permanent
-   Exponential backoff for retrying transient failures
-   Structured logging of errors with trace IDs
-   Context-based cancellation for long-running operations

## OpenRouter API Integration

The service communicates with the OpenRouter API to analyze images using vision models:

-   **Automatic Model Selection**: Best free vision models discovered automatically
-   Base64 encoding of image data
-   Construction of appropriate prompts
-   Processing of API responses to extract structured metadata
-   Fallback strategies for format variations in responses
-   **Rate Limit Handling**: Automatic retry with exponential backoff
-   **Error Recovery**: Retry logic for 5xx and network errors

### New: Model Selector Service

The Model Selector service (`selector/`) provides intelligent, automatic model selection:

#### Responsibilities

1. **Model Discovery**: Fetches all available models from OpenRouter `/api/v1/models` endpoint
2. **Filtering**: Selects only free models with vision/multimodal capabilities
3. **Ranking**: Sorts models by context length (higher is better)
4. **Caching**: Thread-safe in-memory cache of selected model
5. **Periodic Updates**: Automatic updates every 24h (configurable via `OPENROUTER_MODEL_CHECK_INTERVAL`)
6. **Graceful Degradation**: Falls back to `OPENROUTER_MODEL` if no free models available

#### Selection Algorithm

```go
// Pseudo-code for model selection
func SelectBestFreeVisionModel(models []Model) (*Model, error) {
    freeVisionModels := []Model{}

    // Filter for free vision models
    for model in models {
        if model.Pricing.Prompt == "0" {
            if model.Architecture.Modality contains "multimodal" or "image" {
                freeVisionModels.append(model)
            }
        }
    }

    // Sort by context length (descending)
    sort(freeVisionModels, by: ContextLength, descending)

    // Return best model
    return freeVisionModels[0]
}
```

#### Thread Safety

-   Uses `sync.RWMutex` for concurrent-safe access
-   Multiple goroutines can read selected model simultaneously
-   Periodic updates safely modify cached value

#### Lifecycle

1. **Startup**: Initial model selection on service start
2. **Runtime**: Periodic updates every 24h (default)
3. **Shutdown**: Graceful stop via context cancellation

#### Example Log Output

```
INFO  Starting Model Selector              check_interval=24h0m0s
INFO  Updating available models
INFO  Successfully fetched models          models_count=127
INFO  Selected best free vision model
      model_id="google/gemini-2.0-flash-exp:free"
      model_name="Gemini 2.0 Flash (free)"
      context_len=32768
      modality="multimodal"
```
