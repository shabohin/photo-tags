# Testing Strategy

This document outlines the comprehensive testing approach for the Photo Tags Service project, including test types, methodologies, and best practices.

## Documentation Links

-   [Main README](../README.md)
-   [Architecture Documentation](architecture.md)
-   [Development Guide](development.md)
-   [Deployment Guide](deployment.md)

## Testing Philosophy

The Photo Tags Service follows a test-driven development approach with a strong emphasis on automated testing at all levels. Our testing strategy aims to:

-   Ensure functionality meets requirements
-   Prevent regressions when making changes
-   Verify component interactions work as expected
-   Validate system performance under different conditions
-   Provide documentation through tests

## Test Types

### 1. Unit Tests

Unit tests verify that individual components work correctly in isolation.

**Characteristics:**

-   Focus on a single function, method, or small component
-   Fast execution (milliseconds)
-   No external dependencies (use mocks)
-   High coverage (target >80%)

**Example:**

```go
func TestImageValidator_ValidateFormat(t *testing.T) {
    validator := NewImageValidator()

    // Test valid JPEG
    err := validator.ValidateFormat("test.jpg", jpegBytes)
    assert.NoError(t, err)

    // Test invalid format
    err = validator.ValidateFormat("test.txt", textBytes)
    assert.Error(t, err)
}
```

**Key Areas for Unit Testing:**

-   Image format validation
-   Message serialization/deserialization
-   Metadata structure validation
-   Configuration loading
-   Utility functions

### 2. Integration Tests

Integration tests verify that components work correctly together.

**Characteristics:**

-   Test interactions between components
-   May involve external systems (RabbitMQ, MinIO)
-   Medium execution speed (seconds)
-   Cover critical paths

**Example:**

```go
func TestRabbitMQMessaging_SendReceive(t *testing.T) {
    // Setup test RabbitMQ instance
    rabbitmq := setupTestRabbitMQ(t)
    defer cleanupRabbitMQ()

    // Create producer and consumer
    producer := messaging.NewProducer(rabbitmq.URL)
    consumer := messaging.NewConsumer(rabbitmq.URL)

    // Test message sending and receiving
    message := &models.ImageUpload{TraceID: "test-123"}
    err := producer.PublishImageUpload(message)
    assert.NoError(t, err)

    received, err := consumer.ConsumeImageUpload(5*time.Second)
    assert.NoError(t, err)
    assert.Equal(t, message.TraceID, received.TraceID)
}
```

**Key Areas for Integration Testing:**

-   RabbitMQ message publishing and consumption
-   MinIO file upload and download
-   Telegram API interactions
-   OpenAI API interactions
-   ExifTool integration

### 3. End-to-End Tests

End-to-end tests verify complete workflows from user input to output.

**Characteristics:**

-   Test entire system behavior
-   Require all components running
-   Slower execution (minutes)
-   Focus on user scenarios

**Example:**

```go
func TestImageProcessingWorkflow(t *testing.T) {
    // Setup test environment with all services
    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Simulate sending image via Telegram
    imageData := loadTestImage("test.jpg")
    telegramMessage := createTelegramPhotoMessage(imageData)

    // Send to Gateway service
    env.GatewayService.HandleUpdate(telegramMessage)

    // Wait for processing to complete
    processedImage, err := waitForProcessedImage(env, 30*time.Second)
    assert.NoError(t, err)

    // Verify metadata was added
    metadata, err := extractMetadata(processedImage)
    assert.NoError(t, err)
    assert.NotEmpty(t, metadata.Title)
    assert.NotEmpty(t, metadata.Description)
    assert.Len(t, metadata.Keywords, 49)
}
```

**Key End-to-End Test Scenarios:**

-   Processing single image end-to-end
-   Processing multiple images in a group
-   Handling supported image formats
-   Recovery from component failures
-   Error handling for invalid inputs

### 4. Performance Tests

Performance tests validate the system's responsiveness, throughput, and resource usage.

**Characteristics:**

-   Measure system performance metrics
-   Test under various load conditions
-   Identify bottlenecks
-   Verify scalability

**Example:**

```go
func BenchmarkImageProcessingThroughput(b *testing.B) {
    env := setupTestEnvironment(b)
    defer env.Cleanup()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Send image for processing
        env.SendImage(testImages[i%len(testImages)])
    }

    // Measure throughput
    throughput := float64(b.N) / b.Elapsed().Seconds()
    b.ReportMetric(throughput, "images/sec")
}
```

