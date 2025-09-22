# AegisFlux Backend Safety Shim - Cap7.9 Implementation

## Overview

This document outlines the implementation of the AegisFlux Backend Safety Shim (Cap7.9), which provides cryptographic security, mTLS authentication, database persistence, and comprehensive audit logging for the AegisFlux system.

## Architecture

### Core Components

1. **Database Layer** (`internal/db/`)
   - PostgreSQL schema with agents, bundles, assignments, and audit_log tables
   - Comprehensive data store implementation with CRUD operations
   - Support for complex queries and relationships

2. **Signing Layer** (`services/registry/signing/`)
   - Ed25519 digital signature implementation
   - JWS (JSON Web Signature) support
   - Key rotation and management
   - Secure key storage and backup

3. **HTTP Handlers** (`services/registry/handlers/`)
   - Bundle management (POST/GET bundles with signatures)
   - Assignment management with TTL and selectors
   - Health and readiness endpoints
   - Comprehensive error handling

4. **mTLS Security** (`services/registry/mtls/`)
   - Mutual TLS configuration for server and client
   - Certificate validation and extraction
   - Security middleware

5. **Audit Logging** (`internal/audit/`)
   - Comprehensive audit trail for all operations
   - HTTP request logging with client certificate extraction
   - Searchable audit logs with statistics

6. **Admin CLI** (`cmd/admin/`)
   - Key rotation management
   - Assignment creation and management
   - Bundle operations
   - Health checking

## Key Features

### Ed25519 Signing
- Cryptographic signature generation and verification
- JWS support for structured payloads
- Key rotation with seamless cutover
- Secure key storage with encryption

### Bundle Management
- Content hashing (SHA-256) for integrity verification
- Digital signatures for authenticity
- Metadata support (version, description, etc.)
- Duplicate detection and prevention

### Assignment System
- Flexible host selectors (host_id, labels, platform)
- TTL support for time-limited assignments
- Dry-run mode for testing
- Status tracking (active, expired, cancelled)

### mTLS Security
- Mutual certificate authentication
- Client certificate extraction and validation
- Secure communication channels
- Certificate-based user identification

### Audit Logging
- Complete audit trail for all operations
- HTTP request context capture
- Client certificate information logging
- Searchable and filterable logs

### Health Monitoring
- `/healthz` - Basic health status
- `/readyz` - Readiness for traffic
- `/livez` - Liveness probe
- Detailed health information with service status

## Database Schema

### Agents Table
```sql
CREATE TABLE agents (
    agent_uid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_id VARCHAR(255) UNIQUE NOT NULL,
    platform JSONB NOT NULL,
    labels JSONB DEFAULT '[]'::jsonb,
    notes TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Bundles Table
```sql
CREATE TABLE bundles (
    bundle_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hash VARCHAR(64) NOT NULL,
    sig TEXT NOT NULL,
    algo VARCHAR(20) DEFAULT 'Ed25519',
    kid VARCHAR(64) NOT NULL,
    meta JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system'
);
```

### Assignments Table
```sql
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_selector JSONB NOT NULL,
    ttl_ts TIMESTAMP WITH TIME ZONE,
    dry_run BOOLEAN DEFAULT FALSE,
    bundle_id UUID NOT NULL REFERENCES bundles(bundle_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'cancelled'))
);
```

### Audit Log Table
```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    target VARCHAR(255),
    details JSONB DEFAULT '{}'::jsonb,
    at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## API Endpoints

### Bundle Management
- `POST /bundles` - Create new bundle with signature
- `GET /bundles/{id}` - Retrieve bundle by ID
- `GET /bundles` - List bundles with pagination
- `POST /bundles/{id}/verify` - Verify bundle signature

### Assignment Management
- `POST /assignments` - Create new assignment
- `GET /assignments/{id}` - Retrieve assignment by ID
- `GET /assignments` - List assignments with pagination
- `GET /assignments/for-host/{host_id}` - Get assignments for specific host
- `DELETE /assignments/{id}` - Cancel assignment
- `POST /assignments/validate` - Validate assignment parameters

### Health and Monitoring
- `GET /healthz` - Basic health check
- `GET /readyz` - Readiness check
- `GET /livez` - Liveness check
- `GET /health/detailed` - Detailed health information

## Admin CLI Usage

### Key Management
```bash
# Rotate signing keys
./admin key rotate --keys-path ./configs/signer/signer.keys.json

# List available keys
./admin key list --keys-path ./configs/signer/signer.keys.json

# Backup keys
./admin key backup --keys-path ./configs/signer/signer.keys.json --backup-path ./backup/keys.json
```

### Assignment Management
```bash
# Create assignment
./admin assignment create \
  --bundle-id "123e4567-e89b-12d3-a456-426614174000" \
  --host-selector '{"host_id": "testhost-1"}' \
  --ttl-seconds 3600 \
  --created-by "admin"

# List assignments
./admin assignment list --registry-url http://localhost:8090

# Cancel assignment
./admin assignment cancel --assignment-id "123e4567-e89b-12d3-a456-426614174000"
```

### Bundle Management
```bash
# Create bundle
./admin bundle create \
  --name "network-monitor" \
  --content "base64-encoded-content" \
  --description "Network monitoring eBPF program" \
  --version "1.0.0"

# List bundles
./admin bundle list --registry-url http://localhost:8090
```

## Security Considerations

### Key Management
- Private keys are stored encrypted
- Key rotation is supported with seamless cutover
- Backup and restore capabilities
- Secure key generation using crypto/rand

### Certificate Management
- mTLS with client certificate validation
- Certificate-based user identification
- Secure communication channels
- Certificate expiration monitoring

### Audit Security
- Immutable audit logs
- Comprehensive operation tracking
- Client certificate information capture
- Tamper-evident logging

## Deployment

### Prerequisites
- PostgreSQL database
- TLS certificates for mTLS
- Signing key configuration
- NATS messaging system (optional)

### Configuration
1. Set up database schema using migration files
2. Configure mTLS certificates
3. Initialize signing keys
4. Configure service endpoints
5. Set up audit logging

### Monitoring
- Health check endpoints for load balancers
- Readiness probes for Kubernetes
- Audit log monitoring
- Key expiration alerts

## Testing

### Unit Tests
- Database operations
- Signing functionality
- HTTP handlers
- Audit logging

### Integration Tests
- End-to-end bundle creation and assignment
- mTLS authentication flows
- Key rotation procedures
- Health check validation

### Security Tests
- Certificate validation
- Signature verification
- Audit log integrity
- Key management security

## Future Enhancements

### Planned Features
- NATS messaging integration for bundle notifications
- Advanced host selectors with complex criteria
- Bundle versioning and rollback capabilities
- Enhanced audit log analytics
- Certificate auto-renewal
- Distributed key management

### Scalability Considerations
- Database connection pooling
- Horizontal scaling support
- Caching layer for frequently accessed data
- Load balancer integration
- Metrics and monitoring integration

## Conclusion

The AegisFlux Backend Safety Shim provides a robust, secure foundation for the AegisFlux system with comprehensive security features, audit capabilities, and administrative tools. The implementation follows security best practices and provides the necessary infrastructure for secure eBPF program distribution and management.

