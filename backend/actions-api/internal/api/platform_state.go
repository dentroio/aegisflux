package api

import (
	"sync"
	"time"

	"backend/actionsapi/internal/agentsharness"

	"github.com/google/uuid"
)

// PlatformData holds lab-scoped product-platform state (WO-AI / WO-CTRL / WO-PLAT-006 / WO-INT).
type PlatformData struct {
	mu sync.Mutex

	Providers       []AIProviderDTO
	providerSecrets map[string]string
	DefaultProvider string

	Privacy PrivacySettings

	Runs       []AIRunRecord
	Audit      []PrivacyAuditRecord
	Events     []OperationalEvent
	Drafts       []DraftControl
	Research     []ResearchItem
	Candidates   []DetectionCandidate
	AuditBundles []AuditBundle

	// WO-AGENTS-001: governed agent harness job/run audit (lab memory store).
	AgentHarnessJobs []agentsharness.JobRecord
	AgentHarnessRuns []agentsharness.RunRecord
}

// AuditBundle is the foundation for safe enforcement (WO-GROWTH-007).
//
// An audit-mode bundle expresses a control or detection in a form an endpoint
// could one day enforce, but the bundle itself is observe-only by design:
// agents accept it, evaluate it, and report matches as telemetry. The bundle
// never asks the agent to deny, block, or quarantine.
//
// Lifecycle:
//
//	draft -> staged -> [accepted | rejected | incompatible | stale | expired]
//
// "staged" means the bundle has been published for one or more endpoints to
// pick up. The endpoint contract is documented in
// docs/safety/AUDIT_MODE_BUNDLE_CONTRACT.md.
type AuditBundle struct {
	ID                  string                  `json:"id"`
	Version             string                  `json:"version"`
	Mode                string                  `json:"mode"`
	Title               string                  `json:"title"`
	Description         string                  `json:"description,omitempty"`
	Scope               []string                `json:"scope"`
	ExpectedTelemetry   []string                `json:"expected_match_telemetry,omitempty"`
	ApprovalRefs        []string                `json:"approval_refs,omitempty"`
	RollbackNotes       string                  `json:"rollback_notes,omitempty"`
	SourceCandidateID   string                  `json:"source_candidate_id,omitempty"`
	SourceDraftID       string                  `json:"source_draft_id,omitempty"`
	Status              string                  `json:"status"`
	StagedAtMS          int64                   `json:"staged_at_ms,omitempty"`
	ExpiresAtMS         int64                   `json:"expires_at_ms,omitempty"`
	CreatedAtMS         int64                   `json:"created_at_ms"`
	UpdatedAtMS         int64                   `json:"updated_at_ms,omitempty"`
	EndpointStatuses    []AuditBundleStatus     `json:"endpoint_statuses,omitempty"`
	Matches             []AuditBundleMatch      `json:"matches,omitempty"`
	History             []AuditBundleEvent      `json:"history,omitempty"`
}

