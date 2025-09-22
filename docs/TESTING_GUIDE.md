# AegisFlux Backend Safety Shim - Testing Guide

## Quick Start

### Run All Tests
```bash
# Run complete test suite
./tests/run_all_tests.sh

# Run specific test suites
./tests/run_all_tests.sh unit integration
./tests/run_all_tests.sh security performance
```

### Individual Test Suites
```bash
# Unit tests
cd backend && go test -race -cover ./...

# Integration tests
./tests/integration_test.sh

# End-to-end tests
./tests/e2e_test.sh

# Security tests
./tests/security_test.sh

# Performance tests
./tests/performance_test.sh
```

## Test Categories

### 1. Unit Tests
**Purpose**: Test individual components in isolation
**Location**: `backend/internal/*/**_test.go`
**Run**: `cd backend && go test -race -cover ./...`

**Coverage**:
- Database operations (CRUD, constraints, transactions)
- Ed25519 signing (generation, verification, rotation)
- HTTP handlers (validation, parsing, responses)
- Audit logging (event capture, formatting)

### 2. Integration Tests
**Purpose**: Test component interactions and API endpoints
**Location**: `tests/integration_test.sh`
**Run**: `./tests/integration_test.sh`

**Coverage**:
- Bundle management API (POST, GET, verify)
- Assignment management API (create, list, cancel, validate)
- Health check endpoints (healthz, readyz, livez)
- Admin CLI integration
- Database integration

### 3. End-to-End Tests
**Purpose**: Test complete workflows from start to finish
**Location**: `tests/e2e_test.sh`
**Run**: `./tests/e2e_test.sh`

**Coverage**:
- Bundle lifecycle (create → verify → assign)
- Assignment lifecycle (create → validate → cancel)
- Agent registration workflow
- Key rotation workflow
- Audit logging workflow
- NATS messaging workflow

### 4. Security Tests
**Purpose**: Test security features and vulnerability prevention
**Location**: `tests/security_test.sh`
**Run**: `./tests/security_test.sh`

**Coverage**:
- Signature verification and tampering detection
- Input validation and sanitization
- SQL injection prevention
- Authorization and access control
- Rate limiting and abuse prevention
- Data integrity and audit log immutability

### 5. Performance Tests
**Purpose**: Test system performance under various load conditions
**Location**: `tests/performance_test.sh`
**Run**: `./tests/performance_test.sh`

**Coverage**:
- Bundle operations (creation, retrieval, verification)
- Assignment operations (creation, listing, validation)
- Concurrent load handling
- Database performance
- Resource usage monitoring
- Sustained load testing

## Test Environment Setup

### Prerequisites
```bash
# Required tools
docker
docker-compose
postgresql-client
curl
jq
openssl
go (1.21+)
python3
```

### Environment Setup
```bash
# Start test services
docker-compose -f docker-compose.test.yml up -d

# Wait for services
sleep 30

# Run database migrations
psql "postgres://testuser:testpass@localhost:5433/aegisflux_test" -f backend/internal/db/migrate/001_init.sql
```

### Environment Cleanup
```bash
# Stop test services
docker-compose -f docker-compose.test.yml down -v

# Clean up test artifacts
rm -rf ./test-certs
rm -f performance_report_*.txt
```

## Test Configuration

### Environment Variables
```bash
# Test database
export TEST_DB_URL="postgres://testuser:testpass@localhost:5433/aegisflux_test"

# Test services
export REGISTRY_URL="http://localhost:8090"
export ACTIONS_API_URL="http://localhost:8083"
export NATS_URL="nats://localhost:4222"

# Test keys
export KEYS_PATH="./backend/configs/signer/signer.keys.json"
```

### Test Data
- **Test Database**: `aegisflux_test` with isolated schema
- **Test Certificates**: Auto-generated for mTLS testing
- **Test Keys**: Separate signing key configuration
- **Test Containers**: Isolated Docker environment

## CI/CD Integration

### GitHub Actions
The project includes a comprehensive GitHub Actions workflow (`.github/workflows/test.yml`) that runs:

1. **Unit Tests**: Go tests with race detection and coverage
2. **Integration Tests**: API and database integration
3. **Performance Tests**: Load and stress testing
4. **Security Scan**: Gosec and Trivy vulnerability scanning
5. **Docker Build**: Container build and test
6. **Code Linting**: golangci-lint analysis

### Local CI Simulation
```bash
# Run all tests locally (simulates CI)
./tests/run_all_tests.sh

# Run specific CI jobs
./tests/run_all_tests.sh unit
./tests/run_all_tests.sh integration
./tests/run_all_tests.sh security
```

