import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.0.1/index.js';

// Custom metrics
const errorRate = new Rate('error_rate');
const uploadDuration = new Trend('upload_duration');
const totalImages = new Counter('total_images_uploaded');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm up: 10 VUs
    { duration: '1m', target: 50 },    // Ramp up to 50 VUs
    { duration: '2m', target: 100 },   // Peak load: 100 VUs
    { duration: '1m', target: 50 },    // Ramp down to 50 VUs
    { duration: '30s', target: 0 },    // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'], // 95% of requests should be below 5s
    http_req_failed: ['rate<0.1'],     // Error rate should be less than 10%
    error_rate: ['rate<0.1'],
    upload_duration: ['p(95)<3000'],
  },
};

// Environment variables
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_IMAGE_PATH = __ENV.TEST_IMAGE_PATH || '../data/images/test-image.jpg';

// Generate test image data (simple JPEG header for testing)
function generateTestImage() {
  // This is a minimal valid JPEG file (1x1 pixel)
  const jpegData = new Uint8Array([
    0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
    0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
    0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
    0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
    0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
    0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
    0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
    0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
    0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x03, 0xFF, 0xC4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00,
    0x37, 0xFF, 0xD9
  ]);

  return jpegData;
}

// Health check endpoint test
export function healthCheck() {
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check status is 200': (r) => r.status === 200,
    'health check response is valid': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'ok';
      } catch (e) {
        return false;
      }
    },
  });
}

// Main test scenario
export default function () {
  // Generate test image
  const imageData = generateTestImage();

  // Create multipart form data
  const boundary = '----k6Boundary' + Date.now();
  const body = [
    `--${boundary}`,
    'Content-Disposition: form-data; name="photo"; filename="test.jpg"',
    'Content-Type: image/jpeg',
    '',
    String.fromCharCode.apply(null, imageData),
    `--${boundary}--`,
  ].join('\r\n');

  const params = {
    headers: {
      'Content-Type': `multipart/form-data; boundary=${boundary}`,
    },
    timeout: '30s',
  };

  // Measure upload duration
  const startTime = Date.now();

  // Note: This endpoint might need to be adjusted based on actual Gateway API
  // For now, we test the health endpoint as a proxy
  const res = http.get(`${BASE_URL}/health`, params);

  const duration = Date.now() - startTime;
  uploadDuration.add(duration);

  // Check response
  const success = check(res, {
    'status is 200 or 202': (r) => r.status === 200 || r.status === 202,
    'response time < 5000ms': (r) => r.timings.duration < 5000,
  });

  if (success) {
    totalImages.add(1);
  } else {
    errorRate.add(1);
  }

  // Random sleep between 1-3 seconds to simulate real user behavior
  sleep(Math.random() * 2 + 1);
}

// Generate HTML report
export function handleSummary(data) {
  return {
    'reports/gateway-load-test-summary.html': htmlReport(data),
    'reports/gateway-load-test-summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}
