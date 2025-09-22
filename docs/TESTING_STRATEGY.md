# AegisFlux Backend Safety Shim - Testing Strategy

## Overview

This document outlines the comprehensive testing strategy for the AegisFlux Backend Safety Shim (Cap7.9) implementation. The testing approach covers unit tests, integration tests, end-to-end tests, security tests, and performance tests.

## Testing Pyramid

```
                    /\
                   /  \
                  /E2E \     <- End-to-End Tests (Few, High Value)
                 /______\
                /        \
               /Integration\ <- Integration Tests (Some, Medium Value)
              /____________\
             /              \
            /   Unit Tests   \ <- Unit Tests (Many, Low Value)
           /__________________\
```

## Test Categories

### 1. Unit Tests
**Purpose**: Test individual components in isolation
**Coverage**: Core business logic, utility functions, data structures
**Location**: `backend/internal/*/**_test.go`

#### Components Tested:
- **Database Store** (`backend/internal/db/store_test.go`)
  - CRUD operations for agents, bundles, assignments
  - Data integrity and constraint validation
  - Transaction handling
  - Health and readiness checks

- **Signing Service** (`backend/services/registry/signing/signer_test.go`)
  - Ed25519 signature generation and verification
  - JWS creation and validation
  - Key rotation and management
  - Key backup and restore
  - Concurrent access handling

- **HTTP Handlers** (`backend/services/registry/handlers/*_test.go`)
  - Request validation and parsing
  - Response formatting
  - Error handling
  - Input sanitization

- **Audit Logging** (`backend/internal/audit/auditlog_test.go`)
  - Event logging
  - Context extraction
  - Log formatting
  - Search and filtering

### 2. Integration Tests
**Purpose**: Test component interactions and API endpoints
**Coverage**: HTTP APIs, database integration, external service communication
**Location**: `tests/integration_test.sh`

#### Test Areas:
- **API Endpoints**
  - Bundle management (POST, GET, verify)
  - Assignment management (create, list, cancel, validate)
  - Health checks (healthz, readyz, livez)
  - Agent registration workflow

- **Database Integration**
  - Schema migrations
  - Data persistence
  - Transaction handling
  - Constraint validation

- **Admin CLI Integration**
  - Key management operations
  - Bundle operations
  - Assignment operations
  - Health monitoring

- **Service Health**
  - Registry service health
  - Actions API health
  - Database connectivity
  - Signer readiness

### 3. End-to-End Tests
**Purpose**: Test complete workflows from start to finish
**Coverage**: Full user journeys, system integration, business processes
**Location**: `tests/e2e_test.sh`

#### Test Workflows:
- **Bundle Lifecycle**
  1. Create bundle via API
  2. Verify bundle signature
  3. Create bundle via CLI
  4. Verify end-to-end integrity

- **Assignment Lifecycle**
  1. Create assignment via API
  2. Validate assignment parameters
  3. Create dry-run assignment
  4. Retrieve assignments for host
  5. Cancel assignment

- **Agent Workflow**
  1. Register agent via Actions API
  2. Complete agent registration
  3. Create assignment for agent
  4. Simulate agent polling

- **Key Rotation Workflow**
  1. List current keys
  2. Backup keys
  3. Rotate keys
  4. Verify new keys
  5. Test signing with new keys

- **Audit Logging Workflow**
  1. Perform operations
  2. Verify audit log entries
  3. Test log immutability

- **NATS Messaging Workflow**
  1. Create bundle (should emit message)
  2. Create assignment (should emit message)
  3. Verify message delivery

### 4. Security Tests
**Purpose**: Test security features and vulnerability prevention
**Coverage**: Authentication, authorization, input validation, cryptographic security
**Location**: `tests/security_test.sh`

#### Security Test Areas:
- **Signature Security**
  - Valid signature verification
  - Tampered content detection
  - Invalid signature rejection
  - Empty content handling

- **Input Validation**
  - Empty field rejection
  - Invalid JSON handling
  - Missing required fields
  - Invalid data types

- **Authorization**
  - Unauthorized access attempts
  - Certificate-based authentication
  - Access control enforcement
  - Resource access validation

- **SQL Injection Prevention**
  - Malicious input sanitization
  - Parameterized queries
  - Input validation
  - Error message sanitization

- **Rate Limiting**
  - Request throttling
  - Abuse prevention
  - Resource protection
  - Service stability

- **Data Integrity**
  - Hash verification
  - Duplicate detection
  - Consistency checks
  - Immutable audit logs

- **Key Management Security**
  - File permissions
  - Key rotation security
  - Backup security
  - Access control

### 5. Performance Tests
**Purpose**: Test system performance under various load conditions
**Coverage**: Response times, throughput, resource usage, scalability
**Location**: `tests/performance_test.sh`

#### Performance Test Areas:
- **Bundle Operations**
  - Single bundle creation (< 5s)
  - Batch bundle creation (10 bundles < 15s)
  - Large bundle handling (100KB < 10s)
  - Bundle retrieval (< 1s)
  - Signature verification (< 2s)

- **Assignment Operations**
  - Assignment creation (< 3s)
  - Assignment listing (< 2s)
  - Host assignment retrieval (< 1s)
  - Assignment validation (< 2s)

- **Concurrent Load**
  - Concurrent bundle creation (20 requests < 20s)
  - Concurrent retrieval (50 requests < 10s)
  - Mixed operations (30 requests < 25s)

