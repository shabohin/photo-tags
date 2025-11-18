# Integration Tests

This document describes the integration testing setup for the Photo Tags services.

## Overview

Integration tests verify that services work correctly with real external dependencies:

- **Gateway**: Tests with real RabbitMQ and MinIO
- **Analyzer**: Tests with real RabbitMQ, MinIO, and mocked OpenRouter
- **Processor**: Tests with real RabbitMQ, MinIO, ExifTool, and storage

## Test Coverage

### Gateway Integration Tests
- RabbitMQ connection and retry logic
- MinIO connection and file operations
- Concurrent message publishing and consuming
- Concurrent file uploads and downloads
- Graceful shutdown

### Analyzer Integration Tests
- Image analysis with mocked OpenRouter API
- Retry logic with failing dependencies
- Concurrent message processing
- End-to-end workflow from upload to analysis
- Graceful shutdown

### Processor Integration Tests
- ExifTool metadata writing
- Image processing with real storage
- Retry logic for failed operations
- Concurrent image processing
- End-to-end workflow from analysis to processed image
- Graceful shutdown

## Prerequisites

1. **Docker and Docker Compose**
   ```bash
   docker --version
   docker compose version
   ```

2. **ExifTool** (for Processor tests)
   ```bash
   # Ubuntu/Debian
   sudo apt-get install libimage-exiftool-perl

   # macOS
   brew install exiftool
   ```

3. **Go 1.21+**
   ```bash
   go version
   ```

## Running Integration Tests

### Quick Start

Run all integration tests:
```bash
make test-integration
```

This command will:
1. Start test infrastructure (RabbitMQ, MinIO) using docker-compose
2. Run integration tests for all services
3. Stop and clean up test infrastructure

### Running Tests for Specific Services

#### Gateway
```bash
# Start test infrastructure
docker compose -f docker-compose.test.yml up -d

# Run tests
cd services/gateway
go test -v -tags=integration ./tests/integration/... -timeout 5m

# Cleanup
docker compose -f docker-compose.test.yml down -v
```

#### Analyzer
```bash
# Start test infrastructure
docker compose -f docker-compose.test.yml up -d

# Run tests
cd services/analyzer
go test -v -tags=integration ./tests/integration/... -timeout 5m

# Cleanup
docker compose -f docker-compose.test.yml down -v
```

#### Processor
```bash
# Start test infrastructure
docker compose -f docker-compose.test.yml up -d

# Run tests
cd services/processor
go test -v -tags=integration ./tests/integration/... -timeout 5m

# Cleanup
docker compose -f docker-compose.test.yml down -v
```

## Test Infrastructure

The `docker-compose.test.yml` file defines test dependencies:

### RabbitMQ Test Instance
- Port: 5673 (to avoid conflicts with development instance)
- Management UI: http://localhost:15673
- Credentials: testuser/testpass

### MinIO Test Instance
- API Port: 9002 (to avoid conflicts with development instance)
- Console Port: 9003
- Credentials: testuser/testpass123

## Test Configuration

Integration tests use the following configuration:

```go
const (
    testRabbitMQURL    = "amqp://testuser:testpass@localhost:5673/"
    testMinIOEndpoint  = "localhost:9002"
    testMinIOAccessKey = "testuser"
    testMinIOSecretKey = "testpass123"
    testTimeout        = 30 * time.Second
    retryAttempts      = 5
    retryDelay         = 2 * time.Second
)
```

## Writing New Integration Tests

### Test Structure

1. **TestMain**: Setup and teardown infrastructure
2. **Infrastructure checks**: Verify dependencies are available
3. **Individual test functions**: Test specific functionality

Example:
```go
func TestMain(m *testing.M) {
    if !checkInfrastructure() {
        fmt.Println("Test infrastructure not ready")
        os.Exit(1)
    }
    code := m.Run()
    os.Exit(code)
}

func TestFeature(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
    defer cancel()

    // Test implementation
}
```

### Best Practices

1. **Use contexts with timeouts**: Prevent tests from hanging
2. **Clean up resources**: Always defer cleanup operations
3. **Use unique names**: Generate unique queue/bucket names for parallel tests
4. **Test retry logic**: Verify services handle transient failures
5. **Test concurrent operations**: Ensure thread-safety
6. **Test graceful shutdown**: Verify services shutdown cleanly

### Mock Dependencies

For external APIs (like OpenRouter), use mocks:

```go
type MockOpenRouterClient struct {
    mu            sync.Mutex
    callCount     int
    shouldFail    bool
    failCount     int
    failThreshold int
}

func (m *MockOpenRouterClient) AnalyzeImage(ctx context.Context, imageBytes []byte, traceID string) (model.Metadata, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.callCount++

    if m.shouldFail && m.callCount <= m.failThreshold {
        return model.Metadata{}, fmt.Errorf("simulated failure")
    }

    return model.Metadata{
        Title:       "Test Image",
        Description: "Test Description",
        Keywords:    []string{"test"},
    }, nil
}
```

## Troubleshooting

### Tests Fail to Connect to Infrastructure

1. Check if docker containers are running:
   ```bash
   docker compose -f docker-compose.test.yml ps
   ```

2. Check container logs:
   ```bash
   docker compose -f docker-compose.test.yml logs rabbitmq-test
   docker compose -f docker-compose.test.yml logs minio-test
   ```

3. Ensure ports are not in use:
   ```bash
   lsof -i :5673  # RabbitMQ
   lsof -i :9002  # MinIO API
   ```

### ExifTool Tests Fail

1. Verify ExifTool is installed:
   ```bash
   exiftool -ver
   ```

2. Check temp directory permissions:
   ```bash
   ls -la /tmp/processor-integration-tests
   ```

### Timeout Errors

1. Increase test timeout:
   ```bash
   go test -v -tags=integration ./tests/integration/... -timeout 10m
   ```

2. Check system resources (CPU, memory)

### Cleanup Issues

1. Force cleanup:
   ```bash
   docker compose -f docker-compose.test.yml down -v --remove-orphans
   ```

2. Remove stuck containers:
   ```bash
   docker ps -a | grep test
   docker rm -f <container_id>
   ```

## CI/CD Integration

Integration tests can be added to CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
- name: Run Integration Tests
  run: |
    docker compose -f docker-compose.test.yml up -d
    sleep 10
    make test-integration
    docker compose -f docker-compose.test.yml down -v
```

## Performance Considerations

- Integration tests are slower than unit tests
- Run integration tests on CI/CD, not on every local commit
- Use parallel test execution when possible
- Clean up resources to avoid memory leaks

## Metrics

Integration tests verify the following metrics:

- **Throughput**: Concurrent processing of multiple messages/images
- **Reliability**: Retry logic and error handling
- **Stability**: Graceful shutdown and resource cleanup
- **Correctness**: End-to-end workflows produce expected results
