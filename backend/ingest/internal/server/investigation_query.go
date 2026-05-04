package server

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"strings"
)

type investigationResponse struct {
	OK        bool                 `json:"ok"`
	DeviceID  string               `json:"device_id"`
	AgentID   string               `json:"agent_id,omitempty"`
	Filters   investigationFilters `json:"filters"`
	Counts    investigationCounts  `json:"counts"`
	Processes []processRecord      `json:"processes"`
	Flows     []flowRecord         `json:"flows"`
	DNS       []dnsRecord          `json:"dns"`
	Findings  []findingRecord      `json:"findings"`
	Drafts    []draftControlRecord `json:"draft_controls"`
}

type investigationFilters struct {
	ProcessGUID string `json:"process_guid,omitempty"`
	PID         *int   `json:"pid,omitempty"`
}

type investigationCounts struct {
	Processes int `json:"processes"`
	Flows     int `json:"flows"`
	DNS       int `json:"dns"`
	Findings  int `json:"findings"`
	Drafts    int `json:"draft_controls"`
}

type draftControlRecord struct {
	ControlID   string   `json:"control_id"`
	Title       string   `json:"title"`
	Mode        string   `json:"mode"`
	Status      string   `json:"status"`
	Action      string   `json:"action"`
	Target      string   `json:"target"`
	Scope       string   `json:"scope"`
	Reason      string   `json:"reason"`
	Evidence    []string `json:"evidence"`
	BlastRadius []string `json:"blast_radius"`
	Rollback    []string `json:"rollback"`
}

func (s *IngestServer) handleVisibilityInvestigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	limit, err := parseQueryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pid, hasPID, err := parseOptionalInt(r.URL.Query().Get("pid"), "pid")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	processGUID := r.URL.Query().Get("process_guid")
	agentID := r.URL.Query().Get("agent_id")

	investigation, err := s.collectInvestigation(r, deviceID, agentID, processGUID, pid, hasPID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, investigation)
}

