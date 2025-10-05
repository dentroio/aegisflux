-- AegisFlux Backend Safety Shim - Visibility Schema
-- Migration 002: Visibility frames for process tree, exec events, sockets, and network flows

-- Visibility frames table for storing agent telemetry data
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

-- Create indexes for efficient querying
CREATE INDEX idx_visibility_frames_agent_uid_ts ON visibility_frames(agent_uid, ts DESC);
CREATE INDEX idx_visibility_frames_ts ON visibility_frames(ts DESC);
CREATE INDEX idx_visibility_frames_agent_uid ON visibility_frames(agent_uid);

-- Create GIN indexes for JSONB columns for efficient JSON queries
CREATE INDEX idx_visibility_frames_procs_gin ON visibility_frames USING GIN (procs);
CREATE INDEX idx_visibility_frames_flows_gin ON visibility_frames USING GIN (flows);
CREATE INDEX idx_visibility_frames_sockets_gin ON visibility_frames USING GIN (sockets);
CREATE INDEX idx_visibility_frames_exec_events_gin ON visibility_frames USING GIN (exec_events);

-- Assignment snapshots table for storing policy snapshots with signatures
CREATE TABLE assignment_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('observe', 'block')),
    snapshot JSONB NOT NULL,         -- Policy snapshot data
    snapshot_sig TEXT NOT NULL,      -- Signature of snapshot
    snapshot_kid VARCHAR(64) NOT NULL, -- Key ID used for signing
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT NOT NULL
);

-- Create indexes for assignment snapshots
CREATE INDEX idx_assignment_snapshots_assignment_id ON assignment_snapshots(assignment_id);
CREATE INDEX idx_assignment_snapshots_mode ON assignment_snapshots(mode);
CREATE INDEX idx_assignment_snapshots_created_at ON assignment_snapshots(created_at);
CREATE INDEX idx_assignment_snapshots_snapshot_gin ON assignment_snapshots USING GIN (snapshot);

-- Enforcement decisions table for tracking policy enforcement actions
CREATE TABLE enforcement_decisions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    agent_uid TEXT NOT NULL,
    verdict VARCHAR(20) NOT NULL CHECK (verdict IN ('allow', 'deny', 'observe_drop')),
    reason TEXT,
    rule_id TEXT,
    flow_data JSONB,                -- Network flow that triggered the decision
    process_data JSONB,             -- Process information
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('observe', 'block'))
);

-- Create indexes for enforcement decisions
CREATE INDEX idx_enforcement_decisions_assignment_id ON enforcement_decisions(assignment_id);
CREATE INDEX idx_enforcement_decisions_agent_uid ON enforcement_decisions(agent_uid);
CREATE INDEX idx_enforcement_decisions_verdict ON enforcement_decisions(verdict);
CREATE INDEX idx_enforcement_decisions_timestamp ON enforcement_decisions(timestamp DESC);
CREATE INDEX idx_enforcement_decisions_mode ON enforcement_decisions(mode);

-- Network flows table for detailed flow tracking
CREATE TABLE network_flows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_uid TEXT NOT NULL,
    src_ip INET,
    dst_ip INET,
    src_port INTEGER,
    dst_port INTEGER,
    protocol TEXT,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    packets_sent BIGINT DEFAULT 0,
    packets_received BIGINT DEFAULT 0,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    status TEXT,                    -- ESTABLISHED, CLOSED, etc.
    process_id INTEGER,
    process_name TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for network flows
CREATE INDEX idx_network_flows_agent_uid ON network_flows(agent_uid);
CREATE INDEX idx_network_flows_src_ip ON network_flows(src_ip);
CREATE INDEX idx_network_flows_dst_ip ON network_flows(dst_ip);
CREATE INDEX idx_network_flows_protocol ON network_flows(protocol);
CREATE INDEX idx_network_flows_start_time ON network_flows(start_time DESC);
CREATE INDEX idx_network_flows_process_id ON network_flows(process_id);

-- Process information table for detailed process tracking
CREATE TABLE processes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_uid TEXT NOT NULL,
    pid INTEGER NOT NULL,
    ppid INTEGER,
    name TEXT NOT NULL,
    cmdline TEXT,
    executable_path TEXT,
    working_directory TEXT,
    user_id INTEGER,
    group_id INTEGER,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    status TEXT,                    -- RUNNING, TERMINATED, etc.
    memory_usage BIGINT,
    cpu_usage REAL,
    network_connections JSONB,      -- Array of network connections
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for processes
CREATE INDEX idx_processes_agent_uid ON processes(agent_uid);
CREATE INDEX idx_processes_pid ON processes(pid);
CREATE INDEX idx_processes_ppid ON processes(ppid);
CREATE INDEX idx_processes_name ON processes(name);
CREATE INDEX idx_processes_start_time ON processes(start_time DESC);
CREATE INDEX idx_processes_status ON processes(status);

-- Execution events table for tracking process execution
CREATE TABLE exec_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_uid TEXT NOT NULL,
    pid INTEGER NOT NULL,
    ppid INTEGER,
    executable_path TEXT NOT NULL,
    cmdline TEXT,
    working_directory TEXT,
    user_id INTEGER,
    group_id INTEGER,
    exit_code INTEGER,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    duration_ms INTEGER
);

-- Create indexes for exec events
CREATE INDEX idx_exec_events_agent_uid ON exec_events(agent_uid);
CREATE INDEX idx_exec_events_pid ON exec_events(pid);
CREATE INDEX idx_exec_events_timestamp ON exec_events(timestamp DESC);
CREATE INDEX idx_exec_events_executable_path ON exec_events(executable_path);

