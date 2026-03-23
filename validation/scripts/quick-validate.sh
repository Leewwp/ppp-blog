#!/bin/bash
#
# Halo Quick Validation Script
# ============================
# Fast validation for code changes (5-10 minutes)
#
# Usage:
#   ./scripts/quick-validate.sh           # Quick check (no k6)
#   ./scripts/quick-validate.sh --full     # Full check with load tests
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }

PASS_COUNT=0
FAIL_COUNT=0

record() {
    local status="$1"
    local message="$2"
    if [ "$status" = "pass" ]; then
        ((PASS_COUNT++))
        log_success "$message"
    else
        ((FAIL_COUNT++))
        log_error "$message"
    fi
}

# Start Halo in background if not running
start_halo_if_needed() {
    if curl -s http://localhost:8090/actuator/health > /dev/null 2>&1; then
        log_info "Halo is already running"
        return 0
    fi

    log_info "Starting Halo..."
    ./gradlew :application:bootRun &
    local pid=$!

    log_info "Waiting for Halo to start (PID: $pid)..."
    local max_wait=120
    local waited=0

    while [ $waited -lt $max_wait ]; do
        if curl -s http://localhost:8090/actuator/health > /dev/null 2>&1; then
            log_success "Halo is ready"
            return 0
        fi
        sleep 2
        waited=$((waited + 2))
        echo -n "."
    done
    echo ""

    log_error "Halo failed to start within $max_wait seconds"
    return 1
}

# Stop Halo
stop_halo() {
    log_info "Stopping Halo..."
    pkill -f "halo" 2>/dev/null || true
    pkill -f "BootRun" 2>/dev/null || true
    sleep 2
}

# 1. Unit Tests
run_unit_tests() {
    echo ""
    echo "=========================================="
    echo " STEP 1: Unit Tests"
    echo "=========================================="

    if ./gradlew :api:test :application:test 2>&1 | tee /tmp/unit-test.log | grep -q "BUILD SUCCESSFUL"; then
        record "pass" "Unit tests passed"
    else
        record "fail" "Unit tests failed"
        tail -50 /tmp/unit-test.log
    fi
}

# 2. Integration Tests
run_integration_tests() {
    echo ""
    echo "=========================================="
    echo " STEP 2: Integration Tests"
    echo "=========================================="

    if ./gradlew :application:test --tests '*IntegrationTest' 2>&1 | tee /tmp/integration-test.log | grep -q "BUILD SUCCESSFUL"; then
        record "pass" "Integration tests passed"
    else
        record "fail" "Integration tests failed"
        tail -50 /tmp/integration-test.log
    fi
}

# 3. API Validation
run_api_validation() {
    echo ""
    echo "=========================================="
    echo " STEP 3: API Validation"
    echo "=========================================="

    start_halo_if_needed

    if ./e2e/scripts/validate-apis.sh health 2>&1 | grep -q "All API validations passed"; then
        record "pass" "API validation passed"
    else
        record "fail" "API validation failed"
    fi
}

# 4. Load Tests (optional)
run_load_tests() {
    echo ""
    echo "=========================================="
    echo " STEP 4: Load Tests (k6)"
    echo "=========================================="

    if ! command -v k6 &> /dev/null; then
        log_warn "k6 not installed, skipping load tests"
        log_warn "Install with: brew install k6"
        return 0
    fi

    start_halo_if_needed

    if k6 run e2e/load-tests/api-load.js --duration 60s --vus 10 2>&1 | tee /tmp/load-test.log | grep -q "checks"; then
        if grep -q "p(95)<500" /tmp/load-test.log || [ $(grep -c "error rate" /tmp/load-test.log) -gt 0 ]; then
            record "pass" "Load tests completed"
        else
            record "fail" "Load tests failed thresholds"
        fi
    else
        record "fail" "Load tests failed"
    fi
}

# 5. Check Code Coverage
check_coverage() {
    echo ""
    echo "=========================================="
    echo " STEP 5: Code Coverage Check"
    echo "=========================================="

    ./gradlew :application:jacocoTestReport 2>&1 | tail -20

    local coverage=$(grep -o 'Total.*%' application/build/reports/jacoco/test/html/index.html 2>/dev/null | head -1)

    if [ -n "$coverage" ]; then
        log_info "Coverage: $coverage"
    fi
}

# Summary
print_summary() {
    echo ""
    echo "=========================================="
    echo " VALIDATION SUMMARY"
    echo "=========================================="
    echo ""
    echo " Passed: $PASS_COUNT"
    echo " Failed: $FAIL_COUNT"
    echo ""
    echo " Results location: ./build/test-results/"
    echo ""

    if [ $FAIL_COUNT -eq 0 ]; then
        log_success "All validations passed!"
        return 0
    else
        log_error "Some validations failed."
        return 1
    fi
}

# Cleanup
cleanup() {
    stop_halo
}

# Main
main() {
    local full_mode=false
    if [ "$1" = "--full" ] || [ "$1" = "-f" ]; then
        full_mode=true
    fi

    echo ""
    echo "=========================================="
    echo " Halo Quick Validation"
    echo " Mode: $(if $full_mode; then echo 'Full'; else echo 'Quick'; fi)"
    echo "=========================================="

    # Cleanup on exit
    trap cleanup EXIT

    # Run validations
    run_unit_tests
    run_integration_tests
    run_api_validation

    if $full_mode; then
        run_load_tests
        check_coverage
    else
        log_info "Skipping load tests (use --full to include)"
    fi

    print_summary
}

main "$@"
