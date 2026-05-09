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
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Label      string            `json:"label"`
	Detail     string            `json:"detail,omitempty"`
	EvidenceID string            `json:"evidence_id,omitempty"`
	Confidence string            `json:"confidence"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Missing    bool              `json:"missing,omitempty"`
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

type evidencePathResponse struct {
	OK                 bool                  `json:"ok"`
	GeneratedAtMS      int64                 `json:"generated_at_ms"`
	Subject            evidencePathSubject   `json:"subject"`
	Summary            string                `json:"summary"`
	Nodes              []evidenceNode        `json:"nodes"`
	Edges              []evidenceEdge        `json:"edges"`
	MissingEvidence    []string              `json:"missing_evidence"`
	ConfidenceOverall  string                `json:"confidence_overall"`
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

	resp := evidencePathResponse{
		OK:                true,
		GeneratedAtMS:     time.Now().UnixMilli(),
		Subject:           subject,
		Summary:           summary,
		Nodes:             nodes,
		Edges:             edges,
		MissingEvidence:   missing,
		ConfidenceOverall: overall,
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
