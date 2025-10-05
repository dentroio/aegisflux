# 🔌 AGENT WEBSOCKET CONNECTION GUIDE

## 📋 OVERVIEW

This guide explains how agents connect to the AegisFlux backend via WebSocket and the complete communication protocol. Agents use a secure WebSocket connection with Ed25519 authentication and ChaCha20-Poly1305 encryption.

## 🌐 CONNECTION ARCHITECTURE

### **Connection Flow**
```
Agent → WebSocket Gateway (port 8080) → Actions API (port 8083)
```

### **Service Responsibilities**
- **WebSocket Gateway**: Real-time bidirectional communication
- **Actions API**: Agent registration and management
- **BPF Registry**: eBPF artifact distribution

---

## 🔗 CONNECTION SETUP

### **1. WebSocket Connection**
```javascript
// WebSocket endpoint
const wsUrl = "ws://localhost:8080/ws/agent";

// Connection with custom headers
const ws = new WebSocket(wsUrl, {
  headers: {
    'User-Agent': 'AegisFlux-Agent/1.0.0',
    'X-Agent-Version': '1.0.0',
    'X-Protocol-Version': '1.0'
  }
});
```

### **2. Connection States**
- **Connecting**: Initial connection attempt
- **Connected**: WebSocket established
- **Authenticated**: Ed25519 authentication successful
- **Registered**: Agent registered with Actions API
- **Active**: Fully operational

---

## 🔐 AUTHENTICATION PROTOCOL

### **Step 1: Send Authentication Message**
```json
{
  "id": "auth_req_1234567890",
  "type": "request",
  "channel": "auth",
  "timestamp": 1699123456,
  "payload": "base64_encoded_auth_data",
  "headers": {
    "content-type": "application/json"
  }
}
```

### **Authentication Payload (Base64 Encoded)**
```json
{
  "agent_id": "agent-001",
  "public_key": "base64_encoded_ed25519_public_key",
  "timestamp": 1699123456,
  "nonce": "base64_encoded_16_byte_nonce",
  "signature": "base64_encoded_ed25519_signature"
}
```

### **Signature Generation**
```javascript
// Data to sign: agent_id:public_key:timestamp:nonce
const dataToSign = `${agentId}:${publicKey}:${timestamp}:${nonce}`;
const signature = ed25519.sign(privateKey, Buffer.from(dataToSign));
```

### **Authentication Response**
```json
{
  "id": "auth_resp_1234567890",
  "type": "response",
  "channel": "auth",
  "timestamp": 1699123456,
  "payload": "{\"status\":\"success\",\"session_token\":\"jwt_token\"}",
  "headers": {}
}
```

---

## 📝 REGISTRATION PROTOCOL

### **Step 1: Registration Init**
```json
{
  "id": "reg_init_1234567890",
  "type": "request",
  "channel": "agent.registration",
  "timestamp": 1699123456,
  "payload": "base64_encoded_registration_data",
  "headers": {
    "content-type": "application/json"
  }
}
```

### **Registration Init Payload (Base64 Encoded)**
```json
{
  "org_id": "default-org",
  "host_id": "host-001",
  "agent_pubkey": "base64_encoded_ed25519_public_key",
  "machine_id_hash": "sha256_hash_of_machine_id",
  "agent_version": "1.0.0",
  "capabilities": ["ebpf", "network", "process"],
  "platform": {
    "os": "linux",
    "arch": "x86_64",
    "kernel": "5.4.0"
  },
  "network": {
    "interfaces": ["eth0", "lo"],
    "subnet": "192.168.1.0/24"
  }
}
```

### **Registration Init Response**
```json
{
  "id": "reg_init_resp_1234567890",
  "type": "response",
  "channel": "agent.registration",
  "timestamp": 1699123456,
  "payload": "{\"registration_id\":\"reg_123\",\"server_time\":\"2023-11-05T10:30:00Z\",\"nonce\":\"base64_nonce\"}",
  "headers": {}
}
```

### **Step 2: Registration Complete**
```json
{
  "id": "reg_complete_1234567890",
  "type": "request",
  "channel": "registration.complete",
  "timestamp": 1699123456,
  "payload": "base64_encoded_completion_data",
  "headers": {
    "content-type": "application/json"
  }
}
```

