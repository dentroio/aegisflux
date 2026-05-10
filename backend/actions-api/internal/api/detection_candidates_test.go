package api

import (
	"testing"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		ok       bool
	}{
		{candidateStatusNew, candidateStatusSimulated, true},
		{candidateStatusSimulated, candidateStatusReviewed, true},
		{candidateStatusReviewed, candidateStatusSigned, true},
		{candidateStatusSigned, candidateStatusDeployed, true},
		{candidateStatusDeployed, candidateStatusRetired, true},
		{candidateStatusNew, candidateStatusSigned, false},
		{candidateStatusReviewed, candidateStatusDeployed, false},
		{candidateStatusRetired, candidateStatusSimulated, false},
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.to); got != c.ok {
			t.Errorf("canTransition(%s -> %s) = %v, want %v", c.from, c.to, got, c.ok)
		}
	}
}

func TestRecomputeCandidateGate_FlagsMissingFields(t *testing.T) {
	c := DetectionCandidate{ID: "c-1"}
	c = recomputeCandidateGate(c)
	if len(c.QualityGate.MissingFields) == 0 {
		t.Fatalf("expected missing fields, got none")
	}
	expectedMissing := map[string]bool{
		"required_evidence":         true,
		"expected_false_positives":  true,
		"simulation_run":            true,
		"reviewer_notes":            true,
		"expiration_date":           true,
		"rollback_plan":             true,
	}
	for _, m := range c.QualityGate.MissingFields {
		if !expectedMissing[m] {
			t.Fatalf("unexpected missing field %q", m)
		}
		delete(expectedMissing, m)
	}
	if len(expectedMissing) > 0 {
		t.Fatalf("expected gate to flag remaining %v", expectedMissing)
	}
}

func TestRecomputeCandidateGate_AllPresent(t *testing.T) {
	c := DetectionCandidate{
		QualityGate: DetectionCandidateGate{
			RequiredEvidence:       []string{"process telemetry"},
			ExpectedFalsePositives: "low",
		},
		Simulations:   []DetectionCandidateSimulation{{ID: "sim-1"}},
		ReviewerNotes: "reviewed by sec eng",
		ExpiresAtMS:   1_700_000_000_000,
		RollbackPlan:  "deactivate pack and revert to previous version",
	}
	c = recomputeCandidateGate(c)
	if len(c.QualityGate.MissingFields) != 0 {
		t.Fatalf("expected no missing gates, got %+v", c.QualityGate.MissingFields)
	}
}

func TestBuildCandidateSimulation_DeterministicAndAnnotated(t *testing.T) {
	c := DetectionCandidate{ID: "c-9", Title: "Test", Rule: ResearchSuggestedRule{Logic: "a == 1 AND b CONTAINS 'x'"}}
	now := int64(1_700_000_000_000)
	first := buildCandidateSimulation(c, now)
	second := buildCandidateSimulation(c, now+1)
	if first.MatchCount != second.MatchCount || first.MatchedDeviceCount != second.MatchedDeviceCount {
		t.Fatalf("expected deterministic projection counts, got %v vs %v", first, second)
	}
	if len(first.TopIndicators) == 0 {
		t.Fatalf("expected top indicators parsed from rule logic, got %+v", first.TopIndicators)
	}
	if first.Notes == "" {
		t.Fatalf("expected sim notes")
	}
}

func TestActionForCandidateChange(t *testing.T) {
	if got := actionForCandidateChange(candidateStatusNew, candidateStatusSimulated); got != "simulated" {
		t.Fatalf("unexpected: %s", got)
	}
	if got := actionForCandidateChange(candidateStatusReviewed, candidateStatusSigned); got != "signed" {
		t.Fatalf("unexpected: %s", got)
	}
	if got := actionForCandidateChange(candidateStatusSigned, candidateStatusSigned); got != "updated" {
		t.Fatalf("unexpected: %s", got)
	}
}
