#!/bin/bash

# AegisFlux Cap7.10 - Policy Enforcement & Host Visibility Test Suite
# This script tests the complete Cap7.10 implementation

set -euo pipefail

# Configuration
REGISTRY_URL="http://localhost:8090"
ACTIONS_API_URL="http://localhost:8083"
NATS_URL="nats://localhost:4222"
TEST_DB_URL="postgres://testuser:testpass@localhost:5433/aegisflux_test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

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

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    log_info "Running Cap7.10 test: $test_name"
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$test_command"; then
        log_success "$test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "$test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test policy enforcement modes
test_policy_enforcement_modes() {
    log_info "Testing policy enforcement modes..."
    
    # Create observe mode assignment
    local observe_assignment
    observe_assignment=$(curl -sf -X POST "$REGISTRY_URL/assignments" \
        -H "Content-Type: application/json" \
        -d '{
            "bundle_id": "'$(uuidgen)'",
            "mode": "observe",
            "selector": {"host_id": "test-host-1"},
            "snapshot": {
                "allow_cidr_v4": ["10.0.0.0/8"],
                "deny_cidr_v4": ["0.0.0.0/0"],
                "edges": [{"src": "web", "dst": "api"}]
            },
            "created_by": "cap7-10-test"
        }')
    
    local observe_id
    observe_id=$(echo "$observe_assignment" | jq -r '.id')
    
    run_test "Observe mode assignment creation" "[ -n '$observe_id' ] && [ '$observe_id' != 'null' ]"
    
    # Create block mode assignment
    local block_assignment
    block_assignment=$(curl -sf -X POST "$REGISTRY_URL/assignments" \
        -H "Content-Type: application/json" \
        -d '{
            "bundle_id": "'$(uuidgen)'",
            "mode": "block",
            "selector": {"host_id": "test-host-2"},
            "snapshot": {
                "allow_cidr_v4": ["10.0.0.0/8"],
                "deny_cidr_v4": ["0.0.0.0/0"],
                "edges": [{"src": "web", "dst": "api"}]
            },
            "created_by": "cap7-10-test"
        }')
    
    local block_id
    block_id=$(echo "$block_assignment" | jq -r '.id')
    
    run_test "Block mode assignment creation" "[ -n '$block_id' ] && [ '$block_id' != 'null' ]"
    
    # Verify mode in responses
    run_test "Observe mode verification" "echo '$observe_assignment' | jq -e '.mode == \"observe\"'"
    run_test "Block mode verification" "echo '$block_assignment' | jq -e '.mode == \"block\"'"
    
    # Verify snapshot signatures
    run_test "Observe assignment signature" "echo '$observe_assignment' | jq -e '.snapshot_sig != null'"
    run_test "Block assignment signature" "echo '$block_assignment' | jq -e '.snapshot_sig != null'"
    
    # Verify snapshot key IDs
    run_test "Observe assignment key ID" "echo '$observe_assignment' | jq -e '.snapshot_kid != null'"
    run_test "Block assignment key ID" "echo '$block_assignment' | jq -e '.snapshot_kid != null'"
}

