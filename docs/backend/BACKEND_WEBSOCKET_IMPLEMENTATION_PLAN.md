# Backend WebSocket Implementation Plan
## AegisFlux Agent Communication Infrastructure

---

## 🎯 **Implementation Overview**

This document provides a comprehensive roadmap for implementing the backend WebSocket infrastructure to support the new agent communication system. The agent side is **100% complete** and ready for integration.

### **Current Status:**
- ✅ **Agent Side**: Complete WebSocket infrastructure implemented
- 🔄 **Backend Side**: Implementation required (this document)
- 📊 **Progress**: 0% backend implementation, 100% agent implementation

---

## 📋 **Implementation Phases**

### **Phase 1: Core WebSocket Infrastructure (Week 1-2)**
**Priority: HIGH | Effort: 40 hours | Dependencies: None**

#### **1.1 WebSocket Gateway Service**
- [ ] Create `backend/websocket-gateway/` service
- [ ] Implement HTTP to WebSocket upgrade handler
- [ ] Add connection management and health monitoring
- [ ] Create agent connection tracking
- [ ] Add graceful connection handling and cleanup

**Key Files to Create:**
```
backend/websocket-gateway/
├── cmd/
│   └── gateway/
│       └── main.go
├── internal/
│   ├── gateway/
│   │   ├── server.go
│   │   ├── connection.go
│   │   └── handler.go
│   ├── auth/
│   │   ├── service.go
│   │   └── types.go
│   └── types/
│       └── messages.go
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

**Implementation Tasks:**
- [ ] HTTP upgrade handler with WebSocket protocol
- [ ] Agent connection struct with metadata
- [ ] Connection pool management
- [ ] Health check and monitoring
- [ ] Graceful shutdown handling

#### **1.2 Authentication Service**
- [ ] Implement Ed25519 signature verification
- [ ] Add agent public key validation
- [ ] Create session token generation
- [ ] Add authentication request/response handling
- [ ] Implement shared key derivation

**Implementation Tasks:**
- [ ] Ed25519 key pair management
- [ ] Signature verification logic
- [ ] Session token generation (JWT)
- [ ] Agent registry integration
- [ ] Key derivation algorithm (SHA256)

#### **1.3 Message Router Service**
- [ ] Create channel-based message routing
- [ ] Add message decryption and verification
- [ ] Implement handler registration system
- [ ] Add message broadcasting capabilities
- [ ] Create message queue integration

**Implementation Tasks:**
- [ ] Channel management system
- [ ] Message type routing
- [ ] Handler registration and execution
- [ ] Message queue integration
- [ ] Broadcast functionality

### **Phase 2: Security Implementation (Week 2-3)**
**Priority: HIGH | Effort: 30 hours | Dependencies: Phase 1**

#### **2.1 Encryption Service**
- [ ] Implement ChaCha20-Poly1305 encryption/decryption
- [ ] Add Ed25519 signature verification
- [ ] Create nonce generation and validation
- [ ] Add message integrity checking
- [ ] Implement key rotation support

**Implementation Tasks:**
- [ ] ChaCha20-Poly1305 cipher implementation
- [ ] Message encryption/decryption
- [ ] Digital signature verification
- [ ] Nonce generation (cryptographically secure)
- [ ] Message integrity validation

#### **2.2 Connection Manager**
- [ ] Track active agent connections
- [ ] Implement health monitoring
- [ ] Add message broadcasting
- [ ] Create connection cleanup
- [ ] Add metrics and monitoring

**Implementation Tasks:**
- [ ] Active connection tracking
- [ ] Health check implementation
- [ ] Message broadcasting logic
- [ ] Connection lifecycle management
- [ ] Metrics collection and reporting

### **Phase 3: Database & API Integration (Week 3-4)**
**Priority: MEDIUM | Effort: 25 hours | Dependencies: Phase 2**

#### **3.1 Database Schema Updates**
- [ ] Create agent_connections table
- [ ] Create message_queue table
- [ ] Create communication_channels table
- [ ] Add necessary indexes
- [ ] Create migration scripts

**Database Tables:**
```sql
-- Agent connections tracking
CREATE TABLE agent_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    session_token VARCHAR(255) NOT NULL,
    connected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Message queue for reliable delivery
