package server

// Evidence-graph investigation path.
//
// WO-PROD-002 turns an isolated finding (or process selection) into a single,
// explainable path: finding → process → parent process → command → flow → DNS
// → endpoint → detection pack → draft control. The aim is for an operator to
// trust *why* the platform thinks something matters, without having to read
// raw JSON.
//
// We deliberately avoid a real graph database. The view we expose is a curated
// sequence of nodes + edges drawn from the existing visibility store and the
// `investigation_query` draft-control synthesizer.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	evidenceConfidenceHigh   = "high"
	evidenceConfidenceMedium = "medium"
	evidenceConfidenceLow    = "low"
)

// evidenceNode is one step along the curated investigation path.
type evidenceNode struct {
	ID                 string            `json:"id"`
	Type               string            `json:"type"`
	Label              string            `json:"label"`
	OperatorLabel      string            `json:"operator_label,omitempty"`
	Detail             string            `json:"detail,omitempty"`
	EvidenceID         string            `json:"evidence_id,omitempty"`
	Confidence         string            `json:"confidence"`
	ConfidenceReason   string            `json:"confidence_reason,omitempty"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	Missing            bool              `json:"missing,omitempty"`
	RelatedABOMID      string            `json:"related_abom_id,omitempty"`
	RelatedABOMLabel   string            `json:"related_abom_label,omitempty"`
}

// evidenceEdge connects two nodes in the curated path. Edges follow the
// taxonomy described in the WO: launched, resolved, connected, matched,
// observed_on, supports_control.
type evidenceEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label"`
	Confidence string `json:"confidence"`
}

type evidencePathSubject struct {
	Type     string `json:"type"`
	ID       string `json:"id,omitempty"`
	DeviceID string `json:"device_id,omitempty"`
	AgentID  string `json:"agent_id,omitempty"`
}

// evidenceNarrative is the plain-language explanation block surfaced at the
// top of the UI. WO-GROWTH-002 requires we tell the operator: what happened,
// why it matters, what we know with confidence, what is missing, and what to
// do next — without making them read JSON.
type evidenceNarrative struct {
	WhatHappened        string   `json:"what_happened"`
	WhyItMatters        string   `json:"why_it_matters"`
	WhatWeKnow          []string `json:"what_we_know"`
	WhatIsMissing       []string `json:"what_is_missing"`
	RecommendedNextStep string   `json:"recommended_next_step"`
}

type evidencePathResponse struct {
	OK                 bool                  `json:"ok"`
	GeneratedAtMS      int64                 `json:"generated_at_ms"`
	Subject            evidencePathSubject   `json:"subject"`
	Summary            string                `json:"summary"`
	Narrative          evidenceNarrative     `json:"narrative"`
	Nodes              []evidenceNode        `json:"nodes"`
	Edges              []evidenceEdge        `json:"edges"`
	MissingEvidence    []string              `json:"missing_evidence"`
	ConfidenceOverall  string                `json:"confidence_overall"`
	ConfidenceReason   string                `json:"confidence_reason,omitempty"`
	DraftControls      []draftControlRecord  `json:"draft_controls"`
	RawProcesses       []processRecord       `json:"raw_processes"`
	RawFlows           []flowRecord          `json:"raw_flows"`
	RawDNS             []dnsRecord           `json:"raw_dns"`
	RawFindings        []findingRecord       `json:"raw_findings"`
	EmptyHelp          string                `json:"empty_help,omitempty"`
}

