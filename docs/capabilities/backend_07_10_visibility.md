# Backend Cap7.10 - Host Visibility Implementation

## Overview

This document describes the implementation of complete host visibility capabilities in the AegisFlux Backend Safety Shim. Cap7.10 provides comprehensive visibility into agent host behavior including process trees, execution events, network sockets, and network flows.

## Key Features

### 1. Complete Host Visibility
- **Process Tree**: Full process hierarchy with parent-child relationships
- **Execution Events**: Process execution tracking with timing and exit codes
- **Network Sockets**: Active socket enumeration with connection states
- **Network Flows**: Traffic flow monitoring with packet and byte counts

### 2. Real-Time Data Collection
- NATS-based event streaming from agents
- Structured JSON payloads for efficient processing
- High-frequency data collection with configurable intervals
- Reliable message delivery with acknowledgment

### 3. Historical Data Storage
- Time-series database storage for visibility frames
- Efficient indexing for fast queries
- Data retention policies for storage management
- Aggregated views for performance optimization

## Implementation Details

### Database Schema

#### visibility_frames Table
```sql
CREATE TABLE visibility_frames (
    id BIGSERIAL PRIMARY KEY,
    agent_uid TEXT NOT NULL,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    procs JSONB,                    -- Process tree data
    flows JSONB,                    -- Network flows data
    sockets JSONB,                  -- Socket information
    exec_events JSONB,              -- Execution events
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### Specialized Tables
- **network_flows**: Detailed network flow tracking
- **processes**: Process information with resource usage
- **sockets**: Socket enumeration with connection details
- **exec_events**: Process execution event tracking

### NATS Event Processing

#### Event Types
- **Visibility Frames**: Complete host state snapshots
- **Process Updates**: Individual process changes
- **Network Flow Updates**: Flow state changes
- **Execution Events**: Process execution notifications

#### Event Subjects
- `aegis.visibility.host.{agent_uid}`: Host-specific visibility data
- `aegis.visibility.processes.{agent_uid}`: Process-specific data
- `aegis.visibility.flows.{agent_uid}`: Network flow data
- `aegis.visibility.exec.{agent_uid}`: Execution event data

### API Endpoints

#### GET /agents/{id}/visibility/latest
Returns the latest visibility frame for an agent:

```json
{
  "agent_uid": "agent-123",
  "timestamp": "2024-01-15T10:30:00Z",
  "processes": {
    "total": 156,
    "running": 142,
    "terminated": 14,
    "tree": [...]
  },
  "flows": {
    "total": 89,
    "established": 45,
    "closed": 44,
    "flows": [...]
  },
  "sockets": {
    "total": 67,
    "listening": 12,
    "established": 45,
    "sockets": [...]
  },
  "exec_events": {
    "total": 23,
    "recent": [...]
  }
}
```

#### GET /agents/{id}/visibility/history
Returns historical visibility frames with pagination:

```bash
curl "http://localhost:8090/agents/agent-123/visibility/history?limit=50&offset=0"
```

#### GET /agents/{id}/visibility/summary
Returns aggregated visibility statistics:

```json
{
  "agent_uid": "agent-123",
  "latest_frame_ts": "2024-01-15T10:30:00Z",
  "total_processes": 156,
  "total_flows": 89,
  "total_sockets": 67,
  "total_exec_events": 23,
  "active_processes": 142
}
```

#### GET /agents/{id}/flows
Returns network flow details with filtering:

```bash
curl "http://localhost:8090/agents/agent-123/flows?protocol=tcp&status=established&limit=100"
```

#### GET /agents/{id}/processes
Returns process information with filtering:

```bash
curl "http://localhost:8090/agents/agent-123/processes?status=running&name=nginx&limit=50"
```

#### GET /agents/{id}/enforcement/decisions
Returns enforcement decision history:

```bash
curl "http://localhost:8090/agents/agent-123/enforcement/decisions?verdict=deny&mode=block&limit=100"
```

### Data Structures

#### Process Information
```json
{
  "pid": 1234,
  "ppid": 567,
  "name": "nginx",
  "cmdline": "nginx -g daemon off;",
  "executable_path": "/usr/sbin/nginx",
  "working_directory": "/var/www",
  "user_id": 33,
  "group_id": 33,
  "start_time": "2024-01-15T09:00:00Z",
  "end_time": null,
  "status": "running",
  "memory_usage": 1048576,
  "cpu_usage": 0.05,
  "network_connections": [...]
}
```

#### Network Flow Information
```json
{
  "src_ip": "10.0.0.5",
  "dst_ip": "10.0.0.10",
  "src_port": 45678,
  "dst_port": 80,
  "protocol": "tcp",
  "bytes_sent": 1024,
  "bytes_received": 2048,
  "packets_sent": 10,
  "packets_received": 15,
  "start_time": "2024-01-15T10:25:00Z",
  "end_time": null,
  "status": "established",
  "process_id": 1234,
  "process_name": "nginx"
}
```

#### Socket Information
```json
{
  "fd": 5,
  "pid": 1234,
  "family": 2,
  "type": 1,
  "protocol": 6,
  "local_address": "0.0.0.0",
  "local_port": 80,
  "remote_address": "10.0.0.5",
  "remote_port": 45678,
  "state": "established"
}
```

#### Execution Event Information
```json
{
  "pid": 5678,
  "ppid": 1234,
  "executable_path": "/usr/bin/curl",
  "cmdline": "curl -s http://example.com",
  "working_directory": "/tmp",
  "user_id": 1000,
  "group_id": 1000,
  "exit_code": 0,
  "timestamp": "2024-01-15T10:28:00Z",
  "duration_ms": 150
}
```

### Data Collection Strategy

#### Collection Intervals
- **Visibility Frames**: Every 30 seconds (configurable)
- **Process Updates**: Real-time on process state changes
- **Network Flows**: Every 5 seconds for active flows
- **Execution Events**: Real-time on process execution

#### Data Compression
- JSON compression for large payloads
- Delta updates for incremental changes
- Efficient serialization for network transmission
- Batch processing for multiple events

### Performance Considerations

#### Database Optimization
- Partitioned tables by time for large datasets
- Efficient indexes for common query patterns
- GIN indexes for JSONB columns
- Automated cleanup of old data

#### Query Optimization
- Pagination for large result sets
- Filtering at database level
- Caching for frequently accessed data
- Aggregated views for summary statistics

#### Storage Management
- Configurable retention policies
- Automated data cleanup
- Compression for historical data
- Archival strategies for long-term storage

### Security and Privacy

#### Data Access Control
- Agent-specific data isolation
- Role-based access to visibility data
- Audit logging for data access
- Encryption in transit and at rest

#### Data Anonymization
- Configurable data masking
- Sensitive data filtering
- Privacy-preserving analytics
- Compliance with data protection regulations

### Monitoring and Alerting

#### Metrics Collection
- Visibility data collection rate
- Database storage utilization
- Query performance metrics
- Agent connectivity status

#### Alert Conditions
- Agent data collection failures
- Database storage threshold breaches
- Unusual process or network activity
- Performance degradation

### Integration with Enforcement

#### Policy Correlation
- Link visibility data to enforcement decisions
- Track policy effectiveness
- Identify policy violations
- Generate compliance reports

#### Real-Time Analysis
- Process behavior analysis
- Network traffic pattern detection
- Anomaly detection algorithms
- Threat intelligence integration

## Usage Examples

### Basic Visibility Query
```bash
# Get latest visibility for an agent
curl http://localhost:8090/agents/agent-123/visibility/latest

