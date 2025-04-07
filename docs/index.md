# Photo Tags Service Documentation

Welcome to the Photo Tags Service documentation. This index provides an overview of all available documentation resources for the project.

## Core Documentation

-   [Main README](../README.md) - Project overview, installation instructions, and basic usage
-   [Architecture Documentation](architecture.md) - Detailed system architecture, components, and data flows
-   [Development Guide](development.md) - Development workflow, coding standards, and best practices
-   [Testing Strategy](testing.md) - Comprehensive testing approach and methodologies
-   [Deployment Guide](deployment.md) - Deployment options, infrastructure requirements, and monitoring

## Project Overview

Photo Tags Service is an automated image processing service that uses AI to generate metadata for images. The service adds titles, descriptions, and keywords to image metadata using OpenAI's GPT-4o. Users interact with the service through a Telegram bot, sending images and receiving processed versions with embedded metadata.

## Key Features

-   **Automatic Metadata Generation**: Uses GPT-4o to analyze images and generate appropriate metadata
-   **Metadata Embedding**: Writes metadata directly into image files (EXIF, IPTC, XMP)
-   **Simple User Interface**: Easy interaction through a Telegram bot
-   **Microservice Architecture**: Modular design for scalability and maintainability
-   **Comprehensive Testing**: Extensive testing at all levels of the application

## Architecture at a Glance

The system consists of three main services:

1. **Gateway Service** - Handles user interactions via Telegram
2. **Analyzer Service** - Processes images with GPT-4o to generate metadata
3. **Processor Service** - Embeds metadata into image files

These services communicate asynchronously through RabbitMQ message queues, and images are stored in MinIO object storage.

## Quick Start

For quick installation and setup, refer to the [Installation section in the main README](../README.md#installation-and-launch).

## Contributing

For information about contributing to the project, refer to the [Development Guide](development.md).

## Testing

For details about the project's testing approach, refer to the [Testing Strategy](testing.md).

## Deployment

For deployment options and instructions, refer to the [Deployment Guide](deployment.md).
