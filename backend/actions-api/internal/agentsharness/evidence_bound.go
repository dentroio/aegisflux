package agentsharness

import (
	"strings"
)

// EvidenceRefKind identifies what endpoint or platform artifact supports a claim.
const (
	EvidenceRefProcess          = "process"
	EvidenceRefDNS              = "dns"
	EvidenceRefFlow             = "flow"
	EvidenceRefFinding          = "finding"
	EvidenceRefABOMItem         = "abom_item"
	EvidenceRefDetectionPack    = "detection_pack"
	EvidenceRefAuditBundle      = "audit_bundle"
	EvidenceRefOperationalEvent = "operational_event"
	EvidenceRefCollectorHealth  = "collector_health"
	// Lab / integration summaries not tied to a single row ID.
	EvidenceRefIntegrationEvidenceSummary = "integration_evidence_summary"
	EvidenceRefDetectionCandidate         = "detection_candidate"
)

var evidenceRefKinds = map[string]struct{}{
	EvidenceRefProcess:                    {},
	EvidenceRefDNS:                        {},
	EvidenceRefFlow:                       {},
	EvidenceRefFinding:                    {},
	EvidenceRefABOMItem:                   {},
	EvidenceRefDetectionPack:              {},
	EvidenceRefAuditBundle:                {},
	EvidenceRefOperationalEvent:           {},
	EvidenceRefCollectorHealth:            {},
	EvidenceRefIntegrationEvidenceSummary: {},
	EvidenceRefDetectionCandidate:         {},
}

// EvidenceRef is a typed pointer to platform or endpoint evidence (no free-form blobs).
type EvidenceRef struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Detail string `json:"detail,omitempty"`
}

// MissingEvidenceCategory classifies what was not available for this run.
const (
	MissingProcessTelemetry      = "process_telemetry"
	MissingNetworkCapture        = "network_capture"
	MissingUserContext           = "user_context"
	MissingHistoricalWindow      = "historical_window"
	MissingABOMSnapshot          = "abom_snapshot"
	MissingSignedPackAttestation = "signed_pack_attestation"
	MissingCollectorHealth       = "collector_health"
)

var missingEvidenceCategories = map[string]struct{}{
	MissingProcessTelemetry:      {},
	MissingNetworkCapture:        {},
	MissingUserContext:           {},
	MissingHistoricalWindow:      {},
	MissingABOMSnapshot:          {},
	MissingSignedPackAttestation: {},
	MissingCollectorHealth:       {},
}

// MissingEvidenceItem records acknowledged gaps (run may still succeed).
type MissingEvidenceItem struct {
	Category string `json:"category"`
	Detail   string `json:"detail,omitempty"`
}

// ConfidenceBucket is a coarse calibration; rationale is mandatory (WO-AGENTS-002).
const (
	ConfidenceLow     = "low"
	ConfidenceMedium  = "medium"
	ConfidenceHigh    = "high"
	ConfidenceUnknown = "unknown"
)

var confidenceBuckets = map[string]struct{}{
	ConfidenceLow:     {},
	ConfidenceMedium:  {},
	ConfidenceHigh:    {},
	ConfidenceUnknown: {},
}

// EvidenceBoundConclusion is the required shape for governed agent outputs.
type EvidenceBoundConclusion struct {
	Conclusion          string                `json:"conclusion"`
	Evidence            []EvidenceRef         `json:"evidence"`
	Assumptions         []string              `json:"assumptions"`
	MissingEvidence     []MissingEvidenceItem `json:"missing_evidence"`
	ConfidenceBucket    string                `json:"confidence_bucket"`
	ConfidenceRationale string                `json:"confidence_rationale"`
	SafetyBoundaries    []string              `json:"safety_boundaries"`
	Recommendations     []string              `json:"recommendations,omitempty"`
}

// ValidateEvidenceOptions configures completion-time checks.
type ValidateEvidenceOptions struct {
	// ProductImpacting requires full contract (detections, controls, approval packets, and default lab analyst runs).
	ProductImpacting bool
}

const minRationaleRunes = 12

// ValidateEvidenceBoundConclusion returns human-readable violations (empty if valid).
func ValidateEvidenceBoundConclusion(c *EvidenceBoundConclusion, opt ValidateEvidenceOptions) []string {
	if c == nil {
		return []string{"evidence_bound_conclusion is required"}
	}
	var out []string
	if len([]rune(strings.TrimSpace(c.Conclusion))) < 1 {
		out = append(out, "conclusion must be non-empty")
	}
	if len([]rune(strings.TrimSpace(c.ConfidenceRationale))) < minRationaleRunes {
		out = append(out, "confidence_rationale must be at least 12 characters")
	}
	if _, ok := confidenceBuckets[c.ConfidenceBucket]; !ok {
		out = append(out, "confidence_bucket must be one of low, medium, high, unknown")
	}
	for _, e := range c.Evidence {
		if _, ok := evidenceRefKinds[e.Kind]; !ok {
			out = append(out, "unknown evidence kind: "+e.Kind)
		}
		if strings.TrimSpace(e.Ref) == "" {
			out = append(out, "each evidence ref must include ref")
		}
	}
	for _, m := range c.MissingEvidence {
		if _, ok := missingEvidenceCategories[m.Category]; !ok {
			out = append(out, "unknown missing_evidence category: "+m.Category)
		}
	}
	if opt.ProductImpacting {
		if len(c.Evidence)+len(c.MissingEvidence) < 1 {
			out = append(out, "product-impacting runs require at least one evidence ref or missing_evidence item")
		}
		if len(c.SafetyBoundaries) < 1 {
			out = append(out, "safety_boundaries must include at least one boundary for product-impacting runs")
		}
	}
	return out
}