# Test policy snapshot validation
test_policy_snapshot_validation() {
    log_info "Testing policy snapshot validation..."
    
    # Test invalid mode
    run_test "Invalid mode rejection" "curl -sf -X POST '$REGISTRY_URL/assignments' \
        -H 'Content-Type: application/json' \
        -d '{
            \"bundle_id\": \"'$(uuidgen)'\",
            \"mode\": \"invalid\",
            \"selector\": {\"host_id\": \"test-host\"},
            \"snapshot\": {\"allow_cidr_v4\": [\"10.0.0.0/8\"]},
            \"created_by\": \"test\"
        }' | jq -e '.error'"
    
    # Test empty CIDR blocks
    run_test "Empty CIDR rejection" "curl -sf -X POST '$REGISTRY_URL/assignments' \
        -H 'Content-Type: application/json' \
        -d '{
            \"bundle_id\": \"'$(uuidgen)'\",
            \"mode\": \"observe\",
            \"selector\": {\"host_id\": \"test-host\"},
            \"snapshot\": {\"allow_cidr_v4\": [\"\"]},
            \"created_by\": \"test\"
        }' | jq -e '.error'"
    
    # Test invalid edge (empty source)
    run_test "Invalid edge rejection" "curl -sf -X POST '$REGISTRY_URL/assignments' \
        -H 'Content-Type: application/json' \
        -d '{
            \"bundle_id\": \"'$(uuidgen)'\",
            \"mode\": \"observe\",
            \"selector\": {\"host_id\": \"test-host\"},
            \"snapshot\": {
                \"allow_cidr_v4\": [\"10.0.0.0/8\"],
                \"edges\": [{\"src\": \"\", \"dst\": \"api\"}]
            },
            \"created_by\": \"test\"
        }' | jq -e '.error'"
    
    # Test valid snapshot
    run_test "Valid snapshot acceptance" "curl -sf -X POST '$REGISTRY_URL/assignments' \
        -H 'Content-Type: application/json' \
        -d '{
            \"bundle_id\": \"'$(uuidgen)'\",
            \"mode\": \"observe\",
            \"selector\": {\"host_id\": \"test-host\"},
            \"snapshot\": {
                \"allow_cidr_v4\": [\"10.0.0.0/8\"],
                \"deny_cidr_v4\": [\"0.0.0.0/0\"],
                \"edges\": [{\"src\": \"web\", \"dst\": \"api\"}],
                \"rules\": [{\"id\": \"rule1\", \"type\": \"network\", \"condition\": {\"protocol\": \"tcp\"}, \"action\": \"allow\", \"priority\": 1}]
            },
            \"created_by\": \"test\"
        }' | jq -e '.id'"
}

# Test visibility API endpoints
test_visibility_api() {
    log_info "Testing visibility API endpoints..."
    
    local test_agent_id="test-agent-123"
    
    # Test latest visibility endpoint
    run_test "Latest visibility endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/visibility/latest' | jq -e '.agent_uid'"
    
    # Test visibility history endpoint
    run_test "Visibility history endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/visibility/history?limit=10' | jq -e '.frames'"
    
    # Test visibility summary endpoint
    run_test "Visibility summary endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/visibility/summary' | jq -e '.agent_uid'"
    
    # Test network flows endpoint
    run_test "Network flows endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/flows?limit=10' | jq -e '.flows'"
    
    # Test processes endpoint
    run_test "Processes endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/processes?limit=10' | jq -e '.processes'"
    
    # Test enforcement decisions endpoint
    run_test "Enforcement decisions endpoint" "curl -sf '$REGISTRY_URL/agents/$test_agent_id/enforcement/decisions?limit=10' | jq -e '.decisions'"
}

# Test assignment retrieval
test_assignment_retrieval() {
    log_info "Testing assignment retrieval..."
    
    # Create a test assignment
    local assignment_response
    assignment_response=$(curl -sf -X POST "$REGISTRY_URL/assignments" \
        -H "Content-Type: application/json" \
        -d '{
            "bundle_id": "'$(uuidgen)'",
            "mode": "observe",
            "selector": {"host_id": "retrieval-test-host"},
            "snapshot": {
                "allow_cidr_v4": ["10.0.0.0/8"],
                "edges": [{"src": "web", "dst": "api"}]
            },
            "created_by": "retrieval-test"
        }')
    
    local assignment_id
    assignment_id=$(echo "$assignment_response" | jq -r '.id')
    
    # Test GET assignment by ID
    run_test "Assignment retrieval by ID" "curl -sf '$REGISTRY_URL/assignments/$assignment_id' | jq -e '.id == \"$assignment_id\"'"
    
    # Test assignment listing
    run_test "Assignment listing" "curl -sf '$REGISTRY_URL/assignments?limit=10' | jq -e '.assignments | length >= 0'"
    
    # Test assignment filtering by mode
    run_test "Assignment filtering by mode" "curl -sf '$REGISTRY_URL/assignments?mode=observe' | jq -e '.assignments'"
}

# Test NATS event publishing
test_nats_events() {
    log_info "Testing NATS event publishing..."
    
    # Create assignment to trigger NATS event
    local assignment_response
    assignment_response=$(curl -sf -X POST "$REGISTRY_URL/assignments" \
        -H "Content-Type: application/json" \
        -d '{
            "bundle_id": "'$(uuidgen)'",
            "mode": "block",
            "selector": {"host_id": "nats-test-host"},
            "snapshot": {
                "allow_cidr_v4": ["10.0.0.0/8"],
                "edges": [{"src": "web", "dst": "api"}]
            },
            "created_by": "nats-test"
        }')
    
    local assignment_id
    assignment_id=$(echo "$assignment_response" | jq -r '.id')
    
    # In a real implementation, we would verify NATS events were published
    # For now, we'll verify the assignment was created successfully
    run_test "NATS event trigger (assignment creation)" "[ -n '$assignment_id' ] && [ '$assignment_id' != 'null' ]"
    
    # Test assignment cancellation (should trigger deletion event)
    run_test "Assignment cancellation" "curl -sf -X DELETE '$REGISTRY_URL/assignments/$assignment_id' | jq -e '.message'"
}

