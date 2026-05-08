package api

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// PlatformData holds lab-scoped product-platform state (WO-AI / WO-CTRL / WO-PLAT-006 / WO-INT).
type PlatformData struct {
	mu sync.Mutex

	Providers       []AIProviderDTO
	providerSecrets map[string]string
	DefaultProvider string

	Privacy PrivacySettings

	Runs   []AIRunRecord
	Audit  []PrivacyAuditRecord
	Events []OperationalEvent
	Drafts []DraftControl
}

type AIProviderDTO struct {
	ID               string `json:"id"`
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	Enabled          bool   `json:"enabled"`
	SecretConfigured bool   `json:"secret_configured"`
	LastHealthOK     bool   `json:"last_health_ok"`
	LastHealthMS     int64  `json:"last_health_at_ms"`
	LastHealthMsg    string `json:"last_health_message,omitempty"`
}

type PrivacySettings struct {
	AllowExternalAI bool `json:"allow_external_ai"`
	RedactIPs       bool `json:"redact_ips"`
	RedactMACs      bool `json:"redact_macs"`
	RedactUsers     bool `json:"redact_usernames"`
	RedactEmails    bool `json:"redact_emails"`
	RedactHosts     bool `json:"redact_hostnames"`
	RedactCmd       bool `json:"redact_command_lines"`
	RedactPaths     bool `json:"redact_file_paths"`
	RedactSecrets   bool `json:"redact_raw_secrets"`
}

type AIRunRecord struct {
	RunID          string `json:"run_id"`
	AgentID        string `json:"agent_id"`
	DeviceID       string `json:"device_id"`
	CreatedMS      int64  `json:"created_at_ms"`
	ProviderKind   string `json:"provider_kind"`
	Status         string `json:"status"`
	Redacted       bool   `json:"redacted_payload"`
	PrivacyApplied bool   `json:"privacy_applied"`
	Assessment     string `json:"assessment"`
	Evidence       string `json:"evidence_summary"`
	Confidence     string `json:"confidence"`
	NextAction     string `json:"recommended_next_action"`
}

type PrivacyAuditRecord struct {
	ID        string `json:"id"`
	CreatedMS int64  `json:"created_at_ms"`
	Action    string `json:"action"`
	DeviceID  string `json:"device_id,omitempty"`
	Detail    string `json:"detail"`
	Redacted  bool   `json:"redacted"`
}

type OperationalEvent struct {
	ID          string `json:"id"`
	CreatedMS   int64  `json:"created_at_ms"`
	EventType   string `json:"event_type"`
	Status      string `json:"status,omitempty"`
	Subject     string `json:"subject,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	AgentUID    string `json:"agent_uid,omitempty"`
	Description string `json:"description"`
}

type DraftControl struct {
	ID                string   `json:"id"`
	Status            string   `json:"status"`
	SourceFindingID   string   `json:"source_finding_id"`
	ProposedAction    string   `json:"proposed_action"`
	ScopeSelectors    []string `json:"scope_selectors"`
	EvidenceRefs      []string `json:"evidence_refs"`
	ExpectedEffect    string   `json:"expected_effect"`
	BlastRadius       string   `json:"blast_radius"`
	RollbackPlan      string   `json:"rollback_plan"`
	CreatedMS         int64    `json:"created_at_ms"`
	SimulationMatches int      `json:"simulation_match_count,omitempty"`
}

func newPlatformData() *PlatformData {
	return &PlatformData{
		providerSecrets: map[string]string{},
		Providers: []AIProviderDTO{
			{ID: "local", Kind: "local", Name: "Local (deterministic)", Enabled: true, SecretConfigured: false, LastHealthOK: true, LastHealthMS: time.Now().UnixMilli(), LastHealthMsg: "offline-capable"},
			{ID: "openai", Kind: "openai", Name: "OpenAI", Enabled: false, SecretConfigured: false},
			{ID: "anthropic", Kind: "anthropic", Name: "Anthropic", Enabled: false, SecretConfigured: false},
			{ID: "google", Kind: "google", Name: "Google AI", Enabled: false, SecretConfigured: false},
			{ID: "gateway", Kind: "gateway", Name: "Enterprise gateway", Enabled: false, SecretConfigured: false},
		},
		DefaultProvider: "local",
		Privacy: PrivacySettings{
			AllowExternalAI: false,
			RedactIPs:       true,
			RedactMACs:      true,
			RedactUsers:     true,
			RedactEmails:    true,
			RedactHosts:     true,
			RedactCmd:       true,
			RedactPaths:     true,
			RedactSecrets:   true,
		},
	}
}

func (p *PlatformData) touchProviderHealth(id string, ok bool, msg string) {
	for i := range p.Providers {
		if p.Providers[i].ID == id {
			p.Providers[i].LastHealthOK = ok
			p.Providers[i].LastHealthMS = time.Now().UnixMilli()
			p.Providers[i].LastHealthMsg = msg
			return
		}
	}
}

func (p *PlatformData) appendAudit(action, deviceID, detail string, redacted bool) PrivacyAuditRecord {
	rec := PrivacyAuditRecord{
		ID:        uuid.NewString(),
		CreatedMS: time.Now().UnixMilli(),
		Action:    action,
		DeviceID:  deviceID,
		Detail:    detail,
		Redacted:  redacted,
	}
	p.Audit = append(p.Audit, rec)
	if len(p.Audit) > 500 {
		p.Audit = p.Audit[len(p.Audit)-500:]
	}
	return rec
}

func (p *PlatformData) appendRun(r AIRunRecord) {
	p.Runs = append(p.Runs, r)
	if len(p.Runs) > 300 {
		p.Runs = p.Runs[len(p.Runs)-300:]
	}
}

func (p *PlatformData) appendOp(ev OperationalEvent) {
	p.Events = append(p.Events, ev)
	if len(p.Events) > 1000 {
		p.Events = p.Events[len(p.Events)-1000:]
	}
}
