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

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/config/.env.local"
PID_DIR="${PROJECT_ROOT}/tmp/pids"
LOG_DIR="${PROJECT_ROOT}/logs"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Create necessary directories
create_directories() {
    mkdir -p "$PID_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p ~/photo-tags/data/minio
    mkdir -p ~/photo-tags/tmp/processor
}

# Load configuration
load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        log_info "Loading configuration from $CONFIG_FILE"
        set -a
        source "$CONFIG_FILE"
        set +a
    else
        log_warn "Configuration file not found: $CONFIG_FILE"
        log_info "Using default configuration for localhost"
    fi
}

# Start RabbitMQ
start_rabbitmq() {
    log_info "Starting RabbitMQ..."

    if pgrep -f rabbitmq-server > /dev/null; then
        log_info "RabbitMQ is already running"
        return
    fi

    if [[ "$OS" == "darwin" ]]; then
        # macOS - start RabbitMQ with Homebrew
        brew services start rabbitmq
    else
        # Linux - start RabbitMQ with systemd
        if command_exists systemctl; then
            sudo systemctl start rabbitmq-server
        else
            # Fallback to manual start
            rabbitmq-server -detached
        fi
    fi

    # Wait for RabbitMQ to be ready
    log_info "Waiting for RabbitMQ to be ready..."
    local max_attempts=30
    local attempt=0
    while ! rabbitmqctl status >/dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            log_error "RabbitMQ failed to start"
            return 1
        fi
        sleep 1
    done

    log_success "RabbitMQ started"
}

# Start MinIO
start_minio() {
    log_info "Starting MinIO..."

    if pgrep -f "minio server" > /dev/null; then
        log_info "MinIO is already running"
        return
    fi

    local MINIO_ROOT_USER="${MINIO_ROOT_USER:-minioadmin}"
    local MINIO_ROOT_PASSWORD="${MINIO_ROOT_PASSWORD:-minioadmin}"
    local MINIO_DATA_DIR="${MINIO_DATA_DIR:-$HOME/photo-tags/data/minio}"

    if [[ "$OS" == "darwin" ]]; then
        # macOS
        MINIO_ROOT_USER="$MINIO_ROOT_USER" \
        MINIO_ROOT_PASSWORD="$MINIO_ROOT_PASSWORD" \
        minio server "$MINIO_DATA_DIR" --console-address ":9001" \
            > "$LOG_DIR/minio.log" 2>&1 &
        echo $! > "$PID_DIR/minio.pid"
    else
        # Linux
        if command_exists systemctl && systemctl list-unit-files | grep -q minio.service; then
            sudo systemctl start minio
        else
            # Fallback to manual start
            MINIO_ROOT_USER="$MINIO_ROOT_USER" \
            MINIO_ROOT_PASSWORD="$MINIO_ROOT_PASSWORD" \
            nohup minio server "$MINIO_DATA_DIR" --console-address ":9001" \
                > "$LOG_DIR/minio.log" 2>&1 &
            echo $! > "$PID_DIR/minio.pid"
        fi
    fi

    # Wait for MinIO to be ready
    log_info "Waiting for MinIO to be ready..."
    local max_attempts=30
    local attempt=0
    while ! curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1; do
        attempt=$((attempt + 1))
        if [ $attempt -ge $max_attempts ]; then
            log_error "MinIO failed to start"
            return 1
        fi
        sleep 1
    done

    log_success "MinIO started"

    # Configure MinIO
    configure_minio
}

# Configure MinIO
configure_minio() {
    log_info "Configuring MinIO..."

    local MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
    local MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"

    # Set up mc alias
    mc alias set local http://localhost:9000 "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY" > /dev/null 2>&1 || true

    # Create buckets
    mc mb local/original --ignore-existing > /dev/null 2>&1 || true
    mc mb local/processed --ignore-existing > /dev/null 2>&1 || true

    log_success "MinIO configured (buckets: original, processed)"
}

