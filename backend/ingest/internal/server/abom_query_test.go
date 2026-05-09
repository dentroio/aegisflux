package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptrString(value string) *string { return &value }

func TestBuildABOMItems_LocalRuntimeAndGateway(t *testing.T) {
	processes := []processRecord{
		{
			DeviceID:    "win-lab-1",
			TimestampMS: 1_700_000_000_000,
			PID:         4321,
			Name:        ptrString("ollama"),
			Path:        ptrString("/usr/local/bin/ollama"),
			CommandLine: ptrString("ollama serve"),
		},
	}
	dnsRows := []dnsRecord{
		{
			DeviceID:    "win-lab-1",
			TimestampMS: 1_700_000_001_000,
			Query:       "api.openai.com",
			Answers:     []string{"104.18.6.1"},
		},
	}

	items := buildABOMItems(processes, dnsRows, nil, nil, nil)
	if len(items) != 2 {
		t.Fatalf("expected 2 ABOM items, got %d: %#v", len(items), items)
	}

	gotCategories := map[string]abomItem{}
	for _, item := range items {
		gotCategories[item.Category] = item
	}
	if _, ok := gotCategories[abomCategoryLocalModelRuntime]; !ok {
		t.Fatalf("expected local_model_runtime item, got %#v", items)
	}
	if _, ok := gotCategories[abomCategoryModelGateway]; !ok {
		t.Fatalf("expected model_gateway item, got %#v", items)
	}

	runtime := gotCategories[abomCategoryLocalModelRuntime]
	if runtime.Confidence != abomConfidenceHigh {
		t.Errorf("expected high confidence for ollama, got %s", runtime.Confidence)
	}
	if !contains(runtime.CapabilityTags, "local-llm") {
		t.Errorf("expected local-llm capability, got %v", runtime.CapabilityTags)
	}
	if len(runtime.EvidenceRefs) == 0 || !strings.Contains(runtime.EvidenceRefs[0], "process:ollama") {
		t.Errorf("expected process evidence ref, got %v", runtime.EvidenceRefs)
	}

	gateway := gotCategories[abomCategoryModelGateway]
	if gateway.Product != "OpenAI API" {
		t.Errorf("expected OpenAI API product, got %q", gateway.Product)
	}
	if gateway.RecommendedReview == "" {
		t.Errorf("expected non-empty recommended review")
	}
}

func TestBuildABOMItems_BrowserExtensionAndMCP(t *testing.T) {
	extPayload, _ := json.Marshal(map[string]any{
		"extension_id":      "ai-helper-id",
		"name":              "AI Tab Companion",
		"host_permissions":  []any{"*://*/*"},
	})
	extEvent := visibilityEvent{
		EventID:     "ext-1",
		EventType:   "aegis.browser_extension.observed",
		DeviceID:    "linux-lab-1",
		TimestampMS: 1_700_000_002_000,
		Payload:     extPayload,
	}

	processes := []processRecord{
		{
			DeviceID:    "linux-lab-1",
			TimestampMS: 1_700_000_003_000,
			PID:         1234,
			Name:        ptrString("mcp-server-files"),
			Path:        ptrString("/opt/mcp/mcp-server-files"),
			CommandLine: ptrString("mcp-server-files --root /tmp"),
		},
	}

	items := buildABOMItems(processes, nil, nil, []visibilityEvent{extEvent}, nil)
	if len(items) != 2 {
		t.Fatalf("expected 2 items (browser extension + MCP), got %d: %#v", len(items), items)
	}

	gotCategories := map[string]abomItem{}
	for _, item := range items {
		gotCategories[item.Category] = item
	}
	ext, ok := gotCategories[abomCategoryBrowserAIExtension]
	if !ok {
		t.Fatalf("missing browser_ai_extension item, got %#v", items)
	}
	if !contains(ext.CapabilityTags, "broad-host-access") {
		t.Errorf("expected broad-host-access tag for *://*/* permission, got %v", ext.CapabilityTags)
	}
	if ext.Confidence != abomConfidenceHigh {
		t.Errorf("expected high confidence for broad host access, got %s", ext.Confidence)
	}

	mcp, ok := gotCategories[abomCategoryMCPEndpoint]
	if !ok {
		t.Fatalf("missing mcp_endpoint item, got %#v", items)
	}
	if !contains(mcp.CapabilityTags, "protocol:mcp") {
		t.Errorf("expected protocol:mcp tag, got %v", mcp.CapabilityTags)
	}
}

func TestBuildABOMItems_FindingPromotesUnknownAutomation(t *testing.T) {
	finding := findingRecord{
		EventID:     "f-1",
		EventType:   "aegis.risk_finding.created",
		DeviceID:    "win-lab-2",
		TimestampMS: 1_700_000_004_000,
		Title:       ptrString("Suspicious Claude desktop activity"),
		FindingID:   ptrString("find-claude-1"),
		DetectedPatterns: []string{"ai_agent"},
	}

	items := buildABOMItems(nil, nil, []findingRecord{finding}, nil, nil)
	if len(items) != 1 {
		t.Fatalf("expected one ABOM item from finding, got %d", len(items))
	}
	if items[0].Category != abomCategoryUnknownAIAutomation {
		t.Fatalf("expected unknown_ai_automation, got %s", items[0].Category)
	}
	if items[0].Confidence != abomConfidenceLow {
		t.Errorf("expected low confidence for finding-only item, got %s", items[0].Confidence)
	}
	if !contains(items[0].EvidenceRefs, "finding:find-claude-1") {
		t.Errorf("expected finding evidence ref, got %v", items[0].EvidenceRefs)
	}
}

func TestBuildABOMItems_DeduplicatesAcrossDevicesAndPromotesConfidence(t *testing.T) {
	processes := []processRecord{
		{DeviceID: "host-a", TimestampMS: 1, PID: 1, Name: ptrString("ollama"), Path: ptrString("/usr/bin/ollama")},
		{DeviceID: "host-b", TimestampMS: 2, PID: 1, Name: ptrString("ollama"), Path: ptrString("/usr/bin/ollama")},
		{DeviceID: "host-c", TimestampMS: 3, PID: 1, Name: ptrString("ollama"), Path: ptrString("/usr/bin/ollama")},
	}
	items := buildABOMItems(processes, nil, nil, nil, nil)
	if len(items) != 1 {
		t.Fatalf("expected single deduped item, got %d", len(items))
	}
	if len(items[0].DeviceIDs) != 3 {
		t.Errorf("expected 3 devices, got %v", items[0].DeviceIDs)
	}
	if items[0].Confidence != abomConfidenceHigh {
		t.Errorf("expected high confidence (>=3 devices), got %s", items[0].Confidence)
	}
}

func TestBuildABOMItems_EmptyInputs(t *testing.T) {
	items := buildABOMItems(nil, nil, nil, nil, nil)
	if len(items) != 0 {
		t.Fatalf("expected empty result, got %d items", len(items))
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
