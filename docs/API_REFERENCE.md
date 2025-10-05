# AegisFlux API Reference

## Overview

This document provides comprehensive API documentation for all AegisFlux backend services.

## Actions API

**Base URL**: `http://localhost:8083`

### Agent Management

#### Register Agent (Initialize)

```http
POST /agents/register/init
Content-Type: application/json

{
  "org_id": "default-org",
  "host_id": "agent-hostname",
  "agent_pubkey": "base64-encoded-ed25519-public-key",
  "machine_id_hash": "sha256-hash-of-machine-id",
  "agent_version": "1.0.0",
  "capabilities": ["network_monitoring", "policy_enforcement"],
  "platform": {
    "arch": "arm64",
    "os": "linux"
  },
  "network": {
    "interface": "eth0"
  }
}
```

**Response**:
```json
{
  "registration_id": "uuid",
  "nonce": "base64-encoded-nonce",
  "server_time": "2025-09-27T18:55:44Z",
  "status": "pending"
}
```

#### Complete Agent Registration

```http
POST /agents/register/complete
Content-Type: application/json

{
  "registration_id": "uuid-from-init",
  "host_id": "agent-hostname",
  "signature": "base64-encoded-ed25519-signature"
}
```

**Response**:
```json
{
  "agent_uid": "uuid",
  "status": "registered",
  "session_token": "jwt-token"
}
```

#### List Agents

```http
GET /agents
```

**Response**:
```json
{
  "agents": [
    {
      "agent_uid": "uuid",
      "org_id": "default-org",
      "host_id": "agent-hostname",
      "machine_id_hash": "hash",
      "agent_version": "1.0.0",
      "platform": {
        "arch": "arm64",
        "os": "linux"
      },
      "network": {
        "interface": "eth0"
      },
      "labels": [],
      "created": "2025-09-27T18:55:44Z",
      "last_seen": "2025-09-27T18:55:44Z"
    }
  ],
  "total": 1
}
```

#### Get Agent Status

```http
GET /agents/{agent_uid}/status
```

**Response**:
```json
{
  "agent_id": "uuid",
  "connected": true,
  "last_seen": "2025-09-27T18:55:44Z",
  "channels": ["heartbeat", "policies"],
  "session_expires": "2025-09-28T18:55:44Z",
  "websocket_url": "ws://localhost:8080/ws/agent",
  "message_count": 42,
  "uptime": "2h15m30s"
}
```

#### Update Agent Configuration

```http
PUT /agents/{agent_uid}/config
Content-Type: application/json

{
  "channels": ["heartbeat", "policies", "anomalies"],
  "settings": {
    "heartbeat_interval": 30,
    "log_level": "info"
  },
  "policies": ["policy-1", "policy-2"],
  "heartbeat_interval": 30,
  "reconnect_interval": 60,
  "message_queue_size": 1000
}
```

#### Send Message to Agent

```http
POST /agents/{agent_uid}/send
Content-Type: application/json

{
  "channel": "policies",
  "message": {
    "action": "deploy",
    "policy_id": "policy-123"
  },
  "message_type": "request",
  "priority": 1,
  "ttl": 3600
}
```

#### Broadcast to Agents

```http
POST /agents/broadcast
Content-Type: application/json

{
  "channel": "system_update",
  "message": {
    "action": "restart",
    "reason": "maintenance"
  },
  "message_type": "event",
  "agent_filter": ["agent-1", "agent-2"],
  "priority": 2,
  "ttl": 7200
}
```

### Health Check

```http
GET /healthz
```

**Response**:
```
ok
```

## WebSocket Gateway

**Base URL**: `ws://localhost:8080`

### WebSocket Endpoint

```http
GET /ws/agent
Upgrade: websocket
Connection: Upgrade
```

### Message Format

All WebSocket messages use the `SecureMessage` format:

```json
{
  "id": "message-uuid",
  "type": "request|response|event",
  "channel": "auth|heartbeat|policies|anomalies",
  "timestamp": 1695838544,
  "payload": "base64-encoded-json",
  "headers": {
    "content-type": "application/json"
  }
}
```

### Authentication Message

```json
{
  "id": "auth-req-123",
  "type": "request",
  "channel": "auth",
  "timestamp": 1695838544,
  "payload": "base64-encoded-json",
  "headers": {}
}
```

**Payload (decoded)**:
```json
{
  "agent_id": "agent-hostname",
  "public_key": "base64-ed25519-public-key",
  "timestamp": 1695838544,
  "nonce": "base64-16-byte-nonce",
  "signature": "base64-ed25519-signature"
}
```

