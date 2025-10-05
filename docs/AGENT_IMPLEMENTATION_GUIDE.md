# Agent Implementation Guide

**Last Updated**: September 28, 2025  
**Status**: ✅ **Backend Complete** - All services operational and ready for agent integration  
**Priority**: Production-ready agent implementation with full backend support

## 🎯 Executive Summary

The Aegis backend is now **100% complete** with all 8 services fully implemented and operational. This guide provides agent teams with comprehensive information on how to implement real functionality and integrate with the complete backend system.

## ✅ Backend Services Status

### All Services Operational (8/8)

| Service | Status | Port | Capabilities |
|---------|--------|------|--------------|
| **WebSocket Gateway** | ✅ Operational | 8080 | Agent communication, authentication |
| **Actions API** | ✅ Operational | 8083 | Agent registration, management, policy deployment |
| **Decision Service** | ✅ Operational | 8087 | Policy generation from high-level intents |
| **Orchestrator** | ✅ Operational | 8081 | eBPF compilation and deployment |
| **BPF Registry** | ✅ Operational | 8090 | Artifact storage and Vault signing |
| **Ingest Service** | ✅ Operational | 8086/8088 | Event validation and NATS publishing |
| **ETL-Enrich** | ✅ Operational | - | Database integration and data enrichment |
| **Segmenter** | ✅ Operational | 8089 | Network segmentation and policy generation |

## 🔌 Agent Integration Points

### 1. WebSocket Communication (Primary)

**Endpoint**: `ws://localhost:8080/ws/agent`

**Authentication Flow**:
```go
// Step 1: Connect to WebSocket
conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws/agent", nil)

// Step 2: Send authentication message
authMsg := types.SecureMessage{
    ID:        generateMessageID(),
    Type:      types.MessageTypeRequest,
    Channel:   "auth",
    Timestamp: time.Now().Unix(),
    Payload:   base64.StdEncoding.EncodeToString(authData),
    Headers:   make(map[string]string),
}

// Step 3: Send registration message
regMsg := types.SecureMessage{
    ID:        generateMessageID(),
    Type:      types.MessageTypeRequest,
    Channel:   "agent.registration",
    Timestamp: time.Now().Unix(),
    Payload:   base64.StdEncoding.EncodeToString(regData),
    Headers:   make(map[string]string),
}
```

**Message Format**:
```go
type SecureMessage struct {
    ID        string            `json:"id"`
    Type      string            `json:"type"`      // "request", "response", "event"
    Channel   string            `json:"channel"`   // "auth", "agent.registration", etc.
    Timestamp int64             `json:"timestamp"`
    Payload   string            `json:"payload"`   // base64-encoded JSON
    Headers   map[string]string `json:"headers"`
}
```

### 2. Agent Registration (HTTP)

**Endpoints**:
- `POST /agents/register/init` - Initialize registration
- `POST /agents/register/complete` - Complete registration

**Registration Flow**:
```go
// Step 1: Registration Init
initReq := map[string]interface{}{
    "org_id":          "default-org",
    "host_id":         agentHostID,
    "agent_pubkey":    base64.StdEncoding.EncodeToString(publicKey),
    "machine_id_hash": machineIDHash,
    "agent_version":   "1.0.0",
    "capabilities":    []string{"telemetry", "observability", "analysis"},
    "platform":        platformInfo,
    "network":         networkInfo,
}

// Step 2: Registration Complete
completeReq := map[string]interface{}{
    "registration_id": registrationID,
    "host_id":        agentHostID,
    "signature":      base64.StdEncoding.EncodeToString(signature),
}
```

### 3. Policy Deployment Integration

**Policy Reception**:
```go
// Listen for policy messages on WebSocket
func handlePolicyMessage(msg types.SecureMessage) {
    if msg.Channel == "agent.policies" {
        // Decode policy payload
        payload, _ := base64.StdEncoding.DecodeString(msg.Payload)
        var policy PolicyData
        json.Unmarshal(payload, &policy)
        
        // Deploy eBPF policy
        deployPolicy(policy)
    }
}
```

**Policy Deployment**:
```go
func deployPolicy(policy PolicyData) error {
    // 1. Load eBPF program
    collection, err := ebpf.LoadCollection(policy.Program)
    
    // 2. Attach to network interface
    err = collection.Programs["block_icmp"].Attach(networkInterface)
    
    // 3. Apply firewall rules
    err = applyFirewallRules(policy.Rules)
    
    return err
}
```

### 4. Event Reporting

**Event Format**:
```go
type Event struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`        // "connect", "exec", "file", etc.
    Source      string                 `json:"source"`      // Process path
    Timestamp   int64                  `json:"timestamp"`   // Unix timestamp
    Metadata    map[string]interface{} `json:"metadata"`    // Host info, process info
    Payload     []byte                 `json:"payload"`     // Event-specific data
}
```

**Event Submission via gRPC**:
```go
// Connect to Ingest Service
conn, err := grpc.Dial("localhost:8086", grpc.WithInsecure())
client := protos.NewIngestClient(conn)