CREATE TABLE message_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL,
    channel VARCHAR(255) NOT NULL,
    message_type VARCHAR(50) NOT NULL,
    payload TEXT NOT NULL,
    encrypted_payload TEXT NOT NULL,
    nonce TEXT NOT NULL,
    signature TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '1 hour')
);

-- Communication channels management
CREATE TABLE communication_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL,
    channel_type VARCHAR(50) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id, channel_name)
);
```

#### **3.2 New API Endpoints**
- [ ] WebSocket connection endpoint
- [ ] Agent status endpoint
- [ ] Send message to agent endpoint
- [ ] Broadcast to all agents endpoint
- [ ] Connection management endpoints

**API Endpoints:**
```
GET /ws/agent                    # WebSocket connection
GET /api/v1/agents/{id}/status   # Agent status
POST /api/v1/agents/{id}/send    # Send to specific agent
POST /api/v1/agents/broadcast    # Broadcast to all agents
GET /api/v1/agents               # List all agents
DELETE /api/v1/agents/{id}       # Disconnect agent
```

### **Phase 4: Integration & Testing (Week 4-5)**
**Priority: HIGH | Effort: 20 hours | Dependencies: Phase 3**

#### **4.1 Integration Testing**
- [ ] Test agent connection flow
- [ ] Validate authentication process
- [ ] Test message encryption/decryption
- [ ] Verify channel-based routing
- [ ] Test automatic reconnection

#### **4.2 Performance Testing**
- [ ] Load testing with multiple agents
- [ ] Message throughput testing
- [ ] Connection scaling testing
- [ ] Memory usage optimization
- [ ] Latency measurement

#### **4.3 Security Testing**
- [ ] Authentication bypass testing
- [ ] Message tampering detection
- [ ] Replay attack prevention
- [ ] Session management security
- [ ] Encryption strength validation

---

## 🔧 **Technical Specifications**

### **WebSocket Connection Protocol**
```http
GET /ws/agent HTTP/1.1
Host: backend.aegis.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: {base64_encoded_key}
Sec-WebSocket-Version: 13
X-Agent-ID: {agent_id}
X-Agent-Public-Key: {base64_encoded_ed25519_public_key}
User-Agent: Aegis-Agent/1.0
```

### **Authentication Flow**
1. Agent sends authentication request with Ed25519 signature
2. Backend verifies signature using agent's public key
3. Backend generates session token (JWT)
4. Both sides derive shared encryption key using SHA256
5. Connection established with encrypted communication

### **Message Structure**
```json
{
  "id": "msg_1695326400_123456789",
  "type": "request|response|event|heartbeat|ack",
  "channel": "agent.001.policies",
  "payload": "base64_encoded_encrypted_payload",
  "timestamp": 1695326400,
  "nonce": "base64_encoded_12_byte_nonce",
  "signature": "base64_encoded_ed25519_signature",
  "headers": {
    "priority": "high",
    "retry_count": "0"
  }
}
```

### **Communication Channels**
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

---

## 📊 **Implementation Progress Tracking**

### **Phase 1: Core WebSocket Infrastructure**
- [ ] **1.1 WebSocket Gateway Service** (0/5 tasks)
  - [ ] HTTP upgrade handler
  - [ ] Connection management
  - [ ] Health monitoring
  - [ ] Agent tracking
  - [ ] Graceful shutdown
- [ ] **1.2 Authentication Service** (0/5 tasks)
  - [ ] Ed25519 verification
  - [ ] Key validation
  - [ ] Session tokens
  - [ ] Authentication flow
  - [ ] Key derivation
- [ ] **1.3 Message Router Service** (0/5 tasks)
  - [ ] Channel routing
  - [ ] Message decryption
  - [ ] Handler registration
  - [ ] Message queue
  - [ ] Broadcasting

### **Phase 2: Security Implementation**
- [ ] **2.1 Encryption Service** (0/5 tasks)
  - [ ] ChaCha20-Poly1305
  - [ ] Signature verification
  - [ ] Nonce generation
  - [ ] Message integrity
  - [ ] Key rotation
- [ ] **2.2 Connection Manager** (0/5 tasks)
  - [ ] Connection tracking
  - [ ] Health monitoring
  - [ ] Broadcasting
  - [ ] Cleanup
  - [ ] Metrics

### **Phase 3: Database & API Integration**
- [ ] **3.1 Database Schema** (0/5 tasks)
  - [ ] agent_connections table
  - [ ] message_queue table
  - [ ] communication_channels table
  - [ ] Indexes
  - [ ] Migrations
- [ ] **3.2 API Endpoints** (0/5 tasks)
  - [ ] WebSocket endpoint
  - [ ] Agent status
  - [ ] Send message
  - [ ] Broadcast
  - [ ] Management

### **Phase 4: Integration & Testing**
- [ ] **4.1 Integration Testing** (0/5 tasks)
  - [ ] Connection flow
  - [ ] Authentication
  - [ ] Encryption
  - [ ] Routing
  - [ ] Reconnection
- [ ] **4.2 Performance Testing** (0/5 tasks)
  - [ ] Load testing
  - [ ] Throughput
  - [ ] Scaling
  - [ ] Memory
  - [ ] Latency
- [ ] **4.3 Security Testing** (0/5 tasks)
  - [ ] Auth bypass
  - [ ] Tampering
  - [ ] Replay attacks
  - [ ] Session security
  - [ ] Encryption strength

---

## 🎯 **Success Criteria**

### **Functional Requirements**
- [ ] Agents can establish secure WebSocket connections
- [ ] Bidirectional communication works reliably
- [ ] All message types are supported (request, response, event, heartbeat, ack)
- [ ] Authentication and encryption work correctly
- [ ] Channel-based routing functions properly

### **Performance Requirements**
- [ ] Support 1000+ concurrent agent connections
- [ ] Message latency < 100ms
- [ ] 99.9% message delivery success rate
- [ ] Automatic reconnection within 30 seconds
- [ ] Memory usage < 100MB per 100 connections

### **Security Requirements**
- [ ] All communications encrypted end-to-end
- [ ] Mutual authentication working
- [ ] No message tampering possible
- [ ] Session management secure
- [ ] Key rotation support

---

## 🚨 **Risk Assessment**

### **High Risk**
- **Backend Complexity**: Significant increase in backend complexity
- **Database Load**: Additional tables and queries for connection management
- **Network Dependencies**: WebSocket connections require stable network

### **Medium Risk**
- **Migration Complexity**: Gradual migration required
- **Testing**: More complex testing scenarios
- **Monitoring**: Additional monitoring requirements

### **Low Risk**
- **Agent Changes**: Agent-side changes are well-defined
- **Security**: Well-established encryption standards
- **Performance**: Expected performance improvements

---

## 📞 **Resources & Support**

### **Documentation References**
- `new_agent/BACKEND_TEAM_HANDOFF.md` - Complete handoff package
- `new_agent/PHASE_3_BACKEND_ARCHITECTURE.md` - Detailed architecture
- `new_agent/WEBSOCKET_PROTOCOL_SPECIFICATION.md` - Protocol details
- `new_agent/BACKEND_TEST_AGENT.md` - Testing guide

### **Test Agent**
- **Location**: `new_agent/BACKEND_TEST_AGENT.md`
- **Purpose**: Validate backend WebSocket implementation
- **Usage**: Test connection, authentication, and messaging

### **Contact Information**
- **Primary**: Steven Gerhart (steve@dentro.io)
- **Repository**: `/Users/stevengerhart/workspace/github/sgerhart/aegisflux`
- **Agent Documentation**: `new_agent/` folder

---

## 🚀 **Getting Started**

### **Step 1: Choose Implementation Phase**
Select which phase to start with based on priorities:
1. **Phase 1**: Core WebSocket Infrastructure (Recommended)
2. **Phase 2**: Security Implementation
3. **Phase 3**: Database & API Integration
4. **Phase 4**: Integration & Testing

### **Step 2: Set Up Development Environment**
```bash
# Create new service directory
mkdir -p backend/websocket-gateway/{cmd/gateway,internal/{gateway,auth,types}}

# Initialize Go module
cd backend/websocket-gateway
go mod init github.com/sgerhart/aegisflux/websocket-gateway
```

### **Step 3: Start Implementation**
Follow the detailed tasks in each phase section above.

---

**This document will be updated as implementation progresses. Check off completed tasks and update progress percentages.**

**Last Updated**: September 21, 2025
**Status**: Ready for implementation
**Next Action**: Choose starting phase and begin implementation
