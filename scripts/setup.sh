#!/bin/bash

# Setup MinIO buckets

echo "Setting up MinIO..."

# Wait for MinIO to start
echo "Waiting for MinIO to start..."
until curl -s http://localhost:9000/minio/health/live > /dev/null; do
    sleep 1
done

# Install mc (MinIO Client) if not available
if ! command -v mc &> /dev/null; then
    echo "Installing MinIO Client..."
    wget https://dl.min.io/client/mc/release/linux-amd64/mc -O /tmp/mc
    chmod +x /tmp/mc
    MC_CMD="/tmp/mc"
else
    MC_CMD="mc"
fi

# Configure mc
$MC_CMD alias set myminio http://localhost:9000 minioadmin minioadmin

# Create buckets
$MC_CMD mb myminio/original
$MC_CMD mb myminio/processed

echo "MinIO setup complete. Buckets 'original' and 'processed' created."

# Setup RabbitMQ
echo "Setting up RabbitMQ..."

# Wait for RabbitMQ to start
echo "Waiting for RabbitMQ to start..."
until curl -s -u user:password http://localhost:15672/api/aliveness-test/%2F | grep -q "\"status\":\"ok\""; do
    sleep 1
done

# Create queues
curl -s -u user:password -X PUT http://localhost:15672/api/queues/%2F/image_upload
curl -s -u user:password -X PUT http://localhost:15672/api/queues/%2F/metadata_generated
curl -s -u user:password -X PUT http://localhost:15672/api/queues/%2F/image_process
curl -s -u user:password -X PUT http://localhost:15672/api/queues/%2F/image_processed

echo "RabbitMQ setup complete. Queues created."

echo "Setup complete!"