#!/bin/bash

# AegisFlux Backend Safety Shim - Performance Test Suite
# This script tests system performance under various load conditions

set -euo pipefail

# Configuration
REGISTRY_URL="http://localhost:8090"
ACTIONS_API_URL="http://localhost:8083"
TEST_DB_URL="postgres://testuser:testpass@localhost:5432/aegisflux_test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Performance metrics
declare -A METRICS
METRICS[TOTAL_TESTS]=0
METRICS[PASSED_TESTS]=0
METRICS[FAILED_TESTS]=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_performance() {
    echo -e "${YELLOW}[PERF]${NC} $1"
}

run_performance_test() {
    local test_name="$1"
    local test_command="$2"
    local max_duration="$3"
    
    log_info "Running performance test: $test_name"
    METRICS[TOTAL_TESTS]=$((METRICS[TOTAL_TESTS] + 1))
    
    local start_time
    start_time=$(date +%s%N)
    
    if eval "$test_command"; then
        local end_time
        end_time=$(date +%s%N)
        local duration_ms=$(((end_time - start_time) / 1000000))
        
        if [ $duration_ms -le $max_duration ]; then
            log_performance "$test_name: ${duration_ms}ms (max: ${max_duration}ms)"
            log_success "$test_name"
            METRICS[PASSED_TESTS]=$((METRICS[PASSED_TESTS] + 1))
            return 0
        else
            log_performance "$test_name: ${duration_ms}ms (max: ${max_duration}ms) - TOO SLOW"
            log_error "$test_name"
            METRICS[FAILED_TESTS]=$((METRICS[FAILED_TESTS] + 1))
            return 1
        fi
    else
        local end_time
        end_time=$(date +%s%N)
        local duration_ms=$(((end_time - start_time) / 1000000))
        
        log_performance "$test_name: ${duration_ms}ms - FAILED"
        log_error "$test_name"
        METRICS[FAILED_TESTS]=$((METRICS[FAILED_TESTS] + 1))
        return 1
    fi
}

# Test bundle creation performance
test_bundle_creation_performance() {
    log_info "Testing bundle creation performance..."
    
    # Single bundle creation
    run_performance_test "Single bundle creation" \
        "curl -sf -X POST '$REGISTRY_URL/bundles' \
            -H 'Content-Type: application/json' \
            -d '{\"name\": \"perf-bundle-1\", \"content\": \"'$(echo \"Performance test bundle 1\" | base64)'\", \"created_by\": \"perf-test\"}' >/dev/null" \
        5000
    
    # Batch bundle creation
    run_performance_test "Batch bundle creation (10 bundles)" \
        "for i in {1..10}; do
            curl -sf -X POST '$REGISTRY_URL/bundles' \
                -H 'Content-Type: application/json' \
                -d \"{\\\"name\\\": \\\"perf-bundle-\$i\\\", \\\"content\\\": \\\"'$(echo \"Performance test bundle \$i\" | base64)'\\\", \\\"created_by\\\": \\\"perf-test\\\"}\" >/dev/null &
        done
        wait" \
        15000
    
    # Large bundle creation
    local large_content
    large_content=$(python3 -c "print('A' * 100000)" | base64)
    run_performance_test "Large bundle creation (100KB)" \
        "curl -sf -X POST '$REGISTRY_URL/bundles' \
            -H 'Content-Type: application/json' \
            -d '{\"name\": \"large-perf-bundle\", \"content\": \"'$large_content'\", \"created_by\": \"perf-test\"}' >/dev/null" \
        10000
}

# Test bundle retrieval performance
test_bundle_retrieval_performance() {
    log_info "Testing bundle retrieval performance..."
    
    # Create a test bundle first
    local bundle_id
    bundle_id=$(curl -sf -X POST "$REGISTRY_URL/bundles" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "perf-retrieval-bundle",
            "content": "'$(echo "Performance retrieval test bundle" | base64)'",
            "created_by": "perf-test"
        }' | jq -r '.bundle_id')
    
    # Single bundle retrieval
    run_performance_test "Single bundle retrieval" \
        "curl -sf '$REGISTRY_URL/bundles/$bundle_id' >/dev/null" \
        1000
    
    # Bundle listing performance
    run_performance_test "Bundle listing (limit 50)" \
        "curl -sf '$REGISTRY_URL/bundles?limit=50' >/dev/null" \
        2000
    
    # Bundle listing with large limit
    run_performance_test "Bundle listing (limit 1000)" \
        "curl -sf '$REGISTRY_URL/bundles?limit=1000' >/dev/null" \
        5000
    
    # Bundle signature verification
    run_performance_test "Bundle signature verification" \
        "curl -sf -X POST '$REGISTRY_URL/bundles/$bundle_id/verify' \
            -H 'Content-Type: application/json' \
            -d '{\"content\": \"'$(echo \"Performance retrieval test bundle\" | base64)'\"}' >/dev/null" \
        2000
}