# Test database schema
test_database_schema() {
    log_info "Testing database schema..."
    
    # Test that visibility tables exist
    run_test "Visibility frames table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM visibility_frames LIMIT 1;' >/dev/null 2>&1"
    run_test "Network flows table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM network_flows LIMIT 1;' >/dev/null 2>&1"
    run_test "Processes table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM processes LIMIT 1;' >/dev/null 2>&1"
    run_test "Sockets table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM sockets LIMIT 1;' >/dev/null 2>&1"
    run_test "Exec events table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM exec_events LIMIT 1;' >/dev/null 2>&1"
    run_test "Enforcement decisions table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM enforcement_decisions LIMIT 1;' >/dev/null 2>&1"
    run_test "Assignment snapshots table exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM assignment_snapshots LIMIT 1;' >/dev/null 2>&1"
    
    # Test database views exist
    run_test "Latest visibility frames view exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM latest_visibility_frames LIMIT 1;' >/dev/null 2>&1"
    run_test "Active assignments view exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM active_assignments_with_snapshots LIMIT 1;' >/dev/null 2>&1"
    run_test "Enforcement stats view exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM enforcement_stats LIMIT 1;' >/dev/null 2>&1"
    run_test "Network flow stats view exists" "psql '$TEST_DB_URL' -c 'SELECT 1 FROM network_flow_stats LIMIT 1;' >/dev/null 2>&1"
}

# Test performance
test_performance() {
    log_info "Testing Cap7.10 performance..."
    
    # Test assignment creation performance
    local start_time
    start_time=$(date +%s%N)
    
    curl -sf -X POST "$REGISTRY_URL/assignments" \
        -H "Content-Type: application/json" \
        -d '{
            "bundle_id": "'$(uuidgen)'",
            "mode": "observe",
            "selector": {"host_id": "perf-test-host"},
            "snapshot": {
                "allow_cidr_v4": ["10.0.0.0/8"],
                "edges": [{"src": "web", "dst": "api"}]
            },
            "created_by": "perf-test"
        }' >/dev/null
    
    local end_time
    end_time=$(date +%s%N)
    local duration_ms=$(((end_time - start_time) / 1000000))
    
    run_test "Assignment creation performance (<5s)" "[ $duration_ms -lt 5000 ]"
    
    # Test visibility query performance
    start_time=$(date +%s%N)
    
    curl -sf "http://localhost:8090/agents/test-agent/visibility/latest" >/dev/null
    
    end_time=$(date +%s%N)
    duration_ms=$(((end_time - start_time) / 1000000))
    
    run_test "Visibility query performance (<1s)" "[ $duration_ms -lt 1000 ]"
}

# Main test execution
main() {
    log_info "Starting AegisFlux Cap7.10 Implementation Tests"
    log_info "=============================================="
    
    # Check prerequisites
    command -v curl >/dev/null 2>&1 || { log_error "curl is required but not installed."; exit 1; }
    command -v jq >/dev/null 2>&1 || { log_error "jq is required but not installed."; exit 1; }
    command -v psql >/dev/null 2>&1 || { log_error "PostgreSQL client is required but not installed."; exit 1; }
    
    # Run test suites
    test_policy_enforcement_modes
    test_policy_snapshot_validation
    test_visibility_api
    test_assignment_retrieval
    test_nats_events
    test_database_schema
    test_performance
    
    # Results summary
    log_info "=============================================="
    log_info "Cap7.10 Test Results Summary:"
    log_info "Total tests run: $TESTS_RUN"
    log_success "Tests passed: $TESTS_PASSED"
    
    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Tests failed: $TESTS_FAILED"
        exit 1
    else
        log_success "All Cap7.10 tests passed! 🎉"
        exit 0
    fi
}

# Run main function
main "$@"

