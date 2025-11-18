# Quick Start Guide

Get started with performance testing in 5 minutes.

## Prerequisites

You need:
- Docker and Docker Compose (for running services)
- Go 1.24+ (for benchmarks)
- k6 (for load tests) - optional

## Step 1: Install k6 (Optional)

### macOS
```bash
brew install k6
```

### Linux
```bash
curl -s https://dl.k6.io/key.gpg | sudo apt-key add -
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Skip k6
If you don't want to install k6, you can run only Go benchmarks:
```bash
make bench
```

## Step 2: Start Services

From project root:
```bash
./scripts/start.sh
```

Wait until all services are healthy (30-60 seconds).

## Step 3: Run Tests

### Option A: Run Everything (Recommended)

```bash
cd tests/performance
make all
```

This runs all tests and generates HTML reports (~15-30 minutes).

### Option B: Quick Test

```bash
make quick
```

Runs shorter tests (~5 minutes).

### Option C: Individual Tests

```bash
# Only k6 load tests
make k6

# Only Go benchmarks
make benchmarks

# Specific test
make k6-gateway
make bench-exiftool
```

## Step 4: View Results

```bash
make view
```

Or manually open:
```bash
open reports/index.html           # macOS
xdg-open reports/index.html       # Linux
start reports/index.html          # Windows
```

## Understanding Results

### k6 Tests

Look for:
- ✅ **Green thresholds**: All performance targets met
- ⚠️ **Yellow warnings**: Some requests are slow
- ❌ **Red failures**: Performance issues detected

Key metrics:
- `p95 < 5s`: 95% of requests complete in under 5 seconds
- `error_rate < 10%`: Less than 10% of requests fail

### Go Benchmarks

Lower numbers are better:
- **ns/op**: Nanoseconds per operation (speed)
- **B/op**: Bytes allocated per operation (memory)
- **allocs/op**: Number of allocations (GC pressure)

Example:
```
BenchmarkMinIOUpload/Size_100KB-4    1000    1234567 ns/op    102400 B/op    5 allocs/op
```
- Took 1.2ms per upload
- Allocated 100KB per operation
- Made 5 allocations

## Common Issues

### "Services not available"
```bash
# Check services are running
docker ps

# Restart if needed
./scripts/stop.sh
./scripts/start.sh
```

### "k6 not found"
```bash
# Install k6 or skip those tests
make benchmarks  # Run only Go benchmarks
```

### "exiftool not found"
```bash
# Install or skip
brew install exiftool              # macOS
sudo apt-get install exiftool      # Linux

# Or skip ExifTool tests
export SKIP_EXIFTOOL_BENCHMARKS=true
make bench-minio  # Run only MinIO benchmarks
```

## Next Steps

- Read [README.md](README.md) for detailed documentation
- Customize tests in `k6/` and `benchmarks/`
- Integrate into CI/CD
- Set up regular performance monitoring

## Need Help?

Check:
- [README.md](README.md) - Full documentation
- [k6 docs](https://k6.io/docs/) - k6 documentation
- [Go benchmarks](https://golang.org/pkg/testing/#hdr-Benchmarks) - Go testing guide

## Makefile Commands

```bash
make help          # Show all available commands
make all           # Run all tests
make quick         # Quick test (reduced time)
make k6            # Run k6 tests only
make benchmarks    # Run Go benchmarks only
make clean         # Clean reports
make view          # Open reports
make check         # Check prerequisites
make stats         # Show report statistics
```

## Performance Targets

Your system should achieve:
- Gateway: < 500ms (p95)
- Analyzer: < 5s (p95)
- Processor: < 1.5s (p95)
- Throughput: > 10 images/sec
- Error rate: < 5%

If you're not hitting these targets, check:
1. Are all services healthy?
2. Enough CPU/memory allocated to Docker?
3. Any errors in service logs?
4. Network latency issues?

Good luck! 🚀
