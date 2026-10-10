import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const rateLimitedRate = new Rate('rate_limited');
const latencyTrend = new Trend('request_latency');

export const options = {
  stages: [
    { duration: '30s', target: 50 },
    { duration: '1m', target: 100 },
    { duration: '30s', target: 200 },
    { duration: '1m', target: 200 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    errors: ['rate<0.1'],
    rate_limited: ['rate<0.5'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

let authToken = '';

export function setup() {
  const tokenRes = http.post(`${BASE_URL}/gateway/token`, { user_id: 'load-test-user' });
  if (tokenRes.status === 200) {
    const body = JSON.parse(tokenRes.body);
    return { token: body.token };
  }
  return { token: '' };
}

export default function (data) {
  const headers = {};
  if (data.token) {
    headers['Authorization'] = `Bearer ${data.token}`;
  }

  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/api/users`, { headers });
  const latency = Date.now() - startTime;

  latencyTrend.add(latency);

  const isError = res.status >= 500;
  const isRateLimited = res.status === 429;

  errorRate.add(isError);
  rateLimitedRate.add(isRateLimited);

  check(res, {
    'status is not 5xx': (r) => r.status < 500,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  if (isRateLimited) {
    const retryAfter = res.headers['Retry-After'] || '1';
    sleep(parseInt(retryAfter, 10));
  } else {
    sleep(0.1);
  }
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'summary.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const metrics = data.metrics;
  let output = '\n=== Load Test Summary ===\n\n';

  output += `Total Requests: ${metrics.http_reqs?.values?.count || 0}\n`;
  output += `Failed Requests: ${metrics.http_req_failed?.values?.passes || 0}\n`;
  output += `Rate Limited: ${(metrics.rate_limited?.values?.rate || 0) * 100}%\n`;
  output += `Error Rate: ${(metrics.errors?.values?.rate || 0) * 100}%\n\n`;

  output += `Avg Latency: ${metrics.http_req_duration?.values?.avg?.toFixed(2) || 0}ms\n`;
  output += `P95 Latency: ${metrics.http_req_duration?.values?.['p(95)']?.toFixed(2) || 0}ms\n`;
  output += `Max Latency: ${metrics.http_req_duration?.values?.max?.toFixed(2) || 0}ms\n`;

  return output;
}
