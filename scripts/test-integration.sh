#!/bin/bash

# Integration test script for photo-tags services
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting integration tests...${NC}"

# Check if docker-compose is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

# Check if docker compose is available (try both docker-compose and docker compose)
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    echo -e "${RED}Error: docker-compose is not installed${NC}"
    exit 1
fi

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo -e "${YELLOW}Starting test infrastructure...${NC}"

# Start test infrastructure
cd "$PROJECT_ROOT"
$DOCKER_COMPOSE -f docker-compose.test.yml up -d

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to be ready...${NC}"
sleep 10

# Function to check if service is ready
check_service() {
    local service=$1
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if $DOCKER_COMPOSE -f docker-compose.test.yml ps | grep -q "${service}.*healthy"; then
            echo -e "${GREEN}${service} is ready${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -e "${YELLOW}Waiting for ${service}... (${attempt}/${max_attempts})${NC}"
        sleep 2
    done

    echo -e "${RED}${service} failed to become ready${NC}"
    return 1
}

# Check if services are healthy
check_service "rabbitmq-test" || {
    echo -e "${RED}RabbitMQ failed to start${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml logs rabbitmq-test
    $DOCKER_COMPOSE -f docker-compose.test.yml down -v
    exit 1
}

check_service "minio-test" || {
    echo -e "${RED}MinIO failed to start${NC}"
    $DOCKER_COMPOSE -f docker-compose.test.yml logs minio-test
    $DOCKER_COMPOSE -f docker-compose.test.yml down -v
    exit 1
}

echo -e "${GREEN}Test infrastructure is ready${NC}"

# Track test results
GATEWAY_RESULT=0
ANALYZER_RESULT=0
PROCESSOR_RESULT=0

# Run Gateway integration tests
echo -e "${YELLOW}Running Gateway integration tests...${NC}"
cd "$PROJECT_ROOT/services/gateway"
if go test -v -tags=integration ./tests/integration/... -timeout 5m; then
    echo -e "${GREEN}Gateway integration tests passed${NC}"
else
    echo -e "${RED}Gateway integration tests failed${NC}"
    GATEWAY_RESULT=1
fi

# Run Analyzer integration tests
echo -e "${YELLOW}Running Analyzer integration tests...${NC}"
cd "$PROJECT_ROOT/services/analyzer"
if go test -v -tags=integration ./tests/integration/... -timeout 5m; then
    echo -e "${GREEN}Analyzer integration tests passed${NC}"
else
    echo -e "${RED}Analyzer integration tests failed${NC}"
    ANALYZER_RESULT=1
fi

# Run Processor integration tests
echo -e "${YELLOW}Running Processor integration tests...${NC}"
cd "$PROJECT_ROOT/services/processor"
if go test -v -tags=integration ./tests/integration/... -timeout 5m; then
    echo -e "${GREEN}Processor integration tests passed${NC}"
else
    echo -e "${RED}Processor integration tests failed${NC}"
    PROCESSOR_RESULT=1
fi

# Cleanup
echo -e "${YELLOW}Stopping test infrastructure...${NC}"
cd "$PROJECT_ROOT"
$DOCKER_COMPOSE -f docker-compose.test.yml down -v

# Print summary
echo -e "\n${YELLOW}======================================${NC}"
echo -e "${YELLOW}Integration Test Summary${NC}"
echo -e "${YELLOW}======================================${NC}"

if [ $GATEWAY_RESULT -eq 0 ]; then
    echo -e "Gateway:   ${GREEN}PASSED${NC}"
else
    echo -e "Gateway:   ${RED}FAILED${NC}"
fi

if [ $ANALYZER_RESULT -eq 0 ]; then
    echo -e "Analyzer:  ${GREEN}PASSED${NC}"
else
    echo -e "Analyzer:  ${RED}FAILED${NC}"
fi

if [ $PROCESSOR_RESULT -eq 0 ]; then
    echo -e "Processor: ${GREEN}PASSED${NC}"
else
    echo -e "Processor: ${RED}FAILED${NC}"
fi

# Exit with error if any test failed
if [ $GATEWAY_RESULT -ne 0 ] || [ $ANALYZER_RESULT -ne 0 ] || [ $PROCESSOR_RESULT -ne 0 ]; then
    echo -e "\n${RED}Some integration tests failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}All integration tests passed!${NC}"
exit 0
