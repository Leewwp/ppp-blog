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
const BASE_URL = __ENV.HALO_URL || __ENV.BASE_URL || 'http://localhost:8090';
const API_BASE = `${BASE_URL}/apis`;
const PUBLIC_API = `${API_BASE}/api.content.halo.run/v1alpha1`;
const CONSOLE_API = `${API_BASE}/api.console.halo.run/v1alpha1`;
const BASIC_AUTH = __ENV.HALO_BASIC_AUTH || 'Basic YWRtaW46MTIzNDU2';
const JSON_ACCEPT_HEADERS = { Accept: 'application/json' };

function getPostName(postItem) {
    if (!postItem) {
        return '';
    }

    if (postItem.post && postItem.post.metadata && postItem.post.metadata.name) {
        return postItem.post.metadata.name;
    }

    if (postItem.metadata && postItem.metadata.name) {
        return postItem.metadata.name;
    }

    return '';
}

function getMetricValue(data, metricName, valueName) {
    if (!data || !data.metrics || !data.metrics[metricName] || !data.metrics[metricName].values) {
        return 0;
    }

    const value = data.metrics[metricName].values[valueName];
    return value || 0;
}

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
};

// Setup
export function setup() {
    console.log('Starting stress test...');
    return { authHeader: BASIC_AUTH };
}

// Teardown
export function teardown(data) {
    console.log('Cleaning up stress test resources...');
    // Note: In stress test, we might not have time to clean up everything
    // This is acceptable as the test environment should be ephemeral
}

// Main test logic
export default function(data) {
    // Calculate current strain level based on VUs
    const vuCount = __VU;
    systemStrain.add(vuCount / 500);  // Normalize to 0-1 based on max 500 VUs

    // Run mixed operations
    const operations = [
        { weight: 25, fn: () => readOnlyOperation('health') },
        { weight: 25, fn: () => readOnlyOperation('listPosts') },
        { weight: 20, fn: () => readOnlyOperation('getPost') },
        { weight: 10, fn: () => readOnlyOperation('listCategories') },
        { weight: 10, fn: () => readOnlyOperation('listTags') },
        { weight: 5, fn: () => authenticatedOperation('listUsers', data) },
        { weight: 5, fn: () => authenticatedOperation('stats', data) },
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
            res = http.get(`${PUBLIC_API}/posts?page=${Math.floor(Math.random() * 10)}&size=20`, {
                headers: JSON_ACCEPT_HEADERS,
            });
            break;
        case 'listCategories':
            res = http.get(`${PUBLIC_API}/categories?page=0&size=20`, {
                headers: JSON_ACCEPT_HEADERS,
            });
            break;
        case 'listTags':
            res = http.get(`${PUBLIC_API}/tags?page=0&size=20`, {
                headers: JSON_ACCEPT_HEADERS,
            });
            break;
        case 'getPost':
            const listRes = http.get(`${PUBLIC_API}/posts?page=0&size=1`, {
                headers: JSON_ACCEPT_HEADERS,
            });
            if (listRes.status !== 200) {
                errors.add(1);
                failedRequests.add(1);
                return;
            }
            try {
                const listBody = JSON.parse(listRes.body);
                const firstPost = listBody.items && listBody.items.length > 0 ? listBody.items[0] : null;
                const postName = getPostName(firstPost);
                if (!postName) {
                    errors.add(1);
                    failedRequests.add(1);
                    return;
                }
                res = http.get(`${PUBLIC_API}/posts/${postName}`, {
                    headers: JSON_ACCEPT_HEADERS,
                });
            } catch (e) {
                errors.add(1);
                failedRequests.add(1);
                return;
            }
            break;
        default:
            res = http.get(`${BASE_URL}/actuator/health`);
    }

    handleResponse(res, type);
}

// Authenticated operations
function authenticatedOperation(type, data) {
    let res;
    switch (type) {
        case 'listUsers':
            res = http.get(`${CONSOLE_API}/users?page=0&size=10`, {
                headers: {
                    'Accept': 'application/json',
                    'Authorization': data.authHeader
                }
            });
            break;
        case 'stats':
            res = http.get(`${CONSOLE_API}/stats`, {
                headers: {
                    'Accept': 'application/json',
                    'Authorization': data.authHeader
                }
            });
            break;
        default:
            res = http.get(`${BASE_URL}/actuator/health`);
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
    const totalRequests = getMetricValue(data, 'successful_requests', 'count');
    const failedReqs = getMetricValue(data, 'failed_requests', 'count');

    return {
        'stdout': textSummary(data, totalRequests, failedReqs),
    };
}

function textSummary(data, totalRequests, failedReqs) {
    const allRequests = totalRequests + failedReqs;
    const failureRate = allRequests > 0 ? ((failedReqs / allRequests) * 100).toFixed(2) : '0.00';
    let summary = '\n=== Stress Test Summary ===\n\n';

    summary += `Total VUs Reached: ${getMetricValue(data, 'system_strain', 'max')}\n`;
    summary += `Successful Requests: ${totalRequests}\n`;
    summary += `Failed Requests: ${failedReqs}\n`;
    summary += `Failure Rate: ${failureRate}%\n\n`;

    summary += 'Latency Breakdown:\n';
    summary += `  Avg: ${getMetricValue(data, 'http_req_duration', 'avg').toFixed(2)}ms\n`;
    summary += `  P50: ${getMetricValue(data, 'http_req_duration', 'med').toFixed(2)}ms\n`;
    summary += `  P95: ${getMetricValue(data, 'http_req_duration', 'p(95)').toFixed(2)}ms\n`;
    summary += `  P99: ${getMetricValue(data, 'http_req_duration', 'p(99)').toFixed(2)}ms\n`;
    summary += `  Max: ${getMetricValue(data, 'http_req_duration', 'max').toFixed(2)}ms\n\n`;

    summary += 'Recommendations:\n';
    if (failureRate > 10) {
        summary += '  - High failure rate detected. Review error logs.\n';
    }
    if (getMetricValue(data, 'http_req_duration', 'p(95)') > 1000) {
        summary += '  - High latency at P95. Consider scaling or optimizing.\n';
    }

    return summary;
}
