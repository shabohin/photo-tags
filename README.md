# Photo Tags Service

An automated image processing service using AI for metadata generation.

## Project Overview

This service automatically adds titles, descriptions, and keywords to image metadata using artificial intelligence. Users send images through a Telegram bot and receive processed versions with added metadata.

## Architecture

The project is built using a microservice architecture and includes the following components:

-   **Gateway Service** - receives images and sends results via Telegram API
-   **Analyzer Service** - generates metadata using GPT-4o
-   **Processor Service** - writes metadata to images
-   **RabbitMQ** - message exchange between services
-   **MinIO** - image storage

## Installation and Launch

### Prerequisites

-   Docker and Docker Compose
-   Telegram bot token (get from [@BotFather](https://t.me/BotFather))
-   OpenAI API key for GPT-4o access

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
    - `OPENAI_API_KEY` - your OpenAI API key

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