- **Database Performance**
  - Health checks (< 2s)
  - Readiness checks (< 2s)
  - Large result sets (100 records < 5s)

- **Resource Usage**
  - Memory usage during batch operations
  - CPU usage during signature operations
  - Network performance with different payload sizes
  - Sustained load testing (60s, 10+ req/s)

## Test Execution

### Prerequisites
```bash
# Required tools
docker
docker-compose
postgresql-client
curl
jq
openssl
go (for building admin CLI)

# Required services
PostgreSQL database
NATS server
```

### Running Tests

#### Unit Tests
```bash
# Run all unit tests
cd backend
go test ./...

# Run specific test package
go test ./internal/db/...

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

#### Integration Tests
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
./tests/integration_test.sh

# Cleanup
docker-compose -f docker-compose.test.yml down
```

#### End-to-End Tests
```bash
# Ensure services are running
docker-compose -f docker-compose.test.yml up -d

# Run E2E tests
./tests/e2e_test.sh

# Cleanup
docker-compose -f docker-compose.test.yml down
```

#### Security Tests
```bash
# Run security tests
./tests/security_test.sh
```

#### Performance Tests
```bash
# Run performance tests
./tests/performance_test.sh

# View performance report
cat performance_report_*.txt
```

### Continuous Integration

#### GitHub Actions Workflow
```yaml
name: AegisFlux Backend Safety Shim Tests

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - name: Run unit tests
        run: |
          cd backend
          go test -race -cover ./...

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_USER: testuser
          POSTGRES_DB: aegisflux_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      nats:
        image: nats:2.10
        options: >-
          --health-cmd "nats server check jetstream"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3
      - name: Install dependencies
        run: |
          sudo apt-get update
          sudo apt-get install -y postgresql-client jq curl
      - name: Start services
        run: docker-compose -f docker-compose.test.yml up -d
      - name: Wait for services
        run: sleep 30
      - name: Run integration tests
        run: ./tests/integration_test.sh
      - name: Run E2E tests
        run: ./tests/e2e_test.sh
      - name: Run security tests
        run: ./tests/security_test.sh
      - name: Run performance tests
        run: ./tests/performance_test.sh
      - name: Upload test results
        uses: actions/upload-artifact@v3
        with:
          name: test-results
          path: |
            performance_report_*.txt
            test-results.xml
```

## Test Data Management

### Test Database Setup
```sql
-- Create test database
CREATE DATABASE aegisflux_test;

-- Run migrations
\i backend/internal/db/migrate/001_init.sql
```

### Test Certificates
```bash
# Generate test certificates for mTLS testing
./tests/security_test.sh  # Automatically generates test certificates
```

### Test Keys
```bash
# Initialize test signing keys
cp backend/configs/signer/signer.keys.json /tmp/test-keys.json
```

## Test Monitoring and Reporting

### Metrics Collected
- **Test Execution Time**: How long each test takes
- **Success Rate**: Percentage of passing tests
- **Coverage**: Code coverage percentage
- **Performance Metrics**: Response times, throughput
- **Security Metrics**: Vulnerability detection, security compliance

### Reporting
- **Unit Test Reports**: Go test output with coverage
- **Integration Test Reports**: Detailed test results with timing
- **E2E Test Reports**: Workflow completion status
- **Security Test Reports**: Security compliance status
- **Performance Reports**: Performance metrics and benchmarks

### Alerting
- **Test Failures**: Immediate notification of test failures
- **Performance Degradation**: Alerts when performance thresholds are exceeded
- **Security Issues**: Alerts for security test failures
- **Coverage Drops**: Alerts when code coverage decreases

## Best Practices

### Test Development
1. **Write Tests First**: Follow TDD principles where possible
2. **Test Edge Cases**: Include boundary conditions and error cases
3. **Mock External Dependencies**: Use mocks for external services
4. **Keep Tests Independent**: Tests should not depend on each other
5. **Use Descriptive Names**: Test names should clearly describe what is being tested

### Test Maintenance
1. **Regular Updates**: Keep tests updated with code changes
2. **Remove Obsolete Tests**: Clean up tests that are no longer relevant
3. **Performance Monitoring**: Monitor test execution times
4. **Coverage Tracking**: Maintain adequate test coverage
5. **Documentation**: Keep test documentation up to date

### Test Environment
1. **Isolation**: Use separate test databases and services
2. **Consistency**: Ensure test environments are consistent
3. **Cleanup**: Always clean up test data after tests
4. **Security**: Use test credentials and certificates
5. **Monitoring**: Monitor test environment health

## Troubleshooting

### Common Issues
1. **Database Connection Failures**: Check database service status
2. **Certificate Issues**: Verify certificate generation and permissions
3. **Port Conflicts**: Ensure test ports are available
4. **Timeout Issues**: Increase timeout values for slow tests
5. **Resource Exhaustion**: Monitor system resources during tests

### Debug Commands
```bash
# Check service status
docker-compose ps

# View service logs
docker-compose logs registry
docker-compose logs actions-api

# Check database connectivity
psql "$TEST_DB_URL" -c "SELECT 1"

# Test API endpoints
curl -v http://localhost:8090/healthz

# Check signing keys
./backend/cmd/admin/admin key list --keys-path ./backend/configs/signer/signer.keys.json
```

## Conclusion

This comprehensive testing strategy ensures the AegisFlux Backend Safety Shim is thoroughly tested across all dimensions: functionality, security, performance, and reliability. The multi-layered approach provides confidence in the system's correctness and robustness while maintaining high code quality and security standards.

