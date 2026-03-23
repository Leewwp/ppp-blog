/**
 * Halo API Load Test Suite (k6)
 * ==============================
 *
 * This script performs load testing on the Halo API endpoints.
 *
 * Run: k6 run e2e/load-tests/api-load.js
 * Options:
 *   k6 run e2e/load-tests/api-load.js -e BASE_URL=http://localhost:8090
 *   k6 run e2e/load-tests/api-load.js --vus 100 --duration 30s
 *   k6 run e2e/load-tests/api-load.js --stage RAMP_VUS
 *
 * Interview Answer:
 *   "To validate API robustness under load, I use k6:
 *    1. Define realistic user scenarios (login, browse, create, delete)
 *    2. Run with increasing VUs: 10 -> 50 -> 100 -> 500
 *    3. Monitor: p95 latency < 500ms, error rate < 1%, throughput > 1000 RPS
 *    4. Analyze results in JSON format for trends
 *    5. Set up thresholds for automatic pass/fail"
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const latencyTrend = new Trend('latency');
const throughputTrend = new Trend('throughput');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8090';
const API_BASE = `${BASE_URL}/api`;
const CONSOLE_API = `${BASE_URL}/api.console.halo.run/v1alpha1`;

// Test configuration
export const options = {
    stages: [
        { duration: '2m', target: 10 },   // Ramp up to 10 users
        { duration: '5m', target: 10 },   // Stay at 10 users
        { duration: '2m', target: 50 },   // Ramp up to 50 users
        { duration: '5m', target: 50 },   // Stay at 50 users
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 0 },   // Ramp down
    ],
    thresholds: {
        'http_req_duration': ['p(95)<500'],  // 95% of requests under 500ms
        'http_req_failed': ['rate<0.01'],     // Less than 1% failure rate
        'errors': ['rate<0.05'],              // Less than 5% error rate
    },
};

// Test data
const testPost = {
    post: {
        spec: {
            title: 'Load Test Post',
            slug: 'load-test-' + Date.now(),
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
        metadata: { name: 'load-test-' + Date.now() }
    },
    content: {
        raw: '<p>Load test content</p>',
        content: '<p>Load test content</p>',
        rawType: 'HTML'
    }
};

let authToken = '';
let createdPostName = '';

// Setup - runs once before the test
export function setup() {
    console.log('Setting up load test...');

    // Login to get auth token
    const loginRes = http.post(`${API_BASE}/auth/login`, JSON.stringify({
        username: 'admin',
        password: '123456'
    }), {
        headers: { 'Content-Type': 'application/json' }
    });

    if (loginRes.status === 200 || loginRes.status === 204) {
        const body = JSON.parse(loginRes.body);
        authToken = body.token || body.access_token || loginRes.headers['Authorization'] || '';
    } else {
        // Try form-based login
        const formData = {
            username: 'admin',
            password: '123456'
        };
        const formRes = http.post(`${API_BASE}/auth/login`, formData, {
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
        });
        authToken = formRes.headers['Authorization'] || '';
    }

    console.log(`Auth token obtained: ${authToken.substring(0, 20)}...`);
    return { authToken };
}

// Teardown - runs once after the test
export function teardown(data) {
    console.log('Cleaning up...');
    // Delete test posts if any were created
    if (createdPostName) {
        http.del(`${CONSOLE_API}/posts/${createdPostName}`, null, {
            headers: { 'Authorization': `Bearer ${data.authToken}` }
        });
    }
}

// Default function - main test logic
export default function(data) {
    // Use the auth token from setup
    authToken = data.authToken;

    // Define test scenarios
    const scenarios = [
        { name: 'Health Check', weight: 20, fn: healthCheck },
        { name: 'List Posts', weight: 30, fn: listPosts },
        { name: 'Get Post', weight: 20, fn: getPost },
        { name: 'Create Post', weight: 15, fn: createPost },
        { name: 'List Users', weight: 10, fn: listUsers },
        { name: 'Get Settings', weight: 5, fn: getSettings },
    ];

    // Run scenarios based on weight
    const rand = Math.random() * 100;
    let cumulative = 0;
    for (const scenario of scenarios) {
        cumulative += scenario.weight;
        if (rand < cumulative) {
            scenario.fn(data);
            break;
        }
    }

    sleep(1);
}

// Scenario: Health Check
function healthCheck(data) {
    const res = http.get(`${BASE_URL}/actuator/health`);

    const success = check(res, {
        'health check status is 200': (r) => r.status === 200,
        'health check response time < 200ms': (r) => r.timings.duration < 200,
    });

    errorRate.add(!success);
    latencyTrend.add(res.timings.duration);
}

// Scenario: List Posts (Public API)
function listPosts(data) {
    const res = http.get(`${BASE_URL}/api/content.halo.run/v1alpha1/posts?page=0&size=10`);

    const success = check(res, {
        'list posts status is 200': (r) => r.status === 200,
        'list posts has items': (r) => {
            try {
                const body = JSON.parse(r.body);
                return body.items && body.items.length >= 0;
            } catch (e) {
                return false;
            }
        },
    });

    errorRate.add(!success);
    latencyTrend.add(res.timings.duration);
    throughputTrend.add(1);
}

// Scenario: Get Single Post
function getPost(data) {
    // First get the list to get a post name
    const listRes = http.get(`${BASE_URL}/api/content.halo.run/v1alpha1/posts?page=0&size=1`);

    if (listRes.status !== 200) {
        errorRate.add(1);
        return;
    }

    try {
        const listBody = JSON.parse(listRes.body);
        if (listBody.items && listBody.items.length > 0) {
            const postName = listBody.items[0].post.metadata.name;
            const res = http.get(`${BASE_URL}/api/content.halo.run/v1alpha1/posts/${postName}`);

            const success = check(res, {
                'get post status is 200': (r) => r.status === 200,
            });

            errorRate.add(!success);
            latencyTrend.add(res.timings.duration);
        }
    } catch (e) {
        errorRate.add(1);
    }
}

// Scenario: Create Post (requires auth)
function createPost(data) {
    if (!authToken) {
        console.log('No auth token, skipping create post');
        errorRate.add(1);
        return;
    }

    const testPostCopy = JSON.parse(JSON.stringify(testPost));
    testPostCopy.post.metadata.name = 'load-test-' + Date.now() + '-' + Math.random();
    testPostCopy.post.spec.slug = testPostCopy.post.spec.slug + '-' + Math.random();

    const res = http.post(`${CONSOLE_API}/posts`, JSON.stringify(testPostCopy), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${authToken}`
        }
    });

    const success = check(res, {
        'create post status is 201 or 200': (r) => r.status === 201 || r.status === 200,
    });

    errorRate.add(!success);
    latencyTrend.add(res.timings.duration);

    if (success && res.status === 201) {
        try {
            const body = JSON.parse(res.body);
            createdPostName = body.post.metadata.name;
        } catch (e) {}
    }
}

// Scenario: List Users (requires auth)
function listUsers(data) {
    if (!authToken) {
        console.log('No auth token, skipping list users');
        errorRate.add(1);
        return;
    }

    const res = http.get(`${CONSOLE_API}/users?page=0&size=10`, {
        headers: {
            'Authorization': `Bearer ${authToken}`
        }
    });

    const success = check(res, {
        'list users status is 200': (r) => r.status === 200,
    });

    errorRate.add(!success);
    latencyTrend.add(res.timings.duration);
}

// Scenario: Get Settings
function getSettings(data) {
    const res = http.get(`${BASE_URL}/api.console.halo.run/v1alpha1/settings`);

    const success = check(res, {
        'get settings status is 200': (r) => r.status === 200,
    });

    errorRate.add(!success);
    latencyTrend.add(res.timings.duration);
}

// Handle summary to output results
export function handleSummary(data) {
    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
        'load-test-results.json': JSON.stringify(data),
    };
}

function textSummary(data, options) {
    const indent = options.indent || '';
    let summary = '\n' + indent + '=== Load Test Summary ===\n\n';

    summary += indent + 'Requests:\n';
    summary += indent + `  Total: ${data.metrics.http_reqs?.values?.count || 0}\n`;
    summary += indent + `  Failed: ${data.metrics.http_req_failed?.values?.passes || 0}\n`;
    summary += indent + `  Failure Rate: ${((data.metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%\n\n`;

    summary += indent + 'Latency (ms):\n';
    summary += indent + `  Average: ${(data.metrics.http_req_duration?.values?.avg || 0).toFixed(2)}\n`;
    summary += indent + `  P95: ${(data.metrics.http_req_duration?.values?.['p(95)'] || 0).toFixed(2)}\n`;
    summary += indent + `  P99: ${(data.metrics.http_req_duration?.values?.['p(99)'] || 0).toFixed(2)}\n`;
    summary += indent + `  Max: ${(data.metrics.http_req_duration?.values?.max || 0).toFixed(2)}\n\n`;

    summary += indent + 'Custom Metrics:\n';
    summary += indent + `  Error Rate: ${((data.metrics.errors?.values?.rate || 0) * 100).toFixed(2)}%\n`;

    return summary;
}
