# golangci-lint Guide for Photo Tags Service

## Installation

1. **Automatic installation** (recommended):
   ```bash
   ./scripts/install-golangci-lint.sh
   ```

2. **Via Makefile**:
   ```bash
   make install-tools
   ```

3. **Manual installation**:
   ```bash
   # Via Go
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
   
   # Via Homebrew (macOS)
   brew install golangci-lint
   ```

## Usage

### Basic Commands

```bash
# Run linter on all modules
make lint

# Run linter with auto-fix
make lint-fix

# Format code
make fmt

# Full pre-commit check (format + lint + tests)
make pre-commit

# Run all quality checks (tests + linting)
make check
```

### Working with Individual Services

```bash
# Lint specific service
cd services/gateway
golangci-lint run

# With auto-fix
cd services/gateway
golangci-lint run --fix

# Only specific linters
cd services/gateway
golangci-lint run --enable=errcheck,govet
```

## IDE Integration

### VS Code

1. Install Go extension
2. Settings are already configured in `.vscode/settings.json`
3. Linter will run automatically on save

### GoLand/IntelliJ IDEA

1. Go to: Settings → Tools → Go Linter
2. Enable: golangci-lint
3. Set path to golangci-lint binary

## Git Hooks

Install pre-commit hook:

```bash
make install-hooks
```

The hook will run:
1. Code formatting (gofmt + goimports)
2. Linting (golangci-lint)
3. Tests (go test)

## Configuration

Main configuration in `.golangci.yml`:

- **Timeout**: 5 minutes
- **Enabled linters**:
  - errcheck (check for unhandled errors)
  - gosimple (code simplification)
  - govet (static analysis)
  - ineffassign (unused assignments)
  - staticcheck (extended static analysis)
  - gofmt/goimports (formatting)
  - revive (code style)
  - misspell (spelling in comments)

## Common Issues and Solutions

### 1. errcheck Errors

```go
// Bad
file.Close()

// Good
if err := file.Close(); err != nil {
    log.Printf("Failed to close file: %v", err)
}

// Or if error is not critical
_ = file.Close()
```

### 2. Missing Comments for Exported Functions

```go
// Bad
func ProcessImage() {}

// Good
// ProcessImage processes the uploaded image and generates metadata
func ProcessImage() {}
```

### 3. Import Issues

```bash
# Auto-fix imports
make fmt

# Or manually
goimports -w -local github.com/shabohin/photo-tags .
```

### 4. Shadow Variables

```go
// Bad
if err != nil {
    client, err := NewClient() // variable err is shadowed
}

// Good
if err != nil {
    client, clientErr := NewClient()
    if clientErr != nil {
        // handle error
    }
}
```

## CI/CD Integration

GitHub Actions automatically runs linting for each PR:
- Checks all modules
- Blocks merge if errors found
- Shows results in GitHub UI

## Useful Commands

```bash
# Show all available linters
golangci-lint linters

# Run only specific linters
golangci-lint run --enable=errcheck,govet,staticcheck

# Exclude specific files
golangci-lint run --skip-files=".*_test.go"

# Show version
golangci-lint version

# Help
golangci-lint help
```

## Performance Tips

1. **Use cache**: golangci-lint automatically caches results
2. **Run only changed files**: `golangci-lint run --new-from-rev=HEAD~1`
3. **Configure exclusions**: add rules in `.golangci.yml` for known issues
4. **Use fast mode**: `golangci-lint run --fast` (for development)

## Updates

```bash
# Check current version
golangci-lint version

# Update to latest version
./scripts/install-golangci-lint.sh
```

---

**Useful Links:**
- [Official golangci-lint Documentation](https://golangci-lint.run/)
- [List of All Linters](https://golangci-lint.run/usage/linters/)
- [Configuration Guide](https://golangci-lint.run/usage/configuration/)