# Get visibility history
curl "http://localhost:8090/agents/agent-123/visibility/history?limit=10"

# Get visibility summary
curl http://localhost:8090/agents/agent-123/visibility/summary
```

### Network Flow Analysis
```bash
# Get all TCP flows
curl "http://localhost:8090/agents/agent-123/flows?protocol=tcp"

# Get established connections
curl "http://localhost:8090/agents/agent-123/flows?status=established"

# Get flows to specific IP
curl "http://localhost:8090/agents/agent-123/flows?dst_ip=10.0.0.10"
```

### Process Analysis
```bash
# Get all running processes
curl "http://localhost:8090/agents/agent-123/processes?status=running"

# Search for specific process
curl "http://localhost:8090/agents/agent-123/processes?name=nginx"

# Get processes by user
curl "http://localhost:8090/agents/agent-123/processes?user_id=33"
```

### Enforcement Decision Analysis
```bash
# Get all enforcement decisions
curl "http://localhost:8090/agents/agent-123/enforcement/decisions"

# Get denied connections
curl "http://localhost:8090/agents/agent-123/enforcement/decisions?verdict=deny"

# Get observe mode decisions
curl "http://localhost:8090/agents/agent-123/enforcement/decisions?mode=observe"
```

## Testing

### Unit Tests
- Visibility data parsing and validation
- Database operations for visibility tables
- NATS message processing
- API endpoint functionality

### Integration Tests
- End-to-end visibility data flow
- Database query performance
- NATS event processing
- API response validation

### Performance Tests
- High-volume data ingestion
- Database query performance
- API response times
- Memory usage optimization

## Future Enhancements

### Advanced Analytics
- Machine learning for anomaly detection
- Behavioral analysis algorithms
- Predictive analytics for resource usage
- Custom dashboard creation

### Integration Improvements
- SIEM system integration
- Log aggregation systems
- Metrics collection systems
- Alert management platforms

### Performance Optimizations
- Real-time streaming analytics
- Distributed data processing
- Advanced caching strategies
- Query optimization techniques

## Conclusion

Cap7.10 visibility implementation provides comprehensive host visibility capabilities with real-time data collection, efficient storage, and powerful querying capabilities. The system enables detailed analysis of host behavior, policy effectiveness, and security posture while maintaining high performance and scalability.

