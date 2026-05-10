package server

import (
	"strings"
	"testing"
)

func intPtr(i int) *int { return &i }

func TestBuildEvidencePath_FullPath(t *testing.T) {
	parent := "parent-guid-1"
	process := processRecord{
		EventID:           "p-1",
		EventType:         "aegis.process.started",
		TimestampMS:       1_700_000_000_000,
		DeviceID:          "win-lab-1",
		AgentID:           "agent-1",
		ProcessGUID:       "proc-guid-1",
		ParentProcessGUID: &parent,
		PID:               4321,
		Name:              ptrString("ollama"),
		Path:              ptrString("/usr/local/bin/ollama"),
		CommandLine:       ptrString("ollama serve"),
		User:              ptrString("alice"),
	}
	flow := flowRecord{
		EventID:        "f-1",
		EventType:      "aegis.flow.started",
		TimestampMS:    1_700_000_001_000,
		DeviceID:       "win-lab-1",
		AgentID:        "agent-1",
		FlowID:         "flow-1",
		Protocol:       ptrString("tcp"),
		Direction:      ptrString("outbound"),
		RemoteIP:       ptrString("104.18.6.1"),
		RemotePort:     intPtr(443),
		RemoteHostname: ptrString("api.openai.com"),
		ProcessGUID:    &process.ProcessGUID,
	}
	dnsHit := dnsRecord{
		EventID:     "d-1",
		EventType:   "aegis.dns.observed",
		TimestampMS: 1_700_000_001_500,
		DeviceID:    "win-lab-1",
		AgentID:     "agent-1",
		Query:       "api.openai.com",
		Answers:     []string{"104.18.6.1"},
		ProcessGUID: &process.ProcessGUID,
	}
	finding := findingRecord{
		EventID:     "find-1",
		EventType:   "aegis.risk_finding.created",
		TimestampMS: 1_700_000_002_000,
		DeviceID:    "win-lab-1",
		AgentID:     "agent-1",
		Title:       ptrString("AI agent egress"),
		FindingID:   ptrString("find-uuid-1"),
		DetectionID: ptrString("det-ai-egress"),
		ProcessGUID: &process.ProcessGUID,
		RiskScore:   72,
		Severity:    ptrString("high"),
	}
	drafts := buildDraftControls("win-lab-1", investigationFilters{ProcessGUID: process.ProcessGUID}, []processRecord{process}, []flowRecord{flow}, []dnsRecord{dnsHit}, []findingRecord{finding})
	if len(drafts) == 0 {
		t.Fatalf("expected at least one draft control to be derived")
	}

	nodes, edges, missing, overall, summary := buildEvidencePath(
		"win-lab-1", "agent-1", &finding,
		[]processRecord{process}, []flowRecord{flow}, []dnsRecord{dnsHit}, []findingRecord{finding}, drafts,
	)

	requireNode(t, nodes, "endpoint")
	requireNode(t, nodes, "process")
	requireNode(t, nodes, "flow")
	requireNode(t, nodes, "dns")
	requireNode(t, nodes, "finding")
	requireNode(t, nodes, "detection_pack")

	hasMatched := false
	hasObservedOn := false
	hasResolved := false
	hasConnected := false
	hasSupportsControl := false
	for _, edge := range edges {
		switch edge.Label {
		case "matched":
			hasMatched = true
		case "observed_on":
			hasObservedOn = true
		case "resolved":
			hasResolved = true
		case "connected":
			hasConnected = true
		case "supports_control":
			hasSupportsControl = true
		}
	}
	if !hasMatched || !hasObservedOn || !hasResolved || !hasConnected || !hasSupportsControl {
		t.Fatalf("expected matched/observed_on/resolved/connected/supports_control edges, got %+v", edges)
	}

	if overall != evidenceConfidenceHigh && overall != evidenceConfidenceMedium {
		t.Errorf("expected high/medium overall confidence, got %s", overall)
	}
	if !strings.Contains(summary, "AI agent egress") {
		t.Errorf("expected summary to mention finding title, got %q", summary)
	}
	for _, want := range []string{"endpoint", "process", "flow", "dns", "finding"} {
		for _, got := range missing {
			if got == want {
				t.Errorf("expected %s to be present, but it appeared in missing", want)
			}
		}
	}
}

