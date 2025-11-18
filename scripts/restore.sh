#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"

# Docker container names
MINIO_CONTAINER="minio"
RABBITMQ_CONTAINER="rabbitmq"

# MinIO configuration
MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"

# RabbitMQ configuration
RABBITMQ_HOST="${RABBITMQ_HOST:-localhost}"
RABBITMQ_PORT="${RABBITMQ_PORT:-15672}"
RABBITMQ_USER="${RABBITMQ_USER:-user}"
RABBITMQ_PASSWORD="${RABBITMQ_PASSWORD:-password}"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_prompt() {
    echo -e "${YELLOW}[PROMPT]${NC} $1"
}

# Check if docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running or you don't have permissions"
        exit 1
    fi
}

# Check if containers are running
check_containers() {
    log_info "Checking if containers are running..."

    if ! docker ps --format '{{.Names}}' | grep -q "^${MINIO_CONTAINER}$"; then
        log_error "MinIO container '${MINIO_CONTAINER}' is not running"
        exit 1
    fi

    if ! docker ps --format '{{.Names}}' | grep -q "^${RABBITMQ_CONTAINER}$"; then
        log_error "RabbitMQ container '${RABBITMQ_CONTAINER}' is not running"
        exit 1
    fi

    log_success "All required containers are running"
}

# List available backups
list_backups() {
    log_info "Available backups:"

    if [ ! -d "${BACKUP_DIR}" ] || [ -z "$(ls -A "${BACKUP_DIR}"/backup_*.tar.gz 2>/dev/null)" ]; then
        log_error "No backups found in ${BACKUP_DIR}"
        exit 1
    fi

    local index=1
    declare -g -A BACKUP_MAP

    while IFS= read -r backup; do
        local backup_name=$(basename "${backup}" .tar.gz)
        local backup_date=$(echo "${backup_name}" | sed 's/backup_//' | sed 's/_/ /')
        local backup_size=$(du -h "${backup}" | cut -f1)
        local backup_time=$(stat -c %y "${backup}" 2>/dev/null || stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "${backup}" 2>/dev/null)

        echo "  ${index}. ${backup_name} (${backup_size}) - Created: ${backup_time}"
        BACKUP_MAP[$index]="${backup}"
        ((index++))
    done < <(find "${BACKUP_DIR}" -name "backup_*.tar.gz" -type f | sort -r)

    echo ""
}

# Select backup
select_backup() {
    if [ -n "$1" ]; then
        SELECTED_BACKUP="$1"
        if [ ! -f "${SELECTED_BACKUP}" ]; then
            log_error "Backup file not found: ${SELECTED_BACKUP}"
            exit 1
        fi
    else
        list_backups

        read -p "Select backup number (or 'q' to quit): " selection

        if [ "${selection}" = "q" ]; then
            log_info "Restore cancelled"
            exit 0
        fi

        if [ -z "${BACKUP_MAP[$selection]}" ]; then
            log_error "Invalid selection"
            exit 1
        fi

        SELECTED_BACKUP="${BACKUP_MAP[$selection]}"
    fi

    log_info "Selected backup: $(basename "${SELECTED_BACKUP}")"
}

# Extract backup
extract_backup() {
    log_info "Extracting backup..."

    RESTORE_DIR="/tmp/restore_$(date +%s)"
    mkdir -p "${RESTORE_DIR}"

    tar -xzf "${SELECTED_BACKUP}" -C "${RESTORE_DIR}" || {
        log_error "Failed to extract backup"
        rm -rf "${RESTORE_DIR}"
        exit 1
    }

    # Find the extracted backup directory
    BACKUP_CONTENT_DIR=$(find "${RESTORE_DIR}" -maxdepth 1 -type d -name "backup_*" | head -n 1)

    if [ -z "${BACKUP_CONTENT_DIR}" ]; then
        log_error "Invalid backup structure"
        rm -rf "${RESTORE_DIR}"
        exit 1
    fi

    log_success "Backup extracted to: ${RESTORE_DIR}"
}

# Show backup info
show_backup_info() {
    if [ -f "${BACKUP_CONTENT_DIR}/backup.info" ]; then
        log_info "Backup information:"
        cat "${BACKUP_CONTENT_DIR}/backup.info" | while read -r line; do
            echo "  ${line}"
        done
        echo ""
    fi
}

