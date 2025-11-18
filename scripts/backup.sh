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
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_NAME="backup_${TIMESTAMP}"
CURRENT_BACKUP_DIR="${BACKUP_DIR}/${BACKUP_NAME}"

# Docker container names
MINIO_CONTAINER="minio"
RABBITMQ_CONTAINER="rabbitmq"

# MinIO configuration
MINIO_ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
MINIO_ACCESS_KEY="${MINIO_ACCESS_KEY:-minioadmin}"
MINIO_SECRET_KEY="${MINIO_SECRET_KEY:-minioadmin}"
MINIO_BUCKETS="${MINIO_BUCKETS:-original processed}"

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

# Create backup directory
create_backup_dir() {
    log_info "Creating backup directory: ${CURRENT_BACKUP_DIR}"
    mkdir -p "${CURRENT_BACKUP_DIR}/minio"
    mkdir -p "${CURRENT_BACKUP_DIR}/rabbitmq"

    # Create metadata file
    cat > "${CURRENT_BACKUP_DIR}/backup.info" <<EOF
Backup Created: $(date)
Backup Version: 1.0
MinIO Buckets: ${MINIO_BUCKETS}
RabbitMQ Host: ${RABBITMQ_HOST}
EOF

    log_success "Backup directory created"
}

# Backup MinIO buckets
backup_minio() {
    log_info "Starting MinIO backup..."

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
    docker exec "${MINIO_CONTAINER}" mc alias set backup http://localhost:9000 "${MINIO_ACCESS_KEY}" "${MINIO_SECRET_KEY}" --api S3v4 || {
        log_error "Failed to configure MinIO Client"
        return 1
    }

    # Backup each bucket
    for bucket in ${MINIO_BUCKETS}; do
        log_info "Backing up bucket: ${bucket}"

        # Create bucket directory
        mkdir -p "${CURRENT_BACKUP_DIR}/minio/${bucket}"

        # Export bucket using docker exec and mc mirror
        docker exec "${MINIO_CONTAINER}" mc mirror --quiet "backup/${bucket}" "/tmp/backup_${bucket}" 2>&1 | grep -v "mc: <ERROR>" || true

        # Copy from container to host
        docker cp "${MINIO_CONTAINER}:/tmp/backup_${bucket}/." "${CURRENT_BACKUP_DIR}/minio/${bucket}/" 2>/dev/null || {
            log_warning "No data found in bucket ${bucket} or bucket doesn't exist"
        }

        # Cleanup temp directory in container
        docker exec "${MINIO_CONTAINER}" rm -rf "/tmp/backup_${bucket}" || true

        # Count files
        file_count=$(find "${CURRENT_BACKUP_DIR}/minio/${bucket}" -type f 2>/dev/null | wc -l)
        log_success "Bucket ${bucket} backed up (${file_count} files)"
    done

    log_success "MinIO backup completed"
}

# Backup RabbitMQ
backup_rabbitmq() {
    log_info "Starting RabbitMQ backup..."

    # Export definitions (exchanges, queues, bindings, etc.)
    log_info "Exporting RabbitMQ definitions..."
    curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASSWORD}" \
        "http://${RABBITMQ_HOST}:${RABBITMQ_PORT}/api/definitions" \
        -o "${CURRENT_BACKUP_DIR}/rabbitmq/definitions.json" || {
        log_error "Failed to export RabbitMQ definitions"
        return 1
    }

    # Check if definitions were exported
    if [ -s "${CURRENT_BACKUP_DIR}/rabbitmq/definitions.json" ]; then
        log_success "RabbitMQ definitions exported"
    else
        log_error "RabbitMQ definitions file is empty"
        return 1
    fi

    # Export queue messages (optional, for small queues)
    log_info "Exporting queue information..."
    curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASSWORD}" \
        "http://${RABBITMQ_HOST}:${RABBITMQ_PORT}/api/queues" \
        -o "${CURRENT_BACKUP_DIR}/rabbitmq/queues.json" || {
        log_warning "Failed to export queue information"
    }

    # Export overview
    curl -s -u "${RABBITMQ_USER}:${RABBITMQ_PASSWORD}" \
        "http://${RABBITMQ_HOST}:${RABBITMQ_PORT}/api/overview" \
        -o "${CURRENT_BACKUP_DIR}/rabbitmq/overview.json" || {
        log_warning "Failed to export overview"
    }

    log_success "RabbitMQ backup completed"
}

# Create compressed archive
create_archive() {
    log_info "Creating compressed archive..."

    cd "${BACKUP_DIR}"
    tar -czf "${BACKUP_NAME}.tar.gz" "${BACKUP_NAME}" 2>/dev/null || {
        log_error "Failed to create archive"
        return 1
    }

    # Remove uncompressed directory
    rm -rf "${BACKUP_NAME}"

    ARCHIVE_SIZE=$(du -h "${BACKUP_NAME}.tar.gz" | cut -f1)
    log_success "Archive created: ${BACKUP_NAME}.tar.gz (${ARCHIVE_SIZE})"
}

# Cleanup old backups
cleanup_old_backups() {
    log_info "Cleaning up backups older than ${RETENTION_DAYS} days..."

    if [ ! -d "${BACKUP_DIR}" ]; then
        log_warning "Backup directory doesn't exist, skipping cleanup"
        return 0
    fi

    # Find and delete old backup archives
    OLD_BACKUPS=$(find "${BACKUP_DIR}" -name "backup_*.tar.gz" -type f -mtime +${RETENTION_DAYS} 2>/dev/null || true)

    if [ -z "${OLD_BACKUPS}" ]; then
        log_info "No old backups to clean up"
    else
        echo "${OLD_BACKUPS}" | while read -r backup; do
            log_info "Deleting old backup: $(basename "${backup}")"
            rm -f "${backup}"
        done
        log_success "Old backups cleaned up"
    fi
}

# Main backup function
main() {
    log_info "=========================================="
    log_info "Starting backup process..."
    log_info "Backup name: ${BACKUP_NAME}"
    log_info "=========================================="

    check_docker
    check_containers
    create_backup_dir

    # Perform backups
    if ! backup_minio; then
        log_error "MinIO backup failed"
        exit 1
    fi

    if ! backup_rabbitmq; then
        log_error "RabbitMQ backup failed"
        exit 1
    fi

    # Create archive
    if ! create_archive; then
        log_error "Failed to create backup archive"
        exit 1
    fi

    # Cleanup old backups
    cleanup_old_backups

    log_info "=========================================="
    log_success "Backup completed successfully!"
    log_info "Backup location: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
    log_info "=========================================="
}

# Run main function
main "$@"
