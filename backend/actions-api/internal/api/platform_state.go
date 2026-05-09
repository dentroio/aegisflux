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

	Runs     []AIRunRecord
	Audit    []PrivacyAuditRecord
	Events   []OperationalEvent
	Drafts   []DraftControl
	Research []ResearchItem
}

// ResearchItem captures a piece of AI ecosystem intelligence and its
// lifecycle from "new" through "ready_for_pack". It is the seed for governed
// detection opportunities (WO-PROD-004).
type ResearchItem struct {
	ID                  string                  `json:"id"`
	Title               string                  `json:"title"`
	Source              string                  `json:"source"`
	SourceURL           string                  `json:"source_url,omitempty"`
	Category            string                  `json:"category"`
	Summary             string                  `json:"summary"`
	Indicators          []ResearchIndicator     `json:"indicators"`
	EvidenceRequired    []string                `json:"evidence_required"`
	SuggestedDetection  ResearchSuggestedRule   `json:"suggested_detection"`
	ProposedPackID      string                  `json:"proposed_pack_id,omitempty"`
	Status              string                  `json:"status"`
	RiskScore           int                     `json:"risk_score"`
	OperatorNotes       string                  `json:"operator_notes,omitempty"`
	PublishedMS         int64                   `json:"published_at_ms,omitempty"`
	IngestedMS          int64                   `json:"ingested_at_ms"`
	UpdatedMS           int64                   `json:"updated_at_ms,omitempty"`
}

type ResearchIndicator struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Note  string `json:"note,omitempty"`
}

