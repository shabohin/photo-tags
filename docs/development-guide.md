# Development Guide

## Quick Start

1. **Clone and setup**:
   ```bash
   git clone https://github.com/shabohin/photo-tags.git
   cd photo-tags
   ```

2. **Install development tools**:
   ```bash
   make install-tools
   ```

3. **Install dependencies**:
   ```bash
   make deps
   ```

4. **Setup Git hooks** (optional):
   ```bash
   make install-hooks
   ```

## Development Workflow

### Before Starting Work

1. **Pull latest changes**:
   ```bash
   git pull origin main
   ```

2. **Update dependencies** (if needed):
   ```bash
   make deps
   ```

3. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

### During Development

1. **Run tests frequently**:
   ```bash
   make test
   ```

2. **Check code quality**:
   ```bash
   make lint
   ```

3. **Format code**:
   ```bash
   make fmt
   ```

### Before Committing

1. **Run full quality check**:
   ```bash
   make pre-commit
   ```

2. **Fix any issues**:
   ```bash
   make lint-fix  # Auto-fix linting issues
   make fmt       # Format code
   ```

3. **Commit changes**:
   ```bash
   git add .
   git commit -m "feat: your descriptive commit message"
   ```

## Code Standards

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports` for formatting
- Write meaningful variable and function names
- Add comments for exported functions and types

### Error Handling

```go
// Always handle errors
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Use context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### Testing

- Write unit tests for all business logic
- Use table-driven tests when appropriate
- Mock external dependencies
- Aim for good test coverage

```go
func TestProcessImage(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        // test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Project Structure

```
photo-tags/
├── services/           # Microservices
│   ├── gateway/       # Telegram bot and HTTP API
│   ├── analyzer/      # AI metadata generation
│   └── processor/     # Image processing
├── pkg/               # Shared packages
│   ├── models/        # Data models
│   ├── storage/       # Storage interfaces
│   ├── messaging/     # Message queue
│   └── logging/       # Structured logging
├── docker/            # Docker configuration
├── scripts/           # Automation scripts
├── docs/              # Documentation
└── .github/           # GitHub workflows
```

## Environment Setup

### Local Development

1. **Start infrastructure**:
   ```bash
   make start  # Starts RabbitMQ, MinIO, services
   ```

2. **Setup environment**:
   ```bash
   ./scripts/setup.sh  # Creates buckets, queues
   ```

3. **Check services**:
   - RabbitMQ: http://localhost:15672 (user/password)
   - MinIO: http://localhost:9001 (minioadmin/minioadmin)
   - Gateway: http://localhost:8080/health

### Testing

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Generate coverage reports
make test-coverage
```

## Common Tasks

### Adding New Dependencies

1. **Add to go.mod**:
   ```bash
   cd services/gateway  # or relevant service
   go get github.com/some/package
   ```

2. **Update all dependencies**:
   ```bash
   make deps
   ```

3. **Commit changes**:
   ```bash
   git add go.mod go.sum
   git commit -m "deps: add new dependency"
   ```

### Adding New Service

1. **Create service directory**:
   ```bash
   mkdir -p services/newservice/{cmd,internal}
   ```

2. **Initialize go module**:
   ```bash
   cd services/newservice
   go mod init github.com/shabohin/photo-tags/services/newservice
   ```

3. **Update Docker configuration**
4. **Add to CI/CD pipeline**
5. **Update documentation**

### Debugging

1. **Check service logs**:
   ```bash
   docker logs gateway -f
   docker logs analyzer -f
   docker logs processor -f
   ```

2. **Check infrastructure logs**:
   ```bash
   docker logs rabbitmq -f
   docker logs minio -f
   ```

3. **Health checks**:
   ```bash
   curl http://localhost:8080/health
   ```

## Troubleshooting

### Common Issues

1. **Dependencies issues**:
   ```bash
   make deps-clean  # Clean and reinstall
   ```

2. **Linting failures**:
   ```bash
   make lint-fix    # Auto-fix issues
   make fmt         # Format code
   ```

3. **Docker issues**:
   ```bash
   make stop        # Stop all services
   docker system prune -f  # Clean Docker
   make start       # Restart services
   ```

4. **Git hooks not working**:
   ```bash
   make install-hooks  # Reinstall hooks
   chmod +x .git/hooks/pre-commit
   ```

## Best Practices

### Git Workflow

- Use conventional commit messages
- Keep commits atomic and focused
- Write descriptive commit messages
- Use feature branches for new work

### Code Review

- Review code for logic, style, and tests
- Check for proper error handling
- Verify documentation is updated
- Ensure CI/CD passes

### Performance

- Profile code when needed
- Use appropriate data structures
- Handle concurrent operations safely
- Monitor resource usage

---

For more specific guides, see:
- [golangci-lint Guide](./golangci-lint-guide.md)
- [Commands Structure](./commands-structure.md)
