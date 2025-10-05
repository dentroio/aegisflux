#!/bin/bash

# AegisFlux Backend Safety Shim - Complete Test Suite Runner
# This script runs all test suites in the correct order

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
declare -A TEST_RESULTS
TEST_RESULTS[unit]=0
TEST_RESULTS[integration]=0
TEST_RESULTS[e2e]=0
TEST_RESULTS[security]=0
TEST_RESULTS[performance]=0

# Helper functions
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

log_section() {
    echo -e "\n${BLUE}================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================${NC}\n"
}

run_test_suite() {
    local suite_name="$1"
    local script_path="$2"
    local description="$3"
    
    log_section "Running $suite_name Tests"
    log_info "Description: $description"
    log_info "Script: $script_path"
    
    if [ ! -f "$script_path" ]; then
        log_error "$suite_name test script not found: $script_path"
        TEST_RESULTS[$suite_name]=1
        return 1
    fi
    
    if [ ! -x "$script_path" ]; then
        log_warning "Making $script_path executable"
        chmod +x "$script_path"
    fi
    
    log_info "Starting $suite_name tests..."
    
    if eval "$script_path"; then
        log_success "$suite_name tests completed successfully"
        TEST_RESULTS[$suite_name]=0
        return 0
    else
        log_error "$suite_name tests failed"
        TEST_RESULTS[$suite_name]=1
        return 1
    fi
}

check_prerequisites() {
    log_section "Checking Prerequisites"
    
    local missing_deps=()
    
    # Check required commands
    command -v docker >/dev/null 2>&1 || missing_deps+=("docker")
    command -v docker-compose >/dev/null 2>&1 || missing_deps+=("docker-compose")
    command -v curl >/dev/null 2>&1 || missing_deps+=("curl")
    command -v jq >/dev/null 2>&1 || missing_deps+=("jq")
    command -v psql >/dev/null 2>&1 || missing_deps+=("postgresql-client")
    command -v openssl >/dev/null 2>&1 || missing_deps+=("openssl")
    command -v go >/dev/null 2>&1 || missing_deps+=("go")
    command -v python3 >/dev/null 2>&1 || missing_deps+=("python3")
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing required dependencies:"
        for dep in "${missing_deps[@]}"; do
            echo "  - $dep"
        done
        log_info "Please install missing dependencies and try again"
        return 1
    fi
    
    log_success "All prerequisites are available"
    return 0
}

setup_test_environment() {
    log_section "Setting Up Test Environment"
    
    # Stop any existing test containers
    log_info "Stopping existing test containers..."
    docker-compose -f docker-compose.test.yml down 2>/dev/null || true
    
    # Start test services
    log_info "Starting test services..."
    docker-compose -f docker-compose.test.yml up -d
    
    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    sleep 30
    
    # Check service health
    log_info "Checking service health..."
    
    # Check PostgreSQL
    until pg_isready -h localhost -p 5433 -U testuser 2>/dev/null; do
        log_info "Waiting for PostgreSQL..."
        sleep 5
    done
    log_success "PostgreSQL is ready"
    
    # Check NATS
    until curl -sf http://localhost:8223/healthz >/dev/null 2>&1; do
        log_info "Waiting for NATS..."
        sleep 5
    done
    log_success "NATS is ready"
    
    # Run database migrations
    log_info "Running database migrations..."
    psql "postgres://testuser:testpass@localhost:5433/aegisflux_test" -f backend/internal/db/migrate/001_init.sql
    
    log_success "Test environment setup complete"
}

cleanup_test_environment() {
    log_section "Cleaning Up Test Environment"
    
    log_info "Stopping test services..."
    docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
    
    log_info "Cleaning up test artifacts..."
    rm -rf ./test-certs 2>/dev/null || true
    rm -f performance_report_*.txt 2>/dev/null || true
    
    log_success "Cleanup complete"
}

build_admin_cli() {
    log_section "Building Admin CLI"
    
    if [ ! -f "./backend/cmd/admin/admin" ]; then
        log_info "Building admin CLI..."
        cd backend/cmd/admin
        go build -o admin main.go
        cd - >/dev/null
        log_success "Admin CLI built successfully"
    else
        log_info "Admin CLI already exists"
    fi
}

