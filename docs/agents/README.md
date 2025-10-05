# AegisFlux Agent System

The AegisFlux agent system provides comprehensive endpoint monitoring, policy enforcement, and runtime security capabilities through a combination of specialized components.

## Overview

The agent system consists of three main components that work together to provide complete endpoint protection:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   eBPF Sensor   │───►│   Local Agent   │───►│  Policy Bundles │
│ (Data Collection)│    │ (Management)    │    │ (Signed Policies)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Components

### 1. Local Agent (`agents/local-agent/`)

**The main endpoint agent with comprehensive monitoring and policy enforcement capabilities.**

**Key Features:**
- **eBPF Program Management**: Loads, attaches, and manages BPF programs from the registry
- **HTTP API**: RESTful API for monitoring and control (`/healthz`, `/status`, `/metrics`, `/programs`)
- **Telemetry & Rollback**: Automatic monitoring with configurable thresholds and rollback
- **Registry Integration**: Polls BPF Registry for new artifacts with signature verification
- **TTL Management**: Automatic program unloading after TTL expires
- **Systemd Integration**: Production-ready service with notifications and watchdog
- **WebSocket Communication**: Real-time communication with backend services

**Quick Start:**
```bash
# Build and run
cd agents/local-agent
make build
./local-agent

# Docker deployment
make docker-build
make docker-run
```

**Detailed Documentation**: See [agents/local-agent/README.md](../agents/local-agent/README.md) for comprehensive documentation including:
- Complete feature list and architecture
- HTTP API endpoints and examples
- Configuration options and environment variables
- Building, deployment, and operation guides
- Monitoring, troubleshooting, and security considerations

### 2. eBPF Sensor (`agents/ebpf-sensor/`)

**Runtime signal collection and event monitoring using eBPF.**

**Key Features:**
- **Runtime Monitoring**: Collects exec/connect and other system events using eBPF
- **Event Publishing**: Publishes JSON events to NATS `events.raw` subject
- **High Performance**: Rust implementation using aya framework
- **Schema Compliance**: Follows `schemas/Event.json` format
- **Kernel-Level Data**: Captures events at kernel level for minimal overhead

**Event Types:**
- Process execution events
- Network connection events
- File system events
- System call events

**Implementation**: Rust-based using the aya eBPF framework for high-performance kernel-level event collection.

### 3. Policy Bundles (`agents/policy-bundles/`)

**Signed mitigation and segmentation policy packs for secure distribution.**

**Key Features:**
- **Signed Policies**: Cryptographically signed policy definitions
- **YAML Format**: Human-readable policy configuration
- **Orchestrator Verification**: Verified by orchestrator before deployment
- **Multiple Formats**: Supports nftables, Cilium, seccomp examples
- **Policy Distribution**: Packaged for secure distribution to agents

**Policy Types:**
- Network segmentation policies
- Process execution policies
- System call restrictions (seccomp)
- Container runtime policies (Cilium)

## Architecture Integration

### Data Flow

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   eBPF Sensor   │    │   Local Agent   │    │   Backend       │
│                 │    │                 │    │   Services      │
│ - Event Capture │───►│ - Program Mgmt  │───►│ - Orchestrator  │
│ - NATS Publish  │    │ - Policy Enf.   │    │ - Registry      │
│ - JSON Events   │    │ - Telemetry     │    │ - Decision      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Policy Bundles │
                       │                 │
                       │ - Signed YAML   │
                       │ - Verification  │
                       │ - Distribution  │
                       └─────────────────┘
```

### Communication Protocols

**Local Agent ↔ Backend:**
- WebSocket Gateway for real-time communication
- HTTP API for registry operations
- NATS for telemetry and events

**eBPF Sensor → Backend:**
- NATS `events.raw` subject for event publishing
- JSON format following `schemas/Event.json`

**Policy Bundles → Local Agent:**
- Registry distribution with signature verification
- YAML format with cryptographic signatures

## Development

### Prerequisites

- Go 1.21+ (for Local Agent)
- Rust with aya (for eBPF Sensor)
- Docker (for containerized deployment)
- eBPF-enabled kernel

### Building

```bash
# Local Agent
cd agents/local-agent
make build

# eBPF Sensor (when implemented)
cd agents/ebpf-sensor
cargo build

# All agents
make build-all
```

### Testing

```bash
# Local Agent tests
cd agents/local-agent
make test

# Integration tests
make test-integration
```

## Deployment

### Docker Compose

```yaml
services:
  local-agent:
    build: ./agents/local-agent
    privileged: true
    environment:
      - AGENT_HOST_ID=host-001
      - AGENT_REGISTRY_URL=http://bpf-registry:8090
      - AGENT_NATS_URL=nats://nats:4222
    volumes:
      - /sys/fs/bpf:/sys/fs/bpf
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: aegisflux-agent
spec:
  template:
    spec:
      containers:
      - name: agent
        image: aegisflux/local-agent:latest
        securityContext:
          privileged: true
        env:
        - name: AGENT_HOST_ID
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
```

## Monitoring

### Health Checks

```bash
# Local Agent health
curl http://localhost:8080/healthz

# Detailed status
curl http://localhost:8080/status | jq

# Metrics
curl http://localhost:8080/metrics | jq
```

### Logs

```bash
# Local Agent logs
journalctl -u aegisflux-agent -f

# Docker logs
docker logs aegisflux-agent
```

## Security

- **Signature Verification**: All artifacts cryptographically verified
- **Non-root Execution**: Runs as non-root user in containers
- **Minimal Privileges**: Only necessary kernel capabilities
- **Secure Communication**: TLS for backend connections
- **Token Management**: Vault integration for secrets

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure agent runs with eBPF privileges
2. **Registry Connection**: Check network connectivity and URLs
3. **Program Loading**: Verify kernel version and eBPF support
4. **Signature Verification**: Ensure Vault accessibility and keys

### Debug Mode

```bash
export AGENT_LOG_LEVEL=debug
./local-agent
```

## Related Documentation

- **[Engineer Guide](../ENGINEER_GUIDE.md)** - Complete technical documentation
- **[API Reference](../API_REFERENCE.md)** - Backend API documentation
- **[User Guide](../USER_GUIDE.md)** - Getting started and usage
- **[Troubleshooting](../TROUBLESHOOTING.md)** - Common issues and solutions

## Contributing

1. Follow Go and Rust coding standards
2. Add tests for new features
3. Update documentation for changes
4. Ensure security best practices

For detailed development information, see the [Engineer Guide](../ENGINEER_GUIDE.md).