// AuditBundleStatus captures one endpoint's response to a staged audit bundle.
// Status values: pending, accepted, rejected, incompatible, stale.
type AuditBundleStatus struct {
	DeviceID       string `json:"device_id"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	AgentVersion   string `json:"agent_version,omitempty"`
	ReportedAtMS   int64  `json:"reported_at_ms"`
	LastMatchAtMS  int64  `json:"last_match_at_ms,omitempty"`
}

// AuditBundleMatch is a deterministic, observe-only summary of an audit
// match. Agents are expected to send these as ordinary telemetry.
type AuditBundleMatch struct {
	ID         string `json:"id"`
	DeviceID   string `json:"device_id,omitempty"`
	Process    string `json:"process,omitempty"`
	AtMS       int64  `json:"at_ms"`
	Indicator  string `json:"indicator,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// AuditBundleEvent records a lifecycle change for an audit bundle.
type AuditBundleEvent struct {
	ID     string `json:"id"`
	AtMS   int64  `json:"at_ms"`
	Action string `json:"action"`
	From   string `json:"from_status,omitempty"`
	To     string `json:"to_status,omitempty"`
	Note   string `json:"note,omitempty"`
	Actor  string `json:"actor,omitempty"`
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
	LinkedCandidateID   string                  `json:"linked_candidate_id,omitempty"`
	Status              string                  `json:"status"`
	RiskScore           int                     `json:"risk_score"`
	OperatorNotes       string                  `json:"operator_notes,omitempty"`
	PublishedMS         int64                   `json:"published_at_ms,omitempty"`
	IngestedMS          int64                   `json:"ingested_at_ms"`
	UpdatedMS           int64                   `json:"updated_at_ms,omitempty"`
}

// DetectionCandidate links a research item to a detection-pack pipeline
// (WO-GROWTH-005). It carries quality gates, simulation results, reviewer
// notes, and rollout/retirement status so an operator can follow a
// detection from opportunity → candidate → signed pack → rollout → retire.
type DetectionCandidate struct {
	ID                 string                       `json:"id"`
	SourceResearchID   string                       `json:"source_research_id"`
	SourceResearchURL  string                       `json:"source_research_url,omitempty"`
	Title              string                       `json:"title"`
	Category           string                       `json:"category"`
	Status             string                       `json:"status"`
	PackID             string                       `json:"pack_id,omitempty"`
	PackVersion        string                       `json:"pack_version,omitempty"`
	RolloutStatus      string                       `json:"rollout_status,omitempty"`
	QualityGate        DetectionCandidateGate       `json:"quality_gate"`
	Rule               ResearchSuggestedRule        `json:"rule"`
	OperatorNotes      string                       `json:"operator_notes,omitempty"`
	ReviewerNotes      string                       `json:"reviewer_notes,omitempty"`
	ExpiresAtMS        int64                        `json:"expires_at_ms,omitempty"`
	RollbackPlan       string                       `json:"rollback_plan,omitempty"`
	RetirementReason   string                       `json:"retirement_reason,omitempty"`
	Simulations        []DetectionCandidateSimulation `json:"simulations,omitempty"`
	History            []DetectionCandidateEvent    `json:"history,omitempty"`
	CreatedMS          int64                        `json:"created_at_ms"`
	UpdatedMS          int64                        `json:"updated_at_ms,omitempty"`
}

// DetectionCandidateGate captures the answers the operator must provide
// before promotion to a signed pack. Empty/missing fields appear in the
// "missing" list returned by the gate check.
type DetectionCandidateGate struct {
	RequiredEvidence       []string `json:"required_evidence,omitempty"`
	ExpectedFalsePositives string   `json:"expected_false_positives,omitempty"`
	HasSimulation          bool     `json:"has_simulation"`
	HasReviewerNotes       bool     `json:"has_reviewer_notes"`
	HasExpiration          bool     `json:"has_expiration"`
	HasRollback            bool     `json:"has_rollback"`
	MissingFields          []string `json:"missing_fields,omitempty"`
}

// DetectionCandidateSimulation summarizes a simulated match against
// historical telemetry. Bounded and observe-only.
type DetectionCandidateSimulation struct {
	ID                 string   `json:"id"`
	AtMS               int64    `json:"at_ms"`
	MatchCount         int      `json:"match_count"`
	MatchedDeviceCount int      `json:"matched_device_count"`
	TopIndicators      []string `json:"top_indicators,omitempty"`
	Window             string   `json:"window,omitempty"`
	Confidence         string   `json:"confidence,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

// DetectionCandidateEvent records a lifecycle event for the candidate.
type DetectionCandidateEvent struct {
	ID        string `json:"id"`
	AtMS      int64  `json:"at_ms"`
	Action    string `json:"action"`
	From      string `json:"from_status,omitempty"`
	To        string `json:"to_status,omitempty"`
	Note      string `json:"note,omitempty"`
	Actor     string `json:"actor,omitempty"`
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
	ID                   string                 `json:"id"`
	Status               string                 `json:"status"`
	SourceFindingID      string                 `json:"source_finding_id"`
	SourceFindingTitle   string                 `json:"source_finding_title,omitempty"`
	SourceDeviceID       string                 `json:"source_device_id,omitempty"`
	ProposedAction       string                 `json:"proposed_action"`
	ScopeSelectors       []string               `json:"scope_selectors"`
	EvidenceRefs         []string               `json:"evidence_refs"`
	ExpectedEffect       string                 `json:"expected_effect"`
	Confidence           string                 `json:"confidence,omitempty"`
	ExpectedMatches      *int                   `json:"expected_matches,omitempty"`
	ExpectedBreakageRisk string                 `json:"expected_breakage_risk,omitempty"`
	BlastRadius          string                 `json:"blast_radius"`
	BlastRadiusNotes     []string               `json:"blast_radius_notes,omitempty"`
	RollbackPlan         string                 `json:"rollback_plan"`
	RollbackSteps        []string               `json:"rollback_steps,omitempty"`
	OperatorNotes        string                 `json:"operator_notes,omitempty"`
	CreatedMS            int64                  `json:"created_at_ms"`
	UpdatedMS            int64                  `json:"updated_at_ms,omitempty"`
	SimulationMatches    int                    `json:"simulation_match_count,omitempty"`
	SimulationDeviceID   string                 `json:"simulation_device_id,omitempty"`
	SimulationAtMS       int64                  `json:"simulation_at_ms,omitempty"`
	History              []DraftDecisionEntry   `json:"history,omitempty"`
	Simulations          []DraftSimulationResult `json:"simulations,omitempty"`
}

// DraftDecisionEntry records a meaningful transition on a draft control
// (creation, scope edit, simulation run, status change, archive). Each entry
// captures actor, optional note, and before/after snapshots when fields
// changed.
type DraftDecisionEntry struct {
	ID           string         `json:"id"`
	AtMS         int64          `json:"at_ms"`
	Actor        string         `json:"actor,omitempty"`
	Action       string         `json:"action"`
	Note         string         `json:"note,omitempty"`
	Status       string         `json:"status,omitempty"`
	ChangedKeys  []string       `json:"changed_keys,omitempty"`
	Before       *DraftSnapshot `json:"before,omitempty"`
	After        *DraftSnapshot `json:"after,omitempty"`
	SimulationID string         `json:"simulation_id,omitempty"`
}

// DraftSnapshot captures the subset of a draft that operators edit. It is used
// for before/after diffs in decision history.
type DraftSnapshot struct {
	Status             string   `json:"status,omitempty"`
	ScopeSelectors     []string `json:"scope_selectors,omitempty"`
	BlastRadius        string   `json:"blast_radius,omitempty"`
	BlastRadiusNotes   []string `json:"blast_radius_notes,omitempty"`
	RollbackPlan       string   `json:"rollback_plan,omitempty"`
	RollbackSteps      []string `json:"rollback_steps,omitempty"`
	OperatorNotes      string   `json:"operator_notes,omitempty"`
	ExpectedMatches    *int     `json:"expected_matches,omitempty"`
	ExpectedBreakage   string   `json:"expected_breakage_risk,omitempty"`
}

// DraftSimulationResult captures one run of the observe-only simulation: the
// scope used, deterministic lab projections of matches, top process paths,
// top destinations, and an explanation summary.
type DraftSimulationResult struct {
	ID               string   `json:"id"`
	AtMS             int64    `json:"at_ms"`
	DeviceID         string   `json:"device_id,omitempty"`
	Mode             string   `json:"mode"`
	MatchCount       int      `json:"match_count"`
	MatchedDeviceIDs []string `json:"matched_device_ids,omitempty"`
	MatchedUsers     []string `json:"matched_users,omitempty"`
	TopProcessPaths  []string `json:"top_process_paths,omitempty"`
	TopDestinations  []string `json:"top_destinations,omitempty"`
	WindowStartMS    int64    `json:"window_start_ms,omitempty"`
	WindowEndMS      int64    `json:"window_end_ms,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
	ExpectedBreakage string   `json:"expected_breakage_risk,omitempty"`
	Summary          string   `json:"summary"`
	ScopeSnapshot    []string `json:"scope_snapshot,omitempty"`
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

func (p *PlatformData) appendCandidate(c DetectionCandidate) {
	p.Candidates = append(p.Candidates, c)
	if len(p.Candidates) > 200 {
		p.Candidates = p.Candidates[len(p.Candidates)-200:]
	}
}

func (p *PlatformData) appendAuditBundle(b AuditBundle) {
	p.AuditBundles = append(p.AuditBundles, b)
	if len(p.AuditBundles) > 200 {
		p.AuditBundles = p.AuditBundles[len(p.AuditBundles)-200:]
	}
}
