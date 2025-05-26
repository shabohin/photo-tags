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

## Multi-Module Project Structure

### Overview

The Photo Tags Service uses a multi-module Go project structure to organize code into logical units:

-   **Services**: Each service is a separate Go module with its own `go.mod` file
    -   `services/analyzer`: Processes images and generates metadata using AI
    -   `services/gateway`: Handles user input via Telegram and coordinates workflows
    -   `services/processor`: Processes and transforms images
-   **Shared Packages**: Common code used by multiple services
    -   `pkg`: Shared utilities, interfaces, and models

This structure provides several benefits:

-   **Separation of Concerns**: Each module has a clearly defined responsibility
-   **Dependency Management**: Modules can have different dependency versions as needed
-   **Build Efficiency**: Only rebuild what has changed
-   **Deployment Flexibility**: Deploy services independently

### Working with Multiple Modules

When developing in a multi-module project, keep these guidelines in mind:

1. **Module-Specific Commands**: Run Go commands within the specific module directory

    ```bash
    cd services/analyzer
    go test ./...
    ```

2. **Using Shared Code**: Import shared packages with the full module path

    ```go
    import "github.com/shabohin/photo-tags/pkg/messaging"
    ```

3. **Cross-Module Testing**: Test each module in isolation first, then test interactions

4. **Module Versioning**: Each module can be versioned independently as needed

### Module Dependencies

-   Services can depend on the shared `pkg` module
-   Services should not depend directly on other services
-   Communication between services occurs through defined interfaces (e.g., message queues)

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

We use GitHub Actions for CI/CD with a simplified approach based on make commands:

-   Automated testing on each commit
-   Code coverage reporting
-   Linting checks
-   Docker image building
-   Security scanning with Gosec

### Simplified CI Structure

Our CI pipeline has been streamlined to use the same make commands that developers use locally:

-   Consistent workflow between local development and CI
-   Centralized logic through the Makefile
-   Only 3 jobs total (reduced from 15 jobs previously)
-   No matrix strategy needed, reducing complexity

### CI Workflow Jobs

The CI workflow consists of the following jobs:

1. **Quality Checks**: Runs `make check` for all modules

    - Combines linting and testing into a single job
    - Executes unit tests with race detection
    - Runs golangci-lint for code quality
    - Generates and uploads coverage reports to Codecov

2. **Build**: Uses `make build` for all services

    - Only runs after quality checks succeed
    - Builds all services using Docker Compose
    - Ensures consistent build process with local development

3. **Security**: Runs security scanning
    - Uses Gosec to identify security issues
    - Scans each service independently

### Running CI Checks Locally

You can run the same checks locally that are executed in CI, using the identical make commands:

```bash
# Run quality checks (linting + tests) on all modules
make check

# Build all services
make build

# Run individual components manually if needed
make lint
make test
```

This consistency between local and CI environments ensures that if your code passes checks locally, it should also pass in the CI pipeline.

### Benefits of the Simplified Approach

The new CI approach provides several advantages:

-   **Consistent Developer Experience**: The same commands used locally now run in CI
-   **Reduced Maintenance**: Fewer jobs to maintain and configure
-   **Centralized Configuration**: Logic consolidated in Makefile and associated scripts
-   **Faster Execution**: Fewer parallel jobs means less overhead and faster overall pipeline completion
-   **Easier Troubleshooting**: When an issue occurs, it's easier to reproduce locally with the same commands

### Adding a New Module to CI

When adding a new Go module to the project:

1. Update the module-specific scripts (e.g., `scripts/lint.sh`, `scripts/test.sh`)
2. The CI pipeline will automatically include it without changes to the workflow file
3. Update the Makefile if needed for any module-specific build steps

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

1. Create service directory in the services directory:

    ```bash
    mkdir -p services/new-service/cmd services/new-service/internal
    ```

2. Set up the standard structure:

    ```
    services/new-service/
    ├── cmd/
    │   └── main.go           # Application entry point
    ├── internal/
    │   ├── config/           # Service configuration
    │   ├── handler/          # Business logic
    │   └── utils/            # Helper functions
    └── go.mod                # Dependencies
    ```

3. Initialize as a Go module:

    ```bash
    cd services/new-service
    go mod init github.com/shabohin/photo-tags/services/new-service
    ```

4. Configure Docker and Docker Compose files:

    - Add service to docker-compose.yml
    - Create appropriate Dockerfile or use the shared one

5. Add necessary message producers and consumers:

    - Implement interfaces from pkg/messaging
    - Define queue names and message structures

6. Update architecture documentation:

    - Add service to architecture diagrams
    - Document service responsibilities and interactions

7. Integrate with CI/CD pipeline:
    - Update scripts/lint.sh and scripts/test.sh to include the new service
    - Update scripts/pre-commit to include the new service
    - No changes needed to CI workflow file as it uses make commands

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
