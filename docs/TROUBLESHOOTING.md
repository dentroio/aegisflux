# AegisFlux Troubleshooting Guide

## Common Issues and Solutions

### Agent Connection Issues

#### Agent Not Connecting to Backend

**Symptoms**:
- Agent process running but not registered
- WebSocket connection failures
- Authentication errors in logs

**Diagnosis**:
```bash
# Check agent logs
journalctl -u aegis-agent -f

# Check backend connectivity
curl http://localhost:8080/health
curl http://localhost:8083/health

# Test WebSocket connection
wscat -c ws://localhost:8080/ws/agent
```

**Solutions**:
1. **Network connectivity**:
   ```bash
   # Test basic connectivity
   ping backend-host
   telnet backend-host 8080
   telnet backend-host 8083
   ```

2. **Firewall issues**:
   ```bash
   # Check if ports are open
   nmap -p 8080,8083 backend-host
   ```

3. **Authentication problems**:
   - Verify Ed25519 key pair generation
   - Check signature computation
   - Ensure proper message format

#### Agent Registration Loop

**Symptoms**:
- Multiple agent registrations in backend
- Agent repeatedly registering and disconnecting
- Growing number of registered agents

**Root Causes**:
1. **Registration retry logic**: Agent creates new registration on each retry
2. **Session persistence**: Agent doesn't remember successful registrations
3. **Connection management**: Improper WebSocket reconnection handling

**Solutions**:
1. **Fix retry logic**:
   ```go
   // Don't create new registration on retry
   if existingRegistration != nil {
       // Reuse existing registration
       return existingRegistration
   }
   ```

2. **Implement session persistence**:
   ```go
   // Store registration state
   registrationState := &RegistrationState{
       ID: registrationID,
       Completed: false,
       Timestamp: time.Now(),
   }
   saveRegistrationState(registrationState)
   ```

3. **Proper connection management**:
   ```go
   // Reconnect without re-registering
   if isRegistered {
       connectWebSocket()
   } else {
       registerAndConnect()
   }
   ```

### Backend Service Issues

#### Actions API Not Responding

**Symptoms**:
- HTTP 500 errors
- Connection timeouts
- Service unavailable

**Diagnosis**:
```bash
# Check service status
docker compose ps actions-api

# Check logs
docker compose logs -f actions-api

# Test health endpoint
curl http://localhost:8083/healthz
```

**Solutions**:
1. **Restart service**:
   ```bash
   docker compose restart actions-api
   ```

2. **Check database connectivity**:
   ```bash
   # Test database connection
   docker compose exec actions-api ping database-host
   ```

3. **Clear in-memory storage**:
   ```bash
   # Restart to clear agent registry
   docker compose restart actions-api
   ```

#### WebSocket Gateway Issues

**Symptoms**:
- WebSocket connections failing
- Message routing errors
- Connection timeouts

**Diagnosis**:
```bash
# Check WebSocket gateway logs
docker compose logs -f websocket-gateway

# Test WebSocket endpoint
curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: test" \
     http://localhost:8080/ws/agent
```

**Solutions**:
1. **Check message handlers**:
   ```go
   // Ensure handlers are registered
   func (wsg *WebSocketGateway) registerDefaultHandlers() {
       wsg.messageRouter.RegisterHandler("auth", wsg.handleAuthentication)
       wsg.messageRouter.RegisterHandler("heartbeat", wsg.handleHeartbeat)
   }
   ```

2. **Verify authentication flow**:
   ```go
   // Check authentication message format
   if message.Type == types.MessageTypeRequest && message.Channel == "auth" {
       return wsg.handleAuthentication(conn, message)
   }
   ```

### Policy Deployment Issues

#### Policies Not Applying

**Symptoms**:
- Policies created but not enforced
- Agent reports policy loading errors
- eBPF programs not loading

**Diagnosis**:
```bash
# Check agent logs for eBPF errors
journalctl -u aegis-agent | grep -i ebpf

# Check policy status
curl http://localhost:8083/agents/{agent-id}/status

# Verify policy deployment
curl http://localhost:8083/agents/{agent-id}/policies
```

**Solutions**:
1. **Check eBPF compatibility**:
   ```bash
   # Verify kernel version
   uname -r
   
   # Check eBPF features
   cat /proc/sys/net/core/bpf_jit_enable
   ```

2. **Validate policy syntax**:
   ```json
   {
     "name": "valid-policy",
     "rules": [
       {
         "action": "block",
         "protocol": "icmp",
         "destination": "8.8.8.8"
       }
     ]
   }
   ```

3. **Check agent capabilities**:
   ```bash
   # Verify agent has required capabilities
   curl http://localhost:8083/agents/{agent-id} | jq '.capabilities'
   ```

### Authentication and Security Issues

#### Ed25519 Signature Verification Failures

