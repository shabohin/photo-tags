# Photo Tags Service

An automated image processing service using AI for metadata generation with dynamic model selection.

## Project Overview

This service automatically adds titles, descriptions, and keywords to image metadata using artificial intelligence. Users send images through a Telegram bot and receive processed versions with added metadata. The service dynamically selects the best available free vision models from OpenRouter to ensure optimal cost-efficiency.

## Architecture

The project is built using a microservice architecture and includes the following components:

-   **Gateway Service** - receives images and sends results via Telegram API
-   **Analyzer Service** - generates metadata using free vision models from OpenRouter with automatic model selection
-   **Processor Service** - writes metadata to images
-   **RabbitMQ** - message exchange between services
-   **MinIO** - image storage

## Dynamic Model Selection

The Analyzer service automatically:
- Checks OpenRouter API daily and on startup for available free vision models
- Selects the best performing free model with vision capabilities
- Handles service gracefully during model unavailability
- Provides intelligent rate limit management with retry scheduling

## Installation and Launch

### Prerequisites

-   Docker and Docker Compose
-   Telegram bot token (get from [@BotFather](https://t.me/BotFather))
-   OpenRouter API key for accessing vision models (free tier available)

### Starting the Project

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

### Stopping Services

To stop all services, use:

```bash
./scripts/stop.sh
```

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

## Monitoring

After startup, you can access the following interfaces:

-   **RabbitMQ Management**: [http://localhost:15672](http://localhost:15672) (login: user, password: password)
-   **MinIO Console**: [http://localhost:9001](http://localhost:9001) (login: minioadmin, password: minioadmin)
-   **Gateway API**: [http://localhost:8080](http://localhost:8080) (health check available at `/health`)

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
