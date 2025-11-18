# Local Deployment Guide

This guide provides instructions for deploying the Photo Tags system locally without Docker on macOS and Linux ARM64/x86_64.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Platform-Specific Instructions](#platform-specific-instructions)
  - [macOS](#macos)
  - [Linux (Ubuntu/Debian)](#linux-ubuntudebian)
  - [Linux (Raspberry Pi / ARM64)](#linux-raspberry-pi--arm64)
- [Configuration](#configuration)
- [Running Services](#running-services)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [Comparison: Docker vs Native](#comparison-docker-vs-native)

## Prerequisites

Before installing, ensure you have:

- **macOS**: macOS 10.15 or later
- **Linux**: Ubuntu 20.04+, Debian 11+, or Raspberry Pi OS
- At least 2GB RAM
- 5GB free disk space
- Internet connection for downloading dependencies

## Quick Start

For the impatient, here's the fastest way to get started:

```bash
# 1. Clone the repository (if not already done)
git clone https://github.com/shabohin/photo-tags.git
cd photo-tags

# 2. Install all dependencies
./scripts/install-local.sh

# 3. Configure environment
cp config/.env.local.example config/.env.local
# Edit config/.env.local and set your TELEGRAM_TOKEN and OPENROUTER_API_KEY

# 4. Start all services
./scripts/run-local.sh start

# 5. Check status
./scripts/run-local.sh status

# 6. View logs
./scripts/run-local.sh logs
```

## Platform-Specific Instructions

### macOS

#### Installation

The installation script will automatically use Homebrew to install dependencies.

```bash
# Run the installation script
./scripts/install-local.sh
```

This will install:
- **Go** 1.22+ (via Homebrew)
- **ExifTool** (via Homebrew)
- **RabbitMQ** (via Homebrew)
- **MinIO** (via Homebrew)

#### Starting Services

```bash
# Start all services
./scripts/run-local.sh start
```

#### Stopping Services

```bash
# Stop all services
./scripts/run-local.sh stop
```

#### Service Management

On macOS, RabbitMQ is managed via Homebrew services:

```bash
# Start RabbitMQ manually
brew services start rabbitmq

# Stop RabbitMQ manually
brew services stop rabbitmq

# Restart RabbitMQ
brew services restart rabbitmq
```

MinIO runs as a background process managed by the run-local.sh script.

### Linux (Ubuntu/Debian)

#### Installation

```bash
# Run the installation script
./scripts/install-local.sh
```

This will:
1. Download and install Go 1.22+ to `/usr/local/go`
2. Install ExifTool via apt-get
3. Install RabbitMQ from official repository
4. Install MinIO to `/usr/local/bin`
5. Create systemd service files

#### Service Management

On Linux, services can be managed via systemd:

```bash
# Start services via systemd
sudo systemctl start rabbitmq-server
sudo systemctl start minio

# Enable services to start on boot
sudo systemctl enable rabbitmq-server
sudo systemctl enable minio

# Check service status
sudo systemctl status rabbitmq-server
sudo systemctl status minio
```

Or use the run-local.sh script:

```bash
# Start all services (including application services)
./scripts/run-local.sh start

# Stop all services
./scripts/run-local.sh stop
```

#### Post-Installation

After installation, you may need to reload your shell to update PATH:

```bash
source ~/.bashrc
```

Or open a new terminal session.

### Linux (Raspberry Pi / ARM64)

#### Installation on Raspberry Pi

The installation script automatically detects ARM64 architecture and installs compatible binaries.

```bash
# Run the installation script
./scripts/install-local.sh
```

**Note**: On Raspberry Pi, especially with 1-2GB RAM, you may want to reduce worker concurrency:

```bash
# Edit config/.env.local
WORKER_CONCURRENCY=1
```

#### Performance Considerations

For Raspberry Pi 3/4:
- Use `WORKER_CONCURRENCY=1` or `2`
- Set `LOG_LEVEL=warn` to reduce I/O
- Consider using an external SSD instead of SD card for better performance
- Ensure adequate cooling to prevent thermal throttling

#### Memory Optimization

```bash
# In config/.env.local
WORKER_CONCURRENCY=1
RABBITMQ_PREFETCH_COUNT=1
OPENROUTER_MODEL=openai/gpt-4o-mini  # Use smaller/cheaper model
```

## Configuration

### 1. Copy the Example Configuration

```bash
cp config/.env.local.example config/.env.local
```

### 2. Edit Configuration

Edit `config/.env.local` and set the required values:

#### Required Configuration

```bash
# Telegram Bot Token (get from @BotFather)
TELEGRAM_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz

# OpenRouter API Key (get from https://openrouter.ai/)
OPENROUTER_API_KEY=sk-or-v1-abc123...
```

#### Optional Configuration

```bash
# RabbitMQ (default values work for local installation)
RABBITMQ_URL=amqp://user:password@localhost:5672/

# MinIO (default values work for local installation)
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Worker Settings
WORKER_CONCURRENCY=3  # Adjust based on your CPU

# Logging
LOG_LEVEL=info  # debug, info, warn, error
LOG_FORMAT=text  # text or json
```

### 3. Platform-Specific Configuration

#### macOS (Apple Silicon M1/M2/M3)

```bash
# ExifTool path (Homebrew on Apple Silicon)
EXIFTOOL_BINARY_PATH=/opt/homebrew/bin/exiftool
```

#### macOS (Intel)

```bash
# ExifTool path (Homebrew on Intel)
EXIFTOOL_BINARY_PATH=/usr/local/bin/exiftool
```

#### Linux

```bash
# ExifTool path (standard location)
EXIFTOOL_BINARY_PATH=/usr/bin/exiftool
```

## Running Services

### Start All Services

```bash
./scripts/run-local.sh start
```

This will:
1. Start RabbitMQ
2. Start MinIO
3. Build services (if needed)
4. Start Gateway service
5. Start Analyzer service
6. Start Processor service

### Stop All Services

```bash
./scripts/run-local.sh stop
```

### Restart Services

```bash
./scripts/run-local.sh restart
```

### Check Status

```bash
./scripts/run-local.sh status
```

Example output:
```
Service Status:

  ● RabbitMQ       - running
  ● MinIO          - running
  ● gateway        - running (PID: 12345)
  ● analyzer       - running (PID: 12346)
  ● processor      - running (PID: 12347)

Service URLs:
  Gateway:          http://localhost:8080
  RabbitMQ UI:      http://localhost:15672 (user/password)
  MinIO Console:    http://localhost:9001 (minioadmin/minioadmin)

Logs directory: /path/to/photo-tags/logs
```

### View Logs

```bash
# Tail all service logs
./scripts/run-local.sh logs

# View specific service log
tail -f logs/gateway.log
tail -f logs/analyzer.log
tail -f logs/processor.log
```

### Rebuild Services

```bash
./scripts/run-local.sh build
```

## Service URLs

After starting the services, you can access:

| Service | URL | Credentials |
|---------|-----|-------------|
| Gateway API | http://localhost:8080 | - |
| RabbitMQ Management | http://localhost:15672 | user / password |
| MinIO Console | http://localhost:9001 | minioadmin / minioadmin |

## Troubleshooting

### Services Won't Start

1. Check if ports are already in use:
   ```bash
   # Check port usage
   lsof -i :5672  # RabbitMQ
   lsof -i :15672 # RabbitMQ Management
   lsof -i :9000  # MinIO API
   lsof -i :9001  # MinIO Console
   lsof -i :8080  # Gateway
   ```

2. Check service logs:
   ```bash
   tail -f logs/*.log
   ```

3. Verify configuration:
   ```bash
   cat config/.env.local
   ```

### RabbitMQ Issues

#### Connection Refused

```bash
# Check if RabbitMQ is running
# macOS:
brew services list | grep rabbitmq

# Linux:
sudo systemctl status rabbitmq-server

# Start RabbitMQ manually
# macOS:
brew services start rabbitmq

# Linux:
sudo systemctl start rabbitmq-server
```

#### Create User Manually

```bash
# Add user
sudo rabbitmqctl add_user user password

# Set permissions
sudo rabbitmqctl set_permissions -p / user ".*" ".*" ".*"

# Set user tags
sudo rabbitmqctl set_user_tags user administrator
```

### MinIO Issues

#### MinIO Won't Start

```bash
# Check if data directory exists and is writable
ls -la ~/photo-tags/data/minio

# Create if missing
mkdir -p ~/photo-tags/data/minio

# Start MinIO manually
MINIO_ROOT_USER=minioadmin MINIO_ROOT_PASSWORD=minioadmin \
  minio server ~/photo-tags/data/minio --console-address ":9001"
```

#### Create Buckets Manually

```bash
# Configure mc (MinIO client)
mc alias set local http://localhost:9000 minioadmin minioadmin

# Create buckets
mc mb local/original
mc mb local/processed

# List buckets
mc ls local
```

### ExifTool Not Found

```bash
# Find exiftool location
which exiftool

# macOS (Homebrew):
# Apple Silicon: /opt/homebrew/bin/exiftool
# Intel: /usr/local/bin/exiftool

# Linux:
# Usually: /usr/bin/exiftool

# Update config/.env.local with the correct path
EXIFTOOL_BINARY_PATH=/path/to/exiftool
```

### Go Not Found

```bash
# Check Go installation
go version

# If not found, ensure PATH is set
# Add to ~/.bashrc or ~/.zshrc:
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$HOME/go/bin

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

### Low Memory Issues (Raspberry Pi)

If you experience crashes or OOM (Out of Memory) errors:

```bash
# Reduce worker concurrency in config/.env.local
WORKER_CONCURRENCY=1

# Use a smaller AI model
OPENROUTER_MODEL=openai/gpt-4o-mini

# Reduce RabbitMQ prefetch
RABBITMQ_PREFETCH_COUNT=1

# Consider increasing swap
sudo dphys-swapfile swapoff
sudo nano /etc/dphys-swapfile  # Set CONF_SWAPSIZE=2048
sudo dphys-swapfile setup
sudo dphys-swapfile swapon
```

## Development

### Building Individual Services

```bash
# Build Gateway
cd services/gateway
go build -o ../../bin/gateway cmd/main.go

# Build Analyzer
cd services/analyzer
go build -o ../../bin/analyzer cmd/main.go

# Build Processor
cd services/processor
go build -o ../../bin/processor cmd/main.go
```

### Running Individual Services

```bash
# Source configuration
set -a
source config/.env.local
set +a

# Run Gateway
./bin/gateway

# Run Analyzer
./bin/analyzer

# Run Processor
./bin/processor
```

### Development Mode

For development, you can use text logging instead of JSON:

```bash
# In config/.env.local
LOG_LEVEL=debug
LOG_FORMAT=text
```

## Comparison: Docker vs Native

### When to Use Docker

✅ **Pros:**
- Easier setup and consistent environment
- Isolated dependencies
- Easier to replicate production environment
- Better for development on teams

❌ **Cons:**
- Higher resource usage (especially on ARM)
- Additional layer of complexity
- May have performance overhead

### When to Use Native Deployment

✅ **Pros:**
- Better performance (especially on Raspberry Pi)
- Lower memory usage
- Direct access to system resources
- Easier debugging

❌ **Cons:**
- Platform-specific setup
- Dependency management
- Potential conflicts with system packages

### Recommendation

| Platform | Recommendation | Reason |
|----------|---------------|---------|
| **Raspberry Pi 3/4** | Native | Better performance, lower memory usage |
| **macOS (Development)** | Docker or Native | Either works well, choose based on preference |
| **Linux Server (Production)** | Docker | Easier deployment and isolation |
| **Limited RAM (<4GB)** | Native | Lower memory footprint |

## File Locations

### Data Directories

```bash
~/photo-tags/data/minio      # MinIO data
~/photo-tags/tmp/processor   # Temporary processing files
```

### Log Files

```bash
logs/gateway.log     # Gateway service logs
logs/analyzer.log    # Analyzer service logs
logs/processor.log   # Processor service logs
logs/minio.log       # MinIO logs
```

### PID Files

```bash
tmp/pids/gateway.pid    # Gateway process ID
tmp/pids/analyzer.pid   # Analyzer process ID
tmp/pids/processor.pid  # Processor process ID
tmp/pids/minio.pid      # MinIO process ID
```

### Binaries

```bash
bin/gateway     # Gateway binary
bin/analyzer    # Analyzer binary
bin/processor   # Processor binary
```

## Security Considerations

### For Production Use

If deploying to a production environment, consider:

1. **Change Default Credentials**:
   ```bash
   # In config/.env.local
   RABBITMQ_URL=amqp://custom_user:strong_password@localhost:5672/
   MINIO_ACCESS_KEY=custom_access_key
   MINIO_SECRET_KEY=strong_secret_key
   ```

2. **Enable TLS/SSL**:
   ```bash
   MINIO_USE_SSL=true
   ```

3. **Firewall Rules**:
   ```bash
   # Only allow localhost connections
   sudo ufw allow from 127.0.0.1 to any port 5672
   sudo ufw allow from 127.0.0.1 to any port 9000
   ```

4. **Use Environment-Specific Configs**:
   - Development: `config/.env.local`
   - Production: `config/.env.production` (not in git)

## Uninstalling

To remove all installed components:

### macOS

```bash
# Stop services
./scripts/run-local.sh stop

# Remove Homebrew packages
brew uninstall rabbitmq minio exiftool go

# Remove data directories
rm -rf ~/photo-tags/data
rm -rf ~/photo-tags/logs
rm -rf ~/photo-tags/tmp
```

### Linux

```bash
# Stop services
./scripts/run-local.sh stop

# Remove packages (Ubuntu/Debian)
sudo apt-get remove --purge rabbitmq-server libimage-exiftool-perl

# Remove MinIO and Go
sudo rm /usr/local/bin/minio
sudo rm /usr/local/bin/mc
sudo rm -rf /usr/local/go

# Remove systemd service
sudo systemctl disable minio
sudo rm /etc/systemd/system/minio.service
sudo systemctl daemon-reload

# Remove data directories
rm -rf ~/photo-tags/data
rm -rf ~/photo-tags/logs
rm -rf ~/photo-tags/tmp
sudo rm -rf /usr/local/share/minio
```

## Additional Resources

- [RabbitMQ Documentation](https://www.rabbitmq.com/documentation.html)
- [MinIO Documentation](https://min.io/docs/minio/linux/index.html)
- [ExifTool Documentation](https://exiftool.org/)
- [Go Documentation](https://go.dev/doc/)

## Getting Help

If you encounter issues:

1. Check the [Troubleshooting](#troubleshooting) section
2. Review service logs: `./scripts/run-local.sh logs`
3. Check service status: `./scripts/run-local.sh status`
4. Open an issue on GitHub with:
   - Your platform (macOS/Linux, architecture)
   - Error messages from logs
   - Steps to reproduce the issue
