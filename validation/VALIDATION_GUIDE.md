# Halo Test & Validation Framework

Complete guide for verifying code changes and ensuring system robustness.

## Quick Start

```bash
# 1. Start Halo application
cd halo
./gradlew :application:bootRun

# 2. In another terminal, run the full validation pipeline
./e2e/scripts/run-all-tests.sh all
```

## Validation Pipeline Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    CODE CHANGE MADE                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 1: Unit Tests                                             │
│  Command: ./gradlew :api:test :application:test                  │
│  Purpose: Verify individual components work correctly           │
│  Time: ~2-5 minutes                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 2: Integration Tests                                      │
│  Command: ./gradlew :application:test                           │
│  Purpose: Verify component interactions work correctly           │
│  Time: ~5-10 minutes                                            │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 3: E2E Tests (if docker available)                         │
│  Command: ./e2e/start.sh                                        │
│  Purpose: Verify complete user workflows                        │
│  Time: ~10-15 minutes                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 4: Load Tests (if k6 installed)                           │
│  Command: k6 run e2e/load-tests/api-load.js                     │
│  Purpose: Verify system performance under normal load           │
│  Time: ~15-20 minutes                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 5: Stress Tests (if k6 installed)                        │
│  Command: k6 run e2e/load-tests/stress-test.js                  │
│  Purpose: Find system breaking point                            │
│  Time: ~15-20 minutes                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│  STEP 6: API Validation                                          │
│  Command: ./e2e/scripts/validate-apis.sh                         │
│  Purpose: Verify all API endpoints are functional               │
│  Time: ~2-5 minutes                                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    VALIDATION COMPLETE                          │
└─────────────────────────────────────────────────────────────────┘
```

## Interview Answer Template

**Q: "How do you verify the robustness of your code changes?"**

**A:**
```
When I make any code change, I follow a systematic validation pipeline:

1. Unit Tests (2-5 min)
   - Run: ./gradlew :api:test :application:test
   - Verify: All unit tests pass
   - Coverage target: 80%+ via Jacoco
   - Purpose: Verify individual components

2. Integration Tests (5-10 min)
   - Run: ./gradlew :application:test --tests '*IntegrationTest'
   - Verify: Component interactions work
   - Focus: Database operations, controller endpoints

3. E2E Tests (10-15 min)
   - Run: ./e2e/start.sh
   - Verify: Complete user workflows
   - Uses: Real database, real HTTP calls

4. Load Testing (15-20 min)
   - Run: k6 run e2e/load-tests/api-load.js
   - Metrics:
     * P95 latency < 500ms
     * Error rate < 1%
     * Throughput > 1000 RPS
   - Purpose: Verify performance under load

5. Stress Testing (15-20 min)
   - Run: k6 run e2e/load-tests/stress-test.js
   - Purpose: Find breaking points
   - Identify: Max RPS, failure modes, recovery

6. API Contract Validation (2-5 min)
   - Run: ./e2e/scripts/validate-apis.sh
   - Verify: All endpoints respond correctly
   - Check: OpenAPI spec compliance

Total pipeline: ~45-60 minutes
```

## Individual Test Commands

### Unit Tests
```bash
# Run all unit tests
./gradlew :api:test :application:test

# Run specific module tests
./gradlew :api:test
./gradlew :application:test

# Run with coverage
./gradlew :application:test jacocoTestReport

# Run specific test class
./gradlew :application:test --tests '*PostReconcilerTest*

# Run tests in watch mode (continuous)
./gradlew :application:test --continuous
```

### Integration Tests
```bash
# Run all integration tests
./gradlew :application:test --tests '*IntegrationTest'

# Run specific integration test
./gradlew :application:test --tests '*CommentServiceImplIntegrationTest*

# Run with detailed output
./gradlew :application:test --tests '*IntegrationTest' --info

# Debug integration tests
./gradlew :application:test --tests '*IntegrationTest' --debug
```

### E2E Tests
```bash
# Start full E2E test suite
./e2e/start.sh

# With specific compose file
./e2e/start.sh compose-postgres.yaml

# Run specific E2E scenario (after start)
# See e2e/testsuite.yaml for available scenarios
```

### Load Tests (requires k6)
```bash
# Install k6
brew install k6  # macOS
# or: choco install k6  # Windows

# Basic load test
k6 run e2e/load-tests/api-load.js

# With custom target
k6 run e2e/load-tests/api-load.js --vus 100 --duration 60s

# With environment
HALO_URL=http://localhost:8090 k6 run e2e/load-tests/api-load.js

# Output to file
k6 run e2e/load-tests/api-load.js --out json=results.json

# Cloud execution (if using k6 cloud)
k6 cloud e2e/load-tests/api-load.js
```

### Stress Tests (requires k6)
```bash
# Basic stress test
k6 run e2e/load-tests/stress-test.js

