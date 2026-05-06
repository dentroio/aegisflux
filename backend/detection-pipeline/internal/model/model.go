package model

import "encoding/json"

// ResearchItem is the upstream research context for candidate rules.
type ResearchItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Summary     string `json:"summary,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	CreatedAtMS int64  `json:"created_at_ms"`
}

// CandidateStatus is the WO-DET-002 lifecycle state.
type CandidateStatus string

const (
	StatusDraft             CandidateStatus = "draft"
	StatusValidating        CandidateStatus = "validating"
	StatusValidationFailed  CandidateStatus = "validation_failed"
	StatusReadyForReview    CandidateStatus = "ready_for_review"
	StatusApproved          CandidateStatus = "approved"
	StatusRejected          CandidateStatus = "rejected"
	StatusSigned            CandidateStatus = "signed"
)

// Candidate holds a proposed detection-pack rule set pending validation and approval.
type Candidate struct {
	ID             string          `json:"id"`
	ResearchItemID string          `json:"research_item_id"`
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	Status         CandidateStatus `json:"status"`
	CreatedAtMS    int64           `json:"created_at_ms"`
	UpdatedAtMS    int64           `json:"updated_at_ms"`
	RejectReason   string          `json:"reject_reason,omitempty"`

	PackID            string          `json:"pack_id"`
	PackVersion       string          `json:"pack_version"`
	MinAgentVersion   string          `json:"min_agent_version"`
	SupportedOS       []string        `json:"supported_os"`
	Author            string          `json:"author,omitempty"`
	Source            string          `json:"source,omitempty"`
	EvaluatorLimits   json.RawMessage `json:"evaluator_limits"`
	ProposedRules     json.RawMessage `json:"proposed_rules"`
	References        json.RawMessage `json:"references,omitempty"`

	LastValidationID string `json:"last_validation_id,omitempty"`
	SignedPackID     string `json:"signed_pack_id,omitempty"`
}

// ValidationRun records one validation pass against lab telemetry.
type ValidationRun struct {
	ID          string `json:"id"`
	CandidateID string `json:"candidate_id"`
	StartedAtMS int64  `json:"started_at_ms"`
	EndedAtMS   int64  `json:"ended_at_ms"`
	Success     bool   `json:"success"`

	IngestURL     string `json:"ingest_url,omitempty"`
	DeviceID      string `json:"device_id,omitempty"`
	EventsFetched int    `json:"events_fetched"`
	MatchedRules  int    `json:"matched_rules"`
	Details       string `json:"details,omitempty"`
	Errors        string `json:"errors,omitempty"`
}

// SignedPackArtifact is an approved, signed detection_pack.v1 document (not published to endpoints).
type SignedPackArtifact struct {
	ID           string          `json:"id"`
	CandidateID  string          `json:"candidate_id"`
	CreatedAtMS  int64           `json:"created_at_ms"`
	PackJSON     json.RawMessage `json:"pack_json"`
	SignatureAlg string          `json:"signature_algorithm"`
	KeyID        string          `json:"key_id"`
	PackID       string          `json:"pack_id,omitempty"`
	PackVersion  string          `json:"pack_version,omitempty"`
	SHA256Hex    string          `json:"sha256,omitempty"`
}
