package agentsharness

import "encoding/json"

// Job lifecycle (WO-AGENTS-001).
const (
	JobQueued          = "queued"
	JobRunning         = "running"
	JobNeedsHumanInput = "needs_human_input"
	JobCompleted       = "completed"
	JobFailed          = "failed"
	JobCancelled       = "cancelled"
	JobSuperseded      = "superseded"
)

// Read-only tool identifiers registered for the harness (no enforcement / mutation).
const (
	ToolDeviceEvidenceSummary    = "read.device_evidence_summary"
	ToolFindingsEvidencePaths    = "read.findings_evidence_paths"
	ToolDetectionCandidateLookup = "read.detection_candidate_lookup"
)

// SystemAgentMeta is registered agent metadata (no endpoint-side execution).
type SystemAgentMeta struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	AllowedTools []string `json:"allowed_tools"`
}

// ToolMeta is typed tool registry metadata.
type ToolMeta struct {
	ID           string          `json:"id"`
	DisplayName  string          `json:"display_name"`
	Description  string          `json:"description"`
	Mutates      bool            `json:"mutates"`
	InputSchema  json.RawMessage `json:"input_schema"`
	OutputSchema json.RawMessage `json:"output_schema"`
}

// JobRecord mirrors ai_agent_jobs.
type JobRecord struct {
	JobID             string `json:"job_id"`
	AgentID           string `json:"agent_id"`
	DeviceID          string `json:"device_id"`
	FindingID         string `json:"finding_id,omitempty"`
	Status            string `json:"status"`
	Error             string `json:"error,omitempty"`
	SupersededByJobID string `json:"superseded_by_job_id,omitempty"`
	CreatedMS         int64  `json:"created_at_ms"`
	UpdatedMS         int64  `json:"updated_at_ms"`
}

// ToolCallRecord mirrors ai_agent_tool_calls.
type ToolCallRecord struct {
	CallID     string          `json:"call_id"`
	ToolID     string          `json:"tool_id"`
	InputJSON  json.RawMessage `json:"input_json"`
	OutputJSON json.RawMessage `json:"output_json,omitempty"`
	Error      string          `json:"error,omitempty"`
	StartedMS  int64           `json:"started_at_ms"`
	EndedMS    int64           `json:"ended_at_ms"`
	DurationMS int64           `json:"duration_ms"`
}

// RunRecord mirrors ai_agent_runs plus nested tool audit.
type RunRecord struct {
	RunID                 string           `json:"run_id"`
	JobID                 string           `json:"job_id"`
	AgentID               string           `json:"agent_id"`
	DeviceID              string           `json:"device_id"`
	FindingID             string           `json:"finding_id,omitempty"`
	ProviderKind          string           `json:"provider_kind"`
	Model                 string           `json:"model"`
	Status                string           `json:"status"`
	PromptRedactedPreview string           `json:"prompt_redacted_preview,omitempty"`
	Error                 string           `json:"error,omitempty"`
	StartedMS             int64            `json:"started_at_ms"`
	EndedMS               int64            `json:"ended_at_ms"`
	DurationMS            int64            `json:"duration_ms"`
	PrivacyApplied        bool             `json:"privacy_applied"`
	Assessment            string           `json:"assessment,omitempty"`
	EvidenceSummary       string           `json:"evidence_summary,omitempty"`
	Confidence            string           `json:"confidence,omitempty"`
	RecommendedNextAction string           `json:"recommended_next_action,omitempty"`
	ToolCalls             []ToolCallRecord `json:"tool_calls"`
	// WO-AGENTS-002: structured conclusion; required for successful product-impacting completions.
	EvidenceBoundConclusion       *EvidenceBoundConclusion `json:"evidence_bound_conclusion,omitempty"`
	EvidenceBoundValidationErrors []string                 `json:"evidence_bound_validation_errors,omitempty"`
}

// RunSpec is a single synchronous lab invocation through the harness.
type RunSpec struct {
	AgentID     string
	DeviceID    string
	FindingID   string
	CandidateID string
	// Context is arbitrary operator JSON merged into the redacted prompt preview (never raw secrets).
	Context map[string]any
	// ProductImpacting when true (default) enforces WO-AGENTS-002 validation before completed status.
	ProductImpacting *bool
}