func TestBuildEvidencePath_PartialEvidence(t *testing.T) {
	finding := findingRecord{
		EventID:     "find-2",
		EventType:   "aegis.risk_finding.created",
		TimestampMS: 1,
		DeviceID:    "linux-lab-1",
		Title:       ptrString("Telemetry anomaly"),
		FindingID:   ptrString("find-2"),
		RiskScore:   30,
	}

	nodes, _, missing, overall, summary := buildEvidencePath(
		"linux-lab-1", "", &finding,
		nil, nil, nil, []findingRecord{finding}, nil,
	)
	if len(nodes) == 0 {
		t.Fatalf("expected nodes even with partial evidence")
	}

	missingSet := map[string]struct{}{}
	for _, m := range missing {
		missingSet[m] = struct{}{}
	}
	for _, want := range []string{"process", "flow", "dns", "parent_process", "detection_pack"} {
		if _, ok := missingSet[want]; !ok {
			t.Errorf("expected %q in missing list, got %v", want, missing)
		}
	}

	processMissing := false
	for _, node := range nodes {
		if node.Type == "process" && node.Missing {
			processMissing = true
		}
	}
	if !processMissing {
		t.Errorf("expected process node to be flagged as missing")
	}
	if overall == evidenceConfidenceHigh {
		t.Errorf("expected non-high overall confidence with partial evidence, got %s", overall)
	}
	if !strings.Contains(summary, "missing:") {
		t.Errorf("expected summary to call out missing evidence, got %q", summary)
	}
}

func TestBuildEvidencePath_NoInputs(t *testing.T) {
	nodes, _, missing, overall, _ := buildEvidencePath("", "", nil, nil, nil, nil, nil, nil)
	if len(nodes) != 0 {
		t.Fatalf("expected no nodes with no inputs, got %v", nodes)
	}
	if overall != evidenceConfidenceLow {
		t.Errorf("expected low confidence, got %s", overall)
	}
	if len(missing) == 0 {
		t.Errorf("expected missing entries when no inputs")
	}
}