### **Registration Complete Payload (Base64 Encoded)**
```json
{
  "registration_id": "reg_123",
  "host_id": "host-001",
  "signature": "base64_encoded_signature"
}
```

### **Signature for Registration Complete**
```javascript
// Data to sign: nonce + server_time + host_id
const dataToSign = nonce + serverTime + hostId;
const signature = ed25519.sign(privateKey, Buffer.from(dataToSign));
```

---

## 💬 MESSAGE PROTOCOL

### **Message Format**
```json
{
  "id": "unique_message_id",
  "type": "request|response|event",
  "channel": "channel_name",
  "timestamp": 1699123456,
  "payload": "base64_encoded_data",
  "headers": {
    "key": "value"
  }
}
```

### **Message Types**
- **request**: Request requiring response
- **response**: Response to a request
- **event**: One-way notification

### **Channel Types**
- **auth**: Authentication messages
- **agent.registration**: Agent registration
- **registration.complete**: Registration completion
- **agent.{id}.heartbeat**: Heartbeat messages
- **agent.{id}.policies**: Policy updates
- **agent.{id}.status**: Status updates

---

## 💓 HEARTBEAT PROTOCOL

### **Heartbeat Message**
```json
{
  "id": "heartbeat_1234567890",
  "type": "event",
  "channel": "agent.host-001.heartbeat",
  "timestamp": 1699123456,
  "payload": "base64_encoded_heartbeat_data",
  "headers": {}
}
```

### **Heartbeat Payload (Base64 Encoded)**
```json
{
  "agent_id": "host-001",
  "timestamp": 1699123456,
  "status": "healthy",
  "metrics": {
    "cpu_usage": 45.2,
    "memory_usage": 67.8,
    "network_connections": 12,
    "active_policies": 3
  },
  "capabilities": ["ebpf", "network", "process"]
}
```

### **Heartbeat Frequency**
- **Interval**: 30 seconds (configurable)
- **Timeout**: 60 seconds
- **Retry**: Exponential backoff on failure

---

## 📦 POLICY DEPLOYMENT

### **Policy Update Message**
```json
{
  "id": "policy_update_1234567890",
  "type": "request",
  "channel": "agent.host-001.policies",
  "timestamp": 1699123456,
  "payload": "base64_encoded_policy_data",
  "headers": {}
}
```

### **Policy Payload (Base64 Encoded)**
```json
{
  "policy_id": "policy_123",
  "action": "deploy|withdraw|update",
  "artifact": {
    "id": "artifact_456",
    "version": "1.0.0",
    "url": "http://bpf-registry:8090/artifacts/artifact_456.tar.zst",
    "signature": "base64_signature",
    "metadata": {
      "type": "icmp_block",
      "target": "8.8.8.8",
      "direction": "egress"
    }
  }
}
```

### **Policy Response**
```json
{
  "id": "policy_resp_1234567890",
  "type": "response",
  "channel": "agent.host-001.policies",
  "timestamp": 1699123456,
  "payload": "{\"status\":\"success\",\"policy_id\":\"policy_123\"}",
  "headers": {}
}
```

---

## 🔄 CONNECTION MANAGEMENT

### **Connection Lifecycle**
1. **Connect**: Establish WebSocket connection
2. **Authenticate**: Send authentication message
3. **Register**: Complete agent registration
4. **Heartbeat**: Start periodic heartbeat
5. **Operate**: Handle policy updates and commands
6. **Reconnect**: Handle connection failures

### **Reconnection Strategy**
```javascript
class AgentConnection {
  constructor() {
    this.maxRetries = 5;
    this.retryDelay = 1000; // Start with 1 second
    this.maxRetryDelay = 30000; // Max 30 seconds
  }

  async connect() {
    for (let attempt = 0; attempt < this.maxRetries; attempt++) {
      try {
        await this.establishConnection();
        await this.authenticate();
        await this.register();
        this.startHeartbeat();
        return; // Success
      } catch (error) {
        if (attempt === this.maxRetries - 1) throw error;
        await this.delay(this.retryDelay * Math.pow(2, attempt));
      }
    }
  }

  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
```

