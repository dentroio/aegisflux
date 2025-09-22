# AegisFlux Backend Safety Shim

## Overview

The AegisFlux Backend Safety Shim (Cap7.9) provides a secure, auditable foundation for the AegisFlux system. It implements cryptographic security, mutual TLS authentication, database persistence, and comprehensive audit logging for secure eBPF program distribution and management.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    AegisFlux Backend Safety Shim            │
├─────────────────────────────────────────────────────────────┤
│  Admin CLI (cmd/admin/)                                     │
│  ├── Key Management                                         │
│  ├── Assignment Management                                  │
│  ├── Bundle Management                                      │
│  └── Health Monitoring                                      │
├─────────────────────────────────────────────────────────────┤
│  HTTP Handlers (services/registry/handlers/)                │
│  ├── Bundle Handler (bundles.go)                           │
│  ├── Assignment Handler (assignments.go)                   │
│  └── Health Handler (health.go)                            │
├─────────────────────────────────────────────────────────────┤
│  Security Layer                                             │
│  ├── Ed25519 Signing (signing/signer.go)                   │
│  ├── mTLS Configuration (mtls/server_tls.go)               │
│  └── Audit Logging (internal/audit/auditlog.go)            │
├─────────────────────────────────────────────────────────────┤
│  Data Layer                                                 │
│  ├── Database Store (internal/db/store.go)                 │
│  ├── Schema Migration (internal/db/migrate/001_init.sql)   │
│  └── NATS Messaging (messaging/nats.go)                    │
└─────────────────────────────────────────────────────────────┘
```

## Key Features

### 🔐 Cryptographic Security
- **Ed25519 Digital Signatures**: All bundles are cryptographically signed
- **JWS Support**: JSON Web Signatures for structured payloads
- **Key Rotation**: Seamless key rotation with backward compatibility
- **Secure Key Storage**: Encrypted private key storage

### 🔒 Mutual TLS (mTLS)
- **Client Certificate Authentication**: All clients must present valid certificates
- **Certificate-based User Identification**: User identity from client certificates
- **Secure Communication**: All traffic encrypted with TLS 1.2+
- **Certificate Validation**: Comprehensive certificate chain validation

### 💾 Database Persistence
- **PostgreSQL Backend**: Robust, ACID-compliant data storage
- **Comprehensive Schema**: Agents, bundles, assignments, audit logs
- **Data Integrity**: Foreign key constraints and data validation
- **Audit Trail**: Immutable audit logs for all operations

### 📊 Audit Logging
- **Complete Audit Trail**: Every operation is logged with full context
- **Client Certificate Info**: User identification from certificates
- **HTTP Request Context**: Full request details captured
- **Searchable Logs**: Query and filter audit events

### 🏥 Health Monitoring
- **Health Endpoints**: `/healthz`, `/readyz`, `/livez`
- **Service Status**: Database and signer health checks
- **Detailed Diagnostics**: Comprehensive health information
- **Kubernetes Ready**: Compatible with K8s health probes

## Database Schema

### Core Tables

#### Agents
Stores registered agent information with platform details and labels.

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

#### Bundles
Stores eBPF program bundles with cryptographic signatures.

```sql
CREATE TABLE bundles (
    bundle_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hash VARCHAR(64) NOT NULL,        -- SHA-256 hash
    sig TEXT NOT NULL,                -- Ed25519 signature
    algo VARCHAR(20) DEFAULT 'Ed25519',
    kid VARCHAR(64) NOT NULL,         -- Key ID
    meta JSONB DEFAULT '{}'::jsonb,   -- Bundle metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system'
);
```

#### Assignments
Defines which bundles are assigned to which agents.

```sql
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_selector JSONB NOT NULL,     -- Selection criteria
    ttl_ts TIMESTAMP WITH TIME ZONE,  -- Time-to-live
    dry_run BOOLEAN DEFAULT FALSE,    -- Test mode
    bundle_id UUID NOT NULL REFERENCES bundles(bundle_id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active'
);
```

#### Audit Log
Immutable audit trail for all operations.

```sql
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor VARCHAR(255) NOT NULL,      -- Who performed action
    action VARCHAR(100) NOT NULL,     -- What action
    target VARCHAR(255),              -- Target of action
    details JSONB DEFAULT '{}'::jsonb,-- Additional details
    at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## API Endpoints

### Bundle Management
- `POST /bundles` - Create bundle with signature
- `GET /bundles/{id}` - Retrieve bundle
- `GET /bundles` - List bundles
- `POST /bundles/{id}/verify` - Verify signature

### Assignment Management
- `POST /assignments` - Create assignment
- `GET /assignments/{id}` - Retrieve assignment
- `GET /assignments` - List assignments
- `GET /assignments/for-host/{host_id}` - Get agent assignments
- `DELETE /assignments/{id}` - Cancel assignment

### Health & Monitoring
- `GET /healthz` - Health check
- `GET /readyz` - Readiness check
- `GET /livez` - Liveness check

