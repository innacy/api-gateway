// loadtest/gateway.js — k6 load test for the API Gateway extract endpoints
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errorRate = new Rate('errors');
const extractDuration = new Trend('extract_duration');

export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up
    { duration: '1m', target: 500 },    // Sustained load
    { duration: '30s', target: 1000 },  // Peak load
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<200'],
    errors: ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-key-pro';

const endpoints = [
  {
    name: 'extract_url',
    path: '/api/v1/extract/url',
    body: JSON.stringify({ url: 'https://example.com', include_links: true }),
  },
  {
    name: 'extract_data',
    path: '/api/v1/extract/data',
    body: JSON.stringify({ content: '{"name": "test", "value": 42}', schema_hint: 'json' }),
  },
  {
    name: 'extract_metadata',
    path: '/api/v1/extract/metadata',
    body: JSON.stringify({ url: 'https://example.com', metadata_types: ['headers'] }),
  },
];

export function setup() {
  const res = http.get(`${BASE_URL}/api/v1/health`);
  check(res, { 'gateway is healthy': (r) => r.status === 200 });
  return { baseUrl: BASE_URL };
}

export default function () {
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
    },
    tags: { name: endpoint.name },
  };

  const res = http.post(`${BASE_URL}${endpoint.path}`, endpoint.body, params);

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response has body': (r) => r.body.length > 0,
  });

  errorRate.add(res.status !== 200);
  extractDuration.add(res.timings.duration);

  sleep(0.1);
}