# Build services
build_services() {
    log_info "Building services..."

    cd "$PROJECT_ROOT"

    # Build all services
    if [[ -f "$PROJECT_ROOT/Makefile" ]]; then
        make build
    else
        # Manual build
        cd "$PROJECT_ROOT/services/gateway" && go build -o "$PROJECT_ROOT/bin/gateway" cmd/main.go
        cd "$PROJECT_ROOT/services/analyzer" && go build -o "$PROJECT_ROOT/bin/analyzer" cmd/main.go
        cd "$PROJECT_ROOT/services/processor" && go build -o "$PROJECT_ROOT/bin/processor" cmd/main.go
    fi

    log_success "Services built"
}

# Start Gateway service
start_gateway() {
    log_info "Starting Gateway service..."

    export RABBITMQ_URL="${RABBITMQ_URL:-amqp://user:password@localhost:5672/}"
    export MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
    export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
    export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
    export MINIO_USE_SSL="${MINIO_USE_SSL:-false}"
    export SERVER_PORT="${SERVER_PORT:-8080}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export LOG_FORMAT="${LOG_FORMAT:-text}"

    "$PROJECT_ROOT/bin/gateway" > "$LOG_DIR/gateway.log" 2>&1 &
    echo $! > "$PID_DIR/gateway.pid"

    log_success "Gateway started (PID: $(cat $PID_DIR/gateway.pid))"
}

# Start Analyzer service
start_analyzer() {
    log_info "Starting Analyzer service..."

    export RABBITMQ_URL="${RABBITMQ_URL:-amqp://user:password@localhost:5672/}"
    export RABBITMQ_CONSUMER_QUEUE="${RABBITMQ_CONSUMER_QUEUE:-image_upload}"
    export RABBITMQ_PUBLISHER_QUEUE="${RABBITMQ_PUBLISHER_QUEUE:-metadata_generated}"
    export MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
    export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
    export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
    export MINIO_USE_SSL="${MINIO_USE_SSL:-false}"
    export MINIO_ORIGINAL_BUCKET="${MINIO_ORIGINAL_BUCKET:-original}"
    export OPENROUTER_API_KEY="${OPENROUTER_API_KEY:-}"
    export OPENROUTER_MODEL="${OPENROUTER_MODEL:-openai/gpt-4o}"
    export WORKER_CONCURRENCY="${WORKER_CONCURRENCY:-3}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export LOG_FORMAT="${LOG_FORMAT:-text}"

    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        log_error "OPENROUTER_API_KEY is not set in $CONFIG_FILE"
        log_error "Analyzer will not start without API key"
        return 1
    fi

    "$PROJECT_ROOT/bin/analyzer" > "$LOG_DIR/analyzer.log" 2>&1 &
    echo $! > "$PID_DIR/analyzer.pid"

    log_success "Analyzer started (PID: $(cat $PID_DIR/analyzer.pid))"
}

# Start Processor service
start_processor() {
    log_info "Starting Processor service..."

    export RABBITMQ_URL="${RABBITMQ_URL:-amqp://user:password@localhost:5672/}"
    export RABBITMQ_CONSUMER_QUEUE="${RABBITMQ_CONSUMER_QUEUE:-metadata_generated}"
    export RABBITMQ_PUBLISHER_QUEUE="${RABBITMQ_PUBLISHER_QUEUE:-image_processed}"
    export MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
    export MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
    export MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
    export MINIO_USE_SSL="${MINIO_USE_SSL:-false}"
    export MINIO_ORIGINAL_BUCKET="${MINIO_ORIGINAL_BUCKET:-original}"
    export MINIO_PROCESSED_BUCKET="${MINIO_PROCESSED_BUCKET:-processed}"
    export EXIFTOOL_BINARY_PATH="${EXIFTOOL_BINARY_PATH:-$(which exiftool)}"
    export EXIFTOOL_TEMP_DIR="${EXIFTOOL_TEMP_DIR:-$HOME/photo-tags/tmp/processor}"
    export WORKER_CONCURRENCY="${WORKER_CONCURRENCY:-3}"
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export LOG_FORMAT="${LOG_FORMAT:-text}"

    "$PROJECT_ROOT/bin/processor" > "$LOG_DIR/processor.log" 2>&1 &
    echo $! > "$PID_DIR/processor.pid"

    log_success "Processor started (PID: $(cat $PID_DIR/processor.pid))"
}

