/**
 * Halo Stress Test Suite (k6)
 * ===========================
 *
 * This script performs stress testing on the Halo API to find breaking points.
 *
 * Run: k6 run e2e/load-tests/stress-test.js
 *
 * Interview Answer:
 *   "For stress testing, I progressively increase load until the system breaks:
 *    1. Start with baseline: 10 users, 1 minute
 *    2. Increase by 50% each stage until failure
 *    3. Identify: max RPS, max concurrent users, failure modes
 *    4. Measure: response time degradation, error types, recovery time
 *    5. Document: bottleneck points, resource limits, failure recovery"
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Gauge } from 'k6/metrics';

// Custom metrics
const errors = new Rate('errors');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');
const systemStrain = new Gauge('system_strain');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8090';
const API_BASE = `${BASE_URL}/api`;

// Stress test configuration - aggressive escalation
export const options = {
    stages: [
        // Phase 1: Light load (baseline)
        { duration: '1m', target: 10 },
        // Phase 2: Moderate stress
        { duration: '2m', target: 50 },
        // Phase 3: Heavy stress
        { duration: '2m', target: 100 },
        // Phase 4: Breaking point
        { duration: '3m', target: 200 },
        // Phase 5: Beyond breaking point
        { duration: '3m', target: 500 },
        // Phase 6: Recovery check
        { duration: '2m', target: 10 },
    ],
    thresholds: {
        // We expect errors at high load - this is informational
        'http_req_duration': ['p(95)<2000'],
        'http_req_failed': ['rate<0.50'],  // Up to 50% failure is acceptable for stress test
    },
};

// Test data
let authToken = '';
let createdPostNames = [];

// Setup
export function setup() {
    console.log('Starting stress test...');

    // Try to login
    const loginRes = http.post(`${API_BASE}/auth/login`, JSON.stringify({
        username: 'admin',
        password: '123456'
    }), {
        headers: { 'Content-Type': 'application/json' }
    });

    if (loginRes.status === 200) {
        try {
            const body = JSON.parse(loginRes.body);
            authToken = body.token || body.access_token || '';
        } catch (e) {}
    }

    return { authToken };
}

// Teardown
export function teardown(data) {
    console.log('Cleaning up stress test resources...');
    // Note: In stress test, we might not have time to clean up everything
    // This is acceptable as the test environment should be ephemeral
}

// Main test logic
export default function(data) {
    authToken = data.authToken;

    // Calculate current strain level based on VUs
    const vuCount = __VU;
    systemStrain.add(vuCount / 500);  // Normalize to 0-1 based on max 500 VUs

    // Run mixed operations
    const operations = [
        { weight: 30, fn: () => readOnlyOperation('health') },
        { weight: 25, fn: () => readOnlyOperation('listPosts') },
        { weight: 20, fn: () => readOnlyOperation('getPost') },
        { weight: 15, fn: () => readWriteOperation('createPost') },
        { weight: 10, fn: () => authenticatedOperation('listUsers') },
    ];

    const rand = Math.random() * 100;
    let cumulative = 0;
    for (const op of operations) {
        cumulative += op.weight;
        if (rand < cumulative) {
            op.fn();
            break;
        }
    }

    sleep(0.5);  // Shorter sleep for more aggressive load
}

// Read-only operations (should handle high load better)
function readOnlyOperation(type) {
    let res;
    switch (type) {
        case 'health':
            res = http.get(`${BASE_URL}/actuator/health`);
            break;
        case 'listPosts':
            res = http.get(`${BASE_URL}/api/content.halo.run/v1alpha1/posts?page=${Math.floor(Math.random() * 10)}&size=20`);
            break;
        case 'getPost':
            // Try to get a specific post
            res = http.get(`${BASE_URL}/api/content.halo.run/v1alpha1/posts/stress-test-post`);
            break;
        default:
            res = http.get(`${BASE_URL}/actuator/health`);
    }

    handleResponse(res, type);
}

// Read-write operations (more stressful)
function readWriteOperation(type) {
    if (type === 'createPost') {
        const postName = `stress-${Date.now()}-${__VU}-${__ITER}`;
        createdPostNames.push(postName);

        const testPost = {
            post: {
                spec: {
                    title: `Stress Test ${postName}`,
                    slug: postName,
                    template: '',
                    cover: '',
                    deleted: false,
                    publish: false,
                    pinned: false,
                    allowComment: true,
                    visible: 'PUBLIC',
                    priority: 0,
                    excerpt: { autoGenerate: true, raw: '' },
                    categories: [],
                    tags: [],
                    htmlMetas: []
                },
                apiVersion: 'content.halo.run/v1alpha1',
                kind: 'Post',
                metadata: { name: postName }
            },
            content: {
                raw: '<p>Stress test content</p>',
                content: '<p>Stress test content</p>',
                rawType: 'HTML'
            }
        };

        const res = http.post(`${BASE_URL}/api.console.halo.run/v1alpha1/posts`,
            JSON.stringify(testPost),
            {
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                }
            }
        );

        handleResponse(res, 'createPost');
    }
}

// Authenticated operations
function authenticatedOperation(type) {
    if (!authToken) {
        errors.add(1);
        return;
    }

    let res;
    switch (type) {
        case 'listUsers':
            res = http.get(`${BASE_URL}/api.console.halo.run/v1alpha1/users?page=0&size=10`, {
                headers: { 'Authorization': `Bearer ${authToken}` }
            });
            break;
        default:
            res = http.get(`${BASE_URL}/api.console.halo.run/v1alpha1/settings`, {
                headers: { 'Authorization': `Bearer ${authToken}` }
            });
    }

    handleResponse(res, type);
}

// Handle response and update metrics
function handleResponse(res, operation) {
    const isSuccess = res.status >= 200 && res.status < 400;
    const isServerError = res.status >= 500;
    const isClientError = res.status >= 400 && res.status < 500;

    if (isSuccess) {
        successfulRequests.add(1);
    } else {
        failedRequests.add(1);

        if (isServerError) {
            console.log(`[WARN] Server error on ${operation}: ${res.status} at VU ${__VU}`);
        }
    }

    errors.add(!isSuccess);

    // Log if response time exceeds threshold
    if (res.timings.duration > 5000) {
        console.log(`[WARN] Slow response on ${operation}: ${res.timings.duration}ms at VU ${__VU}`);
    }
}

// Custom summary
export function handleSummary(data) {
    const totalRequests = data.metrics.successful_requests?.values?.passes || 0;
    const failedReqs = data.metrics.failed_requests?.values?.fails || 0;
    const failureRate = (failedReqs / (totalRequests + failedReqs) * 100).toFixed(2);

    return {
        'stdout': textSummary(data),
        'stress-test-results.json': JSON.stringify(data),
    };
}

function textSummary(data) {
    let summary = '\n=== Stress Test Summary ===\n\n';

    summary += `Total VUs Reached: ${data.metrics.system_strain?.values?.max || 0}\n`;
    summary += `Successful Requests: ${data.metrics.successful_requests?.values?.passes || 0}\n`;
    summary += `Failed Requests: ${data.metrics.failed_requests?.values?.fails || 0}\n`;
    summary += `Failure Rate: ${failureRate}%\n\n`;

    summary += 'Latency Breakdown:\n';
    summary += `  Avg: ${(data.metrics.http_req_duration?.values?.avg || 0).toFixed(2)}ms\n`;
    summary += `  P50: ${(data.metrics.http_req_duration?.values?.med || 0).toFixed(2)}ms\n`;
    summary += `  P95: ${(data.metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(2)}ms\n`;
    summary += `  P99: ${(data.metrics.http_req_duration?.values?.['p(99)'] || 0).toFixed(2)}ms\n`;
    summary += `  Max: ${(data.metrics.http_req_duration?.values?.max || 0).toFixed(2)}ms\n\n`;

    summary += 'Recommendations:\n';
    if (failureRate > 10) {
        summary += '  - High failure rate detected. Review error logs.\n';
    }
    if ((data.metrics.http_req_duration?.values?.['p(95)'] || 0) > 1000) {
        summary += '  - High latency at P95. Consider scaling or optimizing.\n';
    }

    return summary;
}
