import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

// Custom metrics for each stage
const gatewayLatency = new Trend('gateway_latency');
const analyzerLatency = new Trend('analyzer_latency');
const processorLatency = new Trend('processor_latency');
const endToEndLatency = new Trend('end_to_end_latency');

// Test configuration - focus on latency measurement
export const options = {
  scenarios: {
    latency_test: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
    },
  },
  thresholds: {
    gateway_latency: ['p(50)<100', 'p(95)<500', 'p(99)<1000'],
    analyzer_latency: ['p(50)<2000', 'p(95)<5000', 'p(99)<10000'],
    processor_latency: ['p(50)<500', 'p(95)<1500', 'p(99)<3000'],
    end_to_end_latency: ['p(50)<5000', 'p(95)<15000', 'p(99)<30000'],
  },
};

// Environment variables
const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080';
const ANALYZER_URL = __ENV.ANALYZER_URL || 'http://localhost:8082';
const PROCESSOR_URL = __ENV.PROCESSOR_URL || 'http://localhost:8083';

// Test Gateway latency
function testGatewayLatency() {
  const start = Date.now();
  const res = http.get(`${GATEWAY_URL}/health`);
  const duration = Date.now() - start;

  const success = check(res, {
    'gateway health check ok': (r) => r.status === 200,
  });

  if (success) {
    gatewayLatency.add(duration);
  }

  return duration;
}

// Test Analyzer latency
function testAnalyzerLatency() {
  const start = Date.now();
  const res = http.get(`${ANALYZER_URL}/health`);
  const duration = Date.now() - start;

  const success = check(res, {
    'analyzer health check ok': (r) => r.status === 200,
  });

  if (success) {
    analyzerLatency.add(duration);
  }

  return duration;
}

// Test Processor latency
function testProcessorLatency() {
  const start = Date.now();
  const res = http.get(`${PROCESSOR_URL}/health`);
  const duration = Date.now() - start;

  const success = check(res, {
    'processor health check ok': (r) => r.status === 200,
  });

  if (success) {
    processorLatency.add(duration);
  }

  return duration;
}

// Main test function
export default function () {
  const e2eStart = Date.now();

  // Test each service sequentially to simulate the pipeline
  const gatewayTime = testGatewayLatency();
  sleep(0.1); // Small delay between requests

  const analyzerTime = testAnalyzerLatency();
  sleep(0.1);

  const processorTime = testProcessorLatency();

  // Calculate total end-to-end latency
  const e2eTotal = Date.now() - e2eStart;
  endToEndLatency.add(e2eTotal);

  // Sleep before next iteration
  sleep(1);
}

// Generate HTML report
export function handleSummary(data) {
  return {
    'reports/latency-test-summary.html': htmlReport(data),
    'reports/latency-test-summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
