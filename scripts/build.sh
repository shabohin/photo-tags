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

# Change to directory with docker-compose
cd docker

# Build all services
echo -e "${YELLOW}Building all services...${NC}"

# Parse command line arguments
SERVICES=""
if [ $# -gt 0 ]; then
    SERVICES="$@"
    echo -e "Building specific services: ${GREEN}${SERVICES}${NC}"
else
    echo -e "Building all services"
fi

# Build images
if [ -z "$SERVICES" ]; then
    docker compose build
else
    docker compose build $SERVICES
fi

# Check if build was successful
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Services built successfully!${NC}"
    echo -e "You can now run: ${YELLOW}./scripts/start.sh${NC} to start the services"
else
    echo -e "${RED}Failed to build services.${NC}"
    exit 1
fi
