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

We use GitHub Actions for CI/CD with a matrix strategy optimized for our multi-module project structure:

-   Automated testing on each commit
-   Code coverage reporting
-   Linting checks
-   Docker image building and testing
-   Security scanning with Gosec

### Multi-Module CI Structure

Our CI pipeline is designed to efficiently handle the multi-module nature of the project:

-   Matrix strategy for parallel job execution across modules
-   Module-specific workflows for each Go module:
    -   `services/analyzer`
    -   `services/gateway`
    -   `services/processor`
    -   `pkg`
-   Independent testing and linting for each module
-   Module-specific working directories in GitHub Actions

### CI Workflow Jobs

The CI workflow consists of the following jobs:

1. **Lint**: Runs golangci-lint on each module

    - Uses module-specific configuration
    - Sets working directory to the module path
    - Uses cached dependencies for faster execution

2. **Test**: Runs tests for each module

    - Executes unit tests with race detection
    - Generates coverage reports
    - Uploads coverage data to Codecov with module-specific flags

3. **Build**: Compiles services

    - Only runs after lint and test jobs succeed
    - Builds each service separately
    - Verifies the code can be compiled successfully

4. **Security**: Runs security scanning
    - Uses Gosec to identify security issues
    - Scans each module independently

### Running CI Checks Locally

You can run the same checks locally that are executed in CI:

```bash
# Run linting on all modules
./scripts/lint.sh

# Run tests on all modules
./scripts/test.sh

# Run the pre-commit checks
./scripts/pre-commit
```

The pre-commit hook is configured to run checks on each module separately, mirroring the CI behavior.

### Adding a New Module to CI

To add a new Go module to the CI pipeline:

1. Update the module list in `.github/workflows/ci.yml`:

    ```yaml
    matrix:
        module:
            - services/analyzer
            - services/gateway
            - services/processor
            - pkg
            - your-new-module
    ```

2. Update the module list in `scripts/pre-commit`:

    ```bash
    GO_MODULES=("services/analyzer" "services/gateway" "services/processor" "pkg" "your-new-module")
    ```

3. Update other scripts as needed (e.g., `scripts/lint.sh`, `scripts/test.sh`)

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
    - Add to matrix configuration in .github/workflows/ci.yml
    - Add to GO_MODULES array in scripts/pre-commit
    - Update scripts/lint.sh to include the new service

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
