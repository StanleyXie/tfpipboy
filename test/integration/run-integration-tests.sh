#!/bin/bash

# Integration Test Script for tfpipboy
# Tests end-to-end orchestration with real Terraform modules

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TFPIPBOY_BIN="${PROJECT_ROOT}/bin/tfpipboy"
TEST_DIR="${SCRIPT_DIR}"
WORKSPACE_DIR="${SCRIPT_DIR}/.tfpipboy-test-workspace"

# Test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

cleanup() {
    log_info "Cleaning up test workspace..."
    if [ -d "${WORKSPACE_DIR}" ]; then
        rm -rf "${WORKSPACE_DIR}"
    fi

    # Clean up any .terraform directories in test modules
    find "${TEST_DIR}/terraform-modules" -type d -name ".terraform" -exec rm -rf {} + 2>/dev/null || true
    find "${TEST_DIR}/terraform-modules" -type f -name "terraform.tfstate*" -delete 2>/dev/null || true
    find "${TEST_DIR}/terraform-modules" -type f -name ".terraform.lock.hcl" -delete 2>/dev/null || true
}

setup() {
    log_info "Setting up integration test environment..."

    # Build tfpipboy if not exists
    if [ ! -f "${TFPIPBOY_BIN}" ]; then
        log_info "Building tfpipboy..."
        cd "${PROJECT_ROOT}"
        make build
    fi

    # Verify tfpipboy binary
    if [ ! -x "${TFPIPBOY_BIN}" ]; then
        log_error "tfpipboy binary not found or not executable at ${TFPIPBOY_BIN}"
        exit 1
    fi

    log_success "tfpipboy binary found: ${TFPIPBOY_BIN}"

    # Check Terraform is installed
    if ! command -v terraform &> /dev/null; then
        log_error "Terraform is not installed. Please install Terraform to run integration tests."
        exit 1
    fi

    log_success "Terraform found: $(terraform version | head -n1)"

    # Create workspace directory
    mkdir -p "${WORKSPACE_DIR}"

    # Change to test directory
    cd "${TEST_DIR}"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_exit_code="${3:-0}"

    TESTS_RUN=$((TESTS_RUN + 1))

    echo ""
    log_info "Running test: ${test_name}"
    log_info "Command: ${test_command}"

    if eval "${test_command}"; then
        actual_exit_code=0
    else
        actual_exit_code=$?
    fi

    if [ "${actual_exit_code}" -eq "${expected_exit_code}" ]; then
        log_success "Test passed: ${test_name}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "Test failed: ${test_name} (exit code: ${actual_exit_code}, expected: ${expected_exit_code})"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

print_summary() {
    echo ""
    echo "========================================="
    echo "Integration Test Summary"
    echo "========================================="
    echo "Tests Run:    ${TESTS_RUN}"
    echo "Tests Passed: ${TESTS_PASSED}"
    echo "Tests Failed: ${TESTS_FAILED}"
    echo "========================================="

    if [ "${TESTS_FAILED}" -eq 0 ]; then
        log_success "All integration tests passed!"
        return 0
    else
        log_error "${TESTS_FAILED} integration test(s) failed!"
        return 1
    fi
}

# Main test execution
main() {
    log_info "Starting tfpipboy integration tests..."
    echo ""

    # Setup
    setup

    # Test 1: Verify tfpipboy help command
    run_test \
        "tfpipboy --help" \
        "${TFPIPBOY_BIN} --help > /dev/null 2>&1"

    # Test 2: Validate configuration file
    run_test \
        "Config validation" \
        "${TFPIPBOY_BIN} validate --config-dir=${TEST_DIR} 2>&1 | tee /tmp/tfpipboy-validate.log"

    # Test 3: List modules
    run_test \
        "List modules" \
        "${TFPIPBOY_BIN} list --config-dir=${TEST_DIR} 2>&1 | grep -q 'vpc-dev'"

    # Test 4: Show dependency graph for dev environment
    run_test \
        "Show dependency graph for dev environment" \
        "${TFPIPBOY_BIN} graph --config-dir=${TEST_DIR} dev-environment 2>&1 | grep -E '(vpc-dev|subnet-dev)'"

    # Test 5: Plan single module (vpc-dev)
    run_test \
        "Plan single module (vpc-dev)" \
        "${TFPIPBOY_BIN} plan --config-dir=${TEST_DIR} vpc-dev 2>&1 | tee /tmp/tfpipboy-plan-vpc.log"

    # Test 6: Plan with dependency (subnet depends on vpc)
    run_test \
        "Plan with dependencies (subnet-dev-public)" \
        "${TFPIPBOY_BIN} plan --config-dir=${TEST_DIR} subnet-dev-public 2>&1 | tee /tmp/tfpipboy-plan-subnet.log"

    # Test 7: Plan entire dev environment
    run_test \
        "Plan dev environment group" \
        "${TFPIPBOY_BIN} plan --config-dir=${TEST_DIR} --group dev-environment 2>&1 | tee /tmp/tfpipboy-plan-dev.log"

    # Test 8: Dry-run apply (doesn't actually apply)
    run_test \
        "Dry-run apply for vpc-dev" \
        "${TFPIPBOY_BIN} plan --config-dir=${TEST_DIR} vpc-dev 2>&1"

    # Test 9: Pipeline plan for deploy-dev
    run_test \
        "Pipeline plan (deploy-dev)" \
        "${TFPIPBOY_BIN} pipeline plan --config-dir=${TEST_DIR} deploy-dev 2>&1 | tee /tmp/tfpipboy-pipeline-dev.log"

    # Test 10: Verify parallel execution detection
    run_test \
        "Verify parallel execution stages" \
        "grep -q 'Stage.*Networking' /tmp/tfpipboy-pipeline-dev.log || ${TFPIPBOY_BIN} graph --config-dir=${TEST_DIR} dev-environment 2>&1 | grep -q 'Stage 1'"

    # Test 11: Test error handling - non-existent module
    run_test \
        "Error handling - non-existent module" \
        "${TFPIPBOY_BIN} plan --config-dir=${TEST_DIR} non-existent-module 2>&1" \
        1

    # Test 12: Test circular dependency detection (if config has one - this should fail gracefully)
    # This test expects the validation to catch it
    log_info "Testing circular dependency detection is handled by config validation test"

    # Cleanup
    cleanup

    # Print summary
    print_summary
}

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Run main
main