-- Socket information table for tracking network sockets
CREATE TABLE sockets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_uid TEXT NOT NULL,
    fd INTEGER NOT NULL,
    pid INTEGER NOT NULL,
    family INTEGER,                 -- AF_INET, AF_INET6, etc.
    type INTEGER,                   -- SOCK_STREAM, SOCK_DGRAM, etc.
    protocol INTEGER,
    local_address INET,
    local_port INTEGER,
    remote_address INET,
    remote_port INTEGER,
    state TEXT,                     -- LISTEN, ESTABLISHED, etc.
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for sockets
CREATE INDEX idx_sockets_agent_uid ON sockets(agent_uid);
CREATE INDEX idx_sockets_pid ON sockets(pid);
CREATE INDEX idx_sockets_family ON sockets(family);
CREATE INDEX idx_sockets_type ON sockets(type);
CREATE INDEX idx_sockets_state ON sockets(state);
CREATE INDEX idx_sockets_local_address ON sockets(local_address);
CREATE INDEX idx_sockets_remote_address ON sockets(remote_address);

-- Create views for common queries

-- Active assignments with snapshots view
CREATE VIEW active_assignments_with_snapshots AS
SELECT 
    a.id as assignment_id,
    a.host_selector,
    a.ttl_ts,
    a.dry_run,
    a.bundle_id,
    a.created_by,
    a.status,
    asn.mode,
    asn.snapshot,
    asn.snapshot_sig,
    asn.snapshot_kid,
    b.name as bundle_name,
    b.hash as bundle_hash
FROM assignments a
LEFT JOIN assignment_snapshots asn ON a.id = asn.assignment_id
LEFT JOIN bundles b ON a.bundle_id = b.bundle_id
WHERE a.status = 'active'
  AND (a.ttl_ts IS NULL OR a.ttl_ts > NOW());

-- Latest visibility frame per agent view
CREATE VIEW latest_visibility_frames AS
SELECT DISTINCT ON (agent_uid)
    id,
    agent_uid,
    ts,
    procs,
    flows,
    sockets,
    exec_events,
    created_at
FROM visibility_frames
ORDER BY agent_uid, ts DESC;

-- Enforcement statistics view
CREATE VIEW enforcement_stats AS
SELECT 
    assignment_id,
    agent_uid,
    mode,
    verdict,
    COUNT(*) as count,
    MIN(timestamp) as first_decision,
    MAX(timestamp) as last_decision
FROM enforcement_decisions
GROUP BY assignment_id, agent_uid, mode, verdict;

-- Network flow statistics view
CREATE VIEW network_flow_stats AS
SELECT 
    agent_uid,
    protocol,
    COUNT(*) as flow_count,
    SUM(bytes_sent) as total_bytes_sent,
    SUM(bytes_received) as total_bytes_received,
    SUM(packets_sent) as total_packets_sent,
    SUM(packets_received) as total_packets_received,
    MIN(start_time) as earliest_flow,
    MAX(end_time) as latest_flow
FROM network_flows
GROUP BY agent_uid, protocol;

-- Function to clean up old visibility data (for data retention)
CREATE OR REPLACE FUNCTION cleanup_old_visibility_data(retention_days INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Delete old visibility frames
    DELETE FROM visibility_frames 
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Delete old enforcement decisions
    DELETE FROM enforcement_decisions 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    
    -- Delete old network flows
    DELETE FROM network_flows 
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    
    -- Delete old processes (terminated processes)
    DELETE FROM processes 
    WHERE status = 'TERMINATED' 
    AND end_time < NOW() - INTERVAL '1 day' * retention_days;
    
    -- Delete old exec events
    DELETE FROM exec_events 
    WHERE timestamp < NOW() - INTERVAL '1 day' * retention_days;
    
    -- Delete old sockets
    DELETE FROM sockets 
    WHERE created_at < NOW() - INTERVAL '1 day' * retention_days;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get agent visibility summary
CREATE OR REPLACE FUNCTION get_agent_visibility_summary(p_agent_uid TEXT)
RETURNS TABLE (
    agent_uid TEXT,
    latest_frame_ts TIMESTAMPTZ,
    total_processes BIGINT,
    total_flows BIGINT,
    total_sockets BIGINT,
    total_exec_events BIGINT,
    active_processes BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p_agent_uid,
        vf.ts,
        (SELECT COUNT(*) FROM processes p WHERE p.agent_uid = p_agent_uid),
        (SELECT COUNT(*) FROM network_flows nf WHERE nf.agent_uid = p_agent_uid),
        (SELECT COUNT(*) FROM sockets s WHERE s.agent_uid = p_agent_uid),
        (SELECT COUNT(*) FROM exec_events ee WHERE ee.agent_uid = p_agent_uid),
        (SELECT COUNT(*) FROM processes p WHERE p.agent_uid = p_agent_uid AND p.status = 'RUNNING')
    FROM latest_visibility_frames vf
    WHERE vf.agent_uid = p_agent_uid;
END;
$$ LANGUAGE plpgsql;

-- Insert sample data for testing (optional)
-- This can be removed in production
INSERT INTO assignment_snapshots (assignment_id, mode, snapshot, snapshot_sig, snapshot_kid, created_by)
SELECT 
    a.id,
    'observe',
    '{"allow_cidr_v4": ["10.0.0.0/24"], "deny_cidr_v4": ["8.8.8.8/32"], "edges": [{"src": "frontend", "dst": "auth"}]}'::jsonb,
    'sample-signature',
    'sample-key-id',
    'system'
FROM assignments a
LIMIT 1
ON CONFLICT DO NOTHING;