// Stream events
stream, err := client.PostEvents(ctx)
for _, event := range events {
    err := stream.Send(event)
}
```

## 🏗️ Agent Module Implementation

### 1. Telemetry Module

**Replace Simulation with Real Data**:
```go
type TelemetryModule struct {
    collector *system.Collector
    publisher *EventPublisher
}

func (tm *TelemetryModule) CollectMetrics() (*TelemetryData, error) {
    // Real system metrics collection
    cpu, err := tm.collector.CPUUsage()
    memory, err := tm.collector.MemoryUsage()
    network, err := tm.collector.NetworkStats()
    disk, err := tm.collector.DiskUsage()
    
    return &TelemetryData{
        CPU:     cpu,
        Memory:  memory,
        Network: network,
        Disk:    disk,
        Timestamp: time.Now().Unix(),
    }, nil
}
```

### 2. Observability Module

**Real System Monitoring**:
```go
type ObservabilityModule struct {
    processMonitor *ProcessMonitor
    networkMonitor *NetworkMonitor
    fileMonitor    *FileMonitor
}

func (om *ObservabilityModule) MonitorSystem() {
    // Process monitoring
    go om.processMonitor.WatchProcesses(func(process ProcessEvent) {
        om.publishEvent(process)
    })
    
    // Network monitoring
    go om.networkMonitor.WatchConnections(func(conn NetworkEvent) {
        om.publishEvent(conn)
    })
    
    // File system monitoring
    go om.fileMonitor.WatchFiles(func(file FileEvent) {
        om.publishEvent(file)
    })
}
```

### 3. Analysis Module

**Real Dependency Analysis**:
```go
type AnalysisModule struct {
    scanner *DependencyScanner
    analyzer *VulnerabilityAnalyzer
}

func (am *AnalysisModule) AnalyzeDependencies() (*AnalysisResult, error) {
    // Scan installed packages
    packages, err := am.scanner.ScanInstalledPackages()
    
    // Analyze for vulnerabilities
    vulnerabilities, err := am.analyzer.AnalyzeVulnerabilities(packages)
    
    // Generate dependency graph
    graph, err := am.analyzer.BuildDependencyGraph(packages)
    
    return &AnalysisResult{
        Packages:       packages,
        Vulnerabilities: vulnerabilities,
        DependencyGraph: graph,
    }, nil
}
```

### 4. Threat Intelligence Module

**Real Threat Detection**:
```go
type ThreatIntelligenceModule struct {
    iocScanner    *IOCScanner
    behaviorAnalyzer *BehaviorAnalyzer
    threatFeeds   []ThreatFeed
}

func (tim *ThreatIntelligenceModule) DetectThreats() (*ThreatReport, error) {
    // Scan for IOCs
    iocs, err := tim.iocScanner.ScanSystem()
    
    // Analyze behavior patterns
    behaviors, err := tim.behaviorAnalyzer.AnalyzeBehaviors()
    
    // Check against threat feeds
    threats, err := tim.checkThreatFeeds(iocs, behaviors)
    
    return &ThreatReport{
        IOCs:      iocs,
        Behaviors: behaviors,
        Threats:   threats,
    }, nil
}
```

## 📊 Database Integration

### 1. TimescaleDB Integration

**Time-series Data Storage**:
```go
type MetricsWriter struct {
    db *sql.DB
}

func (mw *MetricsWriter) WriteMetrics(metrics *TelemetryData) error {
    query := `
        INSERT INTO agent_metrics (host_id, timestamp, cpu_usage, memory_usage, network_bytes)
        VALUES ($1, $2, $3, $4, $5)
    `
    
    _, err := mw.db.Exec(query, 
        metrics.HostID,
        time.Unix(metrics.Timestamp, 0),
        metrics.CPU.Usage,
        metrics.Memory.Usage,
        metrics.Network.BytesTransferred,
    )
    
    return err
}
```

### 2. Neo4j Integration

**Graph Data Population**:
```go
type GraphWriter struct {
    driver neo4j.Driver
}

func (gw *GraphWriter) WriteProcessData(process ProcessData) error {
    session := gw.driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()
    
    _, err := session.Run(`
        MERGE (p:Process {pid: $pid, host_id: $host_id})
        SET p.name = $name, p.path = $path, p.owner = $owner
        MERGE (h:Host {host_id: $host_id})
        MERGE (p)-[:RUNS_ON]->(h)
    `, map[string]interface{}{
        "pid":     process.PID,
        "host_id": process.HostID,
        "name":    process.Name,
        "path":    process.Path,
        "owner":   process.Owner,
    })
    
    return err
}
```

## 🔐 Security Implementation

### 1. Ed25519 Authentication

**Key Generation**:
```go
func generateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
    publicKey, privateKey, err := ed25519.GenerateKey(nil)
    return publicKey, privateKey, err
}
```

**Message Signing**:
```go
func signMessage(privateKey ed25519.PrivateKey, message []byte) ([]byte, error) {
    signature := ed25519.Sign(privateKey, message)
    return signature, nil
}
```

**Signature Verification**:
```go
func verifySignature(publicKey ed25519.PublicKey, message, signature []byte) bool {
    return ed25519.Verify(publicKey, message, signature)
}
```

### 2. Message Encryption

**ChaCha20-Poly1305 Encryption**:
```go
func encryptMessage(key []byte, plaintext []byte) ([]byte, error) {
    nonce := make([]byte, 12)
    rand.Read(nonce)
    
    cipher, err := chacha20poly1305.New(key)
    if err != nil {
        return nil, err
    }
    
    ciphertext := cipher.Seal(nil, nonce, plaintext, nil)
    return append(nonce, ciphertext...), nil
}
```

## 🚀 Deployment Guide

### 1. Agent Configuration

**Environment Variables**:
```bash
# WebSocket Gateway connection
WEBSOCKET_URL=ws://localhost:8080/ws/agent

# Agent identification
AGENT_HOST_ID=agent-001
AGENT_VERSION=1.0.0
AGENT_ORG_ID=default-org

# Security
AGENT_PRIVATE_KEY_PATH=/etc/aegis/agent.key
AGENT_PUBLIC_KEY_PATH=/etc/aegis/agent.pub

# Database connections
TIMESCALE_URL=postgresql://postgres:password@localhost:5432/aegisflux
NEO4J_URL=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=password

# NATS connection
NATS_URL=nats://localhost:4222
```

### 2. Service Dependencies

**Required Services**:
- NATS (Message Queue)
- TimescaleDB (Time-series Database)
- Neo4j (Graph Database)
- Vault (Secrets Management)

**Service Startup Order**:
1. Databases (TimescaleDB, Neo4j)
2. NATS
3. Vault
4. Backend Services
5. Agent

### 3. Health Checks

**Agent Health Endpoint**:
```go
func (a *Agent) HealthCheck() *HealthStatus {
    return &HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now().Unix(),
        Modules: map[string]string{
            "telemetry":      a.telemetryModule.Status(),
            "observability":  a.observabilityModule.Status(),
            "analysis":       a.analysisModule.Status(),
            "threat_intel":   a.threatIntelModule.Status(),
        },
        Metrics: a.getMetrics(),
    }
}
```

## 📋 Implementation Checklist

### Phase 1: Core Infrastructure (Week 1)
- [ ] Implement real WebSocket communication
- [ ] Implement Ed25519 authentication
- [ ] Implement agent registration flow
- [ ] Set up database connections

### Phase 2: Module Implementation (Week 2)
- [ ] Replace telemetry simulation with real data collection
- [ ] Replace observability simulation with real monitoring
- [ ] Replace analysis simulation with real dependency scanning
- [ ] Replace threat intelligence simulation with real detection

### Phase 3: Integration (Week 3)
- [ ] Integrate with TimescaleDB for metrics storage
- [ ] Integrate with Neo4j for graph data
- [ ] Implement event streaming to Ingest Service
- [ ] Implement policy deployment from backend

### Phase 4: Testing & Optimization (Week 4)
- [ ] End-to-end testing with backend
- [ ] Performance optimization
- [ ] Security testing
- [ ] Production deployment

## 🆘 Troubleshooting

### Common Issues

1. **Authentication Failures**
   - Verify Ed25519 key pair generation
   - Check signature data format
   - Ensure proper base64 encoding

2. **Connection Issues**
   - Verify WebSocket URL and port
   - Check firewall settings
   - Ensure backend services are running

3. **Database Connection Issues**
   - Verify connection strings
   - Check database credentials
   - Ensure databases are accessible

4. **Policy Deployment Issues**
   - Check eBPF permissions
   - Verify policy format
   - Ensure kernel compatibility

### Support Resources

- **Backend API Documentation**: See `docs/API_REFERENCE.md`
- **WebSocket Protocol**: See `docs/WEBSOCKET_PROTOCOL.md`
- **Database Schema**: See `data/graph/init.cypher` and `data/timeseries/init.sql`
- **Example Implementations**: See `examples/` directory

## 📞 Contact Information

- **Backend Team**: Available for integration support
- **Documentation**: All guides available in `docs/` directory
- **Issue Tracking**: Use GitHub issues for bug reports and feature requests

---

**Status**: ✅ **Backend Complete** - Ready for agent implementation  
**Last Updated**: September 28, 2025  
**Next Review**: October 5, 2025

