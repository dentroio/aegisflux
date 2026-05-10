package api

import (
	"sort"
	"testing"
)

func TestDiffSnapshotKeys_DetectsScopeAndNotes(t *testing.T) {
	before := DraftSnapshot{
		Status:         "draft_observe_only",
		ScopeSelectors: []string{"device:win-1"},
		OperatorNotes:  "",
	}
	after := DraftSnapshot{
		Status:         "draft_observe_only",
		ScopeSelectors: []string{"device:win-1", "user:alice"},
		OperatorNotes:  "Review with platform.",
	}
	keys := diffSnapshotKeys(before, after)
	expected := []string{"operator_notes", "scope_selectors"}
	sort.Strings(expected)
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %v", len(expected), keys)
	}
	for i := range keys {
		if keys[i] != expected[i] {
			t.Fatalf("unexpected key set: got %v want %v", keys, expected)
		}
	}
}

func TestDiffSnapshotKeys_DetectsStatus(t *testing.T) {
	before := DraftSnapshot{Status: "draft_observe_only"}
	after := DraftSnapshot{Status: "draft_reviewed"}
	keys := diffSnapshotKeys(before, after)
	if len(keys) != 1 || keys[0] != "status" {
		t.Fatalf("expected only status to change, got %v", keys)
	}
}

func TestDecideHistoryAction(t *testing.T) {
	if got := decideHistoryAction([]string{"status"}, "draft_observe_only", "draft_archived"); got != "status_changed" {
		t.Fatalf("expected status_changed, got %s", got)
	}
	if got := decideHistoryAction([]string{"scope_selectors"}, "draft_observe_only", "draft_observe_only"); got != "scope_edited" {
		t.Fatalf("expected scope_edited, got %s", got)
	}
	if got := decideHistoryAction([]string{"operator_notes"}, "draft_observe_only", "draft_observe_only"); got != "note_added" {
		t.Fatalf("expected note_added, got %s", got)
	}
	if got := decideHistoryAction(nil, "draft_observe_only", "draft_observe_only"); got != "updated" {
		t.Fatalf("expected updated for empty diff, got %s", got)
	}
}

func TestBuildSimulationResult_ProducesObservableProjection(t *testing.T) {
	draft := DraftControl{
		ID:             "draft-123",
		ScopeSelectors: []string{"device:win-lab-1"},
	}
	now := int64(1_700_000_000_000)
	sim := buildSimulationResult(draft, "win-lab-1", now)

	if sim.Mode != "observe_only" {
		t.Fatalf("expected observe_only mode, got %s", sim.Mode)
	}
	if sim.MatchCount <= 0 {
		t.Fatalf("expected positive match count, got %d", sim.MatchCount)
	}
	if len(sim.MatchedDeviceIDs) == 0 {
		t.Fatalf("expected at least one matched device, got %+v", sim.MatchedDeviceIDs)
	}
	if !containsStr(sim.MatchedDeviceIDs, "win-lab-1") {
		t.Fatalf("expected sim to include source device, got %+v", sim.MatchedDeviceIDs)
	}
	if len(sim.TopProcessPaths) == 0 || len(sim.TopDestinations) == 0 {
		t.Fatalf("expected top process paths and destinations, got %+v", sim)
	}
	if sim.WindowEndMS-sim.WindowStartMS != 24*60*60*1000 {
		t.Fatalf("expected 24h window, got %d", sim.WindowEndMS-sim.WindowStartMS)
	}
	if sim.Summary == "" || sim.ScopeSnapshot == nil {
		t.Fatalf("expected summary and scope snapshot, got %+v", sim)
	}

	// Deterministic: rerun should produce the same match count and top lists.
	sim2 := buildSimulationResult(draft, "win-lab-1", now+1)
	if sim2.MatchCount != sim.MatchCount {
		t.Fatalf("expected deterministic match count, got %d vs %d", sim.MatchCount, sim2.MatchCount)
	}
}

func TestAppendDraftHistory_BoundedAndStamped(t *testing.T) {
	d := &DraftControl{}
	for i := 0; i < 60; i++ {
		appendDraftHistory(d, DraftDecisionEntry{Action: "test"})
	}
	if len(d.History) != 50 {
		t.Fatalf("expected history to be capped at 50, got %d", len(d.History))
	}
	for _, entry := range d.History {
		if entry.ID == "" || entry.AtMS == 0 || entry.Actor == "" {
			t.Fatalf("expected history entry to be stamped, got %+v", entry)
		}
	}
}
