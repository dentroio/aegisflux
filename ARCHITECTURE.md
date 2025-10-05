# 🏗️ AEGISFLUX ARCHITECTURE

## 📋 SYSTEM OVERVIEW

AegisFlux is a comprehensive network security platform that uses eBPF (extended Berkeley Packet Filter) for real-time network policy enforcement. The system consists of distributed agents that collect telemetry, a backend that processes data and makes decisions, and a management interface for policy creation and monitoring.

## 🎯 CORE PRINCIPLES

- **Real-time Policy Enforcement**: eBPF programs deployed to kernel space
- **AI-Powered Decision Making**: LLM agents for intelligent policy creation
- **Scalable Architecture**: Microservices with message-based communication
- **Secure Communication**: WebSocket + Ed25519 signatures + ChaCha20-Poly1305 encryption
- **Event-Driven Processing**: NATS for reliable message delivery

---

## 🏢 SERVICE ARCHITECTURE

### 🔧 **CORE INFRASTRUCTURE SERVICES**

#### **NATS Message Bus**
- **Port**: 4222 (client), 8222 (monitoring)
- **Role**: Central message broker for all inter-service communication
- **Features**: JetStream for reliable delivery, clustering support
- **Used By**: All backend services for async communication

#### **TimescaleDB (PostgreSQL)**
- **Port**: 5432
- **Role**: Primary relational database for time-series data
- **Features**: Time-series optimization, full SQL support
- **Stores**: Agent telemetry, events, metrics, configuration

#### **Neo4j Graph Database**
- **Port**: 7474 (HTTP), 7687 (Bolt)
- **Role**: Graph database for relationship modeling
- **Features**: APOC plugins, Cypher query language
- **Stores**: Network topology, entity relationships, attack graphs

#### **Vault (HashiCorp)**
- **Port**: 8200
- **Role**: Secrets management and cryptographic operations
- **Features**: Key signing, secret storage, PKI management
- **Used By**: BPF Registry for artifact signing

---

### 🎯 **AGENT COMMUNICATION SERVICES**

#### **WebSocket Gateway**
- **Port**: 8080
- **Role**: Real-time bidirectional communication with agents
- **Features**: 
  - WebSocket connection management
  - Ed25519 authentication
  - ChaCha20-Poly1305 encryption
  - Message routing and queuing
  - Heartbeat monitoring
- **Protocol**: Secure WebSocket with custom message format
- **Integrates With**: Actions API for registration

#### **Actions API**
- **Port**: 8083
- **Role**: Agent registration and management
- **Features**:
  - Agent registration (init/complete)
  - Agent status and configuration
  - Message broadcasting to agents
  - Agent lifecycle management
- **Storage**: In-memory agent registry
- **Integrates With**: WebSocket Gateway

---

### 📊 **DATA PROCESSING SERVICES**

#### **Ingest Service**
- **Port**: 50052 (gRPC), 9091 (metrics)
- **Role**: Agent telemetry ingestion and processing
- **Features**:
  - gRPC endpoint for agent data
  - Protocol buffer message handling
  - Data validation and enrichment
  - NATS publishing for downstream services
- **Input**: Agent telemetry via gRPC
- **Output**: Processed events via NATS

#### **ETL-Enrich Service**
- **Language**: Python
- **Role**: Event enrichment and data transformation
- **Features**:
  - NATS message consumption
  - CVE data enrichment
  - Network data correlation
  - Multi-database writing (TimescaleDB, Neo4j)
- **Input**: Raw events from NATS
- **Output**: Enriched data to databases

#### **Correlator Service**
- **Port**: 8082
- **Role**: Rules engine and event correlation
- **Features**:
  - YAML-based rule definitions
  - Temporal correlation windows
  - Finding generation and forwarding
  - Hot-reload rule updates
- **Input**: Enriched events from ETL
- **Output**: Security findings and alerts

---

### 🤖 **AI & DECISION SERVICES**

#### **Decision Service**
- **Port**: 8087
- **Role**: AI-powered policy decision making
- **Features**:
  - LLM agent orchestration (Planner, Explainer, Policy Writer, Segmenter)
  - Natural language policy interpretation
  - Guardrails and safety checks
  - Policy validation and optimization
- **LLM Agents**:
  - **Planner**: High-level policy planning
  - **Explainer**: Policy explanation and documentation
  - **Policy Writer**: eBPF policy generation
  - **Segmenter**: Policy segmentation and deployment
- **Input**: Security findings and policy requests
- **Output**: Structured policy decisions

#### **Segmenter Service** *(Stub - Needs Implementation)*
- **Port**: 8086 (planned)
- **Role**: Policy segmentation and deployment planning
- **Features**: *(To be implemented)*
  - Policy decomposition
  - Deployment strategy planning
  - Rollout coordination
  - Rollback management
- **Status**: Currently stub only, needs full implementation

---

### 🗄️ **STORAGE & REGISTRY SERVICES**

#### **BPF Registry**
- **Port**: 8090
- **Role**: eBPF artifact storage and management
- **Features**:
  - Artifact versioning and storage
  - Ed25519 signature verification
  - Vault integration for signing
  - Artifact distribution to agents
