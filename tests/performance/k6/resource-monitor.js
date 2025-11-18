import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Counter } from 'k6/metrics';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';
import exec from 'k6/execution';

// Custom metrics for resource monitoring
const cpuUsage = new Trend('cpu_usage_percent');
const memoryUsage = new Trend('memory_usage_mb');
const requestsUnderLoad = new Counter('requests_under_load');

// Test configuration - sustained load for resource monitoring
export const options = {
  scenarios: {
    resource_monitoring: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 20 },   // Ramp up
        { duration: '5m', target: 50 },   // Sustained load
        { duration: '2m', target: 100 },  // Peak load
        { duration: '1m', target: 0 },    // Ramp down
      ],
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<5000'],
    http_req_failed: ['rate<0.1'],
  },
};

// Environment variables
const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080';
const ANALYZER_URL = __ENV.ANALYZER_URL || 'http://localhost:8082';
const PROCESSOR_URL = __ENV.PROCESSOR_URL || 'http://localhost:8083';

// Services to monitor
const services = [
  { name: 'Gateway', url: `${GATEWAY_URL}/health` },
  { name: 'Analyzer', url: `${ANALYZER_URL}/health` },
  { name: 'Processor', url: `${PROCESSOR_URL}/health` },
];

// Main test function
export default function () {
  // Rotate through services
  const serviceIndex = exec.scenario.iterationInTest % services.length;
  const service = services[serviceIndex];

  const res = http.get(service.url, {
    tags: { service: service.name },
  });

  check(res, {
    [`${service.name} is healthy`]: (r) => r.status === 200,
    [`${service.name} responds quickly`]: (r) => r.timings.duration < 1000,
  });

  requestsUnderLoad.add(1);

  // Note: Actual CPU/Memory metrics would need to be collected via
  // Docker stats API or system monitoring tools and injected here
  // This is a placeholder for the metric structure

  sleep(1);
}

// Setup function
export function setup() {
  console.log('='.repeat(60));
  console.log('Resource Monitoring Test');
  console.log('='.repeat(60));
  console.log('This test monitors resource usage under sustained load');
  console.log('Services being monitored:');
  services.forEach(s => console.log(`  - ${s.name}: ${s.url}`));
  console.log('='.repeat(60));

  // Verify all services are available
  const unavailable = [];
  services.forEach(service => {
    const res = http.get(service.url);
    if (res.status !== 200) {
      unavailable.push(service.name);
    }
  });

  if (unavailable.length > 0) {
    console.error(`Warning: The following services are unavailable: ${unavailable.join(', ')}`);
  }

  return {
    startTime: Date.now(),
    services: services.map(s => s.name),
  };
}

// Teardown function
export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('='.repeat(60));
  console.log(`Test completed in ${duration.toFixed(2)} seconds`);
  console.log('='.repeat(60));
}

// Generate HTML report
export function handleSummary(data) {
  const summary = {
    ...data,
    custom: {
      services_monitored: services.map(s => s.name),
      test_type: 'Resource Monitoring',
      note: 'For detailed CPU/Memory metrics, check Docker stats or monitoring dashboards',
    },
  };

  return {
    'reports/resource-monitor-summary.html': htmlReport(summary),
    'reports/resource-monitor-summary.json': JSON.stringify(summary),
    stdout: textSummary(summary, { indent: ' ', enableColors: true }),
  };
}
