# E2E Tests for Photo Tags Pipeline

This directory contains end-to-end tests for the complete photo processing pipeline.

## Overview

The E2E tests validate the entire system workflow:
1. **Gateway** receives image uploads
2. **Analyzer** processes images and generates metadata
3. **Processor** embeds metadata into images using exiftool
4. **Gateway** returns processed images

## Architecture

The tests use:
- **testcontainers-go**: For Docker container orchestration and isolation
- **docker-compose**: To start all required services
- **Mock Telegram API**: To simulate Telegram Bot API without real tokens
- **exiftool**: To validate metadata in processed images

## Test Structure

```
tests/e2e/
├── go.mod                    # Go module dependencies
├── pipeline_test.go          # Main pipeline tests
├── resilience_test.go        # Rate limit, timeout, and error handling tests
├── helpers/
│   ├── containers.go         # Testcontainers helpers
│   ├── exiftool.go          # EXIF metadata validation
│   ├── mock_telegram.go     # Mock Telegram API server
│   └── test_image.go        # Test image generation
└── testdata/                # Test data directory
```

## Test Coverage

### Pipeline Tests (`pipeline_test.go`)
- ✅ **TestFullPipelineSuccess**: Complete pipeline from upload to processing
- ✅ **TestPipelineWithInvalidImage**: Failure handling for invalid images

### Resilience Tests (`resilience_test.go`)
- ✅ **TestRateLimitHandling**: Rate limit simulation and retry logic
- ✅ **TestNetworkErrorRecovery**: Network error handling and recovery
- ✅ **TestTimeoutHandling**: Timeout simulation
- ✅ **TestConcurrentProcessing**: Multiple concurrent image processing
- ✅ **TestLargeImageProcessing**: Large image (4K) processing

## Prerequisites

### Required Tools
```bash
# Docker and Docker Compose
docker --version
docker-compose --version

# Go 1.24+
go version

# exiftool (for metadata validation)
# Ubuntu/Debian:
sudo apt-get install libimage-exiftool-perl

# macOS:
brew install exiftool

# Verify installation:
exiftool -ver
```

### Environment Setup

The tests automatically create a temporary `.env.test` file with mock configuration. No manual environment setup is required.

## Running Tests

### Run All E2E Tests
```bash
cd tests/e2e
go test -v -timeout 30m
```

### Run Specific Test
```bash
go test -v -run TestFullPipelineSuccess -timeout 10m
```

### Run Tests in Short Mode (Skip E2E)
```bash
go test -short
```

### Run with Race Detection
```bash
go test -v -race -timeout 30m
```

## Test Configuration

### Timeouts
- **Test Timeout**: 5 minutes per test
- **Operation Timeout**: 30 seconds for individual operations
- **Processing Timeout**: Up to 3 minutes for large images

### Test Containers
The tests start the following containers:
- **RabbitMQ**: Message broker (ports 5672, 15672)
- **MinIO**: Object storage (ports 9000, 9001)
- **Gateway**: API gateway (port 8080)
- **Analyzer**: Image analysis service
- **Processor**: Metadata embedding service

### Resource Isolation
Each test suite:
- Creates an isolated Docker network
- Uses unique container identifiers
- Cleans up all resources after completion
- Runs in parallel with other tests (via testcontainers)

## Debugging

### View Container Logs
```bash
# During test execution, logs are visible in test output
go test -v -run TestFullPipelineSuccess 2>&1 | tee test.log
```

### Manual Container Inspection
If tests fail, containers might still be running:
```bash
# List all containers
docker ps -a | grep e2e_test

# View logs
docker logs <container_id>

# Clean up manually
docker-compose -f ../../docker/docker-compose.yml down
```

### Enable Debug Logging
Set environment variable for verbose logging:
```bash
export TESTCONTAINERS_RYUK_DISABLED=false
go test -v -run TestFullPipelineSuccess
```

## Troubleshooting

### Common Issues

#### 1. Docker Not Available
```
Error: Cannot connect to Docker daemon
```
**Solution**: Ensure Docker is running
```bash
sudo systemctl start docker  # Linux
# or
open -a Docker  # macOS
```

#### 2. Port Already in Use
```
Error: Bind for 0.0.0.0:5672 failed: port is already allocated
```
**Solution**: Stop conflicting services
```bash
docker-compose down
docker ps  # Check for other running containers
```

#### 3. Timeout Errors
```
Error: timeout waiting for processed image
```
**Solution**: Increase timeout or check service logs
```bash
# Increase timeout in test code
# Or check if services are running properly
docker-compose logs analyzer
docker-compose logs processor
```

#### 4. exiftool Not Found
```
Warning: Could not extract EXIF data
```
**Solution**: Install exiftool
```bash
# Ubuntu/Debian
sudo apt-get install libimage-exiftool-perl

# macOS
brew install exiftool
```

## CI/CD Integration

### GitHub Actions Example
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

      - name: Run E2E Tests
        run: |
          cd tests/e2e
          go test -v -timeout 30m
```

## Adding New Tests

### Example Test Structure
```go
func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    suite := setupTestEnvironment(t)

    // Your test logic here

    // Assertions
    assert.NoError(t, err)
    require.NotNil(t, result)
}
```

### Best Practices
1. Always use `setupTestEnvironment(t)` for test setup
2. Use `t.Cleanup()` for resource cleanup
3. Set reasonable timeouts
4. Log important steps with `t.Log()`
5. Use `testing.Short()` to allow skipping E2E tests
6. Validate both success and failure scenarios

## Performance Considerations

### Test Execution Time
- Single test: ~2-5 minutes
- Full suite: ~15-30 minutes
- Parallel execution: Supported via testcontainers

### Resource Usage
- Memory: ~2-4 GB per test suite
- Disk: ~1-2 GB for containers and images
- Network: Local Docker network (no external calls)

## Security

### Mock Telegram API
- Uses randomly generated bot tokens
- Runs on localhost only
- No real API calls
- Automatically cleaned up after tests

### Credentials
All credentials are test-only:
- **RabbitMQ**: user/password
- **MinIO**: minioadmin/minioadmin
- **Telegram**: Random mock tokens

## Contributing

When adding new E2E tests:
1. Follow existing test patterns
2. Document test purpose and expectations
3. Ensure proper cleanup
4. Add timeout handling
5. Update this README

## References

- [testcontainers-go Documentation](https://golang.testcontainers.org/)
- [Testify Documentation](https://github.com/stretchr/testify)
- [ExifTool Documentation](https://exiftool.org/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