- **Storage**: File system with Vault signing
- **Input**: Compiled eBPF artifacts
- **Output**: Signed, versioned artifacts

#### **Config API**
- **Port**: 8085
- **Role**: Configuration management and service discovery
- **Features**:
  - Service configuration storage
  - Dynamic configuration updates
  - Service health monitoring
  - Configuration versioning
- **Storage**: PostgreSQL database
- **Used By**: All services for configuration

---

### 🎛️ **ORCHESTRATION SERVICES**

#### **Orchestrator**
- **Port**: 8084
- **Role**: MapSnapshot orchestration and policy deployment
- **Features**:
  - MapSnapshot processing and compilation
  - eBPF program compilation
  - Policy deployment coordination
  - Rollout management
- **Integrates With**: Decision API, BPF Registry, Vault
- **Input**: Policy decisions and MapSnapshots
- **Output**: Compiled eBPF artifacts

#### **CVE Sync Service**
- **Role**: CVE data synchronization
- **Features**:
  - CVE database updates
  - Vulnerability data enrichment
  - Scheduled synchronization
- **Input**: External CVE databases
- **Output**: CVE data to ETL service

---

### 🖥️ **USER INTERFACE**

#### **Console UI**
- **Technology**: Next.js, React, Tailwind CSS
- **Port**: 3000 (development)
- **Role**: Web-based management interface
- **Features**:
  - Agent management and monitoring
  - Policy creation and deployment
  - Real-time dashboard
  - System health monitoring
- **Integrates With**: All backend services via REST APIs

---

## 🔄 DATA FLOW ARCHITECTURE

### **1. Agent Telemetry Flow**
```
Agent → Ingest (gRPC) → NATS → ETL-Enrich → TimescaleDB/Neo4j
```

### **2. Policy Decision Flow**
```
Security Finding → Correlator → Decision (LLM) → Orchestrator → BPF Registry → Agent
```

### **3. Agent Communication Flow**
```
Agent ↔ WebSocket Gateway ↔ Actions API
```

### **4. Configuration Flow**
```
All Services → Config API → PostgreSQL
```

---

## 🔐 SECURITY ARCHITECTURE

### **Agent Authentication**
1. **Connection**: WebSocket with custom headers
2. **Authentication**: Ed25519 signature verification
3. **Encryption**: ChaCha20-Poly1305 for message encryption
4. **Session**: JWT tokens with expiration

### **Service-to-Service Security**
1. **Internal Network**: Docker network isolation
2. **Authentication**: Service-specific tokens
3. **Encryption**: TLS for external communication
4. **Secrets**: Vault-managed keys and certificates

### **Data Security**
1. **At Rest**: Database encryption
2. **In Transit**: TLS/WebSocket encryption
3. **Processing**: Secure memory handling
4. **Storage**: Signed and verified artifacts

---

## 📊 SCALABILITY CONSIDERATIONS

### **Horizontal Scaling**
- **Stateless Services**: WebSocket Gateway, Actions API, Decision
- **Database Scaling**: TimescaleDB clustering, Neo4j clustering
- **Message Scaling**: NATS clustering and JetStream

### **Performance Optimization**
- **Connection Pooling**: Database connections
- **Caching**: Redis for frequently accessed data
- **Load Balancing**: Multiple service instances
- **Async Processing**: NATS for non-blocking operations

---

## 🚀 DEPLOYMENT ARCHITECTURE

### **Development Environment**
- **Docker Compose**: Single-node deployment
- **Local Development**: All services containerized
- **Hot Reload**: Development-friendly configurations

### **Production Environment**
- **Kubernetes**: Multi-node cluster deployment
- **Service Mesh**: Istio for service communication
- **Monitoring**: Prometheus + Grafana
- **Logging**: ELK stack or similar

---

## 🔧 TECHNOLOGY STACK

### **Backend Services**
- **Language**: Go (primary), Python (ETL)
- **Databases**: PostgreSQL/TimescaleDB, Neo4j
- **Message Queue**: NATS with JetStream
- **Container**: Docker + Kubernetes

### **Agent Communication**
- **Protocol**: WebSocket with custom message format
- **Authentication**: Ed25519 signatures
- **Encryption**: ChaCha20-Poly1305
- **Serialization**: JSON + Base64

### **Frontend**
- **Framework**: Next.js with React
- **Styling**: Tailwind CSS
- **State Management**: React hooks
- **API Communication**: Fetch API

---

## 📈 MONITORING & OBSERVABILITY

### **Metrics Collection**
- **Service Metrics**: Prometheus endpoints
- **Application Metrics**: Custom business metrics
- **System Metrics**: Node exporter
- **Database Metrics**: Database-specific exporters

### **Logging**
- **Structured Logging**: JSON format
- **Log Aggregation**: Centralized log collection
- **Log Analysis**: Search and alerting
- **Retention**: Configurable retention policies

### **Tracing**
- **Distributed Tracing**: OpenTelemetry
- **Request Tracing**: End-to-end request tracking
- **Performance Monitoring**: Latency and throughput
- **Error Tracking**: Exception monitoring

---

*This architecture document provides a comprehensive overview of the AegisFlux system. Each service is designed to be independently deployable, scalable, and maintainable while working together to provide a complete network security solution.*

