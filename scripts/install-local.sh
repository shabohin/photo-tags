#!/usr/bin/env bash

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and Architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin)
            OS="macos"
            ;;
        linux)
            OS="linux"
            ;;
        *)
            log_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    log_info "Detected platform: $OS $ARCH"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install Homebrew on macOS
install_homebrew() {
    if [[ "$OS" != "macos" ]]; then
        return
    fi

    if ! command_exists brew; then
        log_info "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        log_success "Homebrew installed"
    else
        log_info "Homebrew already installed"
    fi
}

# Install Go
install_go() {
    if command_exists go; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go $GO_VERSION already installed"
        return
    fi

    log_info "Installing Go..."

    if [[ "$OS" == "macos" ]]; then
        brew install go
    else
        # Install Go on Linux
        GO_VERSION="1.22.0"
        GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"

        cd /tmp
        curl -LO "https://go.dev/dl/${GO_TARBALL}"
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "${GO_TARBALL}"
        rm "${GO_TARBALL}"

        # Add to PATH if not already there
        if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
        fi

        export PATH=$PATH:/usr/local/go/bin
    fi

    log_success "Go installed: $(go version)"
}

# Install ExifTool
install_exiftool() {
    if command_exists exiftool; then
        log_info "ExifTool already installed: $(exiftool -ver)"
        return
    fi

    log_info "Installing ExifTool..."

    if [[ "$OS" == "macos" ]]; then
        brew install exiftool
    else
        # Install ExifTool on Linux
        if command_exists apt-get; then
            sudo apt-get update
            sudo apt-get install -y libimage-exiftool-perl
        elif command_exists yum; then
            sudo yum install -y perl-Image-ExifTool
        else
            log_warn "Package manager not found, installing from source..."
            cd /tmp
            curl -LO https://exiftool.org/Image-ExifTool-12.76.tar.gz
            tar -xzf Image-ExifTool-12.76.tar.gz
            cd Image-ExifTool-12.76
            perl Makefile.PL
            make
            sudo make install
            cd ..
            rm -rf Image-ExifTool-12.76*
        fi
    fi

    log_success "ExifTool installed: $(exiftool -ver)"
}

# Install RabbitMQ
install_rabbitmq() {
    if command_exists rabbitmq-server; then
        log_info "RabbitMQ already installed"
        return
    fi

    log_info "Installing RabbitMQ..."

    if [[ "$OS" == "macos" ]]; then
        brew install rabbitmq

        # Add to PATH
        if ! grep -q "/usr/local/sbin" ~/.zshrc 2>/dev/null && ! grep -q "/usr/local/sbin" ~/.bashrc 2>/dev/null; then
            echo 'export PATH=$PATH:/usr/local/sbin' >> ~/.zshrc
            echo 'export PATH=$PATH:/usr/local/sbin' >> ~/.bashrc
        fi
        export PATH=$PATH:/usr/local/sbin
    else
        # Install RabbitMQ on Linux
        if command_exists apt-get; then
            # Add RabbitMQ repository
            sudo apt-get update
            sudo apt-get install -y curl gnupg apt-transport-https

            # Import signing key
            curl -fsSL https://github.com/rabbitmq/signing-keys/releases/download/2.0/rabbitmq-release-signing-key.asc | sudo apt-key add -

            # Add repository
            sudo tee /etc/apt/sources.list.d/rabbitmq.list <<EOF
deb https://dl.cloudsmith.io/public/rabbitmq/rabbitmq-server/deb/ubuntu $(lsb_release -cs) main
deb-src https://dl.cloudsmith.io/public/rabbitmq/rabbitmq-server/deb/ubuntu $(lsb_release -cs) main
EOF

            sudo apt-get update
            sudo apt-get install -y rabbitmq-server

            # Enable and start RabbitMQ
            sudo systemctl enable rabbitmq-server
            sudo systemctl start rabbitmq-server

            # Enable management plugin
            sudo rabbitmq-plugins enable rabbitmq_management
        else
            log_error "Unsupported package manager for RabbitMQ installation"
            log_info "Please install RabbitMQ manually from https://www.rabbitmq.com/download.html"
            return 1
        fi
    fi

    log_success "RabbitMQ installed"
}

