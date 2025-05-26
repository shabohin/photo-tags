# Development Guide

This document outlines development workflows, best practices, and processes for contributing to the Photo Tags Service project.

## Documentation Links

-   [Main README](../README.md)
-   [Architecture Documentation](architecture.md)
-   [Testing Strategy](testing.md)
-   [Deployment Guide](deployment.md)

## Development Environment Setup

### Prerequisites

-   Go 1.21+
-   Docker and Docker Compose
-   Git
-   Visual Studio Code (recommended)

### Local Setup

1. Clone the repository:

    ```bash
    git clone https://github.com/shabohin/photo-tags.git
    cd photo-tags
    ```

2. Copy and configure environment variables:

    ```bash
    cp docker/.env.example docker/.env
    # Edit docker/.env and set your TELEGRAM_TOKEN and OPENROUTER_API_KEY
    ```

3. Set up VSCode for Go development:

    - Install Go extension
    - Configure Go linting and formatting
    - Set up test explorer

4. Configure pre-commit hooks:

    ```bash
    ./scripts/setup.sh
    ```

5. Start the development environment:
    ```bash
    ./scripts/start.sh
    ```

## Development Workflow

### Branch Strategy

-   `main`: Stable production code
-   `develop`: Integration branch for completed features
-   `feature/*`: Feature development
-   `bugfix/*`: Bug fixes
-   `release/*`: Release preparation

### Coding Standards

-   Follow Go best practices and idiomatic Go
-   Use the project's linting configuration (.golangci.yml)
-   Ensure all code is covered by tests

### Development Process

1. **Feature Planning**

    - Understand requirements
    - Design interfaces and interactions
    - Plan tests

2. **Implementation**

    - Follow test-driven development practices
    - Write unit tests first
    - Implement functionality to pass tests
    - Add integration tests

3. **Code Review**

    - Submit a pull request to the develop branch
    - Ensure CI tests pass
    - Address review comments

4. **Integration**
    - Merge approved code to develop
    - Verify integration tests pass

## Testing Approach

We follow a comprehensive testing strategy with multiple testing levels:

### Unit Testing

-   Test individual functions and methods
-   Use Go's testing package
-   Mock external dependencies
-   Target high test coverage (>80%)

### Integration Testing

-   Test interactions between components
-   Use Docker Compose for local testing
-   Test messaging, storage, and API interactions

### End-to-End Testing

-   Test complete workflows
-   Verify from user input to user output
-   Use automated Telegram API testing

### Performance Testing

-   Benchmark critical operations
-   Run stress tests to identify bottlenecks
-   Verify system performance meets requirements

### Running Tests

To run all tests:

```bash
./scripts/test.sh
```

To run unit tests only:

```bash
go test ./...
```

To run specific service tests:

```bash
go test ./services/gateway/...
```

## Continuous Integration

We use GitHub Actions for CI/CD:

-   Automated testing on each commit
-   Code coverage reporting
-   Linting checks
-   Docker image building and testing

## Documentation

-   Update documentation when changing functionality
-   Document all exported functions and types
-   Keep architecture documentation up-to-date
-   Update README.md when adding new features

## Common Development Tasks

### Adding a New Queue

1. Define queue name in constants
2. Update both producer and consumer services
3. Add message structure to models package
4. Update documentation

### Modifying Service Communication

1. Update message structures in models package
2. Ensure backward compatibility or version the API
3. Update both the sender and receiver services
4. Update tests to reflect changes

### Adding a New Service

1. Create service directory in the services directory
2. Set up the standard structure (cmd, internal)
3. Configure Docker and Docker Compose files
4. Add necessary message producers and consumers
5. Update architecture documentation

### Debugging Tips

-   Use structured logging with trace_id for tracking requests
-   Monitor RabbitMQ queues for message flow
-   Check MinIO for correct file storage
-   Use Docker logs to view service output

## Dependency Management

-   Use Go modules for dependency management
-   Pin dependencies to specific versions
-   Regularly update dependencies to address security issues
-   Use the fix-deps.sh script to resolve dependency issues:
    ```bash
    ./scripts/fix-deps.sh
    ```

## Code Structure Guidelines

### Service Structure

Each service follows a standard structure:

```
services/[service-name]/
├── cmd/
│   └── main.go           # Application entry point
├── internal/
│   ├── config/           # Service configuration
│   ├── handler/          # Business logic
│   ├── [component]/      # Service-specific components
│   └── utils/            # Helper functions
└── go.mod                # Dependencies
```

### Shared Packages Structure

```
pkg/
├── messaging/            # RabbitMQ communication
├── storage/              # MinIO interactions
├── logging/              # Logging functionality
└── models/               # Shared data structures
```

## Infrastructure Management

### Docker Images

-   Use multi-stage builds for smaller images
-   Follow Docker best practices (minimal images)
-   Tag images with semantic versioning

### Local Development

Use Docker Compose for local development:

```bash
docker-compose -f docker/docker-compose.yml up -d
```

### Staging Environment

-   Mirror production configuration
-   Use separate infrastructure
-   Deploy using CI/CD pipeline

### Troubleshooting

1. Check service logs:

    ```bash
    docker logs [service-name]
    ```

2. Verify RabbitMQ queues:

    - Access RabbitMQ management interface (http://localhost:15672)
    - Check queue depths and message rates

3. Inspect MinIO storage:
    - Access MinIO console (http://localhost:9001)
    - Verify file uploads and permissions