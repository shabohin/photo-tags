#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K6_DIR="${SCRIPT_DIR}/k6"
REPORTS_DIR="${SCRIPT_DIR}/reports"

# Default values
GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080}"
ANALYZER_URL="${ANALYZER_URL:-http://localhost:8082}"
PROCESSOR_URL="${PROCESSOR_URL:-http://localhost:8083}"

# Ensure reports directory exists
mkdir -p "${REPORTS_DIR}"

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if k6 is installed
check_k6() {
    if ! command -v k6 &> /dev/null; then
        print_error "k6 is not installed. Please install it first:"
        echo "  macOS:   brew install k6"
        echo "  Linux:   sudo gpg -k && sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 && echo \"deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main\" | sudo tee /etc/apt/sources.list.d/k6.list && sudo apt-get update && sudo apt-get install k6"
        echo "  Docker:  docker pull grafana/k6"
        echo ""
        echo "Or visit: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi
}

# Function to check if services are available
check_services() {
    print_info "Checking if services are available..."

    local all_ok=true

    if curl -s -f "${GATEWAY_URL}/health" > /dev/null 2>&1; then
        print_info "✓ Gateway is available at ${GATEWAY_URL}"
    else
        print_warning "✗ Gateway is not available at ${GATEWAY_URL}"
        all_ok=false
    fi

    if curl -s -f "${ANALYZER_URL}/health" > /dev/null 2>&1; then
        print_info "✓ Analyzer is available at ${ANALYZER_URL}"
    else
        print_warning "✗ Analyzer is not available at ${ANALYZER_URL}"
        all_ok=false
    fi

    if curl -s -f "${PROCESSOR_URL}/health" > /dev/null 2>&1; then
        print_info "✓ Processor is available at ${PROCESSOR_URL}"
    else
        print_warning "✗ Processor is not available at ${PROCESSOR_URL}"
        all_ok=false
    fi

    if [ "$all_ok" = false ]; then
        print_warning "Some services are not available. Tests may fail."
        read -p "Continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
}

# Function to run a k6 test
run_k6_test() {
    local test_name=$1
    local test_file=$2

    print_info "Running ${test_name}..."

    cd "${SCRIPT_DIR}"

    k6 run \
        --out json="${REPORTS_DIR}/${test_name}-results.json" \
        -e GATEWAY_URL="${GATEWAY_URL}" \
        -e ANALYZER_URL="${ANALYZER_URL}" \
        -e PROCESSOR_URL="${PROCESSOR_URL}" \
        "${test_file}" \
        | tee "${REPORTS_DIR}/${test_name}-output.txt"

    print_info "✓ ${test_name} completed. Reports saved to ${REPORTS_DIR}/"
}

# Function to generate summary report
generate_summary() {
    local summary_file="${REPORTS_DIR}/test-summary.txt"

    print_info "Generating test summary..."

    cat > "${summary_file}" << EOF
===============================================================================
Performance Test Summary
===============================================================================
Date: $(date)
Gateway URL: ${GATEWAY_URL}
Analyzer URL: ${ANALYZER_URL}
Processor URL: ${PROCESSOR_URL}
===============================================================================

Test Reports Generated:
EOF

    for report in "${REPORTS_DIR}"/*.html; do
        if [ -f "$report" ]; then
            echo "  - $(basename "$report")" >> "${summary_file}"
        fi
    done

    echo "" >> "${summary_file}"
    echo "All reports available in: ${REPORTS_DIR}/" >> "${summary_file}"
    echo "===============================================================================" >> "${summary_file}"

    cat "${summary_file}"
}

# Main function
main() {
    print_info "Starting k6 Performance Tests"
    echo "==============================================================================="

    # Check prerequisites
    check_k6
    check_services

    echo ""
    print_info "Test Configuration:"
    echo "  Gateway URL:  ${GATEWAY_URL}"
    echo "  Analyzer URL: ${ANALYZER_URL}"
    echo "  Processor URL: ${PROCESSOR_URL}"
    echo "  Reports Dir:  ${REPORTS_DIR}"
    echo ""

    # Run tests
    case "${1:-all}" in
        gateway)
            run_k6_test "gateway-load-test" "${K6_DIR}/gateway-load-test.js"
            ;;
        latency)
            run_k6_test "latency-test" "${K6_DIR}/latency-test.js"
            ;;
        throughput)
            run_k6_test "throughput-test" "${K6_DIR}/throughput-test.js"
            ;;
        resource)
            run_k6_test "resource-monitor" "${K6_DIR}/resource-monitor.js"
            ;;
        all)
            run_k6_test "gateway-load-test" "${K6_DIR}/gateway-load-test.js"
            echo ""
            run_k6_test "latency-test" "${K6_DIR}/latency-test.js"
            echo ""
            run_k6_test "throughput-test" "${K6_DIR}/throughput-test.js"
            echo ""
            run_k6_test "resource-monitor" "${K6_DIR}/resource-monitor.js"
            ;;
        *)
            print_error "Unknown test: $1"
            echo "Usage: $0 [gateway|latency|throughput|resource|all]"
            exit 1
            ;;
    esac

    echo ""
    generate_summary

    print_info "All tests completed successfully!"
    print_info "Open the HTML reports in ${REPORTS_DIR}/ to view detailed results"
}

# Run main function
main "$@"
