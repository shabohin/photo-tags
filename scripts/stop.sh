#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Ensure script is called from project root
ROOT_DIR=$(dirname "$(dirname "$0")")
cd "$ROOT_DIR/docker"

echo -e "${YELLOW}Stopping services...${NC}"
docker-compose down

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Services stopped successfully!${NC}"
else
    echo -e "${RED}Failed to stop services.${NC}"
    exit 1
fi