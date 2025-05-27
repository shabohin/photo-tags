#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker is not installed. Please install Docker and Docker Compose.${NC}"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null; then
    echo -e "${RED}Docker Compose is not available. Please ensure Docker is properly installed.${NC}"
    exit 1
fi

# Ensure script is called from project root
ROOT_DIR=$(dirname "$(dirname "$0")")
cd "$ROOT_DIR"

# Setup .env file if it doesn't exist
if [ ! -f "docker/.env" ]; then
    echo -e "${YELLOW}.env file not found. Creating from example...${NC}"
    if [ -f "docker/.env.example" ]; then
        cp docker/.env.example docker/.env
        echo -e "${YELLOW}Created .env file from example.${NC}"
        echo -e "${RED}IMPORTANT: Please edit docker/.env to set your TELEGRAM_TOKEN and OPENROUTER_API_KEY!${NC}"
        read -p "Would you like to edit the .env file now? (y/n): " EDIT_ENV
        if [[ $EDIT_ENV == "y" || $EDIT_ENV == "Y" ]]; then
            ${EDITOR:-vi} docker/.env
        fi
    else
        echo -e "${RED}.env.example file not found. Cannot create .env file.${NC}"
        exit 1
    fi
fi

# Change to directory with docker-compose
cd docker

# Check if Telegram token is set
if grep -q "TELEGRAM_TOKEN=your_telegram_bot_token_here" .env; then
    echo -e "${RED}TELEGRAM_TOKEN is not set in .env file.${NC}"
    echo -e "${YELLOW}The Gateway service will start, but Telegram bot will not work.${NC}"
fi

# Check if OpenRouter API key is set
if grep -q "OPENROUTER_API_KEY=your_openrouter_api_key_here" .env; then
    echo -e "${RED}OPENROUTER_API_KEY is not set in .env file.${NC}"
    echo -e "${YELLOW}The Analyzer service will not work properly without an API key.${NC}"
fi

# Start the project
echo -e "${GREEN}Starting services...${NC}"
docker compose up -d

# Check if services started successfully
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Services started successfully!${NC}"
    echo -e "- RabbitMQ UI: ${YELLOW}http://localhost:15672${NC} (user/password)"
    echo -e "- MinIO Console: ${YELLOW}http://localhost:9001${NC} (minioadmin/minioadmin)"
    echo -e "- Gateway Service: ${YELLOW}http://localhost:8080${NC}"

    echo ""
    echo -e "${GREEN}To view logs:${NC}"
    echo -e "  docker logs gateway -f    ${YELLOW}# For Gateway service${NC}"

    echo ""
    echo -e "${GREEN}To stop services:${NC}"
    echo -e "  ./scripts/stop.sh"
else
    echo -e "${RED}Failed to start services.${NC}"
    exit 1
fi
