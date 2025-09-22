# Backend Cap7.10 - Policy Enforcement Implementation

## Overview

This document describes the implementation of backend-pushed policy enforcement in the AegisFlux Backend Safety Shim. Cap7.10 introduces real policy enforcement with two modes: observe (log only) and block (actual enforcement), along with complete host visibility capabilities.

## Key Features

### 1. Mode-Based Enforcement
- **Observe Mode**: Programs classify traffic but always return ALLOW; emit `enforce_decision(verdict="observe_drop")` for any would-be drop
- **Block Mode**: Return DROP where policy denies, actual enforcement takes place
- Safe toggling between observe → block without changing policies

### 2. Policy Snapshot Management
- Signed policy snapshots as part of bundle creation
- Ed25519 signature verification using Cap7.9 signer
- Support for CIDR-based allow/deny lists
- Network flow edge definitions between services
- Custom rule definitions with conditions and actions

### 3. Complete Host Visibility
- Process tree information
- Execution events tracking
- Network socket enumeration
- Network flow monitoring
- Real-time telemetry collection

## Implementation Details

### Assignment Payload Structure

```json
{
  "bundle_id": "net-guard-001",
  "mode": "block",
  "ttl_ts": "2025-10-01T00:00:00Z",
  "selector": { "host_id": "h-123" },
  "snapshot": {
    "allow_cidr_v4": ["10.0.0.0/24"],
    "deny_cidr_v4":  ["8.8.8.8/32", "1.1.1.1/32"],
    "edges": [{"src":"svc:frontend","dst":"svc:auth"}]
  }
}
```

### API Extensions

#### POST /assignments (Extended)
- Accepts `mode` parameter ("observe" | "block")
- Accepts `snapshot` parameter with policy configuration
- Signs snapshot using Ed25519 (Cap7.9 signer)
- Validates snapshot schema
- Emits NATS event `aegis.assignments.created`

#### GET /assignments/{id}
- Returns assignment details including mode and signed snapshot
- Provides signature verification information
- Includes policy snapshot metadata

### Database Schema Extensions

#### assignment_snapshots Table
```sql
CREATE TABLE assignment_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES assignments(id),
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('observe', 'block')),
    snapshot JSONB NOT NULL,
    snapshot_sig TEXT NOT NULL,
    snapshot_kid VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT NOT NULL
);
```

#### enforcement_decisions Table
```sql
CREATE TABLE enforcement_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES assignments(id),
    agent_uid TEXT NOT NULL,
    verdict VARCHAR(20) NOT NULL CHECK (verdict IN ('allow', 'deny', 'observe_drop')),
    reason TEXT,
    rule_id TEXT,
    flow_data JSONB,
    process_data JSONB,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('observe', 'block'))
);
```

### NATS Event Publishing

#### Assignment Events
- **Subject**: `aegis.assignments.created`
- **Payload**: Complete assignment with signed snapshot
- **Usage**: Notify agents of new policy assignments

#### Enforcement Decision Events
- **Subject**: `aegis.enforcement.agent.{agent_uid}`
- **Payload**: Enforcement decision details
- **Usage**: Track policy enforcement actions

### Policy Snapshot Validation

#### CIDR Validation
- IPv4 CIDR blocks for allow/deny lists
- IPv6 CIDR blocks (optional)
- Proper CIDR notation validation

#### Edge Validation
- Source and destination service definitions
- Port and protocol specifications
- Service-to-service communication rules

#### Rule Validation
- Rule ID uniqueness
- Condition syntax validation
- Action type validation
- Priority ordering

### Agent Integration

#### Mode Reading
Agents read enforcement mode from `/sys/fs/bpf/aegis/mode`:
- `observe`: Classify but always allow, emit observe_drop events
- `block`: Enforce actual drops where policy denies

#### Policy Application
- Agents receive signed policy snapshots
- Verify signatures using public keys
- Apply policies to eBPF programs
- Report enforcement decisions back to backend

### Security Considerations

#### Signature Verification
- All policy snapshots are cryptographically signed
- Use Ed25519 signatures (Cap7.9 signer)
- Key rotation support for signature keys
- Signature verification before policy application

#### Audit Logging
- All policy assignments logged
- Enforcement decisions tracked
- Mode changes audited
- Policy violations recorded

## Usage Examples

### Creating an Observe Assignment

```bash
curl -X POST http://localhost:8090/assignments \
  -H "Content-Type: application/json" \
  -d '{
    "bundle_id": "net-guard-001",
    "mode": "observe",
    "selector": {"host_id": "web-01"},
    "snapshot": {
      "allow_cidr_v4": ["10.0.0.0/8"],
      "deny_cidr_v4": ["0.0.0.0/0"],
      "edges": [{"src": "web", "dst": "api"}]
    },
    "created_by": "admin"
  }'
```

### Creating a Block Assignment

```bash
curl -X POST http://localhost:8090/assignments \
  -H "Content-Type: application/json" \
  -d '{
    "bundle_id": "net-guard-001",
    "mode": "block",
    "selector": {"host_id": "web-01"},
    "snapshot": {
      "allow_cidr_v4": ["10.0.0.0/8"],
      "deny_cidr_v4": ["0.0.0.0/0"],
      "edges": [{"src": "web", "dst": "api"}]
    },
    "ttl_seconds": 3600,
    "created_by": "admin"
  }'
```

### Retrieving Assignment Details

```bash
curl http://localhost:8090/assignments/{assignment_id}
```

## Testing

### Unit Tests
- Policy snapshot validation
- Signature generation and verification
- Mode validation
- CIDR block validation

### Integration Tests
- Assignment creation with mode and snapshot
- NATS event publishing
- Database persistence
- API endpoint validation

### End-to-End Tests
- Complete assignment workflow
- Agent policy application
- Enforcement decision reporting
- Mode switching (observe → block)

## Monitoring and Observability

### Metrics
- Assignment creation rate
- Enforcement decision rate by verdict
- Mode distribution (observe vs block)
- Policy violation rates

### Logs
- Assignment creation events
- Policy application events
- Enforcement decisions
- Mode changes

### Alerts
- High policy violation rates
- Assignment failures
- Signature verification failures
- Agent communication issues

## Future Enhancements

### Advanced Policy Features
- Time-based policies
- User/group-based policies
- Application-specific policies
- Dynamic policy updates

### Performance Optimizations
- Policy caching
- Bulk assignment operations
- Compressed policy snapshots
- Efficient signature verification

### Integration Improvements
- Policy versioning
- Rollback capabilities
- A/B testing for policies
- Policy templates

## Conclusion

Cap7.10 implementation provides a robust foundation for backend-pushed policy enforcement with complete host visibility. The observe/block mode system allows for safe policy deployment and testing, while the comprehensive visibility system provides detailed insights into host behavior and policy effectiveness.