### Heartbeat Message

```json
{
  "id": "heartbeat-456",
  "type": "event",
  "channel": "agent.123.heartbeat",
  "timestamp": 1695838544,
  "payload": "base64-encoded-json",
  "headers": {}
}
```

**Payload (decoded)**:
```json
{
  "agent_id": "agent-hostname",
  "status": "healthy",
  "uptime": "2h15m30s",
  "metrics": {
    "cpu_usage": 15.2,
    "memory_usage": 45.8,
    "active_policies": 3
  }
}
```

### Registration Messages

#### Registration Init

```json
{
  "id": "reg-init-789",
  "type": "request",
  "channel": "agent.registration",
  "timestamp": 1695838544,
  "payload": "base64-encoded-json",
  "headers": {}
}
```

#### Registration Complete

```json
{
  "id": "reg-complete-101",
  "type": "request",
  "channel": "registration.complete",
  "timestamp": 1695838544,
  "payload": "base64-encoded-json",
  "headers": {}
}
```

### Health Check

```http
GET /health
```

**Response**:
```json
{
  "status": "healthy",
  "active_connections": 5,
  "total_connections": 42,
  "messages_received": 1250,
  "messages_sent": 1180,
  "uptime": 3600,
  "timestamp": 1695838544
}
```

## BPF Registry API

**Base URL**: `http://localhost:8084`

### Upload Artifact

```http
POST /artifacts
Content-Type: multipart/form-data

{
  "name": "icmp-block-policy",
  "version": "1.0.0",
  "description": "Blocks ICMP traffic to specific hosts",
  "metadata": {
    "type": "network_policy",
    "targets": ["8.8.8.8"]
  },
  "artifact": "tar.zst-file"
}
```

### Get Artifact

```http
GET /artifacts/{artifact_id}
```

### List Artifacts

```http
GET /artifacts
```

## Decision Service API

**Base URL**: `http://localhost:8085`

### Create Policy

```http
POST /policies
Content-Type: application/json

{
  "name": "Block ICMP to DNS",
  "description": "Block ICMP traffic to DNS servers",
  "rules": [
    {
      "action": "block",
      "protocol": "icmp",
      "destination": "8.8.8.8"
    }
  ],
  "targets": ["agent-group-1"]
}
```

### Deploy Policy

```http
POST /policies/{policy_id}/deploy
Content-Type: application/json

{
  "targets": ["agent-1", "agent-2"],
  "schedule": "immediate"
}
```

## Error Responses

All APIs return consistent error responses:

```json
{
  "error": "error-code",
  "message": "Human readable error message",
  "details": {
    "field": "additional error details"
  },
  "timestamp": "2025-09-27T18:55:44Z"
}
```

### Common Error Codes

- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `500` - Internal Server Error

## Authentication

### Ed25519 Signature Verification

For agent authentication, signatures are computed over:

```
data = agent_id + ":" + public_key + ":" + timestamp + ":" + nonce
signature = ed25519_sign(private_key, data)
```

For registration completion:

```
data = nonce + server_time + host_id
signature = ed25519_sign(private_key, data)
```

### JWT Tokens

Session tokens are JWT tokens with the following claims:

```json
{
  "sub": "agent_uid",
  "iss": "aegisflux",
  "exp": 1695924944,
  "iat": 1695838544,
  "agent_id": "agent-hostname"
}
```

## Rate Limiting

APIs implement rate limiting:

- **WebSocket connections**: 1000 per IP
- **Registration requests**: 10 per minute per IP
- **API requests**: 1000 per hour per agent

## Webhooks

### Agent Status Change

```http
POST /webhooks/agent-status
Content-Type: application/json

{
  "event": "agent.registered|agent.disconnected|agent.heartbeat",
  "agent_id": "uuid",
  "timestamp": "2025-09-27T18:55:44Z",
  "data": {
    "status": "connected|disconnected",
    "last_seen": "2025-09-27T18:55:44Z"
  }
}
```

## Complete Service API Reference

### Decision Service (Port 8087)

**Base URL**: `http://localhost:8087`

#### Generate Policy from Intent

```http
POST /plans/policy
Content-Type: application/json

{
  "intent": "Block ICMP traffic to 8.8.8.8",
  "target_hosts": ["192.168.1.100"],
  "priority": "high"
}
```

**Response**:
```json
{
  "plan_id": "plan-12345",
  "controls": [
    {
      "type": "nftables",
      "rule": "drop icmp to 8.8.8.8"
    }
  ]
}
```

### Orchestrator Service (Port 8081)