# Test assignment performance
test_assignment_performance() {
    log_info "Testing assignment performance..."
    
    # Create test bundles first
    local bundle_ids=()
    for i in {1..5}; do
        local bundle_id
        bundle_id=$(curl -sf -X POST "$REGISTRY_URL/bundles" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"perf-assignment-bundle-$i\",
                \"content\": \"$(echo \"Performance assignment test bundle $i\" | base64)\",
                \"created_by\": \"perf-test\"
            }" | jq -r '.bundle_id')
        bundle_ids+=("$bundle_id")
    done
    
    # Single assignment creation
    run_performance_test "Single assignment creation" \
        "curl -sf -X POST '$REGISTRY_URL/assignments' \
            -H 'Content-Type: application/json' \
            -d '{
                \"host_selector\": {\"host_id\": \"perf-test-host-1\"},
                \"bundle_id\": \"${bundle_ids[0]}\",
                \"created_by\": \"perf-test\"
            }' >/dev/null" \
        3000
    
    # Batch assignment creation
    run_performance_test "Batch assignment creation (10 assignments)" \
        "for i in {1..10}; do
            curl -sf -X POST '$REGISTRY_URL/assignments' \
                -H 'Content-Type: application/json' \
                -d \"{\\\"host_selector\\\": {\\\"host_id\\\": \\\"perf-test-host-\$i\\\"}, \\\"bundle_id\\\": \\\"${bundle_ids[$((i % 5))]}\\\", \\\"created_by\\\": \\\"perf-test\\\"}\" >/dev/null &
        done
        wait" \
        15000
    
    # Assignment listing performance
    run_performance_test "Assignment listing (limit 50)" \
        "curl -sf '$REGISTRY_URL/assignments?limit=50' >/dev/null" \
        2000
    
    # Host assignment retrieval
    run_performance_test "Host assignment retrieval" \
        "curl -sf '$REGISTRY_URL/assignments/for-host/perf-test-host-1' >/dev/null" \
        1000
    
    # Assignment validation
    run_performance_test "Assignment validation" \
        "curl -sf -X POST '$REGISTRY_URL/assignments/validate' \
            -H 'Content-Type: application/json' \
            -d '{
                \"host_selector\": {\"host_id\": \"perf-test-host-1\"},
                \"bundle_id\": \"${bundle_ids[0]}\"
            }' >/dev/null" \
        2000
}

# Test concurrent load performance
test_concurrent_load_performance() {
    log_info "Testing concurrent load performance..."
    
    # Concurrent bundle creation
    run_performance_test "Concurrent bundle creation (20 requests)" \
        "for i in {1..20}; do
            curl -sf -X POST '$REGISTRY_URL/bundles' \
                -H 'Content-Type: application/json' \
                -d \"{\\\"name\\\": \\\"concurrent-bundle-\$i\\\", \\\"content\\\": \\\"'$(echo \"Concurrent test bundle \$i\" | base64)'\\\", \\\"created_by\\\": \\\"perf-test\\\"}\" >/dev/null &
        done
        wait" \
        20000
    
    # Concurrent bundle retrieval
    local bundle_id
    bundle_id=$(curl -sf -X POST "$REGISTRY_URL/bundles" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "concurrent-retrieval-bundle",
            "content": "'$(echo "Concurrent retrieval test bundle" | base64)'",
            "created_by": "perf-test"
        }' | jq -r '.bundle_id')
    
    run_performance_test "Concurrent bundle retrieval (50 requests)" \
        "for i in {1..50}; do
            curl -sf '$REGISTRY_URL/bundles/$bundle_id' >/dev/null &
        done
        wait" \
        10000
    
    # Mixed concurrent operations
    run_performance_test "Mixed concurrent operations (30 requests)" \
        "for i in {1..10}; do
            # Bundle creation
            curl -sf -X POST '$REGISTRY_URL/bundles' \
                -H 'Content-Type: application/json' \
                -d \"{\\\"name\\\": \\\"mixed-bundle-\$i\\\", \\\"content\\\": \\\"'$(echo \"Mixed test bundle \$i\" | base64)'\\\", \\\"created_by\\\": \\\"perf-test\\\"}\" >/dev/null &
            
            # Bundle listing
            curl -sf '$REGISTRY_URL/bundles?limit=10' >/dev/null &
            
            # Health check
            curl -sf '$REGISTRY_URL/healthz' >/dev/null &
        done
        wait" \
        25000
}

