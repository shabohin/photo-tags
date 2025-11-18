# E2E Tests - Quick Start Guide

## Overview

This directory contains End-to-End tests for the photo processing pipeline. The tests validate:
- Image upload to MinIO storage
- Message queue flow through RabbitMQ
- EXIF metadata validation
- Mock Telegram API functionality

## Prerequisites

### 1. Install Go 1.24+
```bash
go version
```

### 2. Install Docker & Docker Compose
```bash
docker --version
docker-compose --version
```

### 3. Install exiftool (optional, for metadata validation)
```bash
# Ubuntu/Debian
sudo apt-get install libimage-exiftool-perl

# macOS
brew install exiftool
```

## Running Tests

### Step 1: Start Infrastructure

Start RabbitMQ and MinIO using docker-compose:

```bash
cd ../../docker
docker-compose up -d rabbitmq minio
```

Wait for services to be ready (~10-15 seconds):
```bash
docker-compose logs rabbitmq | grep "Server startup complete"
docker-compose logs minio | grep "API"
```

### Step 2: Run E2E Tests

From the `tests/e2e` directory:

```bash
# Run all tests
make test-verbose

# Or manually:
go test -v -timeout 5m

# Run specific test
go test -v -run TestSimpleImageUpload

# Skip E2E tests (short mode)
go test -short
```

### Step 3: Cleanup

```bash
cd ../../docker
docker-compose down -v
```

## Test Categories

### Simple Tests (simple_e2e_test.go)
- **TestSimpleImageUpload**: Upload image to MinIO and publish to queue
- **TestExifToolValidation**: Validate EXIF metadata extraction
- **TestMockTelegramAPI**: Test mock Telegram API server
- **TestMessageQueueFlow**: Test RabbitMQ message flow
- **TestStorageOperations**: Test MinIO upload/download operations

## Test Structure

```
tests/e2e/
├── simple_e2e_test.go       # Main E2E tests (currently active)
├── pipeline_test.go.disabled # Full pipeline tests (WIP)
├── resilience_test.go.disabled # Resilience tests (WIP)
├── helpers/
│   ├── containers.go        # Container helpers
│   ├── exiftool.go         # EXIF validation
│   ├── mock_telegram.go    # Mock Telegram API
│   └── test_image.go       # Test image generation
├── testdata/               # Test data
├── go.mod                  # Dependencies
├── Makefile               # Build targets
└── README.md              # Full documentation
```

## Environment Variables

The tests use these environment variables (with defaults):

```bash
# RabbitMQ connection
RABBITMQ_URL=amqp://user:password@localhost:5672/

# MinIO connection
MINIO_ENDPOINT=localhost:9000
```

## Troubleshooting

### Tests Skip with "RabbitMQ not available"
```bash
# Check if RabbitMQ is running
docker-compose ps rabbitmq

# Check logs
docker-compose logs rabbitmq

# Restart if needed
docker-compose restart rabbitmq
```

### Tests Skip with "MinIO not available"
```bash
# Check if MinIO is running
docker-compose ps minio

# Check logs
docker-compose logs minio

# Restart if needed
docker-compose restart minio
```

### Port Already in Use
```bash
# Stop all services
docker-compose down

# Check for processes using ports
lsof -i :5672  # RabbitMQ
lsof -i :9000  # MinIO

# Start again
docker-compose up -d rabbitmq minio
```

### Tests Timeout
Increase timeout in test:
```go
go test -v -timeout 10m
```

## CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install exiftool
        run: sudo apt-get update && sudo apt-get install -y libimage-exiftool-perl

      - name: Start services
        run: |
          cd docker
          docker-compose up -d rabbitmq minio
          sleep 15

      - name: Run E2E tests
        run: |
          cd tests/e2e
          go test -v -timeout 10m

      - name: Cleanup
        if: always()
        run: |
          cd docker
          docker-compose down -v
```

## Development

### Adding New Tests

1. Create test function in `simple_e2e_test.go`:
```go
func TestMyNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    suite := setupSimpleTest(t)

    // Your test logic
    assert.NoError(t, err)
}
```

2. Run the test:
```bash
go test -v -run TestMyNewFeature
```

### Running Tests in Docker

```bash
# Build test image
docker build -t photo-tags-e2e-tests -f Dockerfile.test .

# Run tests
docker run --rm --network=docker_default photo-tags-e2e-tests
```

## Performance

- Single test: ~5-10 seconds
- Full suite: ~30-60 seconds
- Memory usage: ~100-200 MB
- Disk usage: ~10-20 MB

## Next Steps

Once the infrastructure is stable, we'll enable the full pipeline tests:
- `pipeline_test.go` - Full end-to-end pipeline with all services
- `resilience_test.go` - Rate limits, timeouts, error handling

To enable them:
```bash
mv pipeline_test.go.disabled pipeline_test.go
mv resilience_test.go.disabled resilience_test.go
```

## Support

For issues or questions:
1. Check service logs: `docker-compose logs <service>`
2. Verify connectivity: `docker-compose ps`
3. Review test output: `go test -v`
4. Check README.md for detailed documentation
