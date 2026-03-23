# Halo Validation Makefile
# =========================
#
# Usage:
#   make help              # Show this help
#   make test              # Run unit tests
#   make test-integration   # Run integration tests
#   make test-api          # Run API validation
#   make test-quick        # Quick validation
#   make test-all          # Run all tests
#   make test-load         # Run load tests
#   make test-stress       # Run stress tests
#   make clean             # Clean results

.PHONY: help test test-integration test-api test-quick test-all test-load test-stress clean

BLUE := \033[0;34m
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
NC := \033[0m

PROJECT_ROOT := $(shell cd $(dir $(lastword $(MAKEFILE_LIST))).. && pwd)
VALIDATION_DIR := $(PROJECT_ROOT)/validation
RESULTS_DIR := $(PROJECT_ROOT)/build/test-results

.DEFAULT_GOAL := help

help:
	@echo ""
	@echo "$(BLUE)Halo Validation Makefile$(NC)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(GREEN)Test Targets:$(NC)"
	@echo "  $(YELLOW)test              $(NC)  Run unit tests"
	@echo "  $(YELLOW)test-integration  $(NC)  Run integration tests"
	@echo "  $(YELLOW)test-api         $(NC)  Run API validation"
	@echo "  $(YELLOW)test-quick       $(NC)  Quick validation (unit + integration + API)"
	@echo "  $(YELLOW)test-all        $(NC)  Run all tests"
	@echo "  $(YELLOW)test-load       $(NC)  Run load tests (requires k6)"
	@echo "  $(YELLOW)test-stress     $(NC)  Run stress tests (requires k6)"
	@echo ""
	@echo "$(GREEN)Utility Targets:$(NC)"
	@echo "  $(YELLOW)clean            $(NC)  Clean test results"
	@echo ""

test:
	@echo "$(BLUE)== Running Unit Tests ==$(NC)"
	cd $(PROJECT_ROOT)/halo && ./gradlew :api:test :application:test

test-integration:
	@echo "$(BLUE)== Running Integration Tests ==$(NC)"
	cd $(PROJECT_ROOT)/halo && ./gradlew :application:test --tests '*IntegrationTest'

test-api:
	@echo "$(BLUE)== Running API Validation ==$(NC)"
	cd $(PROJECT_ROOT)/halo && ./gradlew :application:bootRun &
	sleep 60
	$(VALIDATION_DIR)/scripts/validate-apis.sh all
	pkill -f "halo" 2>/dev/null || true

test-quick:
	@echo "$(BLUE)== Running Quick Validation ==$(NC)"
	@make test
	@make test-integration
	@make test-api

test-load:
	@echo "$(BLUE)== Running Load Tests ==$(NC)"
	@if ! command -v k6 > /dev/null; then \
		echo "$(RED)k6 not found. Install: brew install k6$(NC)"; \
		exit 1; \
	fi
	cd $(PROJECT_ROOT)/halo && ./gradlew :application:bootRun &
	sleep 60
	k6 run $(VALIDATION_DIR)/api-load.js --out json=$(RESULTS_DIR)/load-test-results.json
	pkill -f "halo" 2>/dev/null || true

test-stress:
	@echo "$(BLUE)== Running Stress Tests ==$(NC)"
	@if ! command -v k6 > /dev/null; then \
		echo "$(RED)k6 not found. Install: brew install k6$(NC)"; \
		exit 1; \
	fi
	cd $(PROJECT_ROOT)/halo && ./gradlew :application:bootRun &
	sleep 60
	k6 run $(VALIDATION_DIR)/stress-test.js --out json=$(RESULTS_DIR)/stress-test-results.json
	pkill -f "halo" 2>/dev/null || true

test-all:
	@echo "$(BLUE)== Running Full Test Pipeline ==$(NC)"
	@make test
	@make test-integration
	@make test-api
	@make test-load || echo "$(YELLOW)Load tests skipped$(NC)"
	@make test-stress || echo "$(YELLOW)Stress tests skipped$(NC)"

clean:
	@echo "$(BLUE)== Cleaning Test Results ==$(NC)"
	@rm -rf $(RESULTS_DIR)/*.log $(RESULTS_DIR)/*.json 2>/dev/null || true
	@echo "$(GREEN)Cleaned$(NC)"
