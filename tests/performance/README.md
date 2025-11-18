# Performance and Load Testing Suite

Comprehensive performance testing suite for the Photo Tags service, including load tests and benchmark tests.

## Overview

This test suite provides:

- **k6 Load Tests**: HTTP load testing for Gateway, Analyzer, and Processor services
- **Go Benchmarks**: Performance benchmarks for critical components (ExifTool wrapper, MinIO operations)
- **HTML Reports**: Beautiful, detailed HTML reports for all test results
- **Metrics Tracking**: Latency, throughput, resource usage, and error rates

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Test Types](#test-types)
- [Running Tests](#running-tests)
- [Interpreting Results](#interpreting-results)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required

- **Go 1.24+**: For running Go benchmarks
- **Docker & Docker Compose**: For running the services
- **Bash**: For running test scripts

### For k6 Load Tests

Install k6:

**macOS:**
```bash
brew install k6
```

**Linux (Debian/Ubuntu):**
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Docker:**
```bash
docker pull grafana/k6
```

Visit [k6.io/docs](https://k6.io/docs/getting-started/installation/) for more installation options.

### For Go Benchmarks

**ExifTool** (optional, for ExifTool benchmarks):

**macOS:**
```bash
brew install exiftool
```

**Linux:**
```bash
sudo apt-get install libimage-exiftool-perl
```

**MinIO** must be running (automatically started with `./scripts/start.sh`)

## Quick Start

### 1. Start Services

First, make sure all services are running:

```bash
# From project root
./scripts/start.sh
```

Wait for all services to be healthy (check with `docker ps`).

### 2. Run All Tests

```bash
cd tests/performance
./run-all-tests.sh
```

This will:
1. Run all k6 load tests
2. Run all Go benchmarks
3. Generate HTML reports
4. Create a consolidated index page

### 3. View Results

Open the generated report:

```bash
open reports/index.html
# or
xdg-open reports/index.html  # Linux
```

## Test Types

### k6 Load Tests

#### 1. Gateway Load Test (`gateway-load-test.js`)

Tests the Gateway service under heavy load.

**What it tests:**
- 100 concurrent virtual users
- Simulated image uploads
- Error rate tracking
- Response time distribution

**Stages:**
- Warm up: 10 VUs for 30s
- Ramp up: 50 VUs for 1m
- Peak load: 100 VUs for 2m
- Ramp down: 50 VUs for 1m
- Cool down: 0 VUs for 30s

**Thresholds:**
- 95% of requests < 5s
- Error rate < 10%

#### 2. Latency Test (`latency-test.js`)

Measures latency at each service level.

**What it tests:**
- Gateway latency
- Analyzer latency
- Processor latency
- End-to-end latency

**Thresholds:**
- Gateway p95 < 500ms
- Analyzer p95 < 5s
- Processor p95 < 1.5s
- E2E p95 < 15s

#### 3. Throughput Test (`throughput-test.js`)

Measures system throughput (images/second).

**What it tests:**
- Images processed per second
- Service availability under load
- Processing time distribution

**Configuration:**
- 50 iterations/second
- 3 minute duration
- Up to 200 virtual users

**Thresholds:**
- Minimum 10 images/sec
- 95% processed within 10s
- Error rate < 5%

#### 4. Resource Monitor (`resource-monitor.js`)

Monitors system under sustained load.

**What it tests:**
- Service health under load
- Response times during sustained traffic
- Resource usage patterns

**Load pattern:**
- Ramp to 20 VUs (1m)
- Sustain 50 VUs (5m)
- Peak at 100 VUs (2m)

### Go Benchmarks

#### 1. ExifTool Benchmarks (`exiftool_bench_test.go`)

Benchmarks the ExifTool wrapper performance.

**Tests:**
- `BenchmarkExifToolWriteMetadata`: Metadata writing with different keyword counts
- `BenchmarkExifToolWriteMetadataParallel`: Parallel metadata writes
- `BenchmarkExifToolVerifyMetadata`: Metadata verification
- `BenchmarkExifToolCompleteWorkflow`: Write + verify workflow
- `BenchmarkExifToolCheckAvailability`: ExifTool availability check

**Configurations:**
- 10, 25, 49 keywords per test
- CPU: 1, 2, 4 cores
- Includes memory allocation stats

#### 2. MinIO Benchmarks (`minio_bench_test.go`)

Benchmarks MinIO storage operations.

**Tests:**
- `BenchmarkMinIOUpload`: File uploads (1KB to 5MB)
- `BenchmarkMinIOUploadParallel`: Concurrent uploads
- `BenchmarkMinIODownload`: File downloads (1KB to 5MB)
- `BenchmarkMinIODownloadParallel`: Concurrent downloads
- `BenchmarkMinIOCompleteWorkflow`: Upload + download
- `BenchmarkMinIOGetPresignedURL`: URL generation
- `BenchmarkMinIOAnalyzerDownload`: Analyzer-specific download

**File sizes:**
- 1 KB
- 100 KB
- 1 MB
- 5 MB

## Running Tests

### Run All Tests

```bash
./run-all-tests.sh
```

**Options:**
```bash
./run-all-tests.sh --skip-k6      # Skip k6 tests
./run-all-tests.sh --skip-go      # Skip Go benchmarks
./run-all-tests.sh --no-clean     # Don't clean previous reports
```

### Run k6 Tests Only

```bash
# All k6 tests
./run-k6-tests.sh all

# Individual tests
./run-k6-tests.sh gateway
./run-k6-tests.sh latency
./run-k6-tests.sh throughput
./run-k6-tests.sh resource
```

**Environment variables:**
```bash
export GATEWAY_URL=http://localhost:8080
export ANALYZER_URL=http://localhost:8082
export PROCESSOR_URL=http://localhost:8083

./run-k6-tests.sh all
```

### Run Go Benchmarks Only

```bash
# All benchmarks
./run-benchmarks.sh all

# Individual benchmarks
./run-benchmarks.sh exiftool
./run-benchmarks.sh minio
```

**Environment variables:**
```bash
export BENCH_TIME=10s           # Time per benchmark
export BENCH_COUNT=3            # Number of iterations
export MINIO_ENDPOINT=localhost:9000
export MINIO_ACCESS_KEY=minioadmin
export MINIO_SECRET_KEY=minioadmin
export SKIP_MINIO_BENCHMARKS=true  # Skip MinIO if not available

./run-benchmarks.sh all
```

### Run Specific Go Benchmarks

```bash
cd benchmarks

# Run specific benchmark
go test -bench=BenchmarkExifToolWriteMetadata -benchtime=5s

# Run with CPU profiling
go test -bench=BenchmarkMinIOUpload -cpuprofile=cpu.prof

# Run with memory profiling
go test -bench=BenchmarkMinIODownload -memprofile=mem.prof

# Verbose output
go test -bench=. -benchmem -v
```

## Interpreting Results

### k6 Test Results

**Key metrics:**
- `http_req_duration`: Request duration (avg, min, max, p90, p95, p99)
- `http_req_failed`: Percentage of failed requests
- `iterations`: Number of completed iterations
- `vus`: Virtual users (current, max)

**What to look for:**
- ✅ p95 < threshold: 95% of requests meet performance goals
- ✅ error_rate < 10%: System is stable under load
- ⚠️ High p99: Some requests are very slow (investigate)
- ❌ error_rate > 10%: System struggling under load

### Go Benchmark Results

**Output format:**
```
BenchmarkExifToolWriteMetadata/Keywords_25-4    100    5000000 ns/op    1024 B/op    10 allocs/op
```

**Explanation:**
- `Keywords_25`: Test variant (25 keywords)
- `-4`: GOMAXPROCS (CPU cores)
- `100`: Number of iterations
- `5000000 ns/op`: 5ms per operation
- `1024 B/op`: Bytes allocated per operation
- `10 allocs/op`: Number of allocations per operation

**What to look for:**
- Lower ns/op is better (faster)
- Lower B/op is better (less memory)
- Lower allocs/op is better (less GC pressure)
- Compare parallel vs sequential performance

### HTML Reports

Each report includes:
- **Summary**: Overall test statistics
- **Metrics**: Detailed performance metrics
- **Charts**: Visual representation (k6 reports)
- **Thresholds**: Pass/fail status for each threshold

## Configuration

### k6 Test Configuration

Edit the test files in `k6/` to modify:

- **VU stages**: Number and duration of load stages
- **Thresholds**: Performance thresholds
- **Think time**: Delay between requests
- **Test duration**: How long tests run

### Go Benchmark Configuration

Environment variables:
```bash
BENCH_TIME=10s     # Duration per benchmark (default: 10s)
BENCH_COUNT=3      # Number of runs (default: 3)
```

In code:
```go
// Change benchmark iterations
go test -bench=. -benchtime=100x  // 100 iterations

// Change test duration
go test -bench=. -benchtime=30s   // 30 seconds
```

## Troubleshooting

### k6 Tests Fail

**Problem:** "Service not available"

**Solution:**
```bash
# Check services are running
docker ps

# Check health endpoints
curl http://localhost:8080/health
curl http://localhost:8082/health
curl http://localhost:8083/health

# Restart services
./scripts/start.sh
```

**Problem:** "k6 command not found"

**Solution:** Install k6 (see [Prerequisites](#prerequisites))

### Go Benchmarks Fail

**Problem:** "exiftool not found"

**Solution:**
```bash
# Install exiftool
brew install exiftool  # macOS
sudo apt-get install libimage-exiftool-perl  # Linux

# Or skip ExifTool benchmarks
export SKIP_EXIFTOOL_BENCHMARKS=true
```

**Problem:** "MinIO connection failed"

**Solution:**
```bash
# Check MinIO is running
curl http://localhost:9000/minio/health/live

# Or skip MinIO benchmarks
export SKIP_MINIO_BENCHMARKS=true
./run-benchmarks.sh exiftool  # Run only ExifTool benchmarks
```

**Problem:** "Build errors"

**Solution:**
```bash
cd benchmarks
go mod download
go mod tidy
```

### High Error Rates

**Problem:** Error rate > 10% in tests

**Possible causes:**
- Services not fully started
- Insufficient resources (CPU/memory)
- Rate limiting active
- Network issues

**Solution:**
```bash
# Check Docker resources
docker stats

# Increase Docker resources (Docker Desktop settings)
# Check service logs
docker logs gateway
docker logs analyzer
docker logs processor

# Reduce test load
# Edit test files to reduce VUs or rate
```

### Memory Issues

**Problem:** "Out of memory" during benchmarks

**Solution:**
```bash
# Reduce benchmark time
export BENCH_TIME=5s

# Reduce file sizes in tests
# Edit benchmark test files

# Increase Docker memory limit
```

## Directory Structure

```
tests/performance/
├── README.md                    # This file
├── run-all-tests.sh            # Run all tests
├── run-k6-tests.sh             # Run k6 tests
├── run-benchmarks.sh           # Run Go benchmarks
├── k6/                         # k6 test scripts
│   ├── gateway-load-test.js    # Gateway load test
│   ├── latency-test.js         # Latency measurement
│   ├── throughput-test.js      # Throughput test
│   └── resource-monitor.js     # Resource monitoring
├── benchmarks/                 # Go benchmarks
│   ├── exiftool_bench_test.go  # ExifTool benchmarks
│   ├── minio_bench_test.go     # MinIO benchmarks
│   └── go.mod                  # Go module file
├── data/                       # Test data
│   └── images/                 # Test images
└── reports/                    # Generated reports (git-ignored)
    ├── index.html              # Consolidated report
    ├── *-summary.html          # k6 test reports
    └── *-benchmark.html        # Go benchmark reports
```

## Best Practices

1. **Run tests regularly**: Integrate into CI/CD
2. **Baseline results**: Keep historical data for comparison
3. **Isolate tests**: Run on dedicated test environment
4. **Monitor resources**: Watch CPU, memory, disk during tests
5. **Gradual load**: Start with small load, increase gradually
6. **Multiple runs**: Run tests multiple times for consistency
7. **Document changes**: Note any config changes that affect results

## Performance Targets

Based on test thresholds:

| Metric | Target | Critical |
|--------|--------|----------|
| Gateway latency (p95) | < 500ms | < 1s |
| Analyzer latency (p95) | < 5s | < 10s |
| Processor latency (p95) | < 1.5s | < 3s |
| End-to-end (p95) | < 15s | < 30s |
| Throughput | > 10 images/sec | > 5 images/sec |
| Error rate | < 5% | < 10% |

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Performance Tests

on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:      # Manual trigger

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Install k6
        run: |
          sudo gpg -k
          sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
          echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
          sudo apt-get update
          sudo apt-get install k6

      - name: Install ExifTool
        run: sudo apt-get install libimage-exiftool-perl

      - name: Start services
        run: ./scripts/start.sh

      - name: Run performance tests
        run: |
          cd tests/performance
          ./run-all-tests.sh

      - name: Upload reports
        uses: actions/upload-artifact@v3
        with:
          name: performance-reports
          path: tests/performance/reports/
```

## Contributing

To add new tests:

1. **k6 tests**: Add new `.js` file in `k6/`
2. **Go benchmarks**: Add `Benchmark*` functions in `benchmarks/`
3. **Update scripts**: Add test to appropriate run script
4. **Document**: Update this README

## Resources

- [k6 Documentation](https://k6.io/docs/)
- [Go Benchmarks Guide](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [ExifTool Documentation](https://exiftool.org/)
- [MinIO Go Client](https://docs.min.io/docs/golang-client-api-reference.html)

## License

Same as main project.
