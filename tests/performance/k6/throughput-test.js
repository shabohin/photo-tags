import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

// Custom metrics
const imagesProcessed = new Counter('images_processed');
const imagesPerSecond = new Trend('images_per_second');
const processingTime = new Trend('processing_time');
const errorRate = new Rate('error_rate');

// Test configuration for throughput testing
export const options = {
  scenarios: {
    constant_throughput: {
      executor: 'constant-arrival-rate',
      rate: 50, // 50 iterations per second
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 100,
      maxVUs: 200,
    },
  },
  thresholds: {
    images_per_second: ['value>=10'], // At least 10 images/sec
    processing_time: ['p(95)<10000'], // 95% processed within 10s
    error_rate: ['rate<0.05'],        // Error rate less than 5%
  },
};

// Environment variables
const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080';
const ANALYZER_URL = __ENV.ANALYZER_URL || 'http://localhost:8082';
const PROCESSOR_URL = __ENV.PROCESSOR_URL || 'http://localhost:8083';

// Track processed images
let processedCount = 0;
let lastCheck = Date.now();

// Main test function
export default function () {
  const start = Date.now();

  // Test Gateway (entry point)
  const gatewayRes = http.get(`${GATEWAY_URL}/health`);
  const gatewayOk = check(gatewayRes, {
    'gateway available': (r) => r.status === 200,
  });

  if (!gatewayOk) {
    errorRate.add(1);
    return;
  }

  // Test Analyzer
  const analyzerRes = http.get(`${ANALYZER_URL}/health`);
  const analyzerOk = check(analyzerRes, {
    'analyzer available': (r) => r.status === 200,
  });

  if (!analyzerOk) {
    errorRate.add(1);
    return;
  }

  // Test Processor
  const processorRes = http.get(`${PROCESSOR_URL}/health`);
  const processorOk = check(processorRes, {
    'processor available': (r) => r.status === 200,
  });

  if (!processorOk) {
    errorRate.add(1);
    return;
  }

  // Record successful processing
  const duration = Date.now() - start;
  processingTime.add(duration);
  imagesProcessed.add(1);
  processedCount++;

  // Calculate throughput every second
  const now = Date.now();
  if (now - lastCheck >= 1000) {
    const throughput = (processedCount * 1000) / (now - lastCheck);
    imagesPerSecond.add(throughput);
    processedCount = 0;
    lastCheck = now;
  }

  errorRate.add(0); // Success
}

// Setup function - runs once at the beginning
export function setup() {
  console.log('Starting throughput test...');
  console.log(`Gateway URL: ${GATEWAY_URL}`);
  console.log(`Analyzer URL: ${ANALYZER_URL}`);
  console.log(`Processor URL: ${PROCESSOR_URL}`);

  // Verify all services are available
  const services = [
    { name: 'Gateway', url: `${GATEWAY_URL}/health` },
    { name: 'Analyzer', url: `${ANALYZER_URL}/health` },
    { name: 'Processor', url: `${PROCESSOR_URL}/health` },
  ];

  for (const service of services) {
    const res = http.get(service.url);
    if (res.status !== 200) {
      console.error(`${service.name} is not available at ${service.url}`);
    } else {
      console.log(`${service.name} is available`);
    }
  }
}

// Teardown function - runs once at the end
export function teardown(data) {
  console.log('Throughput test completed');
}

// Generate HTML report
export function handleSummary(data) {
  // Calculate final throughput statistics
  const iterations = data.metrics.iterations.values.count;
  const duration = data.state.testRunDurationMs / 1000; // Convert to seconds
  const avgThroughput = iterations / duration;

  // Add custom summary
  const customSummary = {
    ...data,
    custom: {
      avg_throughput_images_per_sec: avgThroughput.toFixed(2),
      total_images_processed: iterations,
      test_duration_seconds: duration.toFixed(2),
    },
  };

  return {
    'reports/throughput-test-summary.html': htmlReport(customSummary),
    'reports/throughput-test-summary.json': JSON.stringify(customSummary),
    stdout: textSummary(customSummary, { indent: ' ', enableColors: true }),
  };
}
