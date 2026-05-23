package agentsharness

import "encoding/json"

const (
	AgentEndpointAnalyst     = "endpoint_analyst"
	AgentDetectionResearcher = "detection_researcher"
	AgentPackAuthor          = "pack_author"
	AgentSimulationAgent     = "simulation_agent"
	AgentControlDesigner     = "control_designer"
	AgentGovernanceReviewer  = "governance_reviewer"
)

// SystemAgents lists initial system agents (WO-AGENTS-001 deliverable).
var SystemAgents = []SystemAgentMeta{
	{
		ID:           AgentEndpointAnalyst,
		DisplayName:  "Endpoint Analyst",
		Description:  "Summarizes endpoint AI activity and evidence using read-only platform tools.",
		AllowedTools: []string{ToolDeviceEvidenceSummary, ToolFindingsEvidencePaths, ToolDetectionCandidateLookup},
	},
	{
		ID:           AgentDetectionResearcher,
		DisplayName:  "Detection Researcher",
		Description:  "Investigates detection opportunities with bounded read tools (lab stub).",
		AllowedTools: []string{ToolFindingsEvidencePaths, ToolDetectionCandidateLookup},
	},
	{
		ID:           AgentPackAuthor,
		DisplayName:  "Pack Author",
		Description:  "Drafts detection pack narratives from candidates (read-only lookup in lab).",
		AllowedTools: []string{ToolDetectionCandidateLookup},
	},
	{
		ID:           AgentSimulationAgent,
		DisplayName:  "Simulation Agent",
		Description:  "Plans observe-only simulations using evidence path context (lab stub).",
		AllowedTools: []string{ToolDeviceEvidenceSummary, ToolFindingsEvidencePaths},
	},
	{
		ID:           AgentControlDesigner,
		DisplayName:  "Control Designer",
		Description:  "Maps findings to control design context without enforcement tools.",
		AllowedTools: []string{ToolFindingsEvidencePaths, ToolDeviceEvidenceSummary},
	},
	{
		ID:           AgentGovernanceReviewer,
		DisplayName:  "Governance Reviewer",
		Description:  "Reviews agent outputs against privacy and audit posture (lab stub).",
		AllowedTools: []string{ToolDeviceEvidenceSummary, ToolDetectionCandidateLookup},
	},
}

// ToolRegistry returns typed metadata for all harness tools.
func ToolRegistry() []ToolMeta {
	return []ToolMeta{
		{
			ID:           ToolDeviceEvidenceSummary,
			DisplayName:  "Device evidence summary",
			Description:  "Bounded integration evidence summary for one device.",
			Mutates:      false,
			InputSchema:  json.RawMessage(`{"type":"object","properties":{"device_id":{"type":"string"}},"required":["device_id"]}`),
			OutputSchema: json.RawMessage(`{"type":"object","description":"integration.evidence_summary.v1"}`),
		},
		{
			ID:           ToolFindingsEvidencePaths,
			DisplayName:  "Findings / evidence path lookup",
			Description:  "Resolves console-relative paths for findings linked to a device.",
			Mutates:      false,
			InputSchema:  json.RawMessage(`{"type":"object","properties":{"device_id":{"type":"string"},"finding_id":{"type":"string"}},"required":["device_id"]}`),
			OutputSchema: json.RawMessage(`{"type":"object","properties":{"paths":{"type":"array"}}}`),
		},
		{
			ID:           ToolDetectionCandidateLookup,
			DisplayName:  "Detection candidate lookup",
			Description:  "Fetches a detection candidate by id or lists recent lab candidates.",
			Mutates:      false,
			InputSchema:  json.RawMessage(`{"type":"object","properties":{"candidate_id":{"type":"string"},"device_id":{"type":"string"}}}`),
			OutputSchema: json.RawMessage(`{"type":"object"}`),
		},
	}
}

func agentMeta(id string) (SystemAgentMeta, bool) {
	for _, a := range SystemAgents {
		if a.ID == id {
			return a, true
		}
	}
	return SystemAgentMeta{}, false
}

func toolAllowedForAgent(agentID, toolID string) bool {
	meta, ok := agentMeta(agentID)
	if !ok {
		return false
	}
	for _, t := range meta.AllowedTools {
		if t == toolID {
			return true
		}
	}
	return false
}
