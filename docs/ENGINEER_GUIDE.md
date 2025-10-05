# AegisFlux Engineer Guide

## Architecture Overview

AegisFlux is built with a microservices architecture using Go backends and a React frontend.

### System Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Agent (eBPF)  │◄──►│  WebSocket       │◄──►│   Actions API   │
│                 │    │  Gateway         │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │   BPF Registry   │    │   Decision      │
                       │                  │    │   Service       │
                       └──────────────────┘    └─────────────────┘
```

## Backend Services

### Actions API (`backend/actions-api/`)

**Purpose**: Agent registration, management, and policy deployment

**Key Components**:
- Agent registration endpoints (`/agents/register/init`, `/agents/register/complete`)
- Agent management (`/agents/{uid}/status`, `/agents/{uid}/config`)
- Policy deployment (`/agents/{uid}/policies`)

**API Endpoints**:
```go
POST /agents/register/init      // Initialize agent registration
POST /agents/register/complete  // Complete agent registration
GET  /agents                   // List all agents
GET  /agents/{uid}/status      // Get agent status
PUT  /agents/{uid}/config      // Update agent configuration
POST /agents/{uid}/send        // Send message to agent
POST /agents/broadcast         // Broadcast to all agents
```

### WebSocket Gateway (`backend/websocket-gateway/`)

**Purpose**: Real-time bidirectional communication with agents

**Key Components**:
- WebSocket connection management
- Message routing and authentication
- Agent heartbeat monitoring

**Message Types**:
```go
type SecureMessage struct {
    ID        string            `json:"id"`
    Type      MessageType       `json:"type"`      // request, response, event
    Channel   string            `json:"channel"`   // auth, heartbeat, policies
    Timestamp int64             `json:"timestamp"`
    Payload   string            `json:"payload"`   // base64 encoded
    Headers   map[string]string `json:"headers"`
}
```

**Authentication Flow**:
1. Agent connects via WebSocket
2. Agent sends authentication message with Ed25519 signature
3. Backend verifies signature and establishes session
4. Agent can then send other messages

### BPF Registry (`backend/bpf-registry/`)

**Purpose**: Store, sign, and distribute eBPF artifacts

**Key Features**:
- Artifact storage and retrieval
- Cryptographic signing of artifacts
- Version management

### Decision Service (`backend/decision/`)

**Purpose**: Compile high-level policies into eBPF programs

**Key Features**:
- Policy compilation engine
- Rule processing
- Artifact generation

### Orchestrator (`backend/orchestrator/`)

**Purpose**: Orchestrate the deployment pipeline

**Key Features**:
- Workflow management
- Service coordination
- Deployment orchestration

## Agent System

The AegisFlux agent system consists of three main components working together to provide comprehensive endpoint monitoring and policy enforcement.

### Local Agent (`agents/local-agent/`)

**Purpose**: Main endpoint agent with comprehensive monitoring and policy enforcement capabilities

**Key Features**:
- **eBPF Program Management**: Loads, attaches, and manages BPF programs from the registry
- **HTTP API**: RESTful API for monitoring and control (`/healthz`, `/status`, `/metrics`, `/programs`)
- **Telemetry & Rollback**: Automatic monitoring with configurable thresholds and rollback
- **Registry Integration**: Polls BPF Registry for new artifacts with signature verification
- **TTL Management**: Automatic program unloading after TTL expires
- **Systemd Integration**: Production-ready service with notifications and watchdog
- **WebSocket Communication**: Real-time communication with backend services

**Architecture**:
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   BPF Registry  │    │   NATS Cluster  │    │   Vault Server  │
│                 │    │                 │    │                 │
│ - Artifacts     │◄───┤ - Telemetry     │    │ - Signatures    │
│ - Metadata      │    │ - Events        │    │ - Public Keys   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       ▲                       │
         │                       │                       │
         ▼                       │                       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Local Agent                                  │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Registry  │  │    BPF      │  │ Telemetry   │            │
│  │   Client    │  │   Loader    │  │   Sender    │            │
│  │             │  │             │  │             │            │
│  │ - Polling   │  │ - Loading   │  │ - Reporting │            │
│  │ - Download  │  │ - Attaching │  │ - Events    │            │
│  │ - Verify    │  │ - TTL Mgmt  │  │ - Heartbeat │            │
│  └─────────────┘  └─────────────┘  └─────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

**Configuration**:
```bash
# Key environment variables
export AGENT_HOST_ID="host-001"
export AGENT_REGISTRY_URL="http://localhost:8090"
export AGENT_NATS_URL="nats://localhost:4222"
export AGENT_VAULT_URL="http://localhost:8200"
export AGENT_POLL_INTERVAL_SEC="30"
export AGENT_MAX_PROGRAMS="10"
export AGENT_DEFAULT_TTL_SEC="3600"
```

**Communication Protocol**:
1. Connect to WebSocket Gateway
2. Authenticate using Ed25519 keys
3. Register with Actions API via WebSocket
4. Poll registry for assigned artifacts
5. Download, verify, and load eBPF programs
6. Send telemetry and heartbeat updates
7. Handle rollback signals and threshold violations

### eBPF Sensor (`agents/ebpf-sensor/`)

**Purpose**: Runtime signal collection and event monitoring using eBPF

**Key Features**:
- **Runtime Monitoring**: Collects exec/connect and other system events using eBPF
- **Event Publishing**: Publishes JSON events to NATS `events.raw` subject
- **High Performance**: Rust implementation using aya framework
- **Schema Compliance**: Follows `schemas/Event.json` format
- **Kernel-Level Data**: Captures events at kernel level for minimal overhead

**Event Types**:
- Process execution events
- Network connection events
- File system events
- System call events

**Implementation**:
```rust
// Example aya-based eBPF program
use aya::{
    programs::{KProbe, KProbeAttachType},
    Bpf,
};
```

### Policy Bundles (`agents/policy-bundles/`)

**Purpose**: Signed mitigation and segmentation policy packs for distribution

**Key Features**:
- **Signed Policies**: Cryptographically signed policy definitions
- **YAML Format**: Human-readable policy configuration
- **Orchestrator Verification**: Verified by orchestrator before deployment
- **Multiple Formats**: Supports nftables, Cilium, seccomp examples
- **Policy Distribution**: Packaged for secure distribution to agents

**Policy Types**:
- Network segmentation policies
- Process execution policies
- System call restrictions (seccomp)
- Container runtime policies (Cilium)

**Example Policy Structure**:
```yaml
name: "block-icmp-to-dns"
version: "1.0.0"
type: "network_policy"
rules:
  - action: "block"
    protocol: "icmp"
    destination: "8.8.8.8"
