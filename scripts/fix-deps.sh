#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Installing missing Go modules...${NC}"

# Fix gateway dependencies
echo -e "\n${GREEN}Fixing gateway dependencies...${NC}"
cd services/gateway || exit 1
go mod download github.com/go-telegram-bot-api/telegram-bot-api/v5
echo "Downloaded telegram-bot-api"
go get github.com/google/uuid
echo "Downloaded uuid"
go mod tidy
echo "Ran go mod tidy"
cd ../..

# Fix other modules for consistency
echo -e "\n${GREEN}Updating other module dependencies...${NC}"
cd services/analyzer || exit 1
go mod tidy
cd ../..

cd services/processor || exit 1
go mod tidy
cd ../..

cd pkg || exit 1
go mod tidy
cd ..

echo -e "${GREEN}Dependencies fixed successfully!${NC}"