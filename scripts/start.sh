#!/bin/bash

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker and Docker Compose."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "Docker Compose is not installed. Please install Docker Compose."
    exit 1
fi

# Check for OPENAI_API_KEY environment variable
if [ -z "$OPENAI_API_KEY" ]; then
    echo "OPENAI_API_KEY is not set. Please set your OpenAI API key."
    echo "Example: export OPENAI_API_KEY=your-api-key"
    exit 1
fi

# Change to directory with docker-compose
cd "$(dirname "$0")/../docker"

# Start the project
docker-compose up -d

echo "Services started!"
echo "- RabbitMQ UI: http://localhost:15672 (user/password)"
echo "- MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
echo "- Gateway Service: http://localhost:8080"