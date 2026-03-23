#!/bin/bash
#
# Halo API Validation Script
# ==========================
# Validates API contracts and endpoint availability
#
# Usage:
#   ./scripts/validate-apis.sh              # Validate all APIs
#   ./scripts/validate-apis.sh health       # Health checks only
#   ./scripts/validate-apis.sh public        # Public APIs only
#   ./scripts/validate-apis.sh console       # Console APIs only
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

BASE_URL="${HALO_URL:-http://localhost:8090}"
RESULTS_DIR="${RESULTS_DIR:-./build/test-results}"
mkdir -p "$RESULTS_DIR"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }

VALIDATION_RESULTS="$RESULTS_DIR/api-validation-$(date +%Y%m%d-%H%M%S).json"
PASS_COUNT=0
FAIL_COUNT=0

# Record result
record_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    local latency="$4"

    echo "{\"test\":\"$test_name\",\"status\":\"$status\",\"message\":\"$message\",\"latency\":$latency,\"timestamp\":\"$(date -Iseconds)\"}" >> "$VALIDATION_RESULTS"

    if [ "$status" = "pass" ]; then
        ((PASS_COUNT++))
        log_success "$test_name"
    else
        ((FAIL_COUNT++))
        log_error "$test_name: $message"
    fi
}

# Test endpoint
test_endpoint() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    local expected_status="${4:-200}"
    local auth="${5:-}"

    local start_time=$(date +%s%3N)

    local response
    if [ -n "$auth" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" -H "Authorization: Bearer $auth" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" 2>/dev/null)
    fi

    local latency=$(($(date +%s%3N) - start_time))
    local http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "$expected_status" ] || ( [ "$expected_status" = "200" ] && [[ "$http_code" =~ ^2 ]] ); then
        record_result "$name" "pass" "HTTP $http_code" "$latency"
        return 0
    else
        record_result "$name" "fail" "Expected $expected_status, got $http_code" "$latency"
        return 1
    fi
}

# Health checks
validate_health() {
    log_info "Validating health endpoints..."

    test_endpoint "Actuator Health" "$BASE_URL/actuator/health" "GET" "200"
    test_endpoint "Actuator Readiness" "$BASE_URL/actuator/health/readiness" "GET" "200"
    test_endpoint "Actuator Liveness" "$BASE_URL/actuator/health/liveness" "GET" "200"
}

# Public APIs
validate_public_apis() {
    log_info "Validating public APIs..."

    test_endpoint "List Posts (Public)" "$BASE_URL/api/content.halo.run/v1alpha1/posts?page=0&size=10" "GET" "200"
    test_endpoint "List Categories" "$BASE_URL/api/content.halo.run/v1alpha1/categories?page=0&size=10" "GET" "200"
    test_endpoint "List Tags" "$BASE_URL/api/content.halo.run/v1alpha1/tags?page=0&size=10" "GET" "200"
    test_endpoint "List SinglePages" "$BASE_URL/api/content.halo.run/v1alpha1/singlepages?page=0&size=10" "GET" "200"
    test_endpoint "Public API Docs" "$BASE_URL/v3/api-docs/apis_public.api_v1alpha1" "GET" "200"
}

# Console APIs (require auth)
validate_console_apis() {
    log_info "Validating console APIs (requires authentication)..."

    # First, try to get auth token
    local auth_response=$(curl -s -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"123456"}' 2>/dev/null)

    local token=""
    if [ -n "$auth_response" ]; then
        token=$(echo "$auth_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -z "$token" ]; then
            token=$(echo "$auth_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        fi
    fi

    if [ -z "$token" ]; then
        log_warn "Could not obtain auth token, skipping console API tests"
        log_warn "Please ensure Halo is running and admin user exists"
        return
    fi

    test_endpoint "Console Posts" "$BASE_URL/api.console.halo.run/v1alpha1/posts?page=0&size=10" "GET" "200" "$token"
    test_endpoint "Console Users" "$BASE_URL/api.console.halo.run/v1alpha1/users?page=0&size=10" "GET" "200" "$token"
    test_endpoint "Console Plugins" "$BASE_URL/api.console.halo.run/v1alpha1/plugins?page=0&size=10" "GET" "200" "$token"
    test_endpoint "Console Settings" "$BASE_URL/api.console.halo.run/v1alpha1/settings" "GET" "200" "$token"
    test_endpoint "Console API Docs" "$BASE_URL/v3/api-docs/apis_console.api_v1alpha1" "GET" "200"
}

# OpenAPI spec validation
validate_openapi() {
    log_info "Validating OpenAPI specifications..."

    # Check aggregated docs
    test_endpoint "OpenAPI Aggregated" "$BASE_URL/v3/api-docs/apis_aggregated.api_v1alpha1" "GET" "200"

    # Try to fetch and validate JSON structure
    local spec=$(curl -s "$BASE_URL/v3/api-docs/apis_aggregated.api_v1alpha1" 2>/dev/null)

    if echo "$spec" | grep -q '"openapi"'; then
        record_result "OpenAPI Version Check" "pass" "Valid OpenAPI 3.x spec" "0"
    else
        record_result "OpenAPI Version Check" "fail" "Invalid OpenAPI specification" "0"
    fi

    if echo "$spec" | grep -q '"paths"'; then
        record_result "OpenAPI Paths Check" "pass" "Contains path definitions" "0"
    else
        record_result "OpenAPI Paths Check" "fail" "Missing path definitions" "0"
    fi
}

# Summary
print_summary() {
    log_info "========================================"
    log_info "API Validation Summary"
    log_info "========================================"
    log_info "Base URL: $BASE_URL"
    log_info "Results: $VALIDATION_RESULTS"
    log_info ""
    log_info "Passed: $PASS_COUNT"
    log_info "Failed: $FAIL_COUNT"
    log_info ""

    if [ $FAIL_COUNT -eq 0 ]; then
        log_success "All API validations passed!"
        return 0
    else
        log_error "Some API validations failed. Please review the results."
        return 1
    fi
}

# Main
main() {
    local mode="${1:-all}"

    echo "[]" > "$VALIDATION_RESULTS"

    log_info "Starting API validation..."
    log_info "Base URL: $BASE_URL"

    case "$mode" in
        health)
            validate_health
            ;;
        public)
            validate_public_apis
            validate_openapi
            ;;
        console)
            validate_console_apis
            ;;
        openapi)
            validate_openapi
            ;;
        all)
            validate_health
            validate_public_apis
            validate_console_apis
            validate_openapi
            ;;
        *)
            echo "Usage: $0 [health|public|console|openapi|all]"
            exit 1
            ;;
    esac

    print_summary
}

main "$@"