**Base URL**: `http://localhost:8081`

#### Process Segmentation Map

```http
POST /seg/maps
Content-Type: application/json

{
  "map_snapshot": {
    "allow_cidrs": ["192.168.1.0/24"],
    "edges": [
      {
        "src": "192.168.1.10",
        "dst": "192.168.1.20",
        "port": 443,
        "proto": "tcp"
      }
    ]
  }
}
```

### BPF Registry (Port 8090)

**Base URL**: `http://localhost:8090`

#### Create Artifact

```http
POST /artifacts
Content-Type: application/json

{
  "name": "icmp-block-policy",
  "version": "1.0.0",
  "type": "ebpf",
  "data": "base64-encoded-artifact-data"
}
```

#### Assign Artifact to Host

```http
POST /assign/{artifact_id}/{host_id}
Content-Type: application/json

{
  "assignment_id": "assign-12345",
  "priority": 100
}
```

### Ingest Service (Port 8086/8088)

**gRPC Endpoint**: `localhost:8086`  
**HTTP Endpoint**: `http://localhost:8088`

#### Post Events (gRPC)

```protobuf
service Ingest {
  rpc PostEvents(stream Event) returns (PostEventsResponse);
}

message Event {
  string id = 1;
  string type = 2;
  string source = 3;
  int64 timestamp = 4;
  string env = 5;
  string rdns = 6;
  google.protobuf.Struct metadata = 7;
  bytes payload = 8;
}
```

#### Health Check (HTTP)

```http
GET /healthz
```

**Response**:
```json
{
  "ok": true
}
```

### Segmenter Service (Port 8089)

**Base URL**: `http://localhost:8089`

#### Generate Segmentation Proposal

```http
POST /segment/propose
Content-Type: application/json

{
  "hosts": [
    {
      "id": "host-1",
      "ip": "192.168.1.10",
      "labels": ["role:web", "env:production"],
      "services": ["http", "nginx"]
    }
  ],
  "traffic_data": [
    {
      "source_host": "host-1",
      "destination_host": "host-2",
      "destination_port": 443,
      "protocol": "tcp",
      "bytes_transferred": 1024000,
      "packet_count": 500,
      "timestamp": "2025-09-28T17:30:00Z"
    }
  ],
  "traffic_period": "1h",
  "goals": ["reduce_lateral_movement", "security"]
}
```

#### Create Segmentation Plan

```http
POST /segment/plan
Content-Type: application/json

{
  "proposal_id": "proposal-12345",
  "proposal": {
    "id": "microseg-12345",
    "name": "Microsegmentation Proposal",
    "strategy": "microsegmentation",
    "segments": [...]
  },
  "implementation_mode": "balanced"
}
```

#### Get Segmentation Strategies

```http
GET /segment/strategies
```

**Response**:
```json
{
  "strategies": [
    {
      "id": "microsegmentation",
      "name": "Microsegmentation",
      "description": "Fine-grained segmentation based on host characteristics",
      "complexity": "high",
      "security": "high"
    }
  ]
}
```

#### Get Segmentation Goals

```http
GET /segment/goals
```

**Response**:
```json
{
  "goals": [
    {
      "id": "reduce_lateral_movement",
      "name": "Reduce Lateral Movement",
      "description": "Prevent attackers from moving laterally across the network",
      "priority": "high"
    }
  ]
}
```

## Service Integration Matrix

| Service | Protocol | Port | Primary Function |
|---------|----------|------|------------------|
| WebSocket Gateway | WebSocket | 8080 | Agent communication |
| Actions API | HTTP | 8083 | Agent management |
| Decision Service | HTTP | 8087 | Policy generation |
| Orchestrator | HTTP | 8081 | eBPF compilation |
| BPF Registry | HTTP | 8090 | Artifact storage |
| Ingest Service | gRPC/HTTP | 8086/8088 | Event ingestion |
| ETL-Enrich | Python | - | Data enrichment |
| Segmenter | HTTP | 8089 | Network segmentation |

## Complete Data Flow

```
Agent → WebSocket Gateway → Actions API
  ↓
Events → Ingest Service → ETL-Enrich → TimescaleDB/Neo4j
  ↓
Policy Intent → Decision Service → Orchestrator → BPF Registry
  ↓
Network Analysis → Segmenter → Policy Generation → Agent Deployment
```

For more information, see the [Engineer Guide](ENGINEER_GUIDE.md), [Agent Implementation Guide](AGENT_IMPLEMENTATION_GUIDE.md), and [Troubleshooting Guide](TROUBLESHOOTING.md).

