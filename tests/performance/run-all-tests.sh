#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORTS_DIR="${SCRIPT_DIR}/reports"

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

print_section() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC} $1"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

print_header() {
    clear
    echo -e "${MAGENTA}"
    cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║                  Photo Tags Performance Test Suite                       ║
║                                                                           ║
║                    Comprehensive Performance Testing                     ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
}

# Function to clean previous reports
clean_reports() {
    if [ -d "${REPORTS_DIR}" ]; then
        print_info "Cleaning previous test reports..."
        rm -rf "${REPORTS_DIR}"
    fi
    mkdir -p "${REPORTS_DIR}"
}

# Function to run k6 tests
run_k6_tests() {
    print_section "Running k6 Load Tests"

    if ! "${SCRIPT_DIR}/run-k6-tests.sh" all; then
        print_error "k6 tests failed"
        return 1
    fi

    print_info "✓ k6 tests completed successfully"
    return 0
}

# Function to run Go benchmarks
run_go_benchmarks() {
    print_section "Running Go Benchmark Tests"

    if ! "${SCRIPT_DIR}/run-benchmarks.sh" all; then
        print_error "Go benchmarks failed"
        return 1
    fi

    print_info "✓ Go benchmarks completed successfully"
    return 0
}

# Function to generate consolidated report
generate_consolidated_report() {
    print_section "Generating Consolidated Report"

    local report_file="${REPORTS_DIR}/index.html"

    cat > "${report_file}" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Photo Tags - Performance Test Results</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            text-align: center;
        }
        .header h1 {
            font-size: 2.5em;
            margin-bottom: 10px;
        }
        .header p {
            font-size: 1.2em;
            opacity: 0.9;
        }
        .info-section {
            padding: 30px 40px;
            background: #f8f9fa;
            border-bottom: 1px solid #e9ecef;
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
        }
        .info-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .info-card h3 {
            color: #667eea;
            margin-bottom: 10px;
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .info-card p {
            color: #495057;
            font-size: 1.1em;
        }
        .reports-section {
            padding: 40px;
        }
        .section-title {
            font-size: 1.8em;
            color: #333;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 3px solid #667eea;
        }
        .report-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .report-card {
            background: white;
            border: 1px solid #e9ecef;
            border-radius: 8px;
            padding: 20px;
            transition: all 0.3s ease;
            cursor: pointer;
        }
        .report-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 5px 20px rgba(0,0,0,0.1);
            border-color: #667eea;
        }
        .report-card h3 {
            color: #333;
            margin-bottom: 10px;
        }
        .report-card p {
            color: #6c757d;
            font-size: 0.9em;
            margin-bottom: 15px;
        }
        .report-card a {
            display: inline-block;
            padding: 10px 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            text-decoration: none;
            border-radius: 5px;
            transition: all 0.3s ease;
        }
        .report-card a:hover {
            transform: scale(1.05);
        }
        .footer {
            padding: 30px 40px;
            background: #f8f9fa;
            text-align: center;
            color: #6c757d;
        }
        .badge {
            display: inline-block;
            padding: 5px 10px;
            background: #28a745;
            color: white;
            border-radius: 3px;
            font-size: 0.8em;
            margin-left: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Performance Test Results</h1>
            <p>Comprehensive Performance and Load Testing Report</p>
        </div>

        <div class="info-section">
            <div class="info-grid">
                <div class="info-card">
                    <h3>Test Date</h3>
                    <p>TEST_DATE_PLACEHOLDER</p>
                </div>
                <div class="info-card">
                    <h3>Total Tests</h3>
                    <p>TOTAL_TESTS_PLACEHOLDER</p>
                </div>
                <div class="info-card">
                    <h3>Test Duration</h3>
                    <p>~15-30 minutes</p>
                </div>
            </div>
        </div>

        <div class="reports-section">
            <h2 class="section-title">k6 Load Test Reports</h2>
            <div class="report-grid">
                K6_REPORTS_PLACEHOLDER
            </div>

            <h2 class="section-title">Go Benchmark Reports</h2>
            <div class="report-grid">
                GO_REPORTS_PLACEHOLDER
            </div>
        </div>

        <div class="footer">
            <p>Generated by Photo Tags Performance Test Suite</p>
            <p>Project: <a href="https://github.com/shabohin/photo-tags">github.com/shabohin/photo-tags</a></p>
        </div>
    </div>
</body>
</html>
EOF

    # Generate report cards for k6 tests
    local k6_cards=""
    for report in "${REPORTS_DIR}"/*-summary.html; do
        if [ -f "$report" ]; then
            local name=$(basename "$report" | sed 's/-summary.html//' | sed 's/-/ /g' | sed 's/\b\(.\)/\u\1/g')
            k6_cards+="<div class=\"report-card\"><h3>${name}</h3><p>Load testing and performance metrics</p><a href=\"$(basename "$report")\">View Report</a></div>"
        fi
    done

    # Generate report cards for Go benchmarks
    local go_cards=""
    for report in "${REPORTS_DIR}"/*-benchmark.html; do
        if [ -f "$report" ]; then
            local name=$(basename "$report" | sed 's/-benchmark.html//' | sed 's/-/ /g' | sed 's/\b\(.\)/\u\1/g')
            go_cards+="<div class=\"report-card\"><h3>${name} Benchmarks</h3><p>Go performance benchmarks</p><a href=\"$(basename "$report")\">View Report</a></div>"
        fi
    done

    # Count total tests
    local total_tests=$(find "${REPORTS_DIR}" -name "*.html" ! -name "index.html" | wc -l)

    # Replace placeholders
    sed -i "s|TEST_DATE_PLACEHOLDER|$(date)|g" "${report_file}"
    sed -i "s|TOTAL_TESTS_PLACEHOLDER|${total_tests}|g" "${report_file}"
    sed -i "s|K6_REPORTS_PLACEHOLDER|${k6_cards}|g" "${report_file}"
    sed -i "s|GO_REPORTS_PLACEHOLDER|${go_cards}|g" "${report_file}"

    print_info "✓ Consolidated report generated: ${report_file}"
}

# Main function
main() {
    print_header

    print_info "Starting comprehensive performance test suite..."
    print_info "This will run both k6 load tests and Go benchmarks"
    echo ""

    # Parse arguments
    local skip_k6=false
    local skip_go=false
    local skip_clean=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-k6)
                skip_k6=true
                shift
                ;;
            --skip-go)
                skip_go=true
                shift
                ;;
            --no-clean)
                skip_clean=true
                shift
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Usage: $0 [--skip-k6] [--skip-go] [--no-clean]"
                exit 1
                ;;
        esac
    done

    # Clean previous reports
    if [ "$skip_clean" = false ]; then
        clean_reports
    fi

    local start_time=$(date +%s)
    local k6_success=true
    local go_success=true

    # Run k6 tests
    if [ "$skip_k6" = false ]; then
        if ! run_k6_tests; then
            k6_success=false
            print_warning "k6 tests failed but continuing..."
        fi
    else
        print_warning "Skipping k6 tests (--skip-k6 flag)"
    fi

    # Run Go benchmarks
    if [ "$skip_go" = false ]; then
        if ! run_go_benchmarks; then
            go_success=false
            print_warning "Go benchmarks failed but continuing..."
        fi
    else
        print_warning "Skipping Go benchmarks (--skip-go flag)"
    fi

    # Generate consolidated report
    generate_consolidated_report

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Print summary
    print_section "Test Suite Summary"

    echo "Test Results:"
    if [ "$skip_k6" = false ]; then
        if [ "$k6_success" = true ]; then
            echo -e "  ${GREEN}✓${NC} k6 Load Tests: PASSED"
        else
            echo -e "  ${RED}✗${NC} k6 Load Tests: FAILED"
        fi
    fi

    if [ "$skip_go" = false ]; then
        if [ "$go_success" = true ]; then
            echo -e "  ${GREEN}✓${NC} Go Benchmarks: PASSED"
        else
            echo -e "  ${RED}✗${NC} Go Benchmarks: FAILED"
        fi
    fi

    echo ""
    echo "Duration: ${duration} seconds"
    echo "Reports: ${REPORTS_DIR}/"
    echo ""

    print_info "═══════════════════════════════════════════════════════════"
    print_info "  Open ${REPORTS_DIR}/index.html to view all results"
    print_info "═══════════════════════════════════════════════════════════"

    # Return appropriate exit code
    if [ "$k6_success" = true ] && [ "$go_success" = true ]; then
        exit 0
    else
        exit 1
    fi
}

# Run main function
main "$@"