# Stop all services
stop_all() {
    log_info "Stopping all services..."

    # Stop application services
    for service in gateway analyzer processor; do
        if [[ -f "$PID_DIR/$service.pid" ]]; then
            local pid=$(cat "$PID_DIR/$service.pid")
            if kill -0 "$pid" 2>/dev/null; then
                log_info "Stopping $service (PID: $pid)..."
                kill "$pid"
                rm "$PID_DIR/$service.pid"
            fi
        fi
    done

    # Stop MinIO
    if [[ -f "$PID_DIR/minio.pid" ]]; then
        local pid=$(cat "$PID_DIR/minio.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping MinIO..."
            kill "$pid"
            rm "$PID_DIR/minio.pid"
        fi
    elif command_exists systemctl && systemctl list-unit-files | grep -q minio.service; then
        sudo systemctl stop minio
    fi

    # Stop RabbitMQ
    if [[ "$OS" == "darwin" ]]; then
        brew services stop rabbitmq
    elif command_exists systemctl; then
        sudo systemctl stop rabbitmq-server
    else
        rabbitmqctl stop
    fi

    log_success "All services stopped"
}

# Show status
show_status() {
    log_info "Service Status:"
    echo ""

    # Check RabbitMQ
    if pgrep -f rabbitmq-server > /dev/null; then
        echo -e "  ${GREEN}●${NC} RabbitMQ       - running"
    else
        echo -e "  ${RED}●${NC} RabbitMQ       - stopped"
    fi

    # Check MinIO
    if pgrep -f "minio server" > /dev/null; then
        echo -e "  ${GREEN}●${NC} MinIO          - running"
    else
        echo -e "  ${RED}●${NC} MinIO          - stopped"
    fi

    # Check application services
    for service in gateway analyzer processor; do
        if [[ -f "$PID_DIR/$service.pid" ]]; then
            local pid=$(cat "$PID_DIR/$service.pid")
            if kill -0 "$pid" 2>/dev/null; then
                echo -e "  ${GREEN}●${NC} $(printf '%-14s' $service) - running (PID: $pid)"
            else
                echo -e "  ${RED}●${NC} $(printf '%-14s' $service) - stopped (stale PID file)"
                rm "$PID_DIR/$service.pid"
            fi
        else
            echo -e "  ${RED}●${NC} $(printf '%-14s' $service) - stopped"
        fi
    done

    echo ""
    log_info "Service URLs:"
    echo "  Gateway:          http://localhost:8080"
    echo "  RabbitMQ UI:      http://localhost:15672 (user/password)"
    echo "  MinIO Console:    http://localhost:9001 (minioadmin/minioadmin)"
    echo ""
    log_info "Logs directory: $LOG_DIR"
}

# Tail logs
tail_logs() {
    log_info "Tailing all service logs... (Ctrl+C to exit)"
    echo ""

    tail -f "$LOG_DIR"/*.log 2>/dev/null || {
        log_warn "No log files found"
        exit 1
    }
}

# Main function
main() {
    case "${1:-start}" in
        start)
            log_info "Starting photo-tags services locally..."
            echo ""

            create_directories
            load_config

            start_rabbitmq
            echo ""

            start_minio
            echo ""

            # Check if binaries exist, build if not
            if [[ ! -f "$PROJECT_ROOT/bin/gateway" ]] || \
               [[ ! -f "$PROJECT_ROOT/bin/analyzer" ]] || \
               [[ ! -f "$PROJECT_ROOT/bin/processor" ]]; then
                build_services
                echo ""
            fi

            start_gateway
            sleep 2

            start_analyzer
            sleep 2

            start_processor
            echo ""

            log_success "All services started!"
            echo ""
            show_status
            ;;

        stop)
            stop_all
            ;;

        restart)
            stop_all
            echo ""
            sleep 2
            main start
            ;;

        status)
            show_status
            ;;

        logs)
            tail_logs
            ;;

        build)
            build_services
            ;;

        *)
            echo "Usage: $0 {start|stop|restart|status|logs|build}"
            echo ""
            echo "Commands:"
            echo "  start    - Start all services"
            echo "  stop     - Stop all services"
            echo "  restart  - Restart all services"
            echo "  status   - Show service status"
            echo "  logs     - Tail all service logs"
            echo "  build    - Build all services"
            exit 1
            ;;
    esac
}

main "$@"
