# Agent Configuration API

This document describes the new agent configuration API endpoints that allow the backend to configure agents through the WebSocket gateway.

## Overview

The agent configuration API provides endpoints for:
- **Agent Status Monitoring**: Check connection status and session information
- **Agent Configuration**: Update agent settings, channels, and policies
- **Message Sending**: Send messages to specific agents
- **Broadcasting**: Send messages to multiple agents

## API Endpoints

### 1. Agent Status

Get the connection status and session information for a specific agent.

```
GET /agents/{agent_uid}/status
```

**Response:**
```json
{
  "agent_id": "87654321-4321-4321-4321-cba987654321",
  "connected": true,
  "last_seen": "2025-09-24T00:30:15Z",
  "channels": ["agent.87654321-4321-4321-4321-cba987654321.heartbeat", "agent.87654321-4321-4321-4321-cba987654321.policies"],
  "session_expires": "2025-09-25T00:30:15Z",
  "websocket_url": "ws://localhost:8080/ws/agent",
  "message_count": 42,
  "uptime": "2h15m30s"
}
```

**Example:**
```bash
curl -s http://localhost:8083/agents/87654321-4321-4321-4321-cba987654321/status | jq .
```

### 2. Agent Configuration

Update agent configuration settings including channels, policies, and communication parameters.

```
PUT /agents/{agent_uid}/config
```

**Request Body:**
```json
{
  "channels": [
    "agent.87654321-4321-4321-4321-cba987654321.heartbeat",
    "agent.87654321-4321-4321-4321-cba987654321.policies",
    "agent.87654321-4321-4321-4321-cba987654321.anomalies"
  ],
  "settings": {
    "log_level": "info",
    "monitoring_enabled": true,
    "policy_enforcement": "strict"
  },
  "policies": [
    "icmp-block-8.8.8.8",
    "deny-suspicious-connections"
  ],
  "heartbeat_interval": 30,
  "reconnect_interval": 5,
  "message_queue_size": 1000
}
```

**Response:**
```json
{
  "agent_uid": "87654321-4321-4321-4321-cba987654321",
  "config": {
    "channels": ["agent.87654321-4321-4321-4321-cba987654321.heartbeat", "agent.87654321-4321-4321-4321-cba987654321.policies"],
    "settings": {"log_level": "info", "monitoring_enabled": true},
    "policies": ["icmp-block-8.8.8.8"],
    "heartbeat_interval": 30,
    "reconnect_interval": 5,
    "message_queue_size": 1000
  },
  "status": "configuration_updated",
  "message": "Agent configuration updated successfully"
}
```

**Example:**
```bash
curl -s -X PUT http://localhost:8083/agents/87654321-4321-4321-4321-cba987654321/config \
  -H 'Content-Type: application/json' \
  -d '{
    "channels": ["agent.87654321-4321-4321-4321-cba987654321.heartbeat", "agent.87654321-4321-4321-4321-cba987654321.policies"],
    "settings": {"log_level": "info", "monitoring_enabled": true},
    "policies": ["icmp-block-8.8.8.8"],
    "heartbeat_interval": 30,
    "reconnect_interval": 5,
    "message_queue_size": 1000
  }' | jq .
```

### 3. Send Message to Agent

Send a message to a specific agent through the WebSocket gateway.

```
POST /agents/{agent_uid}/send
```

**Request Body:**
```json
{
  "channel": "agent.87654321-4321-4321-4321-cba987654321.policies",
  "message": {
    "action": "deploy_policy",
    "policy_id": "icmp-block-8.8.8.8",
    "policy_config": {
      "type": "network_filter",
      "target": "8.8.8.8",
      "protocol": "icmp",
      "action": "block"
    }
  },
  "message_type": "request",
  "priority": 1,
  "ttl": 300
}
```

**Response:**
```json
{
  "message_id": "msg_1758673983970037841",
  "status": "sent"
}
```

