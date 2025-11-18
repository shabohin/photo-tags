# Photo Tags Service

An automated image processing service using AI for metadata generation with dynamic model selection.

## Project Overview

This service automatically adds titles, descriptions, and keywords to image metadata using artificial intelligence. Users send images through a Telegram bot and receive processed versions with added metadata. The service dynamically selects the best available free vision models from OpenRouter to ensure optimal cost-efficiency.

## Architecture

The project is built using a microservice architecture and includes the following components:

-   **Gateway Service** - receives images and sends results via Telegram API
-   **Analyzer Service** - generates metadata using free vision models from OpenRouter with automatic model selection
-   **Processor Service** - writes metadata to images
-   **File Watcher Service** - monitors directories for batch image processing without Telegram
-   **RabbitMQ** - message exchange between services
-   **MinIO** - image storage

## Dynamic Model Selection

The Analyzer service automatically:
- Checks OpenRouter API daily and on startup for available free vision models
- Selects the best performing free model with vision capabilities
- Handles service gracefully during model unavailability
- Provides intelligent rate limit management with retry scheduling

## Installation and Launch

### Deployment Options

The service supports two deployment methods:

1. **Docker Deployment** (Recommended for most users)
   - Easier setup with consistent environment
   - Best for development teams and production servers
   - See [Docker Setup](#docker-deployment) below

2. **Native/Local Deployment** (Recommended for Raspberry Pi and resource-constrained devices)
   - Better performance on ARM devices
   - Lower memory footprint
   - Direct access to system resources
   - See [Local Deployment Guide](docs/LOCAL_DEPLOYMENT.md)

### Docker Deployment

#### Prerequisites

-   Docker and Docker Compose
-   Telegram bot token (get from [@BotFather](https://t.me/BotFather))
-   OpenRouter API key for accessing vision models (free tier available)

#### Starting the Project

1. Clone the repository:

    ```bash
    git clone https://github.com/shabohin/photo-tags.git
    cd photo-tags
    ```

2. Run the `start.sh` script, which will create an `.env` file from the template:

    ```bash
    chmod +x scripts/*.sh
    ./scripts/start.sh
    ```

3. On first run, you'll be prompted to edit the `.env` file and provide:

    - `TELEGRAM_TOKEN` - your Telegram bot token
    - `OPENROUTER_API_KEY` - your OpenRouter API key (free tier available)

4. The script launches services in Docker and checks all dependencies

5. Run `setup.sh` to complete the initial setup:
    ```bash
    ./scripts/setup.sh
    ```

#### Stopping Services

To stop all services, use:

```bash
./scripts/stop.sh
```

### Local/Native Deployment

For running on bare metal without Docker (recommended for Raspberry Pi, macOS development, or resource-constrained environments):

#### Quick Start

```bash
# 1. Install all dependencies
./scripts/install-local.sh

# 2. Configure environment
cp config/.env.local.example config/.env.local
# Edit config/.env.local and set TELEGRAM_TOKEN and OPENROUTER_API_KEY

# 3. Start all services
./scripts/run-local.sh start

# 4. Check status
./scripts/run-local.sh status
```

#### Platform Support

- ✅ **macOS** (Intel and Apple Silicon M1/M2/M3)
- ✅ **Linux** (Ubuntu, Debian, Raspberry Pi OS)
- ✅ **ARM64** (Raspberry Pi 3/4, other ARM devices)
- ✅ **x86_64** (Standard Linux servers)

#### Managing Services

```bash
./scripts/run-local.sh start    # Start all services
./scripts/run-local.sh stop     # Stop all services
./scripts/run-local.sh restart  # Restart all services
./scripts/run-local.sh status   # Show service status
./scripts/run-local.sh logs     # View all logs
./scripts/run-local.sh build    # Rebuild services
```

For detailed platform-specific instructions, troubleshooting, and configuration options, see the [Local Deployment Guide](docs/LOCAL_DEPLOYMENT.md).

## Usage

After launch, the bot will be accessible through Telegram. Send an image to the bot, and it will automatically process it and return it with added metadata.

### Service Behavior

- **Normal Operation**: Images are processed within 10-30 seconds
- **High Load/Rate Limits**: Users receive confirmation that image is accepted and will be processed (may take longer)
- **Model Unavailability**: Images are queued and processed when models become available
- **Automatic Recovery**: Service automatically detects when models are available again

### User Experience

Users always receive:
1. **Immediate confirmation** when image is uploaded
2. **Processing status** with realistic time estimates
3. **Final result** with AI-generated metadata

The service handles all technical complexities (model selection, rate limits, retries) transparently.

### Supported Formats

-   JPG/JPEG
-   PNG

### Generated Metadata

-   **Title** - brief description of the image
-   **Description** - more detailed description up to 200 characters
-   **Keywords** - 49 keywords describing the image

### File Watcher Service - Batch Processing

In addition to the Telegram bot interface, you can process images in batch mode using the File Watcher Service:

1. **Copy images to input directory**:
```bash
docker cp /path/to/images/* filewatcher:/app/input/
```

2. **Monitor processing**:
```bash
curl http://localhost:8081/stats
```

3. **Retrieve processed images**:
```bash
docker cp filewatcher:/app/output/. /path/to/output/
```

4. **Trigger manual scan**:
```bash
curl -X POST http://localhost:8081/scan
```

For more details, see [File Watcher Service README](services/filewatcher/README.md).

## Monitoring

The service includes comprehensive monitoring with Datadog for APM, metrics, and logs. See the [Monitoring Guide](docs/monitoring.md) for detailed setup instructions.

### Quick Start with Datadog

1. Get a free API key from [datadoghq.com](https://www.datadoghq.com/)
2. Add to `docker/.env`:
   ```bash
   DD_API_KEY=your_api_key_here
   DD_ENV=development
   ```
3. Restart services: `./scripts/start.sh`

### Available Interfaces

After startup, you can access the following interfaces:

-   **RabbitMQ Management**: [http://localhost:15672](http://localhost:15672) (login: user, password: password)
-   **MinIO Console**: [http://localhost:9001](http://localhost:9001) (login: minioadmin, password: minioadmin)
-   **Gateway API**: [http://localhost:8080](http://localhost:8080) (health check available at `/health`)
-   **Dead Letter Queue Admin**: [http://localhost:8080/admin/failed-jobs](http://localhost:8080/admin/failed-jobs) (monitor and retry failed jobs)
-   **Datadog Dashboard**: [app.datadoghq.com](https://app.datadoghq.com/) (if configured)

## Service Logs

The service provides detailed logging for monitoring and debugging:

### Log Categories

- **Model Selection**: Daily checks for available free vision models
- **Rate Limiting**: Automatic handling of API rate limits with retry scheduling
- **Error Handling**: Detailed error tracking for troubleshooting
- **Performance**: Processing time metrics and queue status
- **User Interactions**: Request tracking and processing status

### Viewing Logs

To view service logs, use:

```bash
# Gateway service logs (includes Telegram bot activity)
docker logs gateway -f

# Analyzer service logs (includes model selection and processing)
docker logs analyzer -f

# Processor service logs (includes metadata writing)
docker logs processor -f

# Infrastructure logs
docker logs rabbitmq -f
docker logs minio -f
```

### Log Analysis

Key log patterns to monitor:
- `"Model selection completed"` - Daily model availability checks
- `"Service restored"` - Recovery from model unavailability
- `"Rate limit exceeded"` - API throttling with retry times
- `"Processing queue"` - Backlog processing status

## Architecture Details

### Smart Queue Management

The service implements intelligent queue management:
- **Immediate Acceptance**: All user requests are immediately acknowledged
- **Asynchronous Processing**: Images processed in background when resources available
- **Priority Handling**: Rate-limited requests automatically retried with optimal timing
- **Status Updates**: Users informed of processing progress

### Model Selection Strategy

The Analyzer service:
1. **Daily Checks**: Queries OpenRouter API for available free vision models
2. **Capability Filtering**: Only selects models with image analysis capabilities
3. **Cost Optimization**: Prioritizes free models to minimize operational costs
4. **Performance Ranking**: Uses OpenRouter's model rankings for selection
5. **Automatic Fallback**: Gracefully handles model unavailability

### Error Recovery

Robust error handling includes:
- **Rate Limit Management**: Automatic retry with `X-RateLimit-Reset` timing
- **Model Failover**: Attempts multiple models when primary is unavailable
- **Queue Persistence**: No requests lost during service interruptions
- **User Communication**: Clear status updates without technical jargon

### Dead Letter Queue

The service implements a Dead Letter Queue (DLQ) for managing failed messages:
- **Automatic Failure Tracking**: Failed jobs are automatically sent to DLQ instead of being lost
- **Error Visibility**: View all failed jobs with error reasons and retry counts
- **Manual Retry**: Web UI for reviewing and requeuing failed jobs
- **Full Metadata**: Each failed job includes original queue, message body, and failure details

Access the DLQ admin interface at [http://localhost:8080/admin/failed-jobs](http://localhost:8080/admin/failed-jobs)

For detailed documentation, see [Dead Letter Queue Guide](docs/dead-letter-queue.md).


## Development

### Prerequisites for Development

- Go 1.24+
- Docker and Docker Compose
- golangci-lint (for code quality)

### Setting Up Development Environment

1. **Install golangci-lint**:
   ```bash
   # Using the provided script
   ./scripts/install-golangci-lint.sh

   # Or manually via Go
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.1.6

   # Or via Homebrew (macOS)
   brew install golangci-lint
   ```

2. **Install development dependencies**:
   ```bash
   make install-tools
   make deps
   ```

3. **Setup Git hooks** (optional):
   ```bash
   make install-hooks
   ```

### Code Quality

This project uses golangci-lint for maintaining code quality. The configuration includes:

- **Basic linters**: errcheck, gosimple, govet, staticcheck, unused
- **Formatting**: gofmt, goimports with local prefix support
- **Code quality**: revive, gocritic, cyclop for complexity analysis
- **Security**: gosec for security issues
- **Performance**: prealloc for slice optimization
- **Style**: misspell, whitespace, unconvert

### Available Make Commands

```bash
# Code quality
make lint          # Run linter on all modules
make lint-fix      # Run linter with auto-fix
make fmt           # Format all Go files
make pre-commit    # Run format, lint, and test

# Testing
make test          # Run all tests
make test-race     # Run tests with race detector
make test-coverage # Run tests with coverage reports

# Development
make deps          # Download and tidy dependencies
make build         # Build all services
make check         # Run all quality checks (tests + linting)

# Docker operations
make docker-build  # Build Docker images
make docker-up     # Start services with Docker Compose
make docker-down   # Stop services
make docker-logs   # Show Docker logs

# Environment
make start         # Start all services
make stop          # Stop all services
make setup         # Setup the environment
make clean         # Clean build artifacts and stop services

# Information
make help          # Show all available commands
make version       # Show Go and tool versions
```

### Running Quality Checks

Before committing code, run:

```bash
make pre-commit
```

This will:
1. Format all Go files
2. Run golangci-lint on all modules
3. Run all tests

### Linting Configuration

The project uses a comprehensive golangci-lint configuration (`.golangci.yml`) with:

- Timeout: 5 minutes
- Enabled linters: 25+ linters covering security, performance, style, and bugs
- Custom rules for test files
- Local import prefix: `github.com/shabohin/photo-tags`
- Line length limit: 120 characters

### VS Code Integration

If you use VS Code, the project includes settings for:
- Automatic formatting on save
- golangci-lint integration
- Import organization
- Go-specific editor settings

The configuration is in `.vscode/settings.json`.

### Testing

Run tests with various options:

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Generate coverage reports
make test-coverage
```

Coverage reports are generated in the `coverage/` directory.

### Common Development Tasks

1. **Adding new linter rules**:
   Edit `.golangci.yml` and add new linters to the `enable` section.

2. **Fixing linting issues**:
   ```bash
   make lint-fix  # Auto-fix what can be fixed
   make lint      # Check remaining issues
   ```

3. **Updating dependencies**:
   ```bash
   make deps
   ```

4. **Before creating a PR**:
   ```bash
   make check  # Runs both tests and linting
   ```
