package server

import (
	"sort"
	"testing"
)

func TestBuildABOMInsights_Categories(t *testing.T) {
	now := int64(1_700_000_000_000)
	day := int64(24 * 60 * 60 * 1000)

	items := []abomItem{
		{
			ID: "abom-coding-cursor", Category: abomCategoryCodingAgent, Product: "Cursor",
			Confidence: abomConfidenceMedium, DeviceIDs: []string{"dev-1"},
			FirstSeenMS: now - day/2, LastSeenMS: now - day/2,
			EvidenceRefs: []string{"process_guid:p1"},
		},
		{
			ID: "abom-runtime-ollama", Category: abomCategoryLocalModelRuntime, Product: "Ollama",
			Confidence: abomConfidenceHigh, DeviceIDs: []string{"dev-1", "dev-2", "dev-3", "dev-4"},
			FirstSeenMS: now - 30*day, LastSeenMS: now - day/4,
			EvidenceRefs: []string{"process_guid:p2"},
		},
		{
			ID: "abom-finding-unknown", Category: abomCategoryUnknownAIAutomation, Product: "Unknown automation",
			Confidence: abomConfidenceLow, DeviceIDs: []string{"dev-2"},
			FirstSeenMS: now - 2*day, LastSeenMS: now - 2*day,
		},
		{
			ID: "abom-stale-app", Category: abomCategoryAIDesktopApp, Product: "OldGPT",
			Confidence: abomConfidenceMedium, DeviceIDs: []string{"dev-5"},
			FirstSeenMS: now - 90*day, LastSeenMS: now - 30*day,
		},
	}

	bundle := buildABOMInsights(items, 6, []string{"dev-2"}, abomInsightOptions{
		Now:      now,
		WindowMS: day,
		StaleMS:  7 * day,
	})

	sectionByID := map[string]abomInsightSection{}
	for _, section := range bundle.Sections {
		sectionByID[section.ID] = section
	}

	// newly_observed should pick up the Cursor item (first_seen 12h ago).
	newSection, ok := sectionByID[abomInsightNew]
	if !ok || newSection.Total == 0 {
		t.Fatalf("expected newly_observed section to contain at least one row, got %+v", newSection)
	}
	if !containsItemID(newSection.Items, "abom-coding-cursor") {
		t.Fatalf("expected Cursor in newly_observed, got %+v", newSection.Items)
	}

	// widespread (Ollama on >= 30% of 6 == 2 minimum, plus minimum 3) — Ollama has 4.
	widespread, ok := sectionByID[abomInsightWidespread]
	if !ok || !containsItemID(widespread.Items, "abom-runtime-ollama") {
		t.Fatalf("expected Ollama to be flagged widespread, got %+v", widespread.Items)
	}

	// high_confidence picks up Ollama.
	high, ok := sectionByID[abomInsightHighConfidence]
	if !ok || !containsItemID(high.Items, "abom-runtime-ollama") {
		t.Fatalf("expected Ollama in high-confidence section, got %+v", high)
	}

	// low_confidence_needs_review picks up the unknown finding.
	low, ok := sectionByID[abomInsightLowConfidenceReview]
	if !ok || !containsItemID(low.Items, "abom-finding-unknown") {
		t.Fatalf("expected unknown automation in low-confidence section, got %+v", low)
	}

	// stale picks up OldGPT (last_seen 30d ago > 7d threshold).
	stale, ok := sectionByID[abomInsightStale]
	if !ok || !containsItemID(stale.Items, "abom-stale-app") {
		t.Fatalf("expected stale section to contain OldGPT, got %+v", stale)
	}

	// newly_observed_high_attention should NOT contain Cursor (dev-1 not high
	// attention) but the unknown automation is on dev-2 (high attention) and
	// first_seen is 2 days, outside the 24h window — so this stays empty.
	high2, ok := sectionByID[abomInsightNewHighAttention]
	if !ok {
		t.Fatalf("expected newly_observed_high_attention section")
	}
	if high2.Total != 0 {
		t.Fatalf("expected high-attention new section to be empty, got %+v", high2)
	}
}

func TestBuildABOMInsights_HighAttentionFlagged(t *testing.T) {
	now := int64(1_700_000_000_000)
	day := int64(24 * 60 * 60 * 1000)

	items := []abomItem{
		{
			ID: "abom-runtime-ollama", Category: abomCategoryLocalModelRuntime, Product: "Ollama",
			Confidence: abomConfidenceHigh, DeviceIDs: []string{"dev-2"},
			FirstSeenMS: now - 2*60*60*1000, // 2h ago — within window
			LastSeenMS:  now - 2*60*60*1000,
		},
	}

	bundle := buildABOMInsights(items, 4, []string{"dev-2"}, abomInsightOptions{
		Now:      now,
		WindowMS: day,
		StaleMS:  7 * day,
	})
	sectionByID := map[string]abomInsightSection{}
	for _, section := range bundle.Sections {
		sectionByID[section.ID] = section
	}
	highAttention := sectionByID[abomInsightNewHighAttention]
	if highAttention.Total != 1 {
		t.Fatalf("expected one high-attention new item, got %+v", highAttention)
	}
	if !highAttention.Items[0].HighAttention {
		t.Fatalf("expected HighAttention flag set on insight item")
	}
}

func TestBuildABOMHotspots_RanksHighAttentionFirst(t *testing.T) {
	now := int64(1_700_000_000_000)
	items := []abomItem{
		{ID: "a", Category: abomCategoryCodingAgent, Confidence: abomConfidenceHigh, DeviceIDs: []string{"dev-1", "dev-2"}, LastSeenMS: now},
		{ID: "b", Category: abomCategoryLocalModelRuntime, Confidence: abomConfidenceLow, DeviceIDs: []string{"dev-2"}, LastSeenMS: now - 60_000},
		{ID: "c", Category: abomCategoryModelGateway, Confidence: abomConfidenceMedium, DeviceIDs: []string{"dev-3"}, LastSeenMS: now - 2*60_000},
	}

	bundle := buildABOMInsights(items, 5, []string{"dev-3"}, abomInsightOptions{Now: now, WindowMS: 24 * 60 * 60 * 1000})
	if len(bundle.Hotspots) == 0 {
		t.Fatalf("expected hotspots, got none")
	}
	if !bundle.Hotspots[0].HighAttention || bundle.Hotspots[0].DeviceID != "dev-3" {
		t.Fatalf("expected high-attention dev-3 to rank first, got %+v", bundle.Hotspots[0])
	}
}

func TestBuildABOMInsights_EmptyInputs(t *testing.T) {
	bundle := buildABOMInsights(nil, 0, nil, abomInsightOptions{Now: 1_000_000})
	if len(bundle.Sections) == 0 {
		t.Fatalf("expected section scaffolding even with no items")
	}
	for _, section := range bundle.Sections {
		if section.Total != 0 {
			t.Fatalf("expected empty totals for section %q, got %d", section.ID, section.Total)
		}
	}
	if len(bundle.Hotspots) != 0 {
		t.Fatalf("expected no hotspots, got %v", bundle.Hotspots)
	}
}

func containsItemID(items []abomInsightItem, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

// helper used in legacy tests; kept here for consistency.
var _ = sort.SliceStable
