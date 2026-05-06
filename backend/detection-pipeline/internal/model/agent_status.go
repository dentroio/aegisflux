package model

// RolloutState is the agent-reported detection pack lifecycle (WO-DET-003).
type RolloutState string

const (
	RolloutNotChecked  RolloutState = "not_checked"
	RolloutApplied     RolloutState = "applied"
	RolloutRejected    RolloutState = "rejected"
	RolloutStale       RolloutState = "stale"
	RolloutIncompatible RolloutState = "incompatible"
	RolloutExpired     RolloutState = "expired"
	RolloutRollback    RolloutState = "rollback"
)

// AgentPackStatus is controller-side state for one agent_uid (lab).
type AgentPackStatus struct {
	AgentUID string `json:"agent_uid"`

	ActivePackID      string `json:"active_pack_id,omitempty"`
	ActivePackVersion string `json:"active_pack_version,omitempty"`

	LastCheckAtMS    int64 `json:"last_check_at_ms"`
	LastAppliedAtMS  int64 `json:"last_applied_at_ms,omitempty"`
	LastRejectedAtMS int64 `json:"last_rejected_at_ms,omitempty"`

	LastRejectedPackID   string `json:"last_rejected_pack_id,omitempty"`
	LastRejectedReason   string `json:"last_rejected_reason,omitempty"`
	LastRejectedReasonCodes []string `json:"last_rejected_reason_codes,omitempty"`

	SignatureStatus      string `json:"signature_status,omitempty"`
	HashStatus           string `json:"hash_status,omitempty"`
	SchemaStatus         string `json:"schema_status,omitempty"`
	CompatibilityStatus  string `json:"compatibility_status,omitempty"`

	PreviousPackID      string `json:"previous_pack_id,omitempty"`
	PreviousPackVersion string `json:"previous_pack_version,omitempty"`

	RolloutState   RolloutState `json:"rollout_state"`
	ReasonDetail   string       `json:"reason_detail,omitempty"`
	ReasonCodes    []string     `json:"reason_codes,omitempty"`
	DeviceID       string       `json:"device_id,omitempty"`
	ReportedAgentVersion string `json:"reported_agent_version,omitempty"`
	UpdatedAtMS    int64        `json:"updated_at_ms"`
}