# Restore MinIO buckets
restore_minio() {
    log_info "Starting MinIO restore..."

    if [ ! -d "${BACKUP_CONTENT_DIR}/minio" ]; then
        log_warning "No MinIO backup found, skipping..."
        return 0
    fi

    # Check if mc (MinIO Client) is available in the container
    if ! docker exec "${MINIO_CONTAINER}" which mc &> /dev/null; then
        log_warning "MinIO Client (mc) not found in container, installing..."
        docker exec "${MINIO_CONTAINER}" sh -c "wget -q https://dl.min.io/client/mc/release/linux-amd64/mc -O /usr/local/bin/mc && chmod +x /usr/local/bin/mc" || {
            log_error "Failed to install MinIO Client"
            return 1
        }
    fi

    # Configure mc alias
    log_info "Configuring MinIO Client..."
    docker exec "${MINIO_CONTAINER}" mc alias set restore http://localhost:9000 "${MINIO_ACCESS_KEY}" "${MINIO_SECRET_KEY}" --api S3v4 || {
        log_error "Failed to configure MinIO Client"
        return 1
    }

    # Restore each bucket
    for bucket_dir in "${BACKUP_CONTENT_DIR}/minio"/*; do
        if [ ! -d "${bucket_dir}" ]; then
            continue
        fi

        local bucket=$(basename "${bucket_dir}")
        log_info "Restoring bucket: ${bucket}"

        # Check if bucket has any files
        if [ -z "$(ls -A "${bucket_dir}" 2>/dev/null)" ]; then
            log_warning "Bucket ${bucket} is empty, skipping..."
            continue
        fi

        # Create bucket if it doesn't exist
        docker exec "${MINIO_CONTAINER}" mc mb "restore/${bucket}" --ignore-existing 2>/dev/null || true

        # Copy data to container temp directory
        docker exec "${MINIO_CONTAINER}" mkdir -p "/tmp/restore_${bucket}" || true
        docker cp "${bucket_dir}/." "${MINIO_CONTAINER}:/tmp/restore_${bucket}/" || {
            log_error "Failed to copy bucket data to container"
            return 1
        }

        # Restore bucket using mc mirror
        docker exec "${MINIO_CONTAINER}" mc mirror --overwrite "/tmp/restore_${bucket}/" "restore/${bucket}" || {
            log_error "Failed to restore bucket ${bucket}"
            return 1
        }

        # Cleanup temp directory in container
        docker exec "${MINIO_CONTAINER}" rm -rf "/tmp/restore_${bucket}" || true

        log_success "Bucket ${bucket} restored"
    done

    log_success "MinIO restore completed"
}

# Restore RabbitMQ
restore_rabbitmq() {
    log_info "Starting RabbitMQ restore..."

    if [ ! -d "${BACKUP_CONTENT_DIR}/rabbitmq" ]; then
        log_warning "No RabbitMQ backup found, skipping..."
        return 0
    fi

    # Check if definitions file exists
    if [ ! -f "${BACKUP_CONTENT_DIR}/rabbitmq/definitions.json" ]; then
        log_error "RabbitMQ definitions file not found"
        return 1
    fi

    # Import definitions
    log_info "Importing RabbitMQ definitions..."
    curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASSWORD}" \
        -H "Content-Type: application/json" \
        -X POST \
        --data-binary "@${BACKUP_CONTENT_DIR}/rabbitmq/definitions.json" \
        "http://${RABBITMQ_HOST}:${RABBITMQ_PORT}/api/definitions" || {
        log_error "Failed to import RabbitMQ definitions"
        return 1
    }

    log_success "RabbitMQ restore completed"
}

# Cleanup temporary files
cleanup() {
    if [ -n "${RESTORE_DIR}" ] && [ -d "${RESTORE_DIR}" ]; then
        log_info "Cleaning up temporary files..."
        rm -rf "${RESTORE_DIR}"
        log_success "Cleanup completed"
    fi
}

# Confirm restore
confirm_restore() {
    log_warning "=========================================="
    log_warning "WARNING: This will overwrite existing data!"
    log_warning "=========================================="
    show_backup_info

    if [ "${FORCE_RESTORE}" = "true" ]; then
        log_info "Force restore enabled, skipping confirmation"
        return 0
    fi

    read -p "Are you sure you want to continue? (yes/no): " confirmation

    if [ "${confirmation}" != "yes" ]; then
        log_info "Restore cancelled"
        exit 0
    fi
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS] [BACKUP_FILE]

Restore MinIO and RabbitMQ from backup.

OPTIONS:
    -f, --force         Force restore without confirmation
    -b, --backup-dir    Backup directory (default: ./backups)
    -h, --help          Show this help message

EXAMPLES:
    # Interactive mode - select from available backups
    $0

    # Restore specific backup
    $0 /path/to/backup_20240101_120000.tar.gz

    # Force restore without confirmation
    $0 -f backup_20240101_120000.tar.gz

EOF
}

# Parse arguments
parse_args() {
    FORCE_RESTORE="false"

    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--force)
                FORCE_RESTORE="true"
                shift
                ;;
            -b|--backup-dir)
                BACKUP_DIR="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                if [ -z "${BACKUP_FILE}" ]; then
                    BACKUP_FILE="$1"
                fi
                shift
                ;;
        esac
    done
}

# Main restore function
main() {
    parse_args "$@"

    log_info "=========================================="
    log_info "Starting restore process..."
    log_info "=========================================="

    check_docker
    check_containers
    select_backup "${BACKUP_FILE}"
    extract_backup
    confirm_restore

    # Perform restore
    if ! restore_minio; then
        log_error "MinIO restore failed"
        cleanup
        exit 1
    fi

    if ! restore_rabbitmq; then
        log_error "RabbitMQ restore failed"
        cleanup
        exit 1
    fi

    # Cleanup
    cleanup

    log_info "=========================================="
    log_success "Restore completed successfully!"
    log_info "=========================================="
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main function
main "$@"
