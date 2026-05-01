# WebSocket Gateway Service

The WebSocket Gateway Service provides secure, bidirectional communication between agents and the AegisFlux backend using WebSocket connections with end-to-end encryption.

## Features

- **WebSocket Connection Management**: Handles HTTP to WebSocket upgrades for agent connections
- **Ed25519 Authentication**: Cryptographic agent authentication using Ed25519 signatures
- **Channel-Based Routing**: Message routing based on communication channels
- **Connection Health Monitoring**: Automatic health checks and connection management
- **Session Management**: JWT-based session tokens for authenticated connections
- **Metrics Collection**: Connection and message statistics

## Architecture

```
┌─────────────────┐    WebSocket    ┌──────────────────┐
│   Agent         │ ◄─────────────► │ WebSocket Gateway│
│                 │                 │                  │
│ - Ed25519 Keys  │                 │ - Auth Service   │
│ - Encryption    │                 │ - Message Router │
│ - Channels      │                 │ - Connection Mgr │
└─────────────────┘                 └──────────────────┘
```

## API Endpoints

### WebSocket Connection
- **Endpoint**: `/ws/agent`
- **Method**: WebSocket Upgrade
- **Headers Required**:
  - `X-Agent-ID`: Agent identifier
  - `X-Agent-Public-Key`: Base64 encoded Ed25519 public key
  - `User-Agent`: Must be "Aegis-Agent/1.0"

### Health Check
- **Endpoint**: `/health`
- **Method**: GET
- **Response**: JSON with connection metrics and status

## Communication Protocol

### Authentication Flow
1. Agent initiates WebSocket connection with headers
2. Agent sends authentication request with Ed25519 signature
3. Backend verifies signature and generates session token
4. Backend responds with authentication confirmation
5. Connection established with encrypted communication

### Message Types
- `request`: Request message from agent
- `response`: Response message from backend
- `event`: Event notification
- `heartbeat`: Health check message
- `ack`: Acknowledgment message

### Communication Channels
**Agent → Backend:**
- `agent.{id}.policies` - Policy updates and status
- `agent.{id}.anomalies` - Anomaly detection alerts
- `agent.{id}.threats` - Threat intelligence matches
- `agent.{id}.processes` - Process monitoring events
- `agent.{id}.heartbeat` - Health check messages

**Backend → Agent:**
- `backend.{id}.policies` - Policy commands and updates
- `backend.{id}.investigations` - Investigation requests
- `backend.{id}.tests` - Test execution commands
- `backend.{id}.rollbacks` - Rollback commands

## Configuration

### Environment Variables
- `WEBSOCKET_PORT` - WebSocket gateway port (default: 8080)
- `WEBSOCKET_MAX_CONNECTIONS` - Maximum concurrent connections (default: 1000)
- `WEBSOCKET_HEARTBEAT` - Heartbeat interval (default: 30s)
- `WEBSOCKET_CONNECTION_TIMEOUT` - Connection timeout (default: 60s)
- `WEBSOCKET_SESSION_TIMEOUT` - Session timeout (default: 24h)
- `WEBSOCKET_PRIVATE_KEY_PATH` - Path to Ed25519 private key
- `WEBSOCKET_PUBLIC_KEY_PATH` - Path to Ed25519 public key
- `WEBSOCKET_DATABASE_URL` - Database connection URL
- `ACTIONS_API_URL` - Actions API base URL (default: `http://actions-api:8083`)
- `WEBSOCKET_LOG_LEVEL` - Log level (debug, info, warn, error)

### Command Line Flags
```bash
./websocket-gateway \
  --port 8080 \
  --max-connections 1000 \
  --heartbeat 30s \
  --connection-timeout 60s \
  --session-timeout 24h \
  --actions-api-url http://actions-api:8083 \
  --log-level info
```

## Security

### Encryption
- **ChaCha20-Poly1305**: Message encryption/decryption
- **Ed25519**: Digital signatures for message integrity
- **Shared Key Derivation**: SHA256-based key agreement

### Authentication
- **Mutual Authentication**: Both agent and backend verify each other
- **Session Tokens**: JWT-based session management
- **Key Rotation**: Support for key rotation and updates