func (s *IngestServer) handleEvidencePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()
	findingID := strings.TrimSpace(q.Get("finding_id"))
	deviceID := strings.TrimSpace(q.Get("device_id"))
	processGUID := strings.TrimSpace(q.Get("process_guid"))
	agentID := strings.TrimSpace(q.Get("agent_id"))

	if findingID == "" && deviceID == "" {
		http.Error(w, "finding_id or device_id is required", http.StatusBadRequest)
		return
	}

	// If a finding id was given, locate the finding to anchor the device.
	var anchorFinding *findingRecord
	if findingID != "" {
		anchor, err := s.findFindingByID(ctx, findingID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		if anchor != nil {
			anchorFinding = anchor
			if deviceID == "" {
				deviceID = anchor.DeviceID
			}
			if agentID == "" {
				agentID = anchor.AgentID
			}
			if processGUID == "" && anchor.ProcessGUID != nil && *anchor.ProcessGUID != "" {
				processGUID = *anchor.ProcessGUID
			}
		}
	}

	if deviceID == "" {
		http.Error(w, "could not resolve device for evidence path", http.StatusNotFound)
		return
	}

	limit := 60
	processes, flows, dnsRows, findings, drafts, err := s.collectEvidencePathInputs(ctx, deviceID, agentID, processGUID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// If we have an anchor finding but it isn't in the recent findings list,
	// prepend it so the path is built from the right one.
	if anchorFinding != nil {
		anchorPresent := false
		for _, f := range findings {
			if f.EventID == anchorFinding.EventID {
				anchorPresent = true
				break
			}
		}
		if !anchorPresent {
			findings = append([]findingRecord{*anchorFinding}, findings...)
		}
	}

	subject := evidencePathSubject{Type: "endpoint", ID: deviceID, DeviceID: deviceID, AgentID: agentID}
	if anchorFinding != nil {
		subject.Type = "finding"
		subject.ID = stringValue(anchorFinding.FindingID)
		if subject.ID == "" {
			subject.ID = anchorFinding.EventID
		}
	} else if processGUID != "" {
		subject.Type = "process"
		subject.ID = processGUID
	}

	nodes, edges, missing, overall, summary := buildEvidencePath(deviceID, agentID, anchorFinding, processes, flows, dnsRows, findings, drafts)
	enriched, narrative, overallReason := enrichEvidenceForOperator(nodes, missing, overall, deviceID, anchorFinding, processes, flows, dnsRows, findings, drafts)

	resp := evidencePathResponse{
		OK:                true,
		GeneratedAtMS:     time.Now().UnixMilli(),
		Subject:           subject,
		Summary:           summary,
		Narrative:         narrative,
		Nodes:             enriched,
		Edges:             edges,
		MissingEvidence:   missing,
		ConfidenceOverall: overall,
		ConfidenceReason:  overallReason,
		DraftControls:     drafts,
		RawProcesses:      processes,
		RawFlows:          flows,
		RawDNS:            dnsRows,
		RawFindings:       findings,
	}
	if len(nodes) == 0 {
		resp.EmptyHelp = "No process, flow, DNS, or finding evidence yet for this device. The path populates as the agent reports telemetry."
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *IngestServer) findFindingByID(ctx context.Context, findingID string) (*findingRecord, error) {
	events, err := s.store.Query(ctx, visibilityQueryFilter{Limit: maxVisibilityQueryLimit})
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		if !isFindingEventType(event.EventType) {
			continue
		}
		record, err := event.toFindingRecord()
		if err != nil {
			continue
		}
		if stringValue(record.FindingID) == findingID || record.EventID == findingID {
			clone := record
			return &clone, nil
		}
	}
	return nil, nil
}

func (s *IngestServer) collectEvidencePathInputs(ctx context.Context, deviceID, agentID, processGUID string, limit int) ([]processRecord, []flowRecord, []dnsRecord, []findingRecord, []draftControlRecord, error) {
	events, err := s.store.Query(ctx, visibilityQueryFilter{
		DeviceID: deviceID,
		AgentID:  agentID,
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	processes := make([]processRecord, 0, limit)
	flows := make([]flowRecord, 0, limit)
	dnsRows := make([]dnsRecord, 0, limit)
	findings := make([]findingRecord, 0, limit)
	for _, event := range events {
		switch event.EventType {
		case "aegis.process.started", "aegis.process.ended":
			rec, err := event.toProcessRecord()
			if err != nil {
				continue
			}
			if processGUID != "" && rec.ProcessGUID != processGUID {
				continue
			}
			if len(processes) < limit {
				processes = append(processes, rec)
			}
		case "aegis.flow.started", "aegis.flow.ended":
			rec, err := event.toFlowRecord()
			if err != nil {
				continue
			}
			if processGUID != "" && stringValue(rec.ProcessGUID) != processGUID && stringValue(rec.ProcessGUID) != "" {
				continue
			}
			if len(flows) < limit {
				flows = append(flows, rec)
			}
		case "aegis.dns.observed":
			rec, err := event.toDNSRecord()
			if err != nil {
				continue
			}
			if processGUID != "" && stringValue(rec.ProcessGUID) != processGUID && stringValue(rec.ProcessGUID) != "" {
				continue
			}
			if len(dnsRows) < limit {
				dnsRows = append(dnsRows, rec)
			}
		case "aegis.agent.detected", "aegis.risk_finding.created":
			rec, err := event.toFindingRecord()
			if err != nil {
				continue
			}
			if processGUID != "" && stringValue(rec.ProcessGUID) != processGUID && stringValue(rec.ProcessGUID) != "" {
				continue
			}
			if len(findings) < limit {
				findings = append(findings, rec)
			}
		}
	}

	filters := investigationFilters{ProcessGUID: processGUID}
	drafts := buildDraftControls(deviceID, filters, processes, flows, dnsRows, findings)
	return processes, flows, dnsRows, findings, drafts, nil
}

// buildEvidencePath is the pure path-building function used by the handler and
// tests. It returns the curated nodes, edges, list of missing evidence labels,
// the overall confidence, and a one-line operator summary.
func buildEvidencePath(
	deviceID string,
	agentID string,
	anchorFinding *findingRecord,
	processes []processRecord,
	flows []flowRecord,
	dnsRows []dnsRecord,
	findings []findingRecord,
	drafts []draftControlRecord,
) ([]evidenceNode, []evidenceEdge, []string, string, string) {
	if deviceID == "" && anchorFinding == nil && len(processes) == 0 && len(flows) == 0 && len(findings) == 0 {
		return nil, nil, []string{"endpoint", "process", "flow", "dns", "finding"}, evidenceConfidenceLow, ""
	}

	missing := map[string]struct{}{}
	mark := func(name string) { missing[name] = struct{}{} }

	// Anchor finding: prefer explicit anchor, fall back to highest-risk finding.
	finding := anchorFinding
	if finding == nil && len(findings) > 0 {
		highest := findings[0]
		for _, f := range findings[1:] {
			if f.RiskScore > highest.RiskScore {
				highest = f
			}
		}
		clone := highest
		finding = &clone
	}
	if finding == nil {
		mark("finding")
	}

	// Choose the most relevant process: prefer the one that matches the
	// finding's process_guid, otherwise the most recent observed process.
	var process *processRecord
	if finding != nil && finding.ProcessGUID != nil && *finding.ProcessGUID != "" {
		for i, p := range processes {
			if p.ProcessGUID == *finding.ProcessGUID {
				process = &processes[i]
				break
			}
		}
	}
	if process == nil && len(processes) > 0 {
		// Sort newest-first then pick the head.
		sorted := append([]processRecord{}, processes...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
		process = &sorted[0]
	}
	if process == nil {
		mark("process")
	}

	// Parent process is "missing" when we don't have parent_process_guid set.
	hasParent := false
	if process != nil && process.ParentProcessGUID != nil && *process.ParentProcessGUID != "" {
		hasParent = true
	}
	if !hasParent {
		mark("parent_process")
	}

	// Flow: pick a flow tied to the chosen process, else first observed.
	var flow *flowRecord
	if process != nil {
		for i, f := range flows {
			if stringValue(f.ProcessGUID) == process.ProcessGUID {
				flow = &flows[i]
				break
			}
		}
	}
	if flow == nil && len(flows) > 0 {
		sorted := append([]flowRecord{}, flows...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
		flow = &sorted[0]
	}
	if flow == nil {
		mark("flow")
	}

	// DNS: prefer one tied to the process, else first relevant entry.
	var dnsHit *dnsRecord
	if process != nil {
		for i, d := range dnsRows {
			if stringValue(d.ProcessGUID) == process.ProcessGUID {
				dnsHit = &dnsRows[i]
				break
			}
		}
	}
	if dnsHit == nil && len(dnsRows) > 0 {
		sorted := append([]dnsRecord{}, dnsRows...)
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
		dnsHit = &sorted[0]
	}
	if dnsHit == nil {
		mark("dns")
	}

	// Now build curated nodes + edges.
	nodes := []evidenceNode{}
	edges := []evidenceEdge{}

	endpointID := "endpoint"
	endpointConfidence := evidenceConfidenceHigh
	if deviceID == "" {
		endpointConfidence = evidenceConfidenceLow
	}
	nodes = append(nodes, evidenceNode{
		ID:         endpointID,
		Type:       "endpoint",
		Label:      defaultStringEv(deviceID, "unknown endpoint"),
		Detail:     fmt.Sprintf("Agent %s", defaultStringEv(agentID, "unknown")),
		EvidenceID: fmt.Sprintf("device:%s", deviceID),
		Confidence: endpointConfidence,
	})

	parentID := "parent_process"
	if hasParent {
		nodes = append(nodes, evidenceNode{
			ID:         parentID,
			Type:       "parent_process",
			Label:      fmt.Sprintf("parent guid %s", *process.ParentProcessGUID),
			EvidenceID: fmt.Sprintf("process_guid:%s", *process.ParentProcessGUID),
			Confidence: evidenceConfidenceMedium,
		})
	} else {
		nodes = append(nodes, evidenceNode{
			ID:         parentID,
			Type:       "parent_process",
			Label:      "parent process not linked",
			Confidence: evidenceConfidenceLow,
			Missing:    true,
		})
	}

	processID := "process"
	if process != nil {
		attrs := map[string]string{}
		if process.CommandLine != nil && *process.CommandLine != "" {
			attrs["command_line"] = *process.CommandLine
		}
		if process.User != nil && *process.User != "" {
			attrs["user"] = *process.User
		}
		nodes = append(nodes, evidenceNode{
			ID:         processID,
			Type:       "process",
			Label:      fmt.Sprintf("%s pid %d", defaultStringEv(stringValue(process.Name), "process"), process.PID),
			Detail:     stringValue(process.Path),
			EvidenceID: fmt.Sprintf("process_guid:%s", process.ProcessGUID),
			Confidence: evidenceConfidenceHigh,
			Attributes: attrs,
		})
		edges = append(edges, evidenceEdge{From: parentID, To: processID, Label: "launched", Confidence: ifElse(hasParent, evidenceConfidenceMedium, evidenceConfidenceLow)})
		edges = append(edges, evidenceEdge{From: processID, To: endpointID, Label: "observed_on", Confidence: evidenceConfidenceHigh})
	} else {
		nodes = append(nodes, evidenceNode{
			ID:         processID,
			Type:       "process",
			Label:      "no process linked",
			Confidence: evidenceConfidenceLow,
			Missing:    true,
		})
		edges = append(edges, evidenceEdge{From: parentID, To: processID, Label: "launched", Confidence: evidenceConfidenceLow})
		edges = append(edges, evidenceEdge{From: processID, To: endpointID, Label: "observed_on", Confidence: evidenceConfidenceLow})
	}

	flowID := "flow"
	if flow != nil {
		dest := socketText(flow.RemoteIP, flow.RemotePort)
		nodes = append(nodes, evidenceNode{
			ID:         flowID,
			Type:       "flow",
			Label:      fmt.Sprintf("%s %s to %s", defaultStringEv(stringValue(flow.Protocol), "tcp"), defaultStringEv(stringValue(flow.Direction), "outbound"), dest),
			Detail:     defaultStringEv(stringValue(flow.RemoteHostname), dest),
			EvidenceID: fmt.Sprintf("flow_id:%s", flow.FlowID),
			Confidence: evidenceConfidenceHigh,
		})
		edges = append(edges, evidenceEdge{From: processID, To: flowID, Label: "connected", Confidence: ifElse(process != nil, evidenceConfidenceMedium, evidenceConfidenceLow)})
	} else {
		nodes = append(nodes, evidenceNode{
			ID:         flowID,
			Type:       "flow",
			Label:      "no network flow linked",
			Confidence: evidenceConfidenceLow,
			Missing:    true,
		})
		edges = append(edges, evidenceEdge{From: processID, To: flowID, Label: "connected", Confidence: evidenceConfidenceLow})
	}

	dnsID := "dns"
	if dnsHit != nil {
		nodes = append(nodes, evidenceNode{
			ID:         dnsID,
			Type:       "dns",
			Label:      dnsHit.Query,
			Detail:     strings.Join(dnsHit.Answers, ", "),
			EvidenceID: fmt.Sprintf("event:%s", dnsHit.EventID),
			Confidence: evidenceConfidenceMedium,
		})
		edges = append(edges, evidenceEdge{From: dnsID, To: flowID, Label: "resolved", Confidence: ifElse(flow != nil, evidenceConfidenceMedium, evidenceConfidenceLow)})
	} else {
		nodes = append(nodes, evidenceNode{
			ID:         dnsID,
			Type:       "dns",
			Label:      "no DNS evidence linked",
			Confidence: evidenceConfidenceLow,
			Missing:    true,
		})
		edges = append(edges, evidenceEdge{From: dnsID, To: flowID, Label: "resolved", Confidence: evidenceConfidenceLow})
	}

	findingID := "finding"
	if finding != nil {
		title := stringValue(finding.Title)
		if title == "" {
			title = stringValue(finding.Classification)
		}
		if title == "" {
			title = finding.EventType
		}
		attrs := map[string]string{
			"risk_score": fmt.Sprintf("%d", finding.RiskScore),
		}
		if finding.Severity != nil && *finding.Severity != "" {
			attrs["severity"] = *finding.Severity
		}
		nodes = append(nodes, evidenceNode{
			ID:         findingID,
			Type:       "finding",
			Label:      title,
			Detail:     stringValue(finding.Description),
			EvidenceID: fmt.Sprintf("finding:%s", defaultStringEv(stringValue(finding.FindingID), finding.EventID)),
			Confidence: ifElse(finding.RiskScore >= 60, evidenceConfidenceHigh, evidenceConfidenceMedium),
			Attributes: attrs,
		})
		edges = append(edges, evidenceEdge{From: findingID, To: processID, Label: "matched", Confidence: ifElse(process != nil && finding.ProcessGUID != nil && *finding.ProcessGUID == process.ProcessGUID, evidenceConfidenceHigh, evidenceConfidenceMedium)})
	} else {
		nodes = append(nodes, evidenceNode{
			ID:         findingID,
			Type:       "finding",
			Label:      "no finding anchored",
			Confidence: evidenceConfidenceLow,
			Missing:    true,
		})
		edges = append(edges, evidenceEdge{From: findingID, To: processID, Label: "matched", Confidence: evidenceConfidenceLow})
	}

	// Detection pack node: derive from the finding's detection_id when present.
	if finding != nil && finding.DetectionID != nil && *finding.DetectionID != "" {
		dpID := "detection_pack"
		nodes = append(nodes, evidenceNode{
			ID:         dpID,
			Type:       "detection_pack",
			Label:      fmt.Sprintf("Detection %s", *finding.DetectionID),
			EvidenceID: fmt.Sprintf("detection:%s", *finding.DetectionID),
			Confidence: evidenceConfidenceMedium,
		})
		edges = append(edges, evidenceEdge{From: dpID, To: findingID, Label: "matched", Confidence: evidenceConfidenceMedium})
	} else {
		mark("detection_pack")
	}

	// Draft control nodes — at most two, attached via supports_control.
	for i, draft := range drafts {
		if i >= 2 {
			break
		}
		dcID := fmt.Sprintf("draft_control_%d", i)
		nodes = append(nodes, evidenceNode{
			ID:         dcID,
			Type:       "draft_control",
			Label:      draft.Title,
			Detail:     draft.Action,
			EvidenceID: fmt.Sprintf("draft:%s", draft.ControlID),
			Confidence: evidenceConfidenceMedium,
		})
		edges = append(edges, evidenceEdge{From: findingID, To: dcID, Label: "supports_control", Confidence: ifElse(finding != nil, evidenceConfidenceMedium, evidenceConfidenceLow)})
	}

	overall := overallConfidence(nodes)
	missingList := sortedKeys(missing)
	summary := buildSummary(deviceID, finding, process, flow, dnsHit, drafts, missingList)

	return nodes, edges, missingList, overall, summary
}

func overallConfidence(nodes []evidenceNode) string {
	high := 0
	medium := 0
	low := 0
	for _, node := range nodes {
		switch node.Confidence {
		case evidenceConfidenceHigh:
			high++
		case evidenceConfidenceMedium:
			medium++
		default:
			low++
		}
	}
	if high > 0 && low <= 1 && medium >= 1 {
		return evidenceConfidenceHigh
	}
	if (high + medium) >= 3 {
		return evidenceConfidenceMedium
	}
	if (high + medium) >= 1 {
		return evidenceConfidenceLow
	}
	return evidenceConfidenceLow
}

func buildSummary(deviceID string, finding *findingRecord, process *processRecord, flow *flowRecord, dns *dnsRecord, drafts []draftControlRecord, missing []string) string {
	parts := []string{}
	if finding != nil {
		title := stringValue(finding.Title)
		if title == "" {
			title = stringValue(finding.Classification)
		}
		if title == "" {
			title = finding.EventType
		}
		parts = append(parts, fmt.Sprintf("Finding %q (risk %d) on %s", title, finding.RiskScore, defaultStringEv(deviceID, "unknown endpoint")))
	} else {
		parts = append(parts, fmt.Sprintf("Activity on %s", defaultStringEv(deviceID, "unknown endpoint")))
	}
	if process != nil {
		parts = append(parts, fmt.Sprintf("linked to process %s pid %d", defaultStringEv(stringValue(process.Name), "process"), process.PID))
	}
	if flow != nil {
		parts = append(parts, fmt.Sprintf("connecting to %s", socketText(flow.RemoteIP, flow.RemotePort)))
	}
	if dns != nil {
		parts = append(parts, fmt.Sprintf("via DNS %s", dns.Query))
	}
	if len(drafts) > 0 {
		parts = append(parts, fmt.Sprintf("supports draft control %s", drafts[0].ControlID))
	}
	summary := strings.Join(parts, " · ")
	if len(missing) > 0 {
		summary += fmt.Sprintf(" · missing: %s", strings.Join(missing, ", "))
	}
	return summary
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ifElse(condition bool, a, b string) string {
	if condition {
		return a
	}
	return b
}

func defaultStringEv(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// enrichEvidenceForOperator turns the curated nodes into a plain-language
// explanation block and adds operator labels, confidence reasons, and ABOM
// cross-links to each node.
func enrichEvidenceForOperator(
	nodes []evidenceNode,
	missing []string,
	overallConfidence string,
	deviceID string,
	finding *findingRecord,
	processes []processRecord,
	flows []flowRecord,
	dnsRows []dnsRecord,
	findings []findingRecord,
	drafts []draftControlRecord,
) ([]evidenceNode, evidenceNarrative, string) {
	enriched := make([]evidenceNode, len(nodes))
	copy(enriched, nodes)

	missingSet := map[string]struct{}{}
	for _, m := range missing {
		missingSet[m] = struct{}{}
	}

	leadProcess := pickLeadProcess(finding, processes)
	leadFlow := pickLeadFlow(leadProcess, flows)
	leadDNS := pickLeadDNS(leadProcess, dnsRows)
	leadFinding := finding
	if leadFinding == nil && len(findings) > 0 {
		leadFinding = &findings[0]
	}

	for i := range enriched {
		switch enriched[i].Type {
		case "endpoint":
			enriched[i].OperatorLabel = "Computer where this happened"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No endpoint id resolved — cannot anchor evidence to a single device."
			} else {
				enriched[i].ConfidenceReason = "Endpoint id confirmed by direct telemetry from the agent."
			}
		case "parent_process":
			enriched[i].OperatorLabel = "Program that started it"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No parent process linked. The agent did not report parent_process_guid for this run."
			} else {
				enriched[i].ConfidenceReason = "Parent process id linked through process telemetry."
			}
		case "process":
			enriched[i].OperatorLabel = "Program that ran"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No process record linked. Without a process, this finding cannot be tied to a binary or command."
			} else if leadProcess != nil {
				enriched[i].ConfidenceReason = fmt.Sprintf(
					"Process %q (pid %d) observed by the agent. command_line and user attributes are present.",
					defaultStringEv(stringValue(leadProcess.Name), "process"),
					leadProcess.PID,
				)
				if abomID, label := relatedABOMForProcess(leadProcess); abomID != "" {
					enriched[i].RelatedABOMID = abomID
					enriched[i].RelatedABOMLabel = label
				}
			}
		case "flow":
			enriched[i].OperatorLabel = "Where it talked to"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No matching network flow linked. Either the program made no outbound traffic or the flow telemetry is not yet aggregated."
			} else if leadFlow != nil {
				enriched[i].ConfidenceReason = fmt.Sprintf(
					"Flow telemetry confirms %s connecting to %s.",
					defaultStringEv(stringValue(leadFlow.Direction), "outbound"),
					socketText(leadFlow.RemoteIP, leadFlow.RemotePort),
				)
			}
		case "dns":
			enriched[i].OperatorLabel = "How it resolved the destination"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No DNS lookup linked. The flow may be IP-only or DNS telemetry is not yet captured for this window."
			} else if leadDNS != nil {
				enriched[i].ConfidenceReason = fmt.Sprintf(
					"DNS query %q resolved to %s — confirms the destination domain in plain text.",
					leadDNS.Query,
					defaultStringEv(strings.Join(leadDNS.Answers, ", "), "no answers recorded"),
				)
				if abomID, label := relatedABOMForDNS(leadDNS); abomID != "" {
					enriched[i].RelatedABOMID = abomID
					enriched[i].RelatedABOMLabel = label
				}
			}
		case "finding":
			enriched[i].OperatorLabel = "Why we flagged this"
			if enriched[i].Missing {
				enriched[i].ConfidenceReason = "No finding anchored. Without a finding, the path is observation only — no detection rule fired."
			} else if leadFinding != nil {
				enriched[i].ConfidenceReason = fmt.Sprintf(
					"Detection scored %d (severity %s). The reason this row exists in the path.",
					leadFinding.RiskScore,
					defaultStringEv(stringValue(leadFinding.Severity), "unknown"),
				)
			}
		case "detection_pack":
			enriched[i].OperatorLabel = "Detection that matched"
			enriched[i].ConfidenceReason = "Detection id supplied by the finding — links this evidence to the rule that fired."
		case "draft_control":
			enriched[i].OperatorLabel = "Suggested observe-only response"
			enriched[i].ConfidenceReason = "Synthesized from this evidence by the finding-to-control workflow. No enforcement until you promote it."
		}
	}

	narrative := buildEvidenceNarrativeBlock(deviceID, leadFinding, leadProcess, leadFlow, leadDNS, drafts, missing, missingSet)
	overallReason := buildOverallConfidenceReason(overallConfidence, missing, leadProcess, leadFlow, leadDNS, leadFinding)

	return enriched, narrative, overallReason
}

func pickLeadProcess(finding *findingRecord, processes []processRecord) *processRecord {
	if finding != nil && finding.ProcessGUID != nil && *finding.ProcessGUID != "" {
		for i := range processes {
			if processes[i].ProcessGUID == *finding.ProcessGUID {
				return &processes[i]
			}
		}
	}
	if len(processes) == 0 {
		return nil
	}
	sorted := append([]processRecord{}, processes...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
	return &sorted[0]
}

func pickLeadFlow(process *processRecord, flows []flowRecord) *flowRecord {
	if process != nil {
		for i := range flows {
			if stringValue(flows[i].ProcessGUID) == process.ProcessGUID {
				return &flows[i]
			}
		}
	}
	if len(flows) == 0 {
		return nil
	}
	sorted := append([]flowRecord{}, flows...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
	return &sorted[0]
}

func pickLeadDNS(process *processRecord, rows []dnsRecord) *dnsRecord {
	if process != nil {
		for i := range rows {
			if stringValue(rows[i].ProcessGUID) == process.ProcessGUID {
				return &rows[i]
			}
		}
	}
	if len(rows) == 0 {
		return nil
	}
	sorted := append([]dnsRecord{}, rows...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimestampMS > sorted[j].TimestampMS })
	return &sorted[0]
}

func relatedABOMForProcess(process *processRecord) (string, string) {
	if process == nil {
		return "", ""
	}
	name := strings.ToLower(stringValue(process.Name))
	path := strings.ToLower(stringValue(process.Path))
	cmd := strings.ToLower(stringValue(process.CommandLine))
	hay := name + " " + path + " " + cmd
	switch {
	case abomMCPPattern.MatchString(hay):
		return abomItemID(abomCategoryMCPEndpoint, productNameForProcess(stringValue(process.Name), stringValue(process.Path), "MCP server")), "MCP endpoint"
	case abomLocalRuntimePattern.MatchString(hay):
		return abomItemID(abomCategoryLocalModelRuntime, productNameForProcess(stringValue(process.Name), stringValue(process.Path), "Local model runtime")), "Local model runtime"
	case abomDesktopAppPattern.MatchString(hay):
		return abomItemID(abomCategoryAIDesktopApp, productNameForProcess(stringValue(process.Name), stringValue(process.Path), "AI desktop app")), "AI desktop app"
	case abomCodingAgentPattern.MatchString(hay):
		return abomItemID(abomCategoryCodingAgent, productNameForProcess(stringValue(process.Name), stringValue(process.Path), "Coding agent")), "Coding agent"
	case abomCLIAgentPattern.MatchString(hay):
		return abomItemID(abomCategoryCLIAgent, productNameForProcess(stringValue(process.Name), stringValue(process.Path), "AI CLI agent")), "CLI agent"
	}
	return "", ""
}

func relatedABOMForDNS(record *dnsRecord) (string, string) {
	if record == nil {
		return "", ""
	}
	host := strings.ToLower(record.Query)
	if abomGatewayDomainPattern.MatchString(host) {
		return abomItemID(abomCategoryModelGateway, gatewayProductForHost(record.Query)), "Model gateway"
	}
	return "", ""
}

func buildEvidenceNarrativeBlock(
	deviceID string,
	finding *findingRecord,
	process *processRecord,
	flow *flowRecord,
	dnsHit *dnsRecord,
	drafts []draftControlRecord,
	missing []string,
	missingSet map[string]struct{},
) evidenceNarrative {
	what := narrativeWhatHappened(deviceID, finding, process, flow, dnsHit)
	why := narrativeWhyItMatters(finding, process)

	whatWeKnow := []string{}
	if finding != nil {
		whatWeKnow = append(whatWeKnow, fmt.Sprintf("Detection fired with risk score %d.", finding.RiskScore))
	}
	if process != nil {
		whatWeKnow = append(whatWeKnow, fmt.Sprintf("Program %q ran on %s.", defaultStringEv(stringValue(process.Name), "process"), defaultStringEv(deviceID, "this endpoint")))
	}
	if flow != nil {
		whatWeKnow = append(whatWeKnow, fmt.Sprintf("Outbound flow to %s recorded.", socketText(flow.RemoteIP, flow.RemotePort)))
	}
	if dnsHit != nil {
		whatWeKnow = append(whatWeKnow, fmt.Sprintf("DNS query %q resolved to %s.", dnsHit.Query, defaultStringEv(strings.Join(dnsHit.Answers, ", "), "no answers")))
	}
	if len(whatWeKnow) == 0 {
		whatWeKnow = append(whatWeKnow, "Telemetry has not yet attached strong evidence to this subject.")
	}

	whatIsMissing := []string{}
	for _, m := range missing {
		whatIsMissing = append(whatIsMissing, missingExplanation(m))
	}
	if _, ok := missingSet["parent_process"]; ok && process != nil {
		whatIsMissing = append(whatIsMissing, "Parent process is unknown — verify the agent reports parent_process_guid for richer attribution.")
	}
	if len(whatIsMissing) == 0 {
		whatIsMissing = append(whatIsMissing, "Nothing critical missing in this window — confidence is not bounded by gaps.")
	}

	next := narrativeNextStep(finding, process, drafts, missing)

	return evidenceNarrative{
		WhatHappened:        what,
		WhyItMatters:        why,
		WhatWeKnow:          whatWeKnow,
		WhatIsMissing:       whatIsMissing,
		RecommendedNextStep: next,
	}
}

func narrativeWhatHappened(deviceID string, finding *findingRecord, process *processRecord, flow *flowRecord, dnsHit *dnsRecord) string {
	parts := []string{}
	subject := defaultStringEv(deviceID, "an endpoint")
	if finding != nil {
		title := stringValue(finding.Title)
		if title == "" {
			title = stringValue(finding.Classification)
		}
		if title == "" {
			title = "a finding"
		}
		parts = append(parts, fmt.Sprintf("On %s, %s.", subject, strings.ToLower(title)))
	} else {
		parts = append(parts, fmt.Sprintf("Activity observed on %s.", subject))
	}
	if process != nil {
		parts = append(parts, fmt.Sprintf("Program %q (pid %d) ran the activity.", defaultStringEv(stringValue(process.Name), "process"), process.PID))
	}
	if flow != nil {
		parts = append(parts, fmt.Sprintf("It talked outbound to %s.", socketText(flow.RemoteIP, flow.RemotePort)))
	}
	if dnsHit != nil {
		parts = append(parts, fmt.Sprintf("DNS lookup %q confirms the destination domain.", dnsHit.Query))
	}
	return strings.Join(parts, " ")
}

func narrativeWhyItMatters(finding *findingRecord, process *processRecord) string {
	if finding != nil {
		title := strings.ToLower(stringValue(finding.Title))
		if strings.Contains(title, "ai") || strings.Contains(title, "agent") || strings.Contains(title, "model") {
			return "An AI-related signal is interesting because it lives outside the normal SaaS perimeter — programs talking to model providers can carry secrets and code in real time."
		}
		if process != nil && abomLocalRuntimePattern.MatchString(strings.ToLower(stringValue(process.Name))) {
			return "Local model runtimes can expose APIs to other devices on the same network without authentication. The risk is lateral abuse, not crash damage."
		}
		return fmt.Sprintf("A detection at risk %d is enough to warrant operator attention before drift becomes the norm.", finding.RiskScore)
	}
	if process != nil {
		return "Without a finding this is observation only, but the program path provides starting context for review."
	}
	return "Without a finding or a strong process link, this row is informational only."
}

func narrativeNextStep(finding *findingRecord, process *processRecord, drafts []draftControlRecord, missing []string) string {
	if len(missing) > 2 {
		return "Confirm telemetry coverage on this endpoint (process, flow, DNS) before treating the path as authoritative. The missing-evidence list points to the gaps."
	}
	if len(drafts) > 0 {
		return fmt.Sprintf("Open the finding-to-control designer with this finding and review draft control %s as an observe-only response.", drafts[0].ControlID)
	}
	if finding != nil && process != nil {
		return "Open the finding-to-control designer with this finding to draft an observe-only response with scope and rollback."
	}
	if process != nil {
		return "Cross-link the program to the Agent Bill of Materials. If it is not in ABOM yet, consider whether it should be sanctioned or reviewed."
	}
	return "Investigate adjacent telemetry on this endpoint to build enough evidence for a draft control."
}

func missingExplanation(name string) string {
	switch name {
	case "endpoint":
		return "No endpoint id linked — open the device record manually to verify telemetry."
	case "process":
		return "No process linked — without a process this row cannot be tied to a binary or command."
	case "parent_process":
		return "Parent process not linked — the spawning chain is incomplete."
	case "flow":
		return "No matching outbound flow — the program may be local-only or flow telemetry is missing."
	case "dns":
		return "No DNS lookup linked — confirm DNS collector status before treating the destination as ground truth."
	case "finding":
		return "No finding anchored — the path is observation only, no detection has fired."
	case "detection_pack":
		return "Finding does not reference a detection id — pack lineage cannot be verified."
	}
	return name
}

func buildOverallConfidenceReason(overall string, missing []string, process *processRecord, flow *flowRecord, dnsHit *dnsRecord, finding *findingRecord) string {
	have := []string{}
	if finding != nil {
		have = append(have, "finding")
	}
	if process != nil {
		have = append(have, "process")
	}
	if flow != nil {
		have = append(have, "flow")
	}
	if dnsHit != nil {
		have = append(have, "dns")
	}
	switch overall {
	case evidenceConfidenceHigh:
		return fmt.Sprintf("High confidence: %s aligned without major gaps.", strings.Join(have, ", "))
	case evidenceConfidenceMedium:
		if len(missing) > 0 {
			return fmt.Sprintf("Medium confidence: anchored by %s but missing %s.", strings.Join(have, ", "), strings.Join(missing, ", "))
		}
		return fmt.Sprintf("Medium confidence: %s present but evidence is shallow.", strings.Join(have, ", "))
	default:
		return fmt.Sprintf("Low confidence: limited anchors (%s) and missing %s.", strings.Join(have, ", "), strings.Join(missing, ", "))
	}
}
