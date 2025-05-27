# golangci-lint v2 Guide for Photo Tags Service

## What's New in v2

golangci-lint v2 introduces significant changes:
- **New configuration structure** with `version: "2"`
- **Simplified linter management** with `linters.default`
- **Built-in formatting** with `golangci-lint fmt` command
- **Better exclusion system** with presets
- **Migration command** to upgrade from v1

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
   # Latest v2 version
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.1.6
   
   # Via Homebrew (macOS)
   brew install golangci-lint
   ```

## Migration from v1

If you have an existing v1 configuration:

```bash
# Automatic migration
make migrate-config

# Or manually
golangci-lint migrate --format yaml
```

## Usage

### Basic Commands

```bash
# Run linter
make lint

# Format code (new in v2!)
make fmt
golangci-lint fmt

# Run linter with auto-fix
make lint-fix

# Full pre-commit check
make pre-commit

# Run all quality checks
make check
```

### New v2 Features

#### 1. Built-in Formatting
```bash
# Format code with configured formatters
golangci-lint fmt

# Format specific files
golangci-lint fmt ./services/gateway/...
```

#### 2. Improved Configuration
```yaml
# v2 configuration structure
version: "2"

linters:
  default: standard  # or 'all', 'none', 'fast'
  enable:
    - misspell
    - shadow

formatters:
  enable:
    - gofmt
    - goimports
```

#### 3. Better Exclusions
```yaml
linters:
  exclusions:
    presets:
      - comments
      - std-error-handling
      - common-false-positives
    paths:
      - vendor
      - _test.go
```

## Configuration Structure

### v2 Configuration (.golangci.yml)

```yaml
version: "2"

run:
  timeout: 5m
  go: '1.24'

linters:
  default: standard
  enable:
    - misspell
    - shadow
  settings:
    errcheck:
      check-type-assertions: true
    govet:
      enable:
        - shadow

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes: github.com/shabohin/photo-tags
```

### Key Changes from v1

| v1 | v2 |
|----|-----|
| `linters.enable-all: true` | `linters.default: all` |
| `linters.disable-all: true` | `linters.default: none` |
| `linters-settings:` | `linters.settings:` |
| `issues.exclude-dirs:` | `linters.exclusions.paths:` |
| `govet.check-shadowing: true` | `govet.enable: [shadow]` |

## IDE Integration

### VS Code

Settings are automatically configured in `.vscode/settings.json`:
- Uses golangci-lint for linting and formatting
- Runs on save
- Supports v2 format commands

### GoLand

GoLand 2025.1+ has native golangci-lint v2 support:
1. Go to Settings → Go → Linters
2. Enable golangci-lint
3. Point to your `.golangci.yml` config

## Common Issues and Solutions

### 1. Migration Issues

```bash
# If migration fails
golangci-lint migrate --skip-validation

# Check config validity
golangci-lint config --path .golangci.yml
```

### 2. Performance

```bash
# Use fast linters only
golangci-lint run --fast-only

# Or set in config
linters:
  default: fast
```

### 3. New Exclusion System

```yaml
# v2 exclusions are more powerful
linters:
  exclusions:
    presets:
      - std-error-handling  # Excludes common Go error patterns
      - comments           # Excludes comment-related issues
    rules:
      - path: '_test\.go'
        linters: [errcheck]
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v8
  with:
    version: v2.1.6
    args: --timeout=5m
```

### Performance Tips

1. **Use caching** - GitHub Action automatically caches
2. **Set appropriate timeout** - v2 is faster but complex projects need time
3. **Use exclusions** - Exclude vendor, generated files
4. **Consider fast mode** - For development workflow

## Useful Commands

```bash
# Show enabled linters
golangci-lint linters

# Show enabled formatters  
golangci-lint formatters

# Validate configuration
golangci-lint config --path .golangci.yml

# Run with verbose output
golangci-lint run -v

# Format and lint in one go
golangci-lint fmt && golangci-lint run
```

## Benefits of v2

- **🚀 Better Performance** - Improved caching and parallel execution
- **🎨 Built-in Formatting** - No need for separate formatting tools
- **⚙️ Simpler Configuration** - More intuitive settings structure
- **🔧 Better IDE Integration** - Native support in modern IDEs
- **📦 Preset Exclusions** - Common exclusion patterns built-in
- **🔄 Easy Migration** - Automated migration from v1

---

**Useful Links:**
- [golangci-lint v2 Documentation](https://golangci-lint.run/)
- [Migration Guide](https://golangci-lint.run/product/migration-guide/)
- [Configuration Reference](https://golangci-lint.run/usage/configuration/)
- [GitHub Action v8](https://github.com/golangci/golangci-lint-action)
