# Photo Tags Service

An automated image processing service using AI for metadata generation.

## Project Overview

This service automatically adds titles, descriptions, and keywords to image metadata using AI artificial intelligence.

## Architecture

The project is built using a microservice architecture and includes the following components:

-   Gateway Service - receiving images and sending results via Telegram API
-   Analyzer Service - generating metadata using AI
-   Processor Service - writing metadata to images
-   RabbitMQ - message exchange between services
-   MinIO - image storage

## Installation and Launch

### Prerequisites

-   Docker and Docker Compose
-   OpenAI API key for AI access

### Starting the Project

1.  Clone the repository:
    ```bash
    git clone https://github.com/shabohin/photo-tags.git
    cd photo-tags
    ```
2.  Export your OpenAI API key in docker/env:

    ```bash
    OPENAI_API_KEY=your-api-key
    ```

3.  Start the services:

    ```bash
    ./scripts/start.sh
    ```

4.  Complete initial setup:

    ```bash
    ./scripts/setup.sh
    ```

## Usage

After launch, the bot will be accessible through Telegram. Send an image to the bot, and it will automatically process it and return it with added metadata.

## Development

### Project Structure

```
/
├── services/
│   ├── gateway/        # Service for receiving and sending via Telegram
│   ├── analyzer/       # Service for analysis and metadata generation
│   └── processor/      # Service for writing metadata to images
├── pkg/                # Shared packages
│   ├── messaging/      # RabbitMQ integration
│   ├── storage/        # MinIO integration
│   ├── logging/        # Logging
│   └── models/         # Data structures
├── docker/             # Docker configuration
└── scripts/            # Scripts for launch and setup
```

### Workflow (GitHub Flow)

1. Create a branch from `main` for your changes
2. Make changes and commit them
3. Open a Pull Request to `main`
4. Discuss and refine changes if necessary
5. After passing tests and review, merge changes into `main`