generate_test_report() {
    log_section "Test Results Summary"
    
    local total_suites=5
    local passed_suites=0
    local failed_suites=0
    
    echo "Test Suite Results:"
    echo "==================="
    
    for suite in unit integration e2e security performance; do
        if [ ${TEST_RESULTS[$suite]} -eq 0 ]; then
            echo -e "✅ $suite: ${GREEN}PASSED${NC}"
            ((passed_suites++))
        else
            echo -e "❌ $suite: ${RED}FAILED${NC}"
            ((failed_suites++))
        fi
    done
    
    echo ""
    echo "Overall Results:"
    echo "================"
    echo "Total test suites: $total_suites"
    echo "Passed: $passed_suites"
    echo "Failed: $failed_suites"
    echo "Success rate: $(( (passed_suites * 100) / total_suites ))%"
    
    if [ $failed_suites -gt 0 ]; then
        echo ""
        log_error "Some test suites failed. Please check the logs above for details."
        return 1
    else
        echo ""
        log_success "All test suites passed! 🎉"
        return 0
    fi
}

show_usage() {
    echo "Usage: $0 [OPTIONS] [TEST_SUITES...]"
    echo ""
    echo "OPTIONS:"
    echo "  -h, --help          Show this help message"
    echo "  --setup-only        Only setup test environment"
    echo "  --cleanup-only      Only cleanup test environment"
    echo "  --skip-setup        Skip environment setup"
    echo "  --skip-cleanup      Skip environment cleanup"
    echo ""
    echo "TEST_SUITES:"
    echo "  unit                Run unit tests only"
    echo "  integration         Run integration tests only"
    echo "  e2e                 Run end-to-end tests only"
    echo "  security            Run security tests only"
    echo "  performance         Run performance tests only"
    echo ""
    echo "If no test suites are specified, all tests will be run."
    echo ""
    echo "Examples:"
    echo "  $0                          # Run all tests"
    echo "  $0 unit integration         # Run unit and integration tests only"
    echo "  $0 --setup-only             # Only setup test environment"
    echo "  $0 --skip-setup unit        # Run unit tests without setup"
}

main() {
    local setup_only=false
    local cleanup_only=false
    local skip_setup=false
    local skip_cleanup=false
    local test_suites=()
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            --setup-only)
                setup_only=true
                shift
                ;;
            --cleanup-only)
                cleanup_only=true
                shift
                ;;
            --skip-setup)
                skip_setup=true
                shift
                ;;
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            unit|integration|e2e|security|performance)
                test_suites+=("$1")
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Handle special modes
    if [ "$setup_only" = true ]; then
        check_prerequisites
        setup_test_environment
        exit 0
    fi
    
    if [ "$cleanup_only" = true ]; then
        cleanup_test_environment
        exit 0
    fi
    
    # Set default test suites if none specified
    if [ ${#test_suites[@]} -eq 0 ]; then
        test_suites=(unit integration e2e security performance)
    fi
    
    # Main test execution
    log_section "AegisFlux Backend Safety Shim - Complete Test Suite"
    log_info "Running test suites: ${test_suites[*]}"
    
    # Check prerequisites
    if ! check_prerequisites; then
        exit 1
    fi
    
    # Setup test environment
    if [ "$skip_setup" = false ]; then
        setup_test_environment
    fi
    
    # Build admin CLI
    build_admin_cli
    
    # Run specified test suites
    for suite in "${test_suites[@]}"; do
        case $suite in
            unit)
                run_test_suite "Unit" "cd backend && go test -race -cover ./..." "Go unit tests with race detection and coverage"
                ;;
            integration)
                run_test_suite "Integration" "./tests/integration_test.sh" "API endpoint and database integration tests"
                ;;
            e2e)
                run_test_suite "End-to-End" "./tests/e2e_test.sh" "Complete workflow tests from bundle creation to agent assignment"
                ;;
            security)
                run_test_suite "Security" "./tests/security_test.sh" "Security tests for mTLS, signatures, and vulnerability prevention"
                ;;
            performance)
                run_test_suite "Performance" "./tests/performance_test.sh" "Performance and load testing"
                ;;
            *)
                log_error "Unknown test suite: $suite"
                exit 1
                ;;
        esac
    done
    
    # Cleanup test environment
    if [ "$skip_cleanup" = false ]; then
        cleanup_test_environment
    fi
    
    # Generate test report
    if generate_test_report; then
        exit 0
    else
        exit 1
    fi
}

# Set up trap for cleanup on exit
trap 'if [ "$skip_cleanup" = false ]; then cleanup_test_environment; fi' EXIT

# Run main function
main "$@"