type ResearchSuggestedRule struct {
	Logic         string   `json:"logic"`
	Scope         string   `json:"scope"`
	Confidence    string   `json:"confidence,omitempty"`
	ExpectedNoise string   `json:"expected_noise,omitempty"`
	GuardRails    []string `json:"guard_rails,omitempty"`
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
	ID                   string   `json:"id"`
	Status               string   `json:"status"`
	SourceFindingID      string   `json:"source_finding_id"`
	SourceFindingTitle   string   `json:"source_finding_title,omitempty"`
	SourceDeviceID       string   `json:"source_device_id,omitempty"`
	ProposedAction       string   `json:"proposed_action"`
	ScopeSelectors       []string `json:"scope_selectors"`
	EvidenceRefs         []string `json:"evidence_refs"`
	ExpectedEffect       string   `json:"expected_effect"`
	Confidence           string   `json:"confidence,omitempty"`
	ExpectedMatches      *int     `json:"expected_matches,omitempty"`
	ExpectedBreakageRisk string   `json:"expected_breakage_risk,omitempty"`
	BlastRadius          string   `json:"blast_radius"`
	BlastRadiusNotes     []string `json:"blast_radius_notes,omitempty"`
	RollbackPlan         string   `json:"rollback_plan"`
	RollbackSteps        []string `json:"rollback_steps,omitempty"`
	OperatorNotes        string   `json:"operator_notes,omitempty"`
	CreatedMS            int64    `json:"created_at_ms"`
	UpdatedMS            int64    `json:"updated_at_ms,omitempty"`
	SimulationMatches    int      `json:"simulation_match_count,omitempty"`
	SimulationDeviceID   string   `json:"simulation_device_id,omitempty"`
	SimulationAtMS       int64    `json:"simulation_at_ms,omitempty"`
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
		Research:        seedResearchItems(),
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

func seedResearchItems() []ResearchItem {
	now := time.Now().UnixMilli()
	day := int64(24 * 60 * 60 * 1000)
	return []ResearchItem{
		{
			ID:        uuid.NewString(),
			Title:     "Local Ollama runtime exposes /api/generate without auth by default",
			Source:    "AegisFlux research",
			SourceURL: "https://research.aegisflux.local/notes/ollama-default-bind",
			Category:  "local_model_runtime",
			Summary:   "When operators install Ollama and bind it to 0.0.0.0:11434, peer endpoints can prompt the model without auth. Lateral abuse risk on shared dev networks.",
			Indicators: []ResearchIndicator{
				{Type: "process_name", Value: "ollama", Note: "serve subcommand"},
				{Type: "listen_port", Value: "11434", Note: "default Ollama API"},
				{Type: "command_line_substring", Value: "ollama serve --host 0.0.0.0"},
			},
			EvidenceRequired: []string{
				"process telemetry with command line",
				"listening port telemetry",
				"firewall posture for the device",
			},
			SuggestedDetection: ResearchSuggestedRule{
				Logic:         "process.name == 'ollama' AND command_line CONTAINS '0.0.0.0' AND listen_port == 11434",
				Scope:         "device:* (excluding lab gateway)",
				Confidence:    "medium",
				ExpectedNoise: "low — should only match endpoints actively serving Ollama on the LAN",
				GuardRails: []string{
					"Observe-only for the first 7 days.",
					"Pair with finding for outbound MCP traffic before promoting.",
					"Surface in Agent Bill of Materials for context.",
				},
			},
			Status:      "new",
			RiskScore:   62,
			PublishedMS: now - 2*day,
			IngestedMS:  now - 2*day,
		},
		{
			ID:        uuid.NewString(),
			Title:     "Anthropic Claude Code agent supports MCP servers over stdio and HTTP",
			Source:    "Anthropic docs",
			SourceURL: "https://docs.anthropic.com/claude/docs/mcp",
			Category:  "coding_agent",
			Summary:   "Claude Code spawns and communicates with arbitrary MCP servers. Without governance, untrusted MCPs may exfiltrate code, secrets, and screen content.",
			Indicators: []ResearchIndicator{
				{Type: "process_name", Value: "claude"},
				{Type: "command_line_substring", Value: "claude code"},
				{Type: "command_line_substring", Value: "mcp.json"},
				{Type: "child_process_name", Value: "uvx"},
			},
			EvidenceRequired: []string{
				"process tree (parent + child) with command line",
				"file access patterns near user repos",
				"dns or http to non-vetted MCP hosts",
			},
			SuggestedDetection: ResearchSuggestedRule{
				Logic:         "process.name == 'claude' AND child_process.name IN ('uvx', 'npx', 'python') AND child_process.command_line CONTAINS 'mcp'",
				Scope:         "device:* WITH user.role == 'engineer'",
				Confidence:    "medium",
				ExpectedNoise: "moderate — paired with MCP allow-list will reduce false positives",
				GuardRails: []string{
					"Observe-only until allow-list of MCP hosts is curated.",
					"Block promotion if browser AI extensions are also active without an inventory entry.",
				},
			},
			Status:      "scoped",
			RiskScore:   68,
			PublishedMS: now - 5*day,
			IngestedMS:  now - 4*day,
		},
		{
			ID:        uuid.NewString(),
			Title:     "Browser AI extensions read DOM and clipboard; new sideloaded extensions appearing",
			Source:    "AegisFlux research",
			Category:  "browser_ai_extension",
			Summary:   "Several AI sidebars (e.g. unofficial GPT companions) request <all_urls> and clipboardRead. Sideloaded extensions can bypass managed-store policy.",
			Indicators: []ResearchIndicator{
				{Type: "extension_permission", Value: "<all_urls>"},
				{Type: "extension_permission", Value: "clipboardRead"},
				{Type: "extension_install_source", Value: "sideloaded"},
			},
			EvidenceRequired: []string{
				"browser extension inventory with permissions and install_source",
			},
			SuggestedDetection: ResearchSuggestedRule{
				Logic:         "extension.install_source == 'sideloaded' AND extension.permissions CONTAINS 'clipboardRead'",
				Scope:         "device:* WITH browser IN ('chrome','edge','firefox')",
				Confidence:    "high",
				ExpectedNoise: "very low — sideload is a strong signal",
				GuardRails: []string{
					"Observe-only and surface in Inventory before promoting.",
					"Cross-link to ABOM for fleet exposure.",
				},
			},
			Status:      "ready_for_pack",
			RiskScore:   74,
			PublishedMS: now - 8*day,
			IngestedMS:  now - 7*day,
		},
	}
}

func (p *PlatformData) appendResearch(item ResearchItem) {
	p.Research = append(p.Research, item)
	if len(p.Research) > 200 {
		p.Research = p.Research[len(p.Research)-200:]
	}
}