## Performance Benchmarks

### Response Time Targets
- **Bundle Creation**: < 5 seconds
- **Bundle Retrieval**: < 1 second
- **Assignment Creation**: < 3 seconds
- **Health Checks**: < 2 seconds
- **Concurrent Operations**: < 25 seconds

### Throughput Targets
- **Sustained Load**: 10+ requests/second
- **Batch Operations**: 10 bundles in < 15 seconds
- **Concurrent Requests**: 20 requests in < 20 seconds

### Resource Usage
- **Memory**: Stable under sustained load
- **CPU**: Efficient signature operations
- **Network**: Handles large payloads (100KB+)

## Security Test Coverage

### Authentication & Authorization
- ✅ Client certificate validation
- ✅ mTLS enforcement
- ✅ Access control verification
- ✅ Unauthorized access prevention

### Input Validation
- ✅ Empty field rejection
- ✅ Invalid JSON handling
- ✅ SQL injection prevention
- ✅ XSS protection
- ✅ Large payload handling

### Cryptographic Security
- ✅ Ed25519 signature verification
- ✅ Tampered content detection
- ✅ Key rotation security
- ✅ Secure key storage

### Data Integrity
- ✅ Hash verification
- ✅ Duplicate detection
- ✅ Audit log immutability
- ✅ Transaction consistency

## Troubleshooting

### Common Issues

#### Service Startup Issues
```bash
# Check service status
docker-compose -f docker-compose.test.yml ps

# View service logs
docker-compose -f docker-compose.test.yml logs registry
docker-compose -f docker-compose.test.yml logs actions-api

# Restart services
docker-compose -f docker-compose.test.yml restart
```

#### Database Connection Issues
```bash
# Check database connectivity
pg_isready -h localhost -p 5433 -U testuser

# Test database connection
psql "postgres://testuser:testpass@localhost:5433/aegisflux_test" -c "SELECT 1"

# Check database logs
docker-compose -f docker-compose.test.yml logs postgres-test
```

#### Certificate Issues
```bash
# Check certificate generation
ls -la ./test-certs/

# Verify certificate permissions
stat -c %a ./test-certs/*.crt
stat -c %a ./test-certs/*.key

# Regenerate certificates
./tests/security_test.sh  # Auto-generates certificates
```

#### Test Timeout Issues
```bash
# Increase timeout for slow tests
export TEST_TIMEOUT=300

# Run tests with verbose output
./tests/integration_test.sh 2>&1 | tee test.log
```

### Debug Commands
```bash
# Check API endpoints
curl -v http://localhost:8090/healthz
curl -v http://localhost:8083/healthz

# Test bundle creation
curl -X POST http://localhost:8090/bundles \
  -H "Content-Type: application/json" \
  -d '{"name": "debug-bundle", "content": "dGVzdA==", "created_by": "debug"}'

# Check admin CLI
./backend/cmd/admin/admin health check --service-url http://localhost:8090
./backend/cmd/admin/admin key list --keys-path ./backend/configs/signer/signer.keys.json
```

## Test Reports

### Coverage Reports
```bash
# Generate Go coverage report
cd backend
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# View coverage
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Performance Reports
```bash
# Run performance tests
./tests/performance_test.sh

# View performance report
cat performance_report_*.txt
```

### Security Reports
```bash
# Run security scan
gosec ./backend/...

# Run vulnerability scan
trivy fs .
```

## Best Practices

### Test Development
1. **Write Tests First**: Follow TDD principles
2. **Test Edge Cases**: Include boundary conditions
3. **Mock External Dependencies**: Use mocks for external services
4. **Keep Tests Independent**: Tests should not depend on each other
5. **Use Descriptive Names**: Clear test descriptions

### Test Execution
1. **Run Tests Regularly**: Before every commit
2. **Monitor Performance**: Track test execution times
3. **Clean Up**: Always clean up test data
4. **Document Issues**: Record and fix test failures
5. **Update Tests**: Keep tests current with code changes

### Test Environment
1. **Isolation**: Use separate test databases
2. **Consistency**: Ensure consistent test environments
3. **Security**: Use test credentials and certificates
4. **Monitoring**: Monitor test environment health
5. **Automation**: Automate test environment setup

## Conclusion

This comprehensive testing strategy ensures the AegisFlux Backend Safety Shim is thoroughly tested across all dimensions: functionality, security, performance, and reliability. The multi-layered approach provides confidence in the system's correctness and robustness while maintaining high code quality and security standards.

For questions or issues, please refer to the troubleshooting section or check the test logs for detailed error information.