# Aggressive stress test
k6 run e2e/load-tests/stress-test.js --vus 500 --duration 5m

# With thresholds
k6 run e2e/load-tests/stress-test.js \
    --threshold 'http_req_duration=p(95)<1000' \
    --threshold 'http_req_failed=rate<0.1'
```

### API Validation
```bash
# Validate all APIs
./e2e/scripts/validate-apis.sh all

# Health checks only
./e2e/scripts/validate-apis.sh health

# Public APIs only
./e2e/scripts/validate-apis.sh public

# Console APIs only (requires auth)
./e2e/scripts/validate-apis.sh console

# OpenAPI spec only
./e2e/scripts/validate-apis.sh openapi

# Custom URL
HALO_URL=http://staging:8090 ./e2e/scripts/validate-apis.sh all
```

## Understanding Test Results

### Unit Test Results
```bash
# View report
open application/build/reports/tests/test/index.html

# View coverage
open application/build/reports/jacoco/test/html/index.html
```

### Load Test Results
```bash
# View JSON results
cat load-test-results.json | jq

# Extract key metrics
cat load-test-results.json | jq '.metrics.http_req_duration.values'
```

### API Validation Results
```bash
# Results are saved to
cat build/test-results/api-validation-*.json | jq

# Summary
./e2e/scripts/validate-apis.sh all 2>&1 | tail -20
```

## Test Categories

| Category | Duration | Frequency | Purpose |
|----------|----------|-----------|---------|
| Unit | 2-5 min | Every commit | Fast feedback |
| Integration | 5-10 min | Every PR | Component interactions |
| E2E | 10-15 min | Every PR | Full workflows |
| Load | 15-20 min | Daily/Pre-release | Performance validation |
| Stress | 15-20 min | Weekly/Pre-release | Breaking point discovery |
| API | 2-5 min | Every PR | Contract validation |

## Troubleshooting

### Unit Tests Failing
```bash
# Clean and rebuild
./gradlew clean :application:test

# Run with stacktrace
./gradlew :application:test --stacktrace

# Debug
./gradlew :application:test --tests '*TestClass*' --debug
```

### E2E Tests Failing
```bash
# Check if Halo is running
curl http://localhost:8090/actuator/health

# Check Docker
docker ps
docker-compose -f e2e/compose.yaml logs halo

# Run E2E locally without docker
./gradlew :application:bootRun
# Then in another terminal:
curl http://localhost:8090/api/content.halo.run/v1alpha1/posts
```

### Load Tests Not Working
```bash
# Check k6 installation
k6 version

# If not installed:
brew install k6

# Or use Docker
docker run -it --rm grafana/k6 run \
    -e BASE_URL=http://host.docker.internal:8090 \
    /e2e/load-tests/api-load.js
```

## Adding New Tests

### Adding a New E2E Test Case

Edit `e2e/testsuite.yaml`:

```yaml
- name: myNewTest
  request:
    api: /api.console.halo.run/v1alpha1/my-endpoint
    method: POST
    header:
      Authorization: "{{.param.auth}}"
      Content-Type: application/json
    body: |
      {
        "key": "value"
      }
  expect:
    statusCode: 201
    verify:
      - data.metadata.name != ""
```

### Adding a New Load Test Scenario

Edit `e2e/load-tests/api-load.js`:

```javascript
function myNewScenario(data) {
    const res = http.post(`${CONSOLE_API}/my-endpoint`,
        JSON.stringify({ key: 'value' }),
        {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${data.authToken}`
            }
        }
    );

    check(res, {
        'my check passed': (r) => r.status === 201,
    });

    errorRate.add(res.status !== 201);
    latencyTrend.add(res.timings.duration);
}
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Validation Pipeline
on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up JDK 21
        uses: actions/setup-java@v4
        with:
          java-version: 21

      - name: Unit Tests
        run: ./gradlew :api:test :application:test

      - name: Integration Tests
        run: ./gradlew :application:test --tests '*IntegrationTest'

      - name: Load Tests
        if: github.event_name == 'push'
        run: |
          ./gradlew :application:bootRun &
          sleep 30
          k6 run e2e/load-tests/api-load.js

      - name: API Validation
        run: ./e2e/scripts/validate-apis.sh
```

## Key Metrics to Monitor

| Metric | Good | Warning | Critical |
|--------|------|---------|----------|
| Unit Test Pass Rate | 100% | 95-99% | <95% |
| Integration Test Pass Rate | 100% | 95-99% | <95% |
| Load Test P95 Latency | <200ms | 200-500ms | >500ms |
| Load Test Error Rate | <0.1% | 0.1-1% | >1% |
| Stress Test Max RPS | >1000 | 500-1000 | <500 |
| API Validation Pass Rate | 100% | 90-99% | <90% |