## Security Model

### Authentication
1. **Client Certificate Required**: All requests must include valid client certificate
2. **Certificate Chain Validation**: Full certificate chain validation
3. **User Identification**: User identity extracted from certificate CN
4. **Audit Logging**: All requests logged with certificate info

### Authorization
1. **Certificate-based Access**: Access control based on certificate identity
2. **Role-based Permissions**: Different permissions for different certificate holders
3. **Audit Trail**: All authorization decisions logged

### Data Integrity
1. **Cryptographic Signatures**: All bundles signed with Ed25519
2. **Content Hashing**: SHA-256 hashes for integrity verification
3. **Immutable Audit Logs**: Audit logs cannot be modified
4. **Database Constraints**: Referential integrity enforced

## Admin CLI

### Key Management
```bash
# Rotate signing keys
./admin key rotate --keys-path ./configs/signer/signer.keys.json

# List available keys
./admin key list --keys-path ./configs/signer/signer.keys.json

# Backup keys
./admin key backup --keys-path ./configs/signer/signer.keys.json --backup-path ./backup/
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
  --description "Network monitoring eBPF program"

# List bundles
./admin bundle list --registry-url http://localhost:8090
```

## Deployment

### Prerequisites
- PostgreSQL 12+
- TLS certificates for mTLS
- NATS server (optional)
- Go 1.21+

### Configuration
1. **Database Setup**: Run migration scripts
2. **TLS Certificates**: Configure mTLS certificates
3. **Signing Keys**: Initialize signing key configuration
4. **Service Endpoints**: Configure service URLs
5. **Audit Logging**: Set up audit log monitoring

### Docker Deployment
```yaml
version: '3.8'
services:
  registry:
    build: ./backend/services/registry
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/aegisflux
      - NATS_URL=nats://nats:4222
      - TLS_CERT_PATH=/certs/server.crt
      - TLS_KEY_PATH=/certs/server.key
      - TLS_CA_PATH=/certs/ca.crt
    volumes:
      - ./certs:/certs
      - ./configs:/configs
    ports:
      - "8090:8090"
```

## Monitoring & Observability

### Health Checks
- **Liveness Probe**: `/livez` - Service is running
- **Readiness Probe**: `/readyz` - Service ready for traffic
- **Health Check**: `/healthz` - Overall service health

### Metrics
- Bundle creation rate
- Assignment success rate
- Certificate validation failures
- Audit log volume
- Database connection health

### Alerting
- Certificate expiration warnings
- Key rotation alerts
- Database connectivity issues
- High error rates
- Audit log anomalies

## Security Best Practices

### Key Management
1. **Regular Rotation**: Rotate signing keys regularly
2. **Secure Storage**: Store private keys encrypted
3. **Backup Strategy**: Regular key backups
4. **Access Control**: Limit key access to authorized personnel

### Certificate Management
1. **Short Validity**: Use short certificate validity periods
2. **Regular Renewal**: Automated certificate renewal
3. **Revocation**: Implement certificate revocation
4. **Monitoring**: Monitor certificate expiration

### Audit Logging
1. **Immutable Logs**: Prevent audit log modification
2. **Secure Storage**: Encrypt audit log storage
3. **Retention Policy**: Define log retention periods
4. **Regular Review**: Regular audit log review

## Troubleshooting

### Common Issues

#### Certificate Validation Failures
- Check certificate chain
- Verify certificate validity
- Ensure proper mTLS configuration

#### Database Connection Issues
- Verify database connectivity
- Check credentials and permissions
- Ensure database schema is up to date

#### Signing Key Issues
- Verify key file permissions
- Check key format and validity
- Ensure proper key rotation

### Debug Commands
```bash
# Check service health
./admin health check --service-url http://localhost:8090

# Verify bundle signature
curl -X POST http://localhost:8090/bundles/{id}/verify \
  -H "Content-Type: application/json" \
  -d '{"content": "base64-encoded-content"}'

# Check audit logs
curl http://localhost:8090/audit/logs?limit=10
```

## Future Enhancements

### Planned Features
- **Advanced Host Selectors**: Complex selection criteria
- **Bundle Versioning**: Version management and rollback
- **Distributed Key Management**: Multi-site key management
- **Enhanced Analytics**: Advanced audit log analytics
- **Auto-scaling**: Dynamic scaling based on load

### Integration Opportunities
- **SIEM Integration**: Security information and event management
- **Monitoring Systems**: Prometheus/Grafana integration
- **Secret Management**: Vault integration for key storage
- **Container Orchestration**: Kubernetes operator
- **CI/CD Integration**: Automated bundle deployment

## Conclusion

The AegisFlux Backend Safety Shim provides a robust, secure foundation for eBPF program distribution with comprehensive security features, audit capabilities, and administrative tools. The implementation follows security best practices and provides the necessary infrastructure for secure, auditable eBPF program management in enterprise environments.

