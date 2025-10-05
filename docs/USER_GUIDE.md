# AegisFlux User Guide

## Overview

AegisFlux is a comprehensive network security platform that provides real-time monitoring, policy enforcement, and threat detection using eBPF technology.

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for development)
- Node.js 18+ (for UI development)

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd aegisflux
   ```

2. **Start the backend services**
   ```bash
   docker compose up -d
   ```

3. **Start the UI (optional)**
   ```bash
   cd ui/console
   npm install
   npm run dev
   ```

4. **Access the system**
   - Web UI: http://localhost:3000
   - API: http://localhost:8083

## Core Components

### Backend Services

- **Actions API** (Port 8083): Agent registration and management
- **WebSocket Gateway** (Port 8080): Real-time agent communication
- **BPF Registry**: eBPF artifact storage and distribution
- **Decision Service**: Policy compilation and deployment
- **Orchestrator**: Workflow orchestration

### Agent System

- **Local Agent**: Monitors and enforces policies on endpoints
- **eBPF Sensor**: Kernel-level monitoring and enforcement

### User Interface

- **Console UI**: Web-based management interface
- **Agent Dashboard**: Real-time agent status and monitoring
- **Policy Builder**: Create and manage network security policies

## Basic Usage

### 1. Agent Registration

Agents automatically register with the backend when they start:

```bash
# Start an agent
./agents/local-agent/local-agent --backend-url ws://localhost:8080/ws/agent
```

### 2. Policy Creation

Create network security policies through the UI or API:

```bash
# Create a policy via API
curl -X POST http://localhost:8083/policies \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Block ICMP",
    "type": "network",
    "rules": ["block_icmp_to_8.8.8.8"]
  }'
```

### 3. Policy Deployment

Deploy policies to agents:

```bash
# Deploy policy to specific agent
curl -X POST http://localhost:8083/agents/{agent-id}/policies \
  -H "Content-Type: application/json" \
  -d '{"policy_id": "policy-123"}'
```

## Monitoring and Troubleshooting

### Check System Status

```bash
# Check backend health
curl http://localhost:8083/health

# Check WebSocket gateway
curl http://localhost:8080/health

# List registered agents
curl http://localhost:8083/agents
```

### View Logs

```bash
# Backend logs
docker compose logs -f actions-api
docker compose logs -f websocket-gateway

# Agent logs (on agent machine)
journalctl -u aegis-agent -f
```

### Common Issues

1. **Agent not registering**
   - Check network connectivity
   - Verify WebSocket gateway is running
   - Check agent logs for authentication errors

2. **Policies not applying**
   - Verify agent is registered and online
   - Check policy syntax and validity
   - Review agent logs for eBPF loading errors

3. **Connection issues**
   - Ensure all required ports are open
   - Check firewall settings
   - Verify Docker networking

## Security Considerations

- All agent communications use Ed25519 signatures for authentication
- WebSocket connections are encrypted with ChaCha20-Poly1305
- Agent registration requires proper key management
- Policies are cryptographically signed before deployment

## Support

For technical support and documentation:
- Check the [Engineer Documentation](ENGINEER_GUIDE.md)
- Review [API Documentation](API_REFERENCE.md)
- See [Troubleshooting Guide](TROUBLESHOOTING.md)