### **Error Handling**
- **Connection Errors**: Automatic reconnection with exponential backoff
- **Authentication Errors**: Retry authentication or re-register
- **Message Errors**: Log and continue operation
- **Policy Errors**: Report status and request retry

---

## 🛡️ SECURITY CONSIDERATIONS

### **Key Management**
- **Key Generation**: Use cryptographically secure random number generator
- **Key Storage**: Store private keys securely (HSM, keychain, encrypted storage)
- **Key Rotation**: Implement periodic key rotation
- **Key Backup**: Secure backup of private keys

### **Message Security**
- **Encryption**: All messages encrypted with ChaCha20-Poly1305
- **Authentication**: Every message signed with Ed25519
- **Replay Protection**: Use timestamps and nonces
- **Integrity**: Verify message signatures

### **Network Security**
- **TLS**: Use WSS (WebSocket Secure) in production
- **Certificate Validation**: Verify server certificates
- **Network Isolation**: Use VPN or private networks
- **Firewall**: Restrict access to necessary ports

---

## 📊 MONITORING & DEBUGGING

### **Connection Monitoring**
```javascript
// Monitor connection state
ws.onopen = () => console.log('Connected');
ws.onclose = (event) => console.log('Disconnected:', event.code, event.reason);
ws.onerror = (error) => console.error('Connection error:', error);

// Monitor message flow
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message.channel, message.type);
};
```

### **Debug Logging**
- **Connection Events**: Log all connection state changes
- **Message Flow**: Log all sent/received messages
- **Authentication**: Log authentication attempts and results
- **Registration**: Log registration process steps
- **Heartbeats**: Log heartbeat success/failure
- **Policy Updates**: Log policy deployment status

### **Health Checks**
- **Connection Health**: Monitor WebSocket connection status
- **Authentication Health**: Verify authentication state
- **Registration Health**: Confirm agent registration
- **Heartbeat Health**: Monitor heartbeat success rate
- **Policy Health**: Check policy deployment status

---

## 🚀 IMPLEMENTATION EXAMPLES

### **Complete Agent Connection Example**
```javascript
class AegisFluxAgent {
  constructor(config) {
    this.config = config;
    this.ws = null;
    this.authenticated = false;
    this.registered = false;
    this.heartbeatInterval = null;
  }

  async start() {
    await this.connect();
    await this.authenticate();
    await this.register();
    this.startHeartbeat();
  }

  async connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket('ws://localhost:8080/ws/agent');
      
      this.ws.onopen = () => {
        console.log('Connected to WebSocket Gateway');
        resolve();
      };
      
      this.ws.onerror = reject;
    });
  }

  async authenticate() {
    const authMessage = this.createAuthMessage();
    this.ws.send(JSON.stringify(authMessage));
    
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Auth timeout')), 10000);
      
      this.ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.channel === 'auth' && message.type === 'response') {
          clearTimeout(timeout);
          this.authenticated = true;
          resolve();
        }
      };
    });
  }

  async register() {
    const regMessage = this.createRegistrationMessage();
    this.ws.send(JSON.stringify(regMessage));
    
    // Handle registration init response and complete
    // ... (implementation details)
  }

  startHeartbeat() {
    this.heartbeatInterval = setInterval(() => {
      const heartbeat = this.createHeartbeatMessage();
      this.ws.send(JSON.stringify(heartbeat));
    }, 30000);
  }
}
```

---

## 📚 ADDITIONAL RESOURCES

- **WebSocket Specification**: [RFC 6455](https://tools.ietf.org/html/rfc6455)
- **Ed25519 Signatures**: [RFC 8032](https://tools.ietf.org/html/rfc8032)
- **ChaCha20-Poly1305**: [RFC 8439](https://tools.ietf.org/html/rfc8439)
- **JWT Tokens**: [RFC 7519](https://tools.ietf.org/html/rfc7519)

---

*This guide provides complete information for implementing agent WebSocket connections to the AegisFlux backend. Follow the protocols exactly for successful integration.*