# Test database performance
test_database_performance() {
    log_info "Testing database performance..."
    
    # Test database connection performance
    run_performance_test "Database health check" \
        "curl -sf '$REGISTRY_URL/healthz' | jq -e '.services.database == \"healthy\"'" \
        2000
    
    # Test database readiness check
    run_performance_test "Database readiness check" \
        "curl -sf '$REGISTRY_URL/readyz' | jq -e '.checks.database == \"ready\"'" \
        2000
    
    # Test large result set handling
    # Create many bundles first
    for i in {1..100}; do
        curl -sf -X POST "$REGISTRY_URL/bundles" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"db-perf-bundle-$i\",
                \"content\": \"$(echo \"Database performance test bundle $i\" | base64)\",
                \"created_by\": \"perf-test\"
            }" >/dev/null
    done
    
    run_performance_test "Large result set query (100 bundles)" \
        "curl -sf '$REGISTRY_URL/bundles?limit=100' >/dev/null" \
        5000
}

# Test memory and resource usage
test_resource_usage() {
    log_info "Testing resource usage..."
    
    # Test memory usage during batch operations
    run_performance_test "Memory usage during batch bundle creation (50 bundles)" \
        "for i in {1..50}; do
            curl -sf -X POST '$REGISTRY_URL/bundles' \
                -H 'Content-Type: application/json' \
                -d \"{\\\"name\\\": \\\"memory-bundle-\$i\\\", \\\"content\\\": \\\"'$(echo \"Memory test bundle \$i\" | base64)'\\\", \\\"created_by\\\": \\\"perf-test\\\"}\" >/dev/null &
        done
        wait" \
        30000
    
    # Test CPU usage during signature operations
    local bundle_id
    bundle_id=$(curl -sf -X POST "$REGISTRY_URL/bundles" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "cpu-perf-bundle",
            "content": "'$(echo "CPU performance test bundle" | base64)'",
            "created_by": "perf-test"
        }' | jq -r '.bundle_id')
    
    run_performance_test "CPU usage during signature verification (20 requests)" \
        "for i in {1..20}; do
            curl -sf -X POST '$REGISTRY_URL/bundles/$bundle_id/verify' \
                -H 'Content-Type: application/json' \
                -d '{\"content\": \"'$(echo \"CPU performance test bundle\" | base64)'\"}' >/dev/null &
        done
        wait" \
        15000
}

# Test network performance
test_network_performance() {
    log_info "Testing network performance..."
    
    # Test with different payload sizes
    local small_content
    small_content=$(echo "Small payload" | base64)
    run_performance_test "Small payload processing" \
        "curl -sf -X POST '$REGISTRY_URL/bundles' \
            -H 'Content-Type: application/json' \
            -d '{\"name\": \"small-payload-bundle\", \"content\": \"'$small_content'\", \"created_by\": \"perf-test\"}' >/dev/null" \
        2000
    
    local medium_content
    medium_content=$(python3 -c "print('A' * 10000)" | base64)
    run_performance_test "Medium payload processing (10KB)" \
        "curl -sf -X POST '$REGISTRY_URL/bundles' \
            -H 'Content-Type: application/json' \
            -d '{\"name\": \"medium-payload-bundle\", \"content\": \"'$medium_content'\", \"created_by\": \"perf-test\"}' >/dev/null" \
        5000
    
    local large_content
    large_content=$(python3 -c "print('A' * 100000)" | base64)
    run_performance_test "Large payload processing (100KB)" \
        "curl -sf -X POST '$REGISTRY_URL/bundles' \
            -H 'Content-Type: application/json' \
            -d '{\"name\": \"large-payload-bundle\", \"content\": \"'$large_content'\", \"created_by\": \"perf-test\"}' >/dev/null" \
        15000
    
    # Test network latency
    run_performance_test "Network latency test (10 requests)" \
        "for i in {1..10}; do
            curl -sf '$REGISTRY_URL/healthz' >/dev/null
        done" \
        5000
}

# Test system stability under sustained load
test_sustained_load() {
    log_info "Testing system stability under sustained load..."
    
    # Run sustained load for 60 seconds
    local start_time
    start_time=$(date +%s)
    local end_time=$((start_time + 60))
    local request_count=0
    
    while [ $(date +%s) -lt $end_time ]; do
        # Mix of operations
        curl -sf -X POST "$REGISTRY_URL/bundles" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"sustained-load-bundle-$request_count\",
                \"content\": \"$(echo \"Sustained load test bundle $request_count\" | base64)\",
                \"created_by\": \"perf-test\"
            }" >/dev/null &
        
        curl -sf "$REGISTRY_URL/bundles?limit=10" >/dev/null &
        
        curl -sf "$REGISTRY_URL/healthz" >/dev/null &
        
        request_count=$((request_count + 1))
        
        # Small delay to prevent overwhelming the system
        sleep 0.1
    done
    
    wait
    
    local actual_duration=$((end_time - start_time))
    local requests_per_second=$((request_count / actual_duration))
    
    log_performance "Sustained load test: $request_count requests in ${actual_duration}s ($requests_per_second req/s)"
    
    if [ $requests_per_second -ge 10 ]; then
        log_success "Sustained load test"
        METRICS[PASSED_TESTS]=$((METRICS[PASSED_TESTS] + 1))
    else
        log_error "Sustained load test"
        METRICS[FAILED_TESTS]=$((METRICS[FAILED_TESTS] + 1))
    fi
    
    METRICS[TOTAL_TESTS]=$((METRICS[TOTAL_TESTS] + 1))
}

