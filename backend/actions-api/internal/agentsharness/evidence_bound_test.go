package agentsharness

import "testing"

func TestValidateEvidenceBoundConclusionValid(t *testing.T) {
	c := &EvidenceBoundConclusion{
		Conclusion:          "Endpoint shows AI-adjacent signals consistent with lab summary.",
		Evidence:            []EvidenceRef{{Kind: EvidenceRefFinding, Ref: "finding:f1"}},
		Assumptions:         []string{"Lab harness only."},
		MissingEvidence:     []MissingEvidenceItem{{Category: MissingProcessTelemetry, Detail: "No raw rows."}},
		ConfidenceBucket:    ConfidenceHigh,
		ConfidenceRationale: "Deterministic read-only tools only; bounded integration summary.",
		SafetyBoundaries:    []string{"Observe-only."},
		Recommendations:     []string{"Review findings."},
	}
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateEvidenceBoundConclusionInvalidMissingRationale(t *testing.T) {
	c := &EvidenceBoundConclusion{
		Conclusion:          "x",
		Evidence:            []EvidenceRef{{Kind: EvidenceRefFinding, Ref: "finding:f1"}},
		ConfidenceBucket:    ConfidenceHigh,
		ConfidenceRationale: "short",
		SafetyBoundaries:    []string{"b"},
	}
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) == 0 {
		t.Fatal("expected errors")
	}
}

func TestValidateEvidenceBoundConclusionInvalidNoEvidenceOrMissing(t *testing.T) {
	c := &EvidenceBoundConclusion{
		Conclusion:          "Some conclusion text that is long enough for the validator to accept.",
		Evidence:            nil,
		MissingEvidence:     nil,
		ConfidenceBucket:    ConfidenceMedium,
		ConfidenceRationale: "Twelve+ chars here for product impacting negative test path coverage.",
		SafetyBoundaries:    []string{"observe-only"},
	}
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) == 0 {
		t.Fatal("expected errors for missing evidence and missing_evidence")
	}
}

func TestValidateEvidenceBoundConclusionLowConfidenceValid(t *testing.T) {
	c := &EvidenceBoundConclusion{
		Conclusion:          "Limited telemetry; conclusion is provisional.",
		Evidence:            []EvidenceRef{{Kind: EvidenceRefDNS, Ref: "dns:device-x:q1"}},
		MissingEvidence:     []MissingEvidenceItem{{Category: MissingHistoricalWindow, Detail: "Only 1h window."}},
		ConfidenceBucket:    ConfidenceLow,
		ConfidenceRationale: "Sparse signals and wide uncertainty bands on the operator-supplied context.",
		SafetyBoundaries:    []string{"No enforcement implied."},
	}
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) != 0 {
		t.Fatalf("unexpected: %v", errs)
	}
}

func TestValidateEvidenceBoundUnknownEvidenceKind(t *testing.T) {
	c := &EvidenceBoundConclusion{
		Conclusion:          "Conclusion with enough text for all validators in this package.",
		Evidence:            []EvidenceRef{{Kind: "not_a_kind", Ref: "x"}},
		MissingEvidence:     []MissingEvidenceItem{{Category: MissingUserContext}},
		ConfidenceBucket:    ConfidenceHigh,
		ConfidenceRationale: "Rationale meets minimum length requirement for validation.",
		SafetyBoundaries:    []string{"boundary"},
	}
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) == 0 {
		t.Fatal("expected unknown kind error")
	}
}

func TestBuildEvidenceBoundRoundTripValidates(t *testing.T) {
	spec := RunSpec{DeviceID: "d-1", FindingID: "f-9", Context: map[string]any{"k": 1}}
	calls := []ToolCallRecord{
		{ToolID: ToolDeviceEvidenceSummary, OutputJSON: []byte(`{"integration_event_names":["aegis.device.observed"],"device_id":"d-1"}`)},
		{ToolID: ToolFindingsEvidencePaths, OutputJSON: []byte(`{"paths":[{"finding_id":"f-9","relative":"/x"}]}`)},
	}
	c := BuildEvidenceBoundConclusion(spec, calls, "Assessment line.", "evidence narrative", "High (rules)", "Next step", false)
	errs := ValidateEvidenceBoundConclusion(c, ValidateEvidenceOptions{ProductImpacting: true})
	if len(errs) != 0 {
		t.Fatalf("round trip: %v", errs)
	}
}