### Message Integrity
- **Digital Signatures**: All messages signed with Ed25519
- **Nonce Generation**: Cryptographically secure nonces
- **Replay Protection**: Timestamp-based replay attack prevention

## Building and Running

### Local Development
```bash
# Build the service
go build -o websocket-gateway ./cmd/gateway

# Run with default settings
./websocket-gateway

# Run with custom configuration
./websocket-gateway --port 8080 --log-level debug
```

### Docker
```bash
# Build Docker image
docker build -t aegisflux/websocket-gateway .

# Run container
docker run -p 8080:8080 \
  -e WEBSOCKET_PORT=8080 \
  -e WEBSOCKET_LOG_LEVEL=info \
  aegisflux/websocket-gateway
```

### Docker Compose
```yaml
version: '3.8'
services:
  websocket-gateway:
    build: ./backend/websocket-gateway
    ports:
      - "8080:8080"
    environment:
      - WEBSOCKET_PORT=8080
      - WEBSOCKET_MAX_CONNECTIONS=1000
      - WEBSOCKET_LOG_LEVEL=info
    depends_on:
      - nats
      - database
```

## Testing

### Health Check
```bash
curl http://localhost:8080/health
```

### Actions API Proxy Smoke Test
```bash
backend/websocket-gateway/scripts/smoke_actions_api_proxy.sh
```

The smoke test starts `nats`, `actions-api`, and `websocket-gateway`, then posts
a registration-init request through the gateway to verify the configured
Actions API handoff.

### Actions API NATS Send Smoke Test
```bash
backend/websocket-gateway/scripts/smoke_actions_send_nats.sh
```

This smoke test registers an agent with Actions API, posts an agent send request,
and validates the typed `websocket.messages` payload that the gateway subscribes
to.

### Live Agent Receive Smoke Test
```bash
backend/websocket-gateway/scripts/smoke_live_agent_receive.sh
```

This smoke test registers an agent, authenticates a WebSocket connection, posts
an Actions API send request, and validates the encrypted `SecureMessage` delivered
to the live socket.

### WebSocket Connection Test
```bash
# Using wscat (install with: npm install -g wscat)
wscat -c ws://localhost:8080/ws/agent \
  -H "X-Agent-ID: test-agent-001" \
  -H "X-Agent-Public-Key: <base64-encoded-public-key>" \
  -H "User-Agent: Aegis-Agent/1.0"
```

## Monitoring

### Metrics
The service provides the following metrics:
- Total connections
- Active connections
- Messages received
- Messages sent
- Authentication failures
- Connection errors

### Logs
Structured logging with the following levels:
- `DEBUG`: Detailed debugging information
- `INFO`: General operational messages
- `WARN`: Warning conditions
- `ERROR`: Error conditions

## Integration

### With Agent
The agent must implement:
- WebSocket client with proper headers
- Ed25519 key pair generation and management
- Message encryption/decryption
- Channel-based message routing
- Authentication flow

### With Backend Services
The gateway integrates with:
- **Actions API**: Agent registration and management
- **BPF Registry**: Policy and artifact distribution
- **Message Queue**: Reliable message delivery
- **Database**: Connection and session persistence

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Check if port 8080 is available
   - Verify firewall settings
   - Check service logs

2. **Authentication Failed**
   - Verify agent public key format
   - Check signature generation
   - Validate timestamp and nonce

3. **Message Routing Issues**
   - Check channel naming convention
   - Verify message handler registration
   - Check message format

### Debug Mode
Run with debug logging for detailed information:
```bash
./websocket-gateway --log-level debug
```

## Development

### Project Structure
```
websocket-gateway/
├── cmd/gateway/           # Main application entry point
├── internal/
│   ├── gateway/          # WebSocket gateway implementation
│   ├── auth/             # Authentication service
│   └── types/            # Type definitions
├── Dockerfile
├── go.mod
├── go.sum
└── README.md
```

### Adding New Message Handlers
1. Implement the `MessageHandler` interface
2. Register the handler with the message router
3. Define the channel pattern
4. Test with the test agent

### Adding New Channels
1. Define channel naming convention
2. Implement handler logic
3. Add to default handlers
4. Update documentation

## License

This service is part of the AegisFlux project and follows the same licensing terms.