func TestEnrichEvidenceForOperator_NarrativeAndLabels(t *testing.T) {
	process := processRecord{
		EventID:     "p-1",
		EventType:   "aegis.process.started",
		TimestampMS: 1_700_000_000_000,
		DeviceID:    "win-lab-1",
		ProcessGUID: "proc-guid-1",
		PID:         4321,
		Name:        ptrString("ollama"),
		Path:        ptrString("/usr/local/bin/ollama"),
		CommandLine: ptrString("ollama serve"),
	}
	flow := flowRecord{
		EventID:     "f-1",
		EventType:   "aegis.flow.started",
		TimestampMS: 1_700_000_001_000,
		DeviceID:    "win-lab-1",
		FlowID:      "flow-1",
		Protocol:    ptrString("tcp"),
		Direction:   ptrString("outbound"),
		RemoteIP:    ptrString("104.18.6.1"),
		RemotePort:  intPtr(443),
		ProcessGUID: &process.ProcessGUID,
	}
	dnsHit := dnsRecord{
		EventID:     "d-1",
		EventType:   "aegis.dns.observed",
		TimestampMS: 1_700_000_001_500,
		DeviceID:    "win-lab-1",
		Query:       "api.openai.com",
		Answers:     []string{"104.18.6.1"},
		ProcessGUID: &process.ProcessGUID,
	}
	finding := findingRecord{
		EventID:     "find-1",
		EventType:   "aegis.risk_finding.created",
		TimestampMS: 1_700_000_002_000,
		DeviceID:    "win-lab-1",
		Title:       ptrString("AI agent egress"),
		FindingID:   ptrString("find-uuid-1"),
		ProcessGUID: &process.ProcessGUID,
		RiskScore:   72,
		Severity:    ptrString("high"),
	}
	drafts := buildDraftControls("win-lab-1", investigationFilters{ProcessGUID: process.ProcessGUID}, []processRecord{process}, []flowRecord{flow}, []dnsRecord{dnsHit}, []findingRecord{finding})

	nodes, _, missing, overall, _ := buildEvidencePath(
		"win-lab-1", "agent-1", &finding,
		[]processRecord{process}, []flowRecord{flow}, []dnsRecord{dnsHit}, []findingRecord{finding}, drafts,
	)
	enriched, narrative, overallReason := enrichEvidenceForOperator(nodes, missing, overall, "win-lab-1", &finding, []processRecord{process}, []flowRecord{flow}, []dnsRecord{dnsHit}, []findingRecord{finding}, drafts)

	processNode := requireNode(t, enriched, "process")
	if processNode.OperatorLabel == "" {
		t.Fatalf("expected operator_label to be set on process node")
	}
	if processNode.ConfidenceReason == "" {
		t.Fatalf("expected confidence_reason to be set on process node")
	}
	if processNode.RelatedABOMID == "" || processNode.RelatedABOMLabel == "" {
		t.Fatalf("expected related ABOM cross-link on process node, got %+v", processNode)
	}

	dnsNode := requireNode(t, enriched, "dns")
	if dnsNode.RelatedABOMID == "" || dnsNode.RelatedABOMLabel == "" {
		t.Fatalf("expected related ABOM cross-link on DNS node for openai.com, got %+v", dnsNode)
	}

	if narrative.WhatHappened == "" || narrative.WhyItMatters == "" {
		t.Fatalf("expected what-happened and why-it-matters to be filled, got %+v", narrative)
	}
	if len(narrative.WhatWeKnow) == 0 {
		t.Fatalf("expected what-we-know bullets, got %+v", narrative)
	}
	if narrative.RecommendedNextStep == "" {
		t.Fatalf("expected recommended next step")
	}

	if overallReason == "" {
		t.Fatalf("expected overall confidence reason")
	}
}

func TestEnrichEvidenceForOperator_PartialEvidenceMissingCopy(t *testing.T) {
	finding := findingRecord{
		EventID:     "find-2",
		EventType:   "aegis.risk_finding.created",
		TimestampMS: 1,
		DeviceID:    "linux-lab-1",
		Title:       ptrString("Telemetry anomaly"),
		FindingID:   ptrString("find-2"),
		RiskScore:   30,
	}
	nodes, _, missing, overall, _ := buildEvidencePath(
		"linux-lab-1", "", &finding,
		nil, nil, nil, []findingRecord{finding}, nil,
	)
	enriched, narrative, _ := enrichEvidenceForOperator(nodes, missing, overall, "linux-lab-1", &finding, nil, nil, nil, []findingRecord{finding}, nil)

	processNode := requireNode(t, enriched, "process")
	if !processNode.Missing {
		t.Fatalf("expected process to be flagged missing")
	}
	if processNode.ConfidenceReason == "" || !strings.Contains(strings.ToLower(processNode.ConfidenceReason), "no process") {
		t.Fatalf("expected explicit missing reason on process, got %q", processNode.ConfidenceReason)
	}

	if len(narrative.WhatIsMissing) == 0 {
		t.Fatalf("expected what-is-missing copy to be present")
	}
	hasFlowMissing := false
	for _, line := range narrative.WhatIsMissing {
		if strings.Contains(strings.ToLower(line), "flow") {
			hasFlowMissing = true
			break
		}
	}
	if !hasFlowMissing {
		t.Fatalf("expected flow missing explanation in narrative, got %+v", narrative.WhatIsMissing)
	}
}

func requireNode(t *testing.T, nodes []evidenceNode, nodeType string) evidenceNode {
	t.Helper()
	for _, node := range nodes {
		if node.Type == nodeType {
			return node
		}
	}
	t.Fatalf("expected node of type %q, got %+v", nodeType, nodes)
	return evidenceNode{}
}
