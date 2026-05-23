-- WO-AGENTS-001: Governed AI agent harness contracts (relational target schema).
-- Lab Actions API mirrors these shapes in memory until a shared Postgres store lands.

CREATE TABLE IF NOT EXISTS ai_agent_tools (
    tool_id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    description TEXT NOT NULL,
    mutates BOOLEAN NOT NULL DEFAULT FALSE,
    input_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_agent_tools_no_mutate CHECK (mutates = FALSE)
);

CREATE TABLE IF NOT EXISTS ai_agent_jobs (
    job_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    finding_id TEXT,
    status TEXT NOT NULL,
    error TEXT,
    superseded_by_job_id UUID REFERENCES ai_agent_jobs(job_id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_agent_jobs_status CHECK (status IN (
        'queued', 'running', 'needs_human_input', 'completed', 'failed', 'cancelled', 'superseded'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_agent_jobs_agent_created ON ai_agent_jobs(agent_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_agent_jobs_device_created ON ai_agent_jobs(device_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_agent_jobs_status_created ON ai_agent_jobs(status, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_agent_runs (
    run_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES ai_agent_jobs(job_id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    finding_id TEXT,
    provider_kind TEXT NOT NULL,
    model TEXT NOT NULL,
    status TEXT NOT NULL,
    prompt_redacted_preview TEXT,
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_ms BIGINT,
    privacy_applied BOOLEAN NOT NULL DEFAULT FALSE,
    assessment TEXT,
    evidence_summary TEXT,
    confidence TEXT,
    recommended_next_action TEXT,
    CONSTRAINT ai_agent_runs_status CHECK (status IN (
        'queued', 'running', 'needs_human_input', 'completed', 'failed', 'cancelled', 'superseded'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_agent_runs_agent_started ON ai_agent_runs(agent_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_agent_runs_device_started ON ai_agent_runs(device_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_agent_runs_finding_started ON ai_agent_runs(finding_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_agent_runs_status_started ON ai_agent_runs(status, started_at DESC);

CREATE TABLE IF NOT EXISTS ai_agent_tool_calls (
    call_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES ai_agent_runs(run_id) ON DELETE CASCADE,
    tool_id TEXT NOT NULL REFERENCES ai_agent_tools(tool_id) ON DELETE RESTRICT,
    input_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_json JSONB,
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    duration_ms BIGINT
);

CREATE INDEX IF NOT EXISTS idx_ai_agent_tool_calls_run ON ai_agent_tool_calls(run_id, started_at);

-- Seed read-only tool definitions (harness registry; no enforcement / endpoint mutation tools).
INSERT INTO ai_agent_tools (tool_id, display_name, description, mutates, input_schema, output_schema) VALUES
(
    'read.device_evidence_summary',
    'Device evidence summary',
    'Returns bounded integration evidence summary for a device (read-only).',
    FALSE,
    '{"type":"object","properties":{"device_id":{"type":"string"}},"required":["device_id"]}'::jsonb,
    '{"type":"object","description":"Integration evidence summary v1"}'::jsonb
),
(
    'read.findings_evidence_paths',
    'Findings / evidence path lookup',
    'Resolves relative console paths for findings and evidence navigation (read-only).',
    FALSE,
    '{"type":"object","properties":{"device_id":{"type":"string"},"finding_id":{"type":"string"}},"required":["device_id"]}'::jsonb,
    '{"type":"object","properties":{"paths":{"type":"array"}}}'::jsonb
),
(
    'read.detection_candidate_lookup',
    'Detection candidate lookup',
    'Looks up a detection candidate by id or returns recent lab candidates (read-only stub).',
    FALSE,
    '{"type":"object","properties":{"candidate_id":{"type":"string"},"device_id":{"type":"string"}}}'::jsonb,
    '{"type":"object","description":"Candidate summary or empty stub"}'::jsonb
)
ON CONFLICT (tool_id) DO NOTHING;
