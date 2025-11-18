# Photo Tags Service Documentation

Welcome to the Photo Tags Service documentation. This index provides an overview of all available documentation resources for the project.

## Core Documentation

-   [Main README](../README.md) - Project overview, installation instructions, and basic usage
-   [TODO Status](TODO.md) - Current project status and development roadmap
-   [Architecture Documentation](architecture.md) - Detailed system architecture, components, and data flows
-   [Development Guide](development.md) - Development workflow, coding standards, and best practices
-   [Testing Strategy](testing.md) - Comprehensive testing approach and methodologies
-   [Deployment Guide](deployment.md) - Deployment options, infrastructure requirements, and monitoring

## Project Overview

Photo Tags Service is an automated image processing service that uses AI to generate metadata for images. The service adds titles, descriptions, and keywords to image metadata using free vision models from OpenRouter with automatic model selection. Users interact with the service through a Telegram bot or batch processing, sending images and receiving processed versions with embedded metadata.

## Key Features

-   **Automatic Metadata Generation**: Uses free vision models via OpenRouter with automatic model selection to analyze images and generate appropriate metadata
-   **Metadata Embedding**: Writes metadata directly into image files (EXIF, IPTC, XMP)
-   **Simple User Interface**: Easy interaction through a Telegram bot
-   **Batch Processing**: Process images in bulk via File Watcher Service
-   **Web Dashboard**: Monitor processing statistics and system health
-   **Dead Letter Queue**: Automatic failure tracking with manual retry capability
-   **Statistics API**: PostgreSQL-backed history and analytics
-   **Backup & Recovery**: Automated backup with easy restore functionality
-   **Flexible Deployment**: Docker or native/local deployment options
-   **Microservice Architecture**: Modular design for scalability and maintainability
-   **Comprehensive Monitoring**: Datadog integration for APM, metrics, and logs
-   **Comprehensive Testing**: Extensive testing at all levels of the application

## Architecture at a Glance

The system consists of five main services:

1. **Gateway Service** - Handles user interactions via Telegram and provides Statistics API
2. **Analyzer Service** - Generates metadata using free vision models from OpenRouter with automatic model selection
3. **Processor Service** - Embeds metadata into image files using ExifTool
4. **Filewatcher Service** - Monitors directories for batch image processing
5. **Dashboard Service** - Provides web-based monitoring and statistics interface

These services communicate asynchronously through RabbitMQ message queues with Dead Letter Queue support, images are stored in MinIO object storage, and statistics are tracked in PostgreSQL database.

## Quick Start

For quick installation and setup, refer to the [Installation section in the main README](../README.md#installation-and-launch).

## Contributing

For information about contributing to the project, refer to the [Development Guide](development.md).

## Testing

For details about the project's testing approach, refer to the [Testing Strategy](testing.md).

## Deployment

For deployment options and instructions, refer to the [Deployment Guide](deployment.md).