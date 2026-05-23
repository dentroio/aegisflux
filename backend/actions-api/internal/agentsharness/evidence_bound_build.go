package agentsharness

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildEvidenceBoundConclusion synthesizes WO-AGENTS-002 contract fields from audited tool outputs (lab).
func BuildEvidenceBoundConclusion(spec RunSpec, calls []ToolCallRecord, assessment, evidenceNarrative, legacyConfidence, nextAction string, allowExternal bool) *EvidenceBoundConclusion {
	evidence := make([]EvidenceRef, 0, 8)
	for _, tc := range calls {
		if tc.Error != "" || len(tc.OutputJSON) == 0 {
			continue
		}
		switch tc.ToolID {
		case ToolDeviceEvidenceSummary:
			evidence = append(evidence, EvidenceRef{
				Kind:   EvidenceRefIntegrationEvidenceSummary,
				Ref:    "device:" + spec.DeviceID,
				Detail: "read.device_evidence_summary output (integration schema v1)",
			})
			var payload map[string]any
			if json.Unmarshal(tc.OutputJSON, &payload) == nil {
				if names, ok := payload["integration_event_names"].([]any); ok && len(names) > 0 {
					parts := make([]string, 0, len(names))
					for _, n := range names {
						if s, ok := n.(string); ok && s != "" {
							parts = append(parts, s)
						}
					}
					if len(parts) > 0 {
						detail := strings.Join(parts, ", ")
						if len(detail) > 400 {
							detail = detail[:400] + "…"
						}
						evidence = append(evidence, EvidenceRef{
							Kind:   EvidenceRefOperationalEvent,
							Ref:    "device:" + spec.DeviceID + ":integration_event_names",
							Detail: detail,
						})
					}
				}
			}
		case ToolFindingsEvidencePaths:
			var wrap struct {
				Paths []map[string]string `json:"paths"`
			}
			if json.Unmarshal(tc.OutputJSON, &wrap) == nil {
				for _, p := range wrap.Paths {
					fid := strings.TrimSpace(p["finding_id"])
					if fid != "" {
						evidence = append(evidence, EvidenceRef{
							Kind:   EvidenceRefFinding,
							Ref:    "finding:" + fid,
							Detail: strings.TrimSpace(p["relative"]),
						})
					}
				}
			}
		case ToolDetectionCandidateLookup:
			var payload map[string]any
			if json.Unmarshal(tc.OutputJSON, &payload) != nil {
				break
			}
			if id, ok := payload["candidate_id"].(string); ok && strings.TrimSpace(id) != "" {
				evidence = append(evidence, EvidenceRef{
					Kind:   EvidenceRefDetectionCandidate,
					Ref:    "detection_candidate:" + id,
					Detail: "lookup result",
				})
			} else if recent, ok := payload["recent"].([]any); ok {
				for _, it := range recent {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					cid, _ := m["candidate_id"].(string)
					if cid != "" {
						evidence = append(evidence, EvidenceRef{
							Kind:   EvidenceRefDetectionCandidate,
							Ref:    "detection_candidate:" + cid,
							Detail: "recent candidate list entry",
						})
					}
				}
			}
		}
	}

	missing := []MissingEvidenceItem{
		{
			Category: MissingProcessTelemetry,
			Detail:   "Lab integration summary does not attach raw process/DNS/flow rows; only bounded summary fields.",
		},
	}
	if spec.Context == nil || len(spec.Context) == 0 {
		missing = append(missing, MissingEvidenceItem{
			Category: MissingUserContext,
			Detail:   "No operator context keys were supplied on the run request.",
		})
	}

	bucket := mapLegacyConfidenceToBucket(legacyConfidence, allowExternal)
	rationale := fmt.Sprintf(
		"Based on %d successful read-only tool call(s); external AI %s. Narrative evidence excerpt length=%d runes.",
		countOKTools(calls),
		map[bool]string{true: "allowed by policy (still redacted)", false: "disabled; deterministic path"}[allowExternal],
		len([]rune(evidenceNarrative)),
	)

	assumptions := []string{
		"Tool outputs are produced by the governed Actions API lab harness, not by an endpoint-hosted model.",
		"Evidence links are console-relative paths unless/until deep visibility joins are enabled.",
	}
	if allowExternal {
		assumptions = append(assumptions, "If external providers were invoked in production, latency and token budget would affect completeness (not modeled in this lab stub).")
	}

	safety := []string{
		"Observe-only analysis: no enforcement, blocking, or quarantine actions are implied or executed by this run.",
		"No endpoint mutation tools are registered for governed agents.",
		"Operator prompts are stored only as redacted previews for audit; no hidden chain-of-thought retention.",
	}

	recs := []string{}
	if strings.TrimSpace(nextAction) != "" {
		recs = append(recs, strings.TrimSpace(nextAction))
	}

	return &EvidenceBoundConclusion{
		Conclusion:          strings.TrimSpace(assessment),
		Evidence:            evidence,
		Assumptions:         assumptions,
		MissingEvidence:     missing,
		ConfidenceBucket:    bucket,
		ConfidenceRationale: rationale,
		SafetyBoundaries:    safety,
		Recommendations:     recs,
	}
}

func countOKTools(calls []ToolCallRecord) int {
	n := 0
	for _, tc := range calls {
		if tc.Error == "" {
			n++
		}
	}
	return n
}

func mapLegacyConfidenceToBucket(legacy string, allowExternal bool) string {
	s := strings.ToLower(legacy)
	switch {
	case strings.Contains(s, "high"):
		return ConfidenceHigh
	case strings.Contains(s, "low"):
		return ConfidenceLow
	case strings.Contains(s, "unknown"):
		return ConfidenceUnknown
	case strings.Contains(s, "medium"):
		return ConfidenceMedium
	default:
		if allowExternal {
			return ConfidenceMedium
		}
		return ConfidenceHigh
	}
}
