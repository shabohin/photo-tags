# Photo Tags Service

An automated image processing service using AI for metadata generation with a focus on comprehensive testing and quality assurance.

## Project Overview

This service automatically adds titles, descriptions, and keywords to image metadata using artificial intelligence. Users send images through a Telegram bot and receive processed versions with added metadata. The project is designed with a strong emphasis on testing at all levels, from unit tests to comprehensive integration tests.

## Architecture

The project is built using a microservice architecture and includes the following components:

-   **Gateway Service** - receives images and sends results via Telegram API
-   **Analyzer Service** - generates metadata using GPT-4o via OpenRouter
-   **Processor Service** - writes metadata to images
-   **RabbitMQ** - message exchange between services
-   **MinIO** - image storage

For all project documentation, see the [Documentation Index](docs/index.md).
For more detailed architecture information, see [Architecture Documentation](docs/architecture.md).

## Project Structure

```
/
├── services/
│   ├── gateway/           # Gateway Service for handling user interactions
│   ├── analyzer/          # Analyzer Service for generating metadata
│   └── processor/         # Processor Service for writing metadata to images
│
├── pkg/                   # Shared packages used across services
│   ├── messaging/         # RabbitMQ communication
│   ├── storage/           # MinIO storage operations
│   ├── logging/           # Logging functionality
│   └── models/            # Shared data structures
│
├── docker/                # Docker configuration
│   ├── docker-compose.yml # Service orchestration
│   └── Dockerfile.service # Service container definition
│
└── scripts/               # Utility scripts
    ├── start.sh           # Start the system
    └── setup.sh           # Initial setup
```

## Development and Testing

This project follows a test-driven development approach with extensive testing at all levels:

-   Unit tests for all core components
-   Integration tests for service interactions
-   End-to-end tests for complete user workflows
-   Performance and stress testing

For detailed information about development workflows and testing strategies, see:

-   [Development Guide](docs/development.md)
-   [Testing Strategy](docs/testing.md)

## Installation and Launch

### Prerequisites

-   Docker and Docker Compose
-   Telegram bot token (get from [@BotFather](https://t.me/BotFather))
-   OpenRouter API key for GPT-4o access

### Starting the Project

1. Clone the repository:

    ```bash
    git clone https://github.com/shabohin/photo-tags.git
    cd photo-tags
    ```

2. Copy the environment template and configure it:

    ```bash
    cp docker/.env.example docker/.env
    ```

3. Edit the `.env` file and provide your API keys:

    ```bash
    # Edit docker/.env and set:
    # TELEGRAM_TOKEN - your Telegram bot token from @BotFather
    # OPENROUTER_API_KEY - your OpenRouter API key for GPT-4o access
    ```

4. Run the `start.sh` script to launch services:

    ```bash
    chmod +x scripts/*.sh
    ./scripts/start.sh
    ```

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

For more information about deployment options and monitoring, see [Deployment Guide](docs/deployment.md).

## Viewing Logs

To view service logs, use:

```bash
# Gateway service logs
docker logs gateway -f

# RabbitMQ logs
docker logs rabbitmq -f

# MinIO logs
docker logs minio -f
```