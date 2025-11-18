#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

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

# Check if cron is available
check_cron() {
    if ! command -v crontab &> /dev/null; then
        log_error "crontab command not found. Please install cron."
        exit 1
    fi

    log_success "cron is available"
}

# Setup backup cron job
setup_cron() {
    log_info "Setting up backup cron job..."

    # Create temporary crontab file
    TEMP_CRON=$(mktemp)

    # Get existing crontab (if any)
    crontab -l > "${TEMP_CRON}" 2>/dev/null || true

    # Check if backup job already exists
    if grep -q "photo-tags.*backup.sh" "${TEMP_CRON}"; then
        log_warning "Backup cron job already exists. Do you want to replace it?"
        read -p "Replace existing job? (yes/no): " replace

        if [ "${replace}" != "yes" ]; then
            log_info "Keeping existing cron job"
            rm -f "${TEMP_CRON}"
            return 0
        fi

        # Remove existing backup jobs
        sed -i '/photo-tags.*backup.sh/d' "${TEMP_CRON}"
    fi

    # Add new backup job
    log_info "Adding daily backup job at 2:00 AM..."
    echo "" >> "${TEMP_CRON}"
    echo "# Photo Tags automated backup (added $(date))" >> "${TEMP_CRON}"
    echo "0 2 * * * cd ${PROJECT_DIR} && ${PROJECT_DIR}/scripts/backup.sh >> /var/log/photo-tags-backup.log 2>&1" >> "${TEMP_CRON}"

    # Install new crontab
    crontab "${TEMP_CRON}"
    rm -f "${TEMP_CRON}"

    log_success "Backup cron job installed successfully"
}

# Show current crontab
show_crontab() {
    log_info "Current crontab:"
    echo ""
    crontab -l | grep -A 1 "photo-tags" || log_warning "No photo-tags backup jobs found"
    echo ""
}

# Create log directory
setup_log_directory() {
    log_info "Setting up log directory..."

    LOG_DIR="/var/log"
    if [ ! -w "${LOG_DIR}" ]; then
        log_warning "Cannot write to /var/log, using ${PROJECT_DIR}/logs instead"
        LOG_DIR="${PROJECT_DIR}/logs"
        mkdir -p "${LOG_DIR}"
    fi

    LOG_FILE="${LOG_DIR}/photo-tags-backup.log"
    touch "${LOG_FILE}" 2>/dev/null || {
        log_error "Cannot create log file: ${LOG_FILE}"
        exit 1
    }

    log_success "Log file: ${LOG_FILE}"
}

# Test backup script
test_backup() {
    log_info "Testing backup script..."

    if [ ! -x "${PROJECT_DIR}/scripts/backup.sh" ]; then
        log_error "backup.sh is not executable or not found"
        exit 1
    fi

    log_success "Backup script is executable"

    read -p "Do you want to run a test backup now? (yes/no): " test_now

    if [ "${test_now}" = "yes" ]; then
        log_info "Running test backup..."
        cd "${PROJECT_DIR}"
        ./scripts/backup.sh || {
            log_error "Test backup failed"
            exit 1
        }
        log_success "Test backup completed"
    fi
}

# Show usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Setup automated backup cron job for Photo Tags.

OPTIONS:
    -t, --test-only     Only test backup script without installing cron
    -r, --remove        Remove backup cron job
    -h, --help          Show this help message

EXAMPLES:
    # Install backup cron job
    $0

    # Test backup script only
    $0 --test-only

    # Remove backup cron job
    $0 --remove

EOF
}

# Remove cron job
remove_cron() {
    log_info "Removing backup cron job..."

    TEMP_CRON=$(mktemp)
    crontab -l > "${TEMP_CRON}" 2>/dev/null || true

    if ! grep -q "photo-tags.*backup.sh" "${TEMP_CRON}"; then
        log_warning "No backup cron job found"
        rm -f "${TEMP_CRON}"
        return 0
    fi

    # Remove backup jobs
    sed -i '/photo-tags.*backup.sh/d' "${TEMP_CRON}"
    sed -i '/Photo Tags automated backup/d' "${TEMP_CRON}"

    # Install updated crontab
    crontab "${TEMP_CRON}"
    rm -f "${TEMP_CRON}"

    log_success "Backup cron job removed"
}

# Parse arguments
parse_args() {
    TEST_ONLY=false
    REMOVE=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--test-only)
                TEST_ONLY=true
                shift
                ;;
            -r|--remove)
                REMOVE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Main function
main() {
    parse_args "$@"

    log_info "=========================================="
    log_info "Photo Tags Backup Cron Setup"
    log_info "=========================================="
    log_info "Project directory: ${PROJECT_DIR}"
    echo ""

    check_cron

    if [ "${REMOVE}" = "true" ]; then
        remove_cron
        show_crontab
        exit 0
    fi

    if [ "${TEST_ONLY}" = "true" ]; then
        test_backup
        exit 0
    fi

    setup_log_directory
    test_backup
    setup_cron
    show_crontab

    log_info "=========================================="
    log_success "Backup cron setup completed!"
    log_info "=========================================="
    log_info "Backup will run daily at 2:00 AM"
    log_info "Logs will be written to: ${LOG_FILE:-/var/log/photo-tags-backup.log}"
    log_info ""
    log_info "To view logs: tail -f ${LOG_FILE:-/var/log/photo-tags-backup.log}"
    log_info "To edit cron: crontab -e"
    log_info "To remove: $0 --remove"
    log_info "=========================================="
}

# Run main function
main "$@"
