package server

import "net/http"

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

	events, err := s.store.Query(r.Context(), visibilityQueryFilter{
		DeviceID: deviceID,
		AgentID:  agentID,
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
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

	writeJSON(w, http.StatusOK, investigationResponse{
		OK:       true,
		DeviceID: deviceID,
		AgentID:  agentID,
		Filters:  filters,
		Counts: investigationCounts{
			Processes: len(processes),
			Flows:     len(flows),
			DNS:       len(dns),
			Findings:  len(findings),
		},
		Processes: processes,
		Flows:     flows,
		DNS:       dns,
		Findings:  findings,
	})
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
