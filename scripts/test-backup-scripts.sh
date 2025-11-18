#!/bin/bash

# Test script for backup and restore scripts
# This tests syntax and basic functionality without Docker

# Don't exit on error - we want to collect all test results
# set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((TESTS_FAILED++))
}

# Test 1: Check if scripts exist
test_scripts_exist() {
    log_info "Test 1: Checking if scripts exist..."

    if [ -f "scripts/backup.sh" ]; then
        log_success "backup.sh exists"
    else
        log_error "backup.sh not found"
    fi

    if [ -f "scripts/restore.sh" ]; then
        log_success "restore.sh exists"
    else
        log_error "restore.sh not found"
    fi

    if [ -f "scripts/setup-backup-cron.sh" ]; then
        log_success "setup-backup-cron.sh exists"
    else
        log_error "setup-backup-cron.sh not found"
    fi
}

# Test 2: Check if scripts are executable
test_scripts_executable() {
    log_info "Test 2: Checking if scripts are executable..."

    if [ -x "scripts/backup.sh" ]; then
        log_success "backup.sh is executable"
    else
        log_error "backup.sh is not executable"
    fi

    if [ -x "scripts/restore.sh" ]; then
        log_success "restore.sh is executable"
    else
        log_error "restore.sh is not executable"
    fi

    if [ -x "scripts/setup-backup-cron.sh" ]; then
        log_success "setup-backup-cron.sh is executable"
    else
        log_error "setup-backup-cron.sh is not executable"
    fi
}

# Test 3: Check syntax with bash -n
test_syntax() {
    log_info "Test 3: Checking script syntax..."

    if bash -n scripts/backup.sh 2>/dev/null; then
        log_success "backup.sh syntax is valid"
    else
        log_error "backup.sh syntax is invalid"
    fi

    if bash -n scripts/restore.sh 2>/dev/null; then
        log_success "restore.sh syntax is valid"
    else
        log_error "restore.sh syntax is invalid"
    fi

    if bash -n scripts/setup-backup-cron.sh 2>/dev/null; then
        log_success "setup-backup-cron.sh syntax is valid"
    else
        log_error "setup-backup-cron.sh syntax is invalid"
    fi
}

# Test 4: Check for required functions
test_required_functions() {
    log_info "Test 4: Checking for required functions in scripts..."

    # Check backup.sh functions
    if grep -q "backup_minio()" scripts/backup.sh; then
        log_success "backup.sh contains backup_minio function"
    else
        log_error "backup.sh missing backup_minio function"
    fi

    if grep -q "backup_rabbitmq()" scripts/backup.sh; then
        log_success "backup.sh contains backup_rabbitmq function"
    else
        log_error "backup.sh missing backup_rabbitmq function"
    fi

    if grep -q "cleanup_old_backups()" scripts/backup.sh; then
        log_success "backup.sh contains cleanup_old_backups function"
    else
        log_error "backup.sh missing cleanup_old_backups function"
    fi

    # Check restore.sh functions
    if grep -q "restore_minio()" scripts/restore.sh; then
        log_success "restore.sh contains restore_minio function"
    else
        log_error "restore.sh missing restore_minio function"
    fi

    if grep -q "restore_rabbitmq()" scripts/restore.sh; then
        log_success "restore.sh contains restore_rabbitmq function"
    else
        log_error "restore.sh missing restore_rabbitmq function"
    fi
}

# Test 5: Check for configuration variables
test_configuration() {
    log_info "Test 5: Checking for configuration variables..."

    # Check backup.sh configuration
    if grep -q "BACKUP_DIR=" scripts/backup.sh; then
        log_success "backup.sh has BACKUP_DIR configuration"
    else
        log_error "backup.sh missing BACKUP_DIR configuration"
    fi

    if grep -q "RETENTION_DAYS=" scripts/backup.sh; then
        log_success "backup.sh has RETENTION_DAYS configuration"
    else
        log_error "backup.sh missing RETENTION_DAYS configuration"
    fi

    if grep -q "MINIO_ENDPOINT=" scripts/backup.sh; then
        log_success "backup.sh has MINIO_ENDPOINT configuration"
    else
        log_error "backup.sh missing MINIO_ENDPOINT configuration"
    fi

    if grep -q "RABBITMQ_HOST=" scripts/backup.sh; then
        log_success "backup.sh has RABBITMQ_HOST configuration"
    else
        log_error "backup.sh missing RABBITMQ_HOST configuration"
    fi
}

# Test 6: Check help option
test_help_option() {
    log_info "Test 6: Checking help option..."

    if ./scripts/restore.sh --help &>/dev/null; then
        log_success "restore.sh --help works"
    else
        log_error "restore.sh --help doesn't work"
    fi

    if ./scripts/setup-backup-cron.sh --help &>/dev/null; then
        log_success "setup-backup-cron.sh --help works"
    else
        log_error "setup-backup-cron.sh --help doesn't work"
    fi
}

# Test 7: Check documentation
test_documentation() {
    log_info "Test 7: Checking documentation..."

    if [ -f "docs/backup-and-recovery.md" ]; then
        log_success "backup-and-recovery.md exists"
    else
        log_error "backup-and-recovery.md not found"
    fi

    if grep -q "Backup and Disaster Recovery" README.md; then
        log_success "README.md contains backup section"
    else
        log_error "README.md missing backup section"
    fi

    if grep -q "backup-and-recovery.md" README.md; then
        log_success "README.md links to backup documentation"
    else
        log_error "README.md doesn't link to backup documentation"
    fi
}

# Test 8: Check .gitignore
test_gitignore() {
    log_info "Test 8: Checking .gitignore..."

    if grep -q "backups/" .gitignore; then
        log_success ".gitignore contains backups/ directory"
    else
        log_error ".gitignore missing backups/ directory"
    fi

    if grep -q "*.tar.gz" .gitignore; then
        log_success ".gitignore contains *.tar.gz pattern"
    else
        log_error ".gitignore missing *.tar.gz pattern"
    fi
}

# Test 9: Check crontab file
test_crontab_file() {
    log_info "Test 9: Checking crontab file..."

    if [ -f "scripts/backup.crontab" ]; then
        log_success "backup.crontab exists"
    else
        log_error "backup.crontab not found"
    fi

    if [ -f "scripts/backup.crontab" ] && grep -q "backup.sh" scripts/backup.crontab; then
        log_success "backup.crontab contains backup.sh command"
    else
        log_error "backup.crontab doesn't contain backup.sh command"
    fi
}

# Test 10: Check error handling
test_error_handling() {
    log_info "Test 10: Checking error handling..."

    if grep -q "set -e" scripts/backup.sh; then
        log_success "backup.sh has error handling (set -e)"
    else
        log_error "backup.sh missing error handling"
    fi

    if grep -q "set -e" scripts/restore.sh; then
        log_success "restore.sh has error handling (set -e)"
    else
        log_error "restore.sh missing error handling"
    fi
}

# Run all tests
main() {
    echo "=========================================="
    echo "Running Backup Scripts Tests"
    echo "=========================================="
    echo ""

    test_scripts_exist
    echo ""
    test_scripts_executable
    echo ""
    test_syntax
    echo ""
    test_required_functions
    echo ""
    test_configuration
    echo ""
    test_help_option
    echo ""
    test_documentation
    echo ""
    test_gitignore
    echo ""
    test_crontab_file
    echo ""
    test_error_handling
    echo ""

    echo "=========================================="
    echo "Test Results"
    echo "=========================================="
    echo -e "${GREEN}Tests Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Tests Failed: ${TESTS_FAILED}${NC}"
    echo ""

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

main