**Example:**
```bash
curl -s -X POST http://localhost:8083/agents/87654321-4321-4321-4321-cba987654321/send \
  -H 'Content-Type: application/json' \
  -d '{
    "channel": "agent.87654321-4321-4321-4321-cba987654321.policies",
    "message": {
      "action": "deploy_policy",
      "policy_id": "icmp-block-8.8.8.8",
      "policy_config": {
        "type": "network_filter",
        "target": "8.8.8.8",
        "protocol": "icmp",
        "action": "block"
      }
    },
    "message_type": "request",
    "priority": 1,
    "ttl": 300
  }' | jq .
```

### 4. Broadcast to Agents

Send a message to multiple agents or all agents.

```
POST /agents/broadcast
```

**Request Body:**
```json
{
  "channel": "agent.*.policies",
  "message": {
    "action": "policy_update",
    "message": "New security policies available for deployment",
    "policies": ["new-ssl-policy", "updated-firewall-rules"]
  },
  "message_type": "event",
  "agent_filter": ["87654321-4321-4321-4321-cba987654321", "12345678-1234-1234-1234-123456789abc"],
  "priority": 2,
  "ttl": 600
}
```

**Response:**
```json
{
  "message_id": "broadcast_1758673983970037841",
  "sent_to": ["87654321-4321-4321-4321-cba987654321", "12345678-1234-1234-1234-123456789abc"],
  "failed": [],
  "total_sent": 2
}
```

**Example:**
```bash
curl -s -X POST http://localhost:8083/agents/broadcast \
  -H 'Content-Type: application/json' \
  -d '{
    "channel": "agent.*.policies",
    "message": {
      "action": "policy_update",
      "message": "New security policies available for deployment"
    },
    "message_type": "event",
    "agent_filter": ["87654321-4321-4321-4321-cba987654321"],
    "priority": 2,
    "ttl": 600
  }' | jq .
```

## Integration with WebSocket Gateway

These API endpoints are designed to integrate with the WebSocket gateway service. The actual message delivery and agent communication happens through the WebSocket connections established by the gateway.

### Message Flow

1. **Backend API** receives configuration request
2. **Actions API** validates the request and agent existence
3. **WebSocket Gateway** receives the message and routes it to the appropriate agent
4. **Agent** receives the configuration and applies the changes
5. **Agent** sends acknowledgment back through the WebSocket connection

### Channel Types

The system supports various channel types for different message categories:

- `agent.{agent_id}.heartbeat` - Health check and status messages
- `agent.{agent_id}.policies` - Policy deployment and updates
- `agent.{agent_id}.anomalies` - Security anomaly reports
- `agent.{agent_id}.threats` - Threat intelligence updates
- `agent.{agent_id}.processes` - Process monitoring data
- `agent.{agent_id}.status` - Agent status updates
- `agent.{agent_id}.logs` - Log data transmission

### Message Types

- `request` - Commands requiring a response
- `response` - Responses to previous requests
- `event` - Notifications and status updates
- `heartbeat` - Health check messages
- `ack` - Acknowledgments

## Error Handling

All endpoints return appropriate HTTP status codes:

- `200 OK` - Successful operation
- `400 Bad Request` - Invalid request format
- `404 Not Found` - Agent not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

Error responses include detailed error messages:

```json
{
  "error": "Agent not found",
  "message": "No agent found with UID: invalid-agent-id",
  "code": 404
}
```

## Security Considerations

- All messages are encrypted using ChaCha20-Poly1305
- Messages are signed with Ed25519 for integrity verification
- Agent authentication is required for WebSocket connections
- Session tokens are used for message authorization
- TTL (Time To Live) prevents message replay attacks

## Monitoring and Metrics

The API provides visibility into:

- Agent connection status
- Message delivery success/failure rates
- Session expiration times
- Channel subscription status
- Message queue sizes
- Agent uptime and health

This enables comprehensive monitoring of the agent communication infrastructure.
