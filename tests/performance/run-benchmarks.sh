#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="${SCRIPT_DIR}/benchmarks"
REPORTS_DIR="${SCRIPT_DIR}/reports"

# Benchmark options
BENCH_TIME="${BENCH_TIME:-10s}"
BENCH_COUNT="${BENCH_COUNT:-3}"

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
    echo -e "${BLUE}$1${NC}"
}

# Ensure reports directory exists
mkdir -p "${REPORTS_DIR}"

# Function to check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.24 or later."
        exit 1
    fi

    local go_version=$(go version | awk '{print $3}')
    print_info "Using Go version: ${go_version}"
}

# Function to check if exiftool is installed
check_exiftool() {
    if ! command -v exiftool &> /dev/null; then
        print_warning "exiftool is not installed. ExifTool benchmarks will be skipped."
        print_warning "To install exiftool:"
        echo "  macOS:  brew install exiftool"
        echo "  Linux:  sudo apt-get install libimage-exiftool-perl"
        return 1
    fi
    print_info "✓ exiftool is available"
    return 0
}

# Function to check if MinIO is available
check_minio() {
    local minio_endpoint="${MINIO_ENDPOINT:-localhost:9000}"

    if ! curl -s -f "http://${minio_endpoint}/minio/health/live" > /dev/null 2>&1; then
        print_warning "MinIO is not available at ${minio_endpoint}. MinIO benchmarks will be skipped."
        print_warning "To run MinIO benchmarks, start MinIO with: ./scripts/start.sh"
        export SKIP_MINIO_BENCHMARKS=true
        return 1
    fi
    print_info "✓ MinIO is available at ${minio_endpoint}"
    return 0
}

# Function to install benchstat if not present
install_benchstat() {
    if ! command -v benchstat &> /dev/null; then
        print_info "Installing benchstat for benchmark comparison..."
        go install golang.org/x/perf/cmd/benchstat@latest
    fi
}

