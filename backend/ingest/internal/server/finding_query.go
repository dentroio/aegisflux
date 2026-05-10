package server

import (
	"encoding/json"
	"net/http"
)

type findingQueryResponse struct {
	OK       bool            `json:"ok"`
	Count    int             `json:"count"`
	Findings []findingRecord `json:"findings"`
}

type findingRecord struct {
	EventID             string          `json:"event_id"`
	EventType           string          `json:"event_type"`
	TimestampMS         int64           `json:"timestamp_ms"`
	DeviceID            string          `json:"device_id"`
	AgentID             string          `json:"agent_id"`
	DetectionID         *string         `json:"detection_id,omitempty"`
	FindingID           *string         `json:"finding_id,omitempty"`
	Severity            *string         `json:"severity,omitempty"`
	Title               *string         `json:"title,omitempty"`
	Description         *string         `json:"description,omitempty"`
	Classification      *string         `json:"classification,omitempty"`
	ApplicationCategory *string         `json:"application_category,omitempty"`
	RiskSignal          *string         `json:"risk_signal,omitempty"`
	AgentLikelihood     *float64        `json:"agent_likelihood,omitempty"`
	Confidence          *float64        `json:"confidence,omitempty"`
	RiskScore           int             `json:"risk_score"`
	ProcessGUID         *string         `json:"process_guid,omitempty"`
	FlowID              *string         `json:"flow_id,omitempty"`
	DetectedPatterns    []string        `json:"detected_patterns,omitempty"`
	Evidence            json.RawMessage `json:"evidence"`
	RecommendedAction   string          `json:"recommended_action"`
	Payload             json.RawMessage `json:"payload"`
}

type findingPayload struct {
	DetectionID         *string         `json:"detection_id"`
	FindingID           *string         `json:"finding_id"`
	Severity            *string         `json:"severity"`
	Title               *string         `json:"title"`
	Description         *string         `json:"description"`
	Classification      *string         `json:"classification"`
	ApplicationCategory *string         `json:"application_category"`
	RiskSignal          *string         `json:"risk_signal"`
	AgentLikelihood     *float64        `json:"agent_likelihood"`
	Confidence          *float64        `json:"confidence"`
	RiskScore           int             `json:"risk_score"`
	ProcessGUID         *string         `json:"process_guid"`
	FlowID              *string         `json:"flow_id"`
	DetectedPatterns    []string        `json:"detected_patterns"`
	Evidence            json.RawMessage `json:"evidence"`
	RecommendedAction   string          `json:"recommended_action"`
}

func (s *IngestServer) handleVisibilityFindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	limit, err := parseQueryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	events, err := s.store.Query(r.Context(), visibilityQueryFilter{
		DeviceID: r.URL.Query().Get("device_id"),
		AgentID:  r.URL.Query().Get("agent_id"),
		Limit:    maxVisibilityQueryLimit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	processGUID := r.URL.Query().Get("process_guid")
	flowID := r.URL.Query().Get("flow_id")
	detectionID := r.URL.Query().Get("detection_id")
	findingID := r.URL.Query().Get("finding_id")
	severity := r.URL.Query().Get("severity")

	findings := make([]findingRecord, 0, limit)
	for _, event := range events {
		if !isFindingEventType(event.EventType) {
			continue
		}
		record, err := event.toFindingRecord()
		if err != nil {
			s.logger.Warn("Skipping malformed finding visibility event",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"error", err)
			continue
		}
		if processGUID != "" && stringValue(record.ProcessGUID) != processGUID {
			continue
		}
		if flowID != "" && stringValue(record.FlowID) != flowID {
			continue
		}
		if detectionID != "" && stringValue(record.DetectionID) != detectionID {
			continue
		}
		if findingID != "" && stringValue(record.FindingID) != findingID {
			continue
		}
		if severity != "" && stringValue(record.Severity) != severity {
			continue
		}
		findings = append(findings, record)
		if len(findings) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, findingQueryResponse{
		OK:       true,
		Count:    len(findings),
		Findings: findings,
	})
}

func isFindingEventType(eventType string) bool {
	return eventType == "aegis.agent.detected" || eventType == "aegis.risk_finding.created"
}

func (e visibilityEvent) toFindingRecord() (findingRecord, error) {
	var payload findingPayload
	if len(e.Payload) > 0 {
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return findingRecord{}, err
		}
	}
	if len(payload.Evidence) == 0 {
		payload.Evidence = []byte("[]")
	}

	record := findingRecord{
		EventID:             e.EventID,
		EventType:           e.EventType,
		TimestampMS:         e.TimestampMS,
		DeviceID:            e.DeviceID,
		AgentID:             e.AgentID,
		DetectionID:         payload.DetectionID,
		FindingID:           payload.FindingID,
		Severity:            payload.Severity,
		Title:               payload.Title,
		Description:         payload.Description,
		Classification:      payload.Classification,
		ApplicationCategory: payload.ApplicationCategory,
		RiskSignal:          payload.RiskSignal,
		AgentLikelihood:     payload.AgentLikelihood,
		Confidence:          payload.Confidence,
		RiskScore:           payload.RiskScore,
		ProcessGUID:         payload.ProcessGUID,
		FlowID:              payload.FlowID,
		DetectedPatterns:    payload.DetectedPatterns,
		Evidence:            payload.Evidence,
		RecommendedAction:   payload.RecommendedAction,
		Payload:             e.Payload,
	}
	return record, nil
}
