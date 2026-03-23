#!/bin/bash
#
# Halo E2E Test Runner - Complete Validation Pipeline
# ====================================================
# This script provides a complete validation pipeline for code changes.
#
# Usage:
#   ./scripts/run-all-tests.sh              # Run all tests
#   ./scripts/run-all-tests.sh unit         # Unit tests only
#   ./scripts/run-all-tests.sh integration  # Integration tests only
#   ./scripts/run-all-tests.sh e2e          # E2E tests only
#   ./scripts/run-all-tests.sh load         # Load tests only
#   ./scripts/run-all-tests.sh validate     # Validation only
#
# Interview Answer Template:
#   "When I make a code change, I verify it through a complete pipeline:
#    1. First, run unit tests: ./gradlew :api:test :application:test
#    2. Then, run integration tests with coverage: ./gradlew :application:integTest
#    3. Run E2E tests: ./e2e/start.sh
#    4. Run load tests: k6 run e2e/load-tests/api-load.js
#    5. Validate API contracts: ./scripts/validate-apis.sh
#    6. Check results in: build/reports/"
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Test results directory
RESULTS_DIR="$PROJECT_ROOT/build/test-results"
COVERAGE_DIR="$PROJECT_ROOT/build/coverage"
mkdir -p "$RESULTS_DIR" "$COVERAGE_DIR"

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo ""
    echo "=========================================="
    echo " $1"
    echo "=========================================="
}

# =============================================================================
# STEP 1: Unit Tests
# =============================================================================
run_unit_tests() {
    print_header "STEP 1: Running Unit Tests"

    log_info "Running API module unit tests..."
    ./gradlew :api:test --info 2>&1 | tee "$RESULTS_DIR/api-unit.log"

    log_info "Running Application module unit tests..."
    ./gradlew :application:test --info 2>&1 | tee "$RESULTS_DIR/application-unit.log"

    log_success "Unit tests completed"
    log_info "Results: $RESULTS_DIR/"
}

# =============================================================================
# STEP 2: Integration Tests
# =============================================================================
run_integration_tests() {
    print_header "STEP 2: Running Integration Tests"

    log_info "Running integration tests with coverage..."
    ./gradlew :application:test --tests '*IntegrationTest' --info 2>&1 | tee "$RESULTS_DIR/integration.log"

    log_success "Integration tests completed"
}

# =============================================================================
# STEP 3: E2E Tests
# =============================================================================
run_e2e_tests() {
    print_header "STEP 3: Running E2E Tests"

    log_info "Starting E2E test suite..."
    ./e2e/start.sh

    log_success "E2E tests completed"
}

# =============================================================================
# STEP 4: Load Tests (k6)
# =============================================================================
run_load_tests() {
    print_header "STEP 4: Running Load Tests"

    if ! command -v k6 &> /dev/null; then
        log_warn "k6 not found. Installing..."
        brew install k6 2>/dev/null || (
            curl -sL https://github.com/grafana/k6/releases/download/v0.49.0/k6-v0.49.0-linux-amd64.tar.gz | tar xz -C /usr/local/bin
        )
    fi

    log_info "Starting Halo application for load testing..."
    ./gradlew :application:bootRun &
    APP_PID=$!
    sleep 30  # Wait for startup

    # Run load tests
    k6 run e2e/load-tests/api-load.js --out json="$RESULTS_DIR/load-test-results.json"

    # Cleanup
    kill $APP_PID 2>/dev/null || true

    log_success "Load tests completed"
}

# =============================================================================
# STEP 5: API Validation
# =============================================================================
run_api_validation() {
    print_header "STEP 5: Validating API Contracts"

    log_info "Validating OpenAPI specifications..."
    ./scripts/validate-apis.sh

    log_success "API validation completed"
}

# =============================================================================
# STEP 6: Generate Report
# =============================================================================
generate_report() {
    print_header "STEP 6: Generating Test Report"

    log_info "Collecting test results..."

    echo "# Halo Test Report" > "$RESULTS_DIR/TEST_REPORT.md"
    echo "" >> "$RESULTS_DIR/TEST_REPORT.md"
    echo "Generated: $(date)" >> "$RESULTS_DIR/TEST_REPORT.md"
    echo "" >> "$RESULTS_DIR/TEST_REPORT.md"

    # Unit test summary
    echo "## Unit Tests" >> "$RESULTS_DIR/TEST_REPORT.md"
    grep -r "tests completed" "$RESULTS_DIR"/*.log 2>/dev/null | head -5 >> "$RESULTS_DIR/TEST_REPORT.md" || echo "No unit test results" >> "$RESULTS_DIR/TEST_REPORT.md"
    echo "" >> "$RESULTS_DIR/TEST_REPORT.md"

    # Load test summary
    echo "## Load Tests" >> "$RESULTS_DIR/TEST_REPORT.md"
    if [ -f "$RESULTS_DIR/load-test-results.json" ]; then
        tail -20 "$RESULTS_DIR/load-test-results.json" >> "$RESULTS_DIR/TEST_REPORT.md"
    fi

    log_success "Report generated: $RESULTS_DIR/TEST_REPORT.md"
}

# =============================================================================
# Main
# =============================================================================
main() {
    local mode="${1:-all}"

    log_info "Halo Test Pipeline - Mode: $mode"
    log_info "Results will be saved to: $RESULTS_DIR"

    case "$mode" in
        unit)
            run_unit_tests
            ;;
        integration)
            run_integration_tests
            ;;
        e2e)
            run_e2e_tests
            ;;
        load)
            run_load_tests
            ;;
        validate)
            run_api_validation
            ;;
        all)
            run_unit_tests
            run_integration_tests
            # run_e2e_tests  # Uncomment if docker available
            # run_load_tests  # Uncomment if k6 installed
            run_api_validation
            generate_report
            ;;
        *)
            log_error "Unknown mode: $mode"
            echo "Usage: $0 [unit|integration|e2e|load|validate|all]"
            exit 1
            ;;
    esac

    log_success "Test pipeline completed!"
}

main "$@"