metadata:
  description: "Block ICMP traffic to DNS servers"
  severity: "medium"
signature: "base64-encoded-signature"
```

## Development Setup

### Prerequisites

```bash
# Go 1.21+
go version

# Docker and Docker Compose
docker --version
docker compose version

# Node.js 18+ (for UI)
node --version
```

### Building Services

```bash
# Build all Go services
make build

# Build specific service
cd backend/actions-api
go build -o actions-api ./cmd/actions-api

# Build UI
cd ui/console
npm install
npm run build
```

### Running Tests

```bash
# Run all tests
make test

# Run specific service tests
cd backend/actions-api
go test ./...

# Run UI tests
cd ui/console
npm test
```

## API Development

### Adding New Endpoints

1. **Define the handler in `agents_api.go`**:
```go
func (s *Server) newEndpoint(w http.ResponseWriter, r *http.Request, agentUID string) {
    // Implementation
}
```

2. **Add route in `server.go`**:
```go
func (s *Server) routes() {
    s.mux.HandleFunc("/new-endpoint", s.newEndpoint)
}
```

3. **Add tests**:
```go
func TestNewEndpoint(t *testing.T) {
    // Test implementation
}
```

### WebSocket Message Handling

1. **Define message type**:
```go
type NewMessage struct {
    Field1 string `json:"field1"`
    Field2 int    `json:"field2"`
}
```

2. **Add handler**:
```go
func (wsg *WebSocketGateway) handleNewMessage(agentID string, message types.SecureMessage) error {
    // Implementation
}
```

3. **Register handler**:
```go
func (wsg *WebSocketGateway) registerDefaultHandlers() {
    wsg.messageRouter.RegisterHandler("new.message", wsg.handleNewMessage)
}
```

## Security Implementation

### Ed25519 Authentication

```go
// Generate key pair
publicKey, privateKey, err := ed25519.GenerateKey(nil)

// Sign message
data := fmt.Sprintf("%s:%s:%d:%s", agentID, publicKey, timestamp, nonce)
signature := ed25519.Sign(privateKey, []byte(data))

// Verify signature
valid := ed25519.Verify(publicKey, []byte(data), signature)
```

### Message Encryption

```go
// ChaCha20-Poly1305 encryption
cipher, err := chacha20poly1305.New(key)
encrypted := cipher.Seal(nil, nonce, plaintext, nil)
```

## Deployment

### Docker Compose

```yaml
services:
  actions-api:
    build: ./backend/actions-api
    ports:
      - "8083:8083"
    environment:
      - DATABASE_URL=postgres://...
  
  websocket-gateway:
    build: ./backend/websocket-gateway
    ports:
      - "8080:8080"
    depends_on:
      - actions-api
```

### Production Considerations

1. **Database**: Use persistent storage (PostgreSQL)
2. **Secrets**: Use proper secret management (Vault)
3. **Monitoring**: Implement metrics and logging
4. **Scaling**: Consider load balancing for high availability

## Troubleshooting

### Common Issues

1. **Build failures**: Check Go version and dependencies
2. **Connection issues**: Verify network configuration
3. **Authentication failures**: Check key management
4. **Policy deployment**: Verify agent registration status

### Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Check service health
curl http://localhost:8083/health
curl http://localhost:8080/health

# View logs
docker compose logs -f [service-name]
```

## Contributing

1. Follow Go coding standards
2. Add tests for new features
3. Update documentation
4. Submit pull requests with clear descriptions

For more detailed information, see the [API Reference](API_REFERENCE.md) and [Troubleshooting Guide](TROUBLESHOOTING.md).