# Function to convert benchmark output to HTML
generate_html_report() {
    local bench_file=$1
    local html_file=$2
    local title=$3

    print_info "Generating HTML report: $(basename ${html_file})"

    cat > "${html_file}" << 'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TITLE_PLACEHOLDER</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }
        h2 {
            color: #555;
            margin-top: 30px;
        }
        .info {
            background-color: #e3f2fd;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
        }
        .benchmark-results {
            background-color: white;
            padding: 20px;
            border-radius: 5px;
            margin: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        pre {
            background-color: #f8f8f8;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            font-family: 'Courier New', monospace;
            font-size: 12px;
            line-height: 1.4;
        }
        .metric {
            display: inline-block;
            margin: 10px 20px 10px 0;
        }
        .metric-label {
            font-weight: bold;
            color: #666;
        }
        .metric-value {
            color: #4CAF50;
            font-size: 1.2em;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background-color: #4CAF50;
            color: white;
        }
        tr:hover {
            background-color: #f5f5f5;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <h1>TITLE_PLACEHOLDER</h1>

    <div class="info">
        <p><strong>Generated:</strong> DATE_PLACEHOLDER</p>
        <p><strong>Benchmark Time:</strong> BENCH_TIME_PLACEHOLDER per test</p>
        <p><strong>Iterations:</strong> COUNT_PLACEHOLDER</p>
    </div>

    <div class="benchmark-results">
        <h2>Benchmark Results</h2>
        <pre>RESULTS_PLACEHOLDER</pre>
    </div>

    <div class="footer">
        <p>Generated by Photo Tags Performance Test Suite</p>
        <p>For more information about Go benchmarks, visit: <a href="https://golang.org/pkg/testing/#hdr-Benchmarks">https://golang.org/pkg/testing/#hdr-Benchmarks</a></p>
    </div>
</body>
</html>
EOF

    # Replace placeholders
    sed -i "s|TITLE_PLACEHOLDER|${title}|g" "${html_file}"
    sed -i "s|DATE_PLACEHOLDER|$(date)|g" "${html_file}"
    sed -i "s|BENCH_TIME_PLACEHOLDER|${BENCH_TIME}|g" "${html_file}"
    sed -i "s|COUNT_PLACEHOLDER|${BENCH_COUNT}|g" "${html_file}"

    # Insert benchmark results
    local results=$(cat "${bench_file}" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
    sed -i "s|RESULTS_PLACEHOLDER|${results}|g" "${html_file}"
}

# Function to run ExifTool benchmarks
run_exiftool_benchmarks() {
    print_section "=== Running ExifTool Benchmarks ==="

    cd "${BENCHMARKS_DIR}"

    local output_file="${REPORTS_DIR}/exiftool-benchmark.txt"
    local html_file="${REPORTS_DIR}/exiftool-benchmark.html"

    print_info "Running benchmarks (this may take a few minutes)..."

    go test -bench=BenchmarkExifTool \
        -benchtime="${BENCH_TIME}" \
        -count="${BENCH_COUNT}" \
        -benchmem \
        -cpu=1,2,4 \
        -timeout=30m \
        . 2>&1 | tee "${output_file}"

    generate_html_report "${output_file}" "${html_file}" "ExifTool Benchmark Results"

    print_info "✓ ExifTool benchmarks completed"
    print_info "  Text report: ${output_file}"
    print_info "  HTML report: ${html_file}"
}

# Function to run MinIO benchmarks
run_minio_benchmarks() {
    print_section "=== Running MinIO Benchmarks ==="

    cd "${BENCHMARKS_DIR}"

    local output_file="${REPORTS_DIR}/minio-benchmark.txt"
    local html_file="${REPORTS_DIR}/minio-benchmark.html"

    print_info "Running benchmarks (this may take a few minutes)..."

    go test -bench=BenchmarkMinIO \
        -benchtime="${BENCH_TIME}" \
        -count="${BENCH_COUNT}" \
        -benchmem \
        -cpu=1,2,4 \
        -timeout=30m \
        . 2>&1 | tee "${output_file}"

    generate_html_report "${output_file}" "${html_file}" "MinIO Benchmark Results"

    print_info "✓ MinIO benchmarks completed"
    print_info "  Text report: ${output_file}"
    print_info "  HTML report: ${html_file}"
}

# Function to run all benchmarks
run_all_benchmarks() {
    print_section "=== Running All Benchmarks ==="

    cd "${BENCHMARKS_DIR}"

    local output_file="${REPORTS_DIR}/all-benchmarks.txt"
    local html_file="${REPORTS_DIR}/all-benchmarks.html"

    print_info "Running all benchmarks (this may take several minutes)..."

    go test -bench=. \
        -benchtime="${BENCH_TIME}" \
        -count="${BENCH_COUNT}" \
        -benchmem \
        -cpu=1,2,4 \
        -timeout=60m \
        . 2>&1 | tee "${output_file}"

    generate_html_report "${output_file}" "${html_file}" "All Benchmark Results"

    print_info "✓ All benchmarks completed"
    print_info "  Text report: ${output_file}"
    print_info "  HTML report: ${html_file}"
}

# Function to generate summary
generate_summary() {
    local summary_file="${REPORTS_DIR}/benchmark-summary.txt"

    print_info "Generating benchmark summary..."

    cat > "${summary_file}" << EOF
===============================================================================
Go Benchmark Summary
===============================================================================
Date: $(date)
Benchmark Time: ${BENCH_TIME} per test
Iterations: ${BENCH_COUNT}
===============================================================================

Reports Generated:
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
    print_info "Starting Go Benchmark Tests"
    echo "==============================================================================="

    # Check prerequisites
    check_go
    install_benchstat

    # Initialize Go module
    cd "${BENCHMARKS_DIR}"
    print_info "Downloading Go dependencies..."
    go mod download

    local run_exiftool=true
    local run_minio=true

    # Check optional dependencies
    check_exiftool || run_exiftool=false
    check_minio || run_minio=false

    echo ""
    print_info "Benchmark Configuration:"
    echo "  Benchmark Time: ${BENCH_TIME} per test"
    echo "  Iterations:     ${BENCH_COUNT}"
    echo "  Reports Dir:    ${REPORTS_DIR}"
    echo ""

    # Run benchmarks based on argument
    case "${1:-all}" in
        exiftool)
            if [ "$run_exiftool" = true ]; then
                run_exiftool_benchmarks
            else
                print_error "Cannot run ExifTool benchmarks: exiftool not available"
                exit 1
            fi
            ;;
        minio)
            if [ "$run_minio" = true ]; then
                run_minio_benchmarks
            else
                print_error "Cannot run MinIO benchmarks: MinIO not available"
                exit 1
            fi
            ;;
        all)
            if [ "$run_exiftool" = true ] || [ "$run_minio" = true ]; then
                run_all_benchmarks
            else
                print_error "Cannot run benchmarks: no dependencies available"
                exit 1
            fi
            ;;
        *)
            print_error "Unknown benchmark: $1"
            echo "Usage: $0 [exiftool|minio|all]"
            exit 1
            ;;
    esac

    echo ""
    generate_summary

    print_info "All benchmarks completed successfully!"
    print_info "Open the HTML reports in ${REPORTS_DIR}/ to view detailed results"
}

# Run main function
main "$@"
