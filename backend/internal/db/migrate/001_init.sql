-- AegisFlux Backend Safety Shim Database Schema
-- Migration 001: Initial schema for agents, bundles, assignments, and audit logging

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Agents table - stores registered agent information
CREATE TABLE agents (
    agent_uid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_id VARCHAR(255) UNIQUE NOT NULL,
    platform JSONB NOT NULL,
    labels JSONB DEFAULT '[]'::jsonb,
    notes TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index on host_id for fast lookups
CREATE INDEX idx_agents_host_id ON agents(host_id);
CREATE INDEX idx_agents_created_at ON agents(created_at);

-- Bundles table - stores eBPF program bundles with signatures
CREATE TABLE bundles (
    bundle_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hash VARCHAR(64) NOT NULL, -- SHA-256 hash of bundle content
    sig TEXT NOT NULL, -- Ed25519 signature
    algo VARCHAR(20) DEFAULT 'Ed25519', -- Signature algorithm
    kid VARCHAR(64) NOT NULL, -- Key ID for signature verification
    meta JSONB DEFAULT '{}'::jsonb, -- Bundle metadata (version, description, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system'
);

-- Create unique index on hash to prevent duplicate bundles
CREATE UNIQUE INDEX idx_bundles_hash ON bundles(hash);
CREATE INDEX idx_bundles_kid ON bundles(kid);
CREATE INDEX idx_bundles_created_at ON bundles(created_at);

-- Assignments table - defines which bundles are assigned to which agents
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    host_selector JSONB NOT NULL, -- Selection criteria (host_id, labels, etc.)
    ttl_ts TIMESTAMP WITH TIME ZONE, -- Time-to-live for assignment
    dry_run BOOLEAN DEFAULT FALSE, -- If true, don't actually deploy
    bundle_id UUID NOT NULL REFERENCES bundles(bundle_id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'expired', 'cancelled'))
);

-- Create index on bundle_id for efficient lookups
CREATE INDEX idx_assignments_bundle_id ON assignments(bundle_id);
CREATE INDEX idx_assignments_ttl_ts ON assignments(ttl_ts);
CREATE INDEX idx_assignments_status ON assignments(status);
CREATE INDEX idx_assignments_created_at ON assignments(created_at);

-- Audit log table - tracks all administrative actions
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor VARCHAR(255) NOT NULL, -- Who performed the action
    action VARCHAR(100) NOT NULL, -- What action was performed
    target VARCHAR(255), -- Target of the action (bundle_id, agent_uid, etc.)
    details JSONB DEFAULT '{}'::jsonb, -- Additional details about the action
    at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create index on actor and timestamp for audit queries
CREATE INDEX idx_audit_log_actor ON audit_log(actor);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_at ON audit_log(at);

-- Signing keys table - stores active and rotated signing keys
CREATE TABLE signing_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kid VARCHAR(64) UNIQUE NOT NULL, -- Key identifier
    public_key TEXT NOT NULL, -- Base64 encoded public key
    private_key_encrypted TEXT, -- Encrypted private key (if stored)
    algorithm VARCHAR(20) DEFAULT 'Ed25519',
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'rotated', 'revoked')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    rotated_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create index on kid and status for key lookups
CREATE INDEX idx_signing_keys_kid ON signing_keys(kid);
CREATE INDEX idx_signing_keys_status ON signing_keys(status);

-- Bundle assignments view - shows active assignments with bundle details
CREATE VIEW active_bundle_assignments AS
SELECT 
    a.id as assignment_id,
    a.host_selector,
    a.ttl_ts,
    a.dry_run,
    a.created_at,
    a.created_by,
    b.bundle_id,
    b.name as bundle_name,
    b.hash as bundle_hash,
    b.sig as bundle_sig,
    b.kid as bundle_kid,
    b.meta as bundle_meta
FROM assignments a
JOIN bundles b ON a.bundle_id = b.bundle_id
WHERE a.status = 'active'
  AND (a.ttl_ts IS NULL OR a.ttl_ts > NOW());

-- Agent bundle assignments view - shows which bundles are assigned to specific agents
CREATE VIEW agent_bundle_assignments AS
SELECT 
    ag.agent_uid,
    ag.host_id,
    ag.platform,
    ag.labels as agent_labels,
    aba.assignment_id,
    aba.bundle_id,
    aba.bundle_name,
    aba.bundle_hash,
    aba.bundle_sig,
    aba.bundle_kid,
    aba.bundle_meta,
    aba.ttl_ts,
    aba.dry_run
FROM agents ag
CROSS JOIN active_bundle_assignments aba
WHERE (
    -- Match by host_id if specified in selector
    (aba.host_selector->>'host_id' IS NULL OR aba.host_selector->>'host_id' = ag.host_id)
    -- Match by labels if specified in selector
    AND (
        aba.host_selector->>'labels' IS NULL 
        OR ag.labels ?| (SELECT array_agg(value::text) FROM jsonb_array_elements_text(aba.host_selector->'labels'))
    )
    -- Match by platform if specified in selector
    AND (
        aba.host_selector->>'platform' IS NULL 
        OR ag.platform @> aba.host_selector->'platform'
    )
);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger to automatically update updated_at on agents table
CREATE TRIGGER update_agents_updated_at 
    BEFORE UPDATE ON agents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to log audit events
CREATE OR REPLACE FUNCTION log_audit_event(
    p_actor VARCHAR(255),
    p_action VARCHAR(100),
    p_target VARCHAR(255) DEFAULT NULL,
    p_details JSONB DEFAULT '{}'::jsonb
)
RETURNS UUID AS $$
DECLARE
    audit_id UUID;
BEGIN
    INSERT INTO audit_log (actor, action, target, details)
    VALUES (p_actor, p_action, p_target, p_details)
    RETURNING id INTO audit_id;
    
    RETURN audit_id;
END;
$$ LANGUAGE plpgsql;

-- Insert default signing key (this should be replaced with actual key generation)
INSERT INTO signing_keys (kid, public_key, algorithm, status) 
VALUES (
    'default-key-1',
    'dummy-public-key-replace-with-actual-key',
    'Ed25519',
    'active'
) ON CONFLICT (kid) DO NOTHING;

