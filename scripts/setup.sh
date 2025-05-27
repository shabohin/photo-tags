#!/bin/bash

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Setting up the environment...${NC}"

# Check if docker-compose is running
if ! docker ps &>/dev/null; then
    echo -e "${RED}Docker not running. Please start Docker first.${NC}"
    exit 1
fi

# Check if our containers are running
if ! docker ps | grep -q "minio"; then
    echo -e "${RED}MinIO container not running. Please run './scripts/start.sh' first.${NC}"
    exit 1
fi

if ! docker ps | grep -q "rabbitmq"; then
    echo -e "${RED}RabbitMQ container not running. Please run './scripts/start.sh' first.${NC}"
    exit 1
fi

# Wait for services to be ready
echo -e "${YELLOW}Waiting for services to be ready...${NC}"

# Wait for RabbitMQ to start (retry for 30 seconds)
echo -e "Waiting for RabbitMQ..."
RETRY=0
MAX_RETRY=30
while [ $RETRY -lt $MAX_RETRY ]; do
    if curl -s -u user:password http://localhost:9100/api/aliveness-test/%2F | grep -q "\"status\":\"ok\""; then
        echo -e "${GREEN}RabbitMQ is ready!${NC}"
        break
    fi
    RETRY=$((RETRY+1))
    sleep 1
    echo -n "."
done

if [ $RETRY -eq $MAX_RETRY ]; then
    echo -e "${RED}Failed to connect to RabbitMQ after $MAX_RETRY seconds.${NC}"
    exit 1
fi

# Wait for MinIO to start (retry for 30 seconds)
echo -e "Waiting for MinIO..."
RETRY=0
while [ $RETRY -lt $MAX_RETRY ]; do
    if curl -s http://localhost:9000/minio/health/live &>/dev/null; then
        echo -e "${GREEN}MinIO is ready!${NC}"
        break
    fi
    RETRY=$((RETRY+1))
    sleep 1
    echo -n "."
done

if [ $RETRY -eq $MAX_RETRY ]; then
    echo -e "${RED}Failed to connect to MinIO after $MAX_RETRY seconds.${NC}"
    exit 1
fi

# Setup MinIO buckets
echo -e "${YELLOW}Setting up MinIO buckets...${NC}"

# Create a temporary mc config and use it directly from the host
MC_DIR=$(mktemp -d)
echo -e "Downloading MinIO client to temporary directory: ${MC_DIR}"

# Download mc client to temporary directory
if [ "$(uname)" == "Darwin" ]; then
    # macOS
    curl -s -o ${MC_DIR}/mc https://dl.min.io/client/mc/release/darwin-amd64/mc
else
    # Linux
    curl -s -o ${MC_DIR}/mc https://dl.min.io/client/mc/release/linux-amd64/mc
fi

chmod +x ${MC_DIR}/mc

# Configure MinIO client
echo -e "Configuring MinIO client..."
${MC_DIR}/mc config host add myminio http://localhost:9000 minioadmin minioadmin

# Create buckets
echo -e "Creating buckets..."
${MC_DIR}/mc mb -p myminio/original
${MC_DIR}/mc mb -p myminio/processed

# Cleanup mc client
rm -rf ${MC_DIR}

echo -e "${GREEN}MinIO setup complete. Buckets 'original' and 'processed' created.${NC}"

# Setup RabbitMQ queues
echo -e "${YELLOW}Setting up RabbitMQ queues...${NC}"

# Create queues using RabbitMQ HTTP API
for QUEUE in image_upload metadata_generated image_process image_processed; do
    echo -e "Creating queue: ${QUEUE}"
    curl -s -u user:password -X PUT "http://localhost:9100/api/queues/%2F/${QUEUE}" \
         -H "Content-Type: application/json" \
         -d '{"durable": true}' > /dev/null
done

echo -e "${GREEN}RabbitMQ setup complete. Queues created.${NC}"

echo -e "${GREEN}Environment setup complete!${NC}"
echo -e "You can now access the services at:"
echo -e "- Gateway: ${YELLOW}http://localhost:9003/health${NC}"
echo -e "- RabbitMQ UI: ${YELLOW}http://localhost:9100${NC} (login: user, password: password)"
echo -e "- MinIO Console: ${YELLOW}http://localhost:9001${NC} (login: minioadmin, password: minioadmin)"