**Symptoms**:
- "signature verify failed" errors
- Authentication rejected
- 401 Unauthorized responses

**Root Causes**:
1. **Wrong signature data**: Signing incorrect data
2. **Key mismatch**: Using wrong key pair
3. **Timestamp issues**: Clock synchronization problems

**Solutions**:
1. **Correct signature data**:
   ```go
   // For authentication
   data := fmt.Sprintf("%s:%s:%d:%s", agentID, publicKey, timestamp, nonce)
   
   // For registration completion
   data := nonce + serverTime + hostID
   ```

2. **Verify key generation**:
   ```go
   publicKey, privateKey, err := ed25519.GenerateKey(nil)
   if err != nil {
       return fmt.Errorf("key generation failed: %w", err)
   }
   ```

3. **Check timestamp synchronization**:
   ```go
   // Ensure timestamps are within acceptable range
   if time.Now().Unix() - timestamp > 300 { // 5 minutes
       return fmt.Errorf("timestamp too old")
   }
   ```

### Performance Issues

#### High CPU Usage

**Symptoms**:
- Backend services using excessive CPU
- Slow response times
- System becoming unresponsive

**Diagnosis**:
```bash
# Check resource usage
docker stats

# Check process CPU usage
top -p $(docker compose ps -q actions-api)
```

**Solutions**:
1. **Optimize message processing**:
   ```go
   // Use worker pools for message processing
   workers := make(chan struct{}, 10)
   for msg := range messageChannel {
       workers <- struct{}{}
       go func(m Message) {
           defer func() { <-workers }()
           processMessage(m)
       }(msg)
   }
   ```

2. **Implement connection limits**:
   ```go
   // Limit concurrent connections
   if len(connections) >= maxConnections {
       return fmt.Errorf("connection limit exceeded")
   }
   ```

#### Memory Leaks

**Symptoms**:
- Continuously growing memory usage
- Out of memory errors
- Service crashes

**Diagnosis**:
```bash
# Monitor memory usage
docker stats --no-stream

# Check for goroutine leaks
curl http://localhost:8083/debug/pprof/goroutine
```

**Solutions**:
1. **Proper resource cleanup**:
   ```go
   defer func() {
       conn.Close()
       delete(connections, agentID)
   }()
   ```

2. **Implement connection timeouts**:
   ```go
   conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
   ```

### Network Issues

#### Port Conflicts

**Symptoms**:
- Services failing to start
- "Address already in use" errors
- Connection refused errors

**Diagnosis**:
```bash
# Check port usage
netstat -tlnp | grep :8080
netstat -tlnp | grep :8083

# Check Docker port mappings
docker compose ps
```

**Solutions**:
1. **Kill conflicting processes**:
   ```bash
   # Find process using port
   lsof -i :8080
   
   # Kill process
   kill -9 <PID>
   ```

2. **Change port configuration**:
   ```yaml
   # docker-compose.yml
   services:
     websocket-gateway:
       ports:
         - "8081:8080"  # Use different external port
   ```

#### DNS Resolution Issues

**Symptoms**:
- "no such host" errors
- Service discovery failures
- Inter-service communication issues

**Diagnosis**:
```bash
# Test DNS resolution
nslookup service-name
dig service-name

# Check Docker networking
docker network ls
docker network inspect aegisflux_default
```

**Solutions**:
1. **Use service names in Docker**:
   ```go
   // Use service name instead of localhost
   url := "http://actions-api:8083/agents"
   ```

2. **Check Docker Compose networking**:
   ```yaml
   # Ensure services are on same network
   services:
     actions-api:
       networks:
         - aegisflux
     websocket-gateway:
       networks:
         - aegisflux
   ```

## Debugging Tools

### Log Analysis

```bash
# Follow all logs
docker compose logs -f

# Follow specific service
docker compose logs -f actions-api

# Search for specific errors
docker compose logs | grep -i error

# Show last 100 lines
docker compose logs --tail=100
```

### Network Debugging

```bash
# Test connectivity
curl -v http://localhost:8083/health

# Test WebSocket
wscat -c ws://localhost:8080/ws/agent

# Check port connectivity
telnet localhost 8080
telnet localhost 8083
```

### Performance Monitoring

```bash
# Monitor resource usage
docker stats

# Check system resources
htop
iostat -x 1

# Monitor network traffic
iftop
```

## Getting Help

1. **Check logs first**: Most issues can be diagnosed from logs
2. **Verify configuration**: Ensure all settings are correct
3. **Test connectivity**: Verify network and service connectivity
4. **Check documentation**: Review API docs and guides
5. **Search issues**: Check for similar problems in logs
6. **Contact support**: Provide detailed error information

For more information, see the [User Guide](USER_GUIDE.md) and [Engineer Guide](ENGINEER_GUIDE.md).