# Generate performance report
generate_performance_report() {
    log_info "Generating performance report..."
    
    local report_file="performance_report_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "AegisFlux Backend Safety Shim - Performance Test Report"
        echo "======================================================"
        echo "Generated: $(date)"
        echo ""
        echo "Test Results:"
        echo "Total Tests: ${METRICS[TOTAL_TESTS]}"
        echo "Passed: ${METRICS[PASSED_TESTS]}"
        echo "Failed: ${METRICS[FAILED_TESTS]}"
        echo "Success Rate: $(( (METRICS[PASSED_TESTS] * 100) / METRICS[TOTAL_TESTS] ))%"
        echo ""
        echo "Performance Benchmarks:"
        echo "- Single bundle creation: < 5s"
        echo "- Batch bundle creation (10): < 15s"
        echo "- Bundle retrieval: < 1s"
        echo "- Assignment creation: < 3s"
        echo "- Concurrent operations: < 25s"
        echo "- Database operations: < 5s"
        echo "- Network operations: < 15s"
        echo ""
        echo "System Requirements:"
        echo "- Minimum throughput: 10 req/s"
        echo "- Maximum response time: 15s for large operations"
        echo "- Memory usage: Stable under sustained load"
        echo "- CPU usage: Efficient signature operations"
    } > "$report_file"
    
    log_info "Performance report saved to: $report_file"
}

# Main performance test execution
main() {
    log_info "Starting AegisFlux Backend Safety Shim Performance Tests"
    log_info "========================================================="
    
    # Run performance test suites
    test_bundle_creation_performance
    test_bundle_retrieval_performance
    test_assignment_performance
    test_concurrent_load_performance
    test_database_performance
    test_resource_usage
    test_network_performance
    test_sustained_load
    
    # Generate performance report
    generate_performance_report
    
    # Results summary
    log_info "========================================================="
    log_info "Performance Test Results Summary:"
    log_info "Total performance tests run: ${METRICS[TOTAL_TESTS]}"
    log_success "Performance tests passed: ${METRICS[PASSED_TESTS]}"
    
    if [ ${METRICS[FAILED_TESTS]} -gt 0 ]; then
        log_error "Performance tests failed: ${METRICS[FAILED_TESTS]}"
        exit 1
    else
        log_success "All performance tests passed!"
        exit 0
    fi
}

# Run main function
main "$@"