func (s *IngestServer) collectInvestigation(r *http.Request, deviceID, agentID, processGUID string, pid int, hasPID bool, limit int) (investigationResponse, error) {
	events, err := s.store.Query(r.Context(), visibilityQueryFilter{
		DeviceID: deviceID,
		AgentID:  agentID,
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		return investigationResponse{}, err
	}

	processes := make([]processRecord, 0, limit)
	flows := make([]flowRecord, 0, limit)
	dns := make([]dnsRecord, 0, limit)
	findings := make([]findingRecord, 0, limit)

	for _, event := range events {
		switch event.EventType {
		case "aegis.process.started", "aegis.process.ended":
			record, err := event.toProcessRecord()
			if err != nil {
				s.logger.Warn("Skipping malformed process visibility event",
					"event_id", event.EventID,
					"event_type", event.EventType,
					"error", err)
				continue
			}
			if !matchesInvestigationProcess(record.ProcessGUID, record.PID, processGUID, pid, hasPID) {
				continue
			}
			if len(processes) < limit {
				processes = append(processes, record)
			}
		case "aegis.flow.started", "aegis.flow.ended":
			record, err := event.toFlowRecord()
			if err != nil {
				s.logger.Warn("Skipping malformed flow visibility event",
					"event_id", event.EventID,
					"event_type", event.EventType,
					"error", err)
				continue
			}
			if !matchesInvestigationOptionalProcess(record.ProcessGUID, record.PID, processGUID, pid, hasPID) {
				continue
			}
			if len(flows) < limit {
				flows = append(flows, record)
			}
		case "aegis.dns.observed":
			record, err := event.toDNSRecord()
			if err != nil {
				s.logger.Warn("Skipping malformed DNS visibility event",
					"event_id", event.EventID,
					"event_type", event.EventType,
					"error", err)
				continue
			}
			if !matchesInvestigationOptionalProcess(record.ProcessGUID, record.PID, processGUID, pid, hasPID) {
				continue
			}
			if len(dns) < limit {
				dns = append(dns, record)
			}
		case "aegis.agent.detected", "aegis.risk_finding.created":
			record, err := event.toFindingRecord()
			if err != nil {
				s.logger.Warn("Skipping malformed finding visibility event",
					"event_id", event.EventID,
					"event_type", event.EventType,
					"error", err)
				continue
			}
			if !matchesInvestigationFinding(record, processGUID, pid, hasPID) {
				continue
			}
			if len(findings) < limit {
				findings = append(findings, record)
			}
		}
	}

	filters := investigationFilters{ProcessGUID: processGUID}
	if hasPID {
		filters.PID = &pid
	}
	drafts := buildDraftControls(deviceID, filters, processes, flows, dns, findings)

	return investigationResponse{
		OK:       true,
		DeviceID: deviceID,
		AgentID:  agentID,
		Filters:  filters,
		Counts: investigationCounts{
			Processes: len(processes),
			Flows:     len(flows),
			DNS:       len(dns),
			Findings:  len(findings),
			Drafts:    len(drafts),
		},
		Processes: processes,
		Flows:     flows,
		DNS:       dns,
		Findings:  findings,
		Drafts:    drafts,
	}, nil
}

func buildDraftControls(deviceID string, filters investigationFilters, processes []processRecord, flows []flowRecord, dns []dnsRecord, findings []findingRecord) []draftControlRecord {
	if len(processes) == 0 && len(flows) == 0 && len(findings) == 0 {
		return nil
	}

	var process *processRecord
	if len(processes) > 0 {
		process = &processes[0]
	}
	var flow *flowRecord
	if len(flows) > 0 {
		flow = &flows[0]
	}
	var finding *findingRecord
	if len(findings) > 0 {
		finding = &findings[0]
	}
	var dnsRecord *dnsRecord
	if len(dns) > 0 {
		dnsRecord = &dns[0]
	}

	processName := "selected activity"
	if process != nil && stringValue(process.Name) != "" {
		processName = stringValue(process.Name)
	} else if flow != nil && stringValue(flow.ProcessName) != "" {
		processName = stringValue(flow.ProcessName)
	}
	processScope := processScopeText(process, filters)
	remoteTarget := "remote destination not yet linked"
	if flow != nil {
		remoteTarget = socketText(flow.RemoteIP, flow.RemotePort)
	}
	findingTitle := "linked endpoint evidence"
	if finding != nil {
		findingTitle = findingTitleText(*finding)
	}

	evidence := []string{}
	if process != nil {
		evidence = append(evidence, fmt.Sprintf("Process: %s pid %d", processName, process.PID))
		if commandLine := stringValue(process.CommandLine); commandLine != "" {
			evidence = append(evidence, "Command line: "+commandLine)
		}
	} else if filters.ProcessGUID != "" {
		evidence = append(evidence, "Selection: "+filters.ProcessGUID)
	}
	if flow != nil {
		evidence = append(evidence, fmt.Sprintf("Flow: %s %s to %s", defaultString(stringValue(flow.Direction), "unknown"), defaultString(stringValue(flow.Protocol), "tcp"), remoteTarget))
	}
	if dnsRecord != nil {
		answer := strings.Join(dnsRecord.Answers, ", ")
		if answer == "" {
			answer = defaultString(stringValue(dnsRecord.Resolver), "unknown")
		}
		evidence = append(evidence, fmt.Sprintf("DNS: %s resolved to %s", dnsRecord.Query, answer))
	}
	if finding != nil {
		evidence = append(evidence, fmt.Sprintf("Finding: %s risk %d", findingTitle, finding.RiskScore))
	}

	if flow != nil {
		return []draftControlRecord{{
			ControlID:   draftControlID(deviceID, processScope, remoteTarget),
			Title:       "Observe outbound access for " + processName,
			Mode:        "observe-only",
			Status:      "draft",
			Action:      "Stage a monitor rule that records matches before deny or restrict is considered.",
			Target:      remoteTarget,
			Scope:       fmt.Sprintf("%s; %s %s traffic to %s", processScope, defaultString(stringValue(flow.Protocol), "tcp"), defaultString(stringValue(flow.Direction), "outbound"), remoteTarget),
			Reason:      fmt.Sprintf("Aegis linked %s to a concrete process and network destination. The next safe step is to measure repeat matches and affected activity before enforcement.", findingTitle),
			Evidence:    evidence,
			BlastRadius: []string{"Count historical matches for the same process, destination, port, and protocol.", "Check whether other processes on the device use the same destination.", "Require DNS and process evidence before expanding beyond this endpoint."},
			Rollback:    []string{"Disable the staged monitor rule.", "Clear any pending enforcement candidate for this process and destination.", "Keep collected evidence for audit and future simulation."},
		}}
	}

	return []draftControlRecord{{
		ControlID:   draftControlID(deviceID, processScope, remoteTarget),
		Title:       "Observe process behavior for " + processName,
		Mode:        "observe-only",
		Status:      "draft",
		Action:      "Stage a process watch that records future flows, DNS, and findings before control design.",
		Target:      processTargetText(process, filters),
		Scope:       processScope,
		Reason:      "Aegis has process or finding evidence but not enough linked network evidence for a network control. The next safe step is richer observation.",
		Evidence:    evidence,
		BlastRadius: []string{"Wait for at least one linked flow or DNS record before proposing a restrict or deny rule.", "Compare command-line markers to avoid matching unrelated processes.", "Keep scope limited to this device until cross-device evidence exists."},
		Rollback:    []string{"Remove the process watch candidate.", "Keep the investigation record available for review.", "Return the device to passive collection only."},
	}}
}

func processScopeText(process *processRecord, filters investigationFilters) string {
	if process != nil {
		scope := fmt.Sprintf("%s pid %d", defaultString(stringValue(process.Name), "unknown process"), process.PID)
		if path := stringValue(process.Path); path != "" {
			scope += " at " + path
		}
		return scope
	}
	if filters.ProcessGUID != "" {
		return filters.ProcessGUID
	}
	if filters.PID != nil {
		return fmt.Sprintf("pid %d", *filters.PID)
	}
	return "device activity"
}

func processTargetText(process *processRecord, filters investigationFilters) string {
	if process != nil && stringValue(process.Path) != "" {
		return stringValue(process.Path)
	}
	if filters.ProcessGUID != "" {
		return filters.ProcessGUID
	}
	if filters.PID != nil {
		return fmt.Sprintf("pid %d", *filters.PID)
	}
	return "device activity"
}

func findingTitleText(finding findingRecord) string {
	for _, value := range []string{
		stringValue(finding.Title),
		stringValue(finding.Classification),
		finding.EventType,
	} {
		if value != "" {
			return value
		}
	}
	return "linked endpoint evidence"
}

func socketText(ip *string, port *int) string {
	if port != nil && *port > 0 {
		return fmt.Sprintf("%s:%d", defaultString(stringValue(ip), "unknown"), *port)
	}
	return defaultString(stringValue(ip), "unknown")
}

func draftControlID(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return fmt.Sprintf("draft-%x", sum[:6])
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func matchesInvestigationProcess(recordGUID string, recordPID int, filterGUID string, filterPID int, hasFilterPID bool) bool {
	if filterGUID != "" && recordGUID != filterGUID {
		return false
	}
	if hasFilterPID && recordPID != filterPID {
		return false
	}
	return true
}

func matchesInvestigationOptionalProcess(recordGUID *string, recordPID *int, filterGUID string, filterPID int, hasFilterPID bool) bool {
	if filterGUID != "" && stringValue(recordGUID) != filterGUID {
		return false
	}
	if hasFilterPID && (recordPID == nil || *recordPID != filterPID) {
		return false
	}
	return true
}

func matchesInvestigationFinding(record findingRecord, filterGUID string, filterPID int, hasFilterPID bool) bool {
	if filterGUID != "" && stringValue(record.ProcessGUID) != filterGUID {
		return false
	}
	// Findings do not currently carry PID in the v1 schema. If a PID filter is
	// provided, include process-linked findings by GUID and leave PID-only
	// matching to process/flow/DNS records.
	if hasFilterPID && filterGUID == "" {
		return false
	}
	return true
}