# Install MinIO
install_minio() {
    if command_exists minio; then
        log_info "MinIO already installed"
        return
    fi

    log_info "Installing MinIO..."

    if [[ "$OS" == "macos" ]]; then
        brew install minio/stable/minio
        brew install minio/stable/mc
    else
        # Install MinIO on Linux
        cd /tmp

        # Download MinIO server
        curl -LO "https://dl.min.io/server/minio/release/${OS}-${ARCH}/minio"
        chmod +x minio
        sudo mv minio /usr/local/bin/

        # Download MinIO client (mc)
        curl -LO "https://dl.min.io/client/mc/release/${OS}-${ARCH}/mc"
        chmod +x mc
        sudo mv mc /usr/local/bin/

        # Create MinIO data directory
        sudo mkdir -p /usr/local/share/minio
        sudo chown $(whoami) /usr/local/share/minio

        # Create systemd service file
        if command_exists systemctl; then
            sudo tee /etc/systemd/system/minio.service > /dev/null <<EOF
[Unit]
Description=MinIO
Documentation=https://docs.min.io
Wants=network-online.target
After=network-online.target

[Service]
User=$(whoami)
Group=$(id -gn)
WorkingDirectory=/usr/local/share/minio

ExecStart=/usr/local/bin/minio server /usr/local/share/minio --console-address ":9001"

# Let systemd restart this service always
Restart=always

# Specifies the maximum file descriptor number that can be opened by this process
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

            sudo systemctl daemon-reload
            sudo systemctl enable minio
        fi
    fi

    log_success "MinIO installed"
}

# Create necessary directories
create_directories() {
    log_info "Creating necessary directories..."

    mkdir -p ~/photo-tags/data/minio
    mkdir -p ~/photo-tags/data/rabbitmq
    mkdir -p ~/photo-tags/logs
    mkdir -p ~/photo-tags/tmp/processor

    log_success "Directories created"
}

# Setup RabbitMQ user
setup_rabbitmq() {
    log_info "Setting up RabbitMQ..."

    # Wait for RabbitMQ to start
    sleep 5

    if [[ "$OS" == "macos" ]]; then
        # Enable management plugin
        rabbitmq-plugins enable rabbitmq_management || true

        # Create user (if not exists)
        rabbitmqctl add_user user password 2>/dev/null || true
        rabbitmqctl set_user_tags user administrator
        rabbitmqctl set_permissions -p / user ".*" ".*" ".*"
    fi

    log_success "RabbitMQ configured"
}

# Main installation function
main() {
    log_info "Starting local installation for photo-tags..."
    echo ""

    detect_platform
    echo ""

    # Install dependencies based on platform
    if [[ "$OS" == "macos" ]]; then
        install_homebrew
        echo ""
    fi

    install_go
    echo ""

    install_exiftool
    echo ""

    install_rabbitmq
    echo ""

    install_minio
    echo ""

    create_directories
    echo ""

    # Build the services
    log_info "Building services..."
    cd "$(dirname "$0")/.."
    make build || {
        log_warn "Make build failed, trying manual build..."
        cd services/gateway && go build -o ../../bin/gateway cmd/main.go && cd ../..
        cd services/analyzer && go build -o ../../bin/analyzer cmd/main.go && cd ../..
        cd services/processor && go build -o ../../bin/processor cmd/main.go && cd ../..
    }
    log_success "Services built"
    echo ""

    log_success "Installation completed!"
    echo ""
    log_info "Next steps:"
    log_info "  1. Copy config/.env.local.example to config/.env.local and configure"
    log_info "  2. Start services with: ./scripts/run-local.sh"
    log_info "  3. Access RabbitMQ management UI at: http://localhost:15672 (user/password)"
    log_info "  4. Access MinIO console at: http://localhost:9001 (minioadmin/minioadmin)"
    echo ""
}

main "$@"