**Key Performance Test Areas:**

-   Image processing throughput
-   Message queue performance
-   Response times under load
-   Memory and CPU utilization
-   Concurrent user handling

## Test Infrastructure

### Local Testing

Local tests run during development to provide quick feedback:

-   Unit tests run without Docker
-   Integration tests use Docker Compose for dependencies
-   End-to-end tests use a complete local Docker Compose environment

**Running Local Tests:**

```bash
# Run all tests
./scripts/test.sh

# Run with verbose output
go test -v ./...

# Run tests for specific service
go test ./services/gateway/...

# Run tests with race detection
go test -race ./...
```

### CI/CD Testing

Tests in the CI/CD pipeline verify code quality before integration:

-   GitHub Actions runs tests on every commit
-   Test failures prevent merging
-   Coverage reports highlight untested code
-   Performance benchmarks track changes over time

## Test Data Management

### Test Images

-   Standard test images stored in `testdata/`
-   Various formats (JPG, PNG) and dimensions
-   Different content types (landscapes, objects, people)
-   Edge cases (very large, small, or corrupted)

### Mock Services

-   Mock Telegram API for simulating user interactions
-   Mock OpenAI API for deterministic responses
-   MockMinio for storage testing without external dependencies

## Testing Best Practices

### Writing Effective Tests

1. **Test One Thing Per Test**: Each test should verify a single behavior
2. **Use Descriptive Test Names**: Names should describe what is being tested
3. **Follow AAA Pattern**: Arrange, Act, Assert
4. **Keep Tests Independent**: Tests should not depend on each other
5. **Use Test Tables**: For testing multiple inputs and outputs

### Test Coverage

-   Aim for >80% code coverage overall
-   Focus on critical paths and error handling
-   Use coverage reports to identify untested code
-   Don't pursue 100% coverage at the expense of meaningful tests

### Test Maintenance

-   Review and update tests when functionality changes
-   Refactor tests to improve clarity and reduce duplication
-   Keep test code as clean as production code
-   Document complex test setups and scenarios

## Mocking Strategy

### External Services

Use interface-based mocking for external dependencies:

```go
// Define interface
type OpenAIClient interface {
    GenerateMetadata(image []byte) (*models.Metadata, error)
}

// Real implementation
type RealOpenAIClient struct {
    apiKey string
}

// Mock implementation for testing
type MockOpenAIClient struct {
    mock.Mock
}

func (m *MockOpenAIClient) GenerateMetadata(image []byte) (*models.Metadata, error) {
    args := m.Called(image)
    return args.Get(0).(*models.Metadata), args.Error(1)
}
```

### Storage

Use in-memory storage for unit tests:

```go
type InMemoryStorage struct {
    files map[string][]byte
}

func (s *InMemoryStorage) UploadFile(path string, data []byte) error {
    s.files[path] = data
    return nil
}

func (s *InMemoryStorage) DownloadFile(path string) ([]byte, error) {
    data, ok := s.files[path]
    if !ok {
        return nil, errors.New("file not found")
    }
    return data, nil
}
```

## Testing Specific Components

### RabbitMQ Testing

-   Use test containers for integration tests
-   Create temporary queues for tests
-   Clean up queues after tests

### MinIO Testing

-   Use MinIO test server
-   Create test buckets for each test run
-   Clean up test files after tests

### Telegram Bot Testing

-   Mock Telegram API responses
-   Simulate message updates
-   Verify bot responses

### AI Integration Testing

-   Mock OpenAI API responses
-   Use deterministic responses for consistent testing
-   Test with various image types

## Troubleshooting Tests

### Common Issues

1. **Flaky Tests**: Tests that sometimes pass and sometimes fail
    - Add retry mechanisms
    - Increase timeouts
    - Fix race conditions
2. **Slow Tests**: Tests that take too long to run
    - Separate unit and integration tests
    - Use parallelism where appropriate
    - Optimize test setup/teardown
3. **Dependencies Between Tests**: Tests that depend on each other
    - Isolate test environments
    - Clean up after each test
    - Don't share state between tests

### Debugging Failed Tests

1. Use verbose mode to see detailed output:

    ```bash
    go test -v ./...
    ```

2. Add temporary debug statements:

    ```go
    t.Logf("Received message: %+v", message)
    ```

3. Use Go's race detector:
    ```bash
    go test -race ./...
    ```

## Continuous Testing

-   Run unit tests on every save
-   Run integration tests before committing
-   Run complete test suite before merging
-   Schedule nightly performance tests
