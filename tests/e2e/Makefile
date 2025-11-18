.PHONY: help test test-verbose test-short test-race clean deps lint

# Default target
help:
	@echo "E2E Test Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make test         - Run all E2E tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-short   - Skip E2E tests (short mode)"
	@echo "  make test-race    - Run tests with race detector"
	@echo "  make deps         - Download dependencies"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Clean up containers and temp files"

# Run all E2E tests
test:
	@echo "Running E2E tests..."
	go test -timeout 30m

# Run tests with verbose output
test-verbose:
	@echo "Running E2E tests (verbose)..."
	go test -v -timeout 30m

# Skip E2E tests
test-short:
	@echo "Running tests in short mode (E2E tests skipped)..."
	go test -short

# Run tests with race detector
test-race:
	@echo "Running E2E tests with race detector..."
	go test -v -race -timeout 30m

# Run specific test
test-one:
	@echo "Running specific test: $(TEST)"
	go test -v -run $(TEST) -timeout 10m

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean up
clean:
	@echo "Cleaning up..."
	docker-compose -f ../../docker/docker-compose.yml down -v || true
	docker network prune -f || true
	docker volume prune -f || true
	go clean -testcache

# Check prerequisites
check-prereqs:
	@echo "Checking prerequisites..."
	@command -v docker >/dev/null 2>&1 || { echo "Error: docker not found"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "Error: docker-compose not found"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "Error: go not found"; exit 1; }
	@command -v exiftool >/dev/null 2>&1 || { echo "Warning: exiftool not found (some tests may fail)"; }
	@echo "All prerequisites satisfied!"

# Initialize test environment
init: check-prereqs deps
	@echo "Test environment initialized!"
