package server

import (
	"encoding/json"
	"net/http"
)

const clarionExportContractVersion = "aegis-clarion.export.v1"

type clarionEventExportResponse struct {
	OK              bool                 `json:"ok"`
	ContractVersion string               `json:"contract_version"`
	Count           int                  `json:"count"`
	Events          []clarionEventExport `json:"events"`
}

type clarionEventExport struct {
	ContractVersion       string          `json:"contract_version"`
	SchemaVersion         string          `json:"schema_version"`
	EventID               string          `json:"event_id"`
	EventType             string          `json:"event_type"`
	TimestampMS           int64           `json:"timestamp_ms"`
	Source                string          `json:"source"`
	TenantID              string          `json:"tenant_id,omitempty"`
	DeviceID              string          `json:"device_id"`
	AgentID               string          `json:"agent_id"`
	SensorVersion         string          `json:"sensor_version"`
	Sequence              int64           `json:"sequence"`
	ClarionContextObjects []string        `json:"clarion_context_objects"`
	Payload               json.RawMessage `json:"payload"`
	ReceivedAtMS          int64           `json:"received_at_ms,omitempty"`
}

func (s *IngestServer) handleClarionEventExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
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
		EventID:   r.URL.Query().Get("event_id"),
		TenantID:  r.URL.Query().Get("tenant_id"),
		DeviceID:  r.URL.Query().Get("device_id"),
		AgentID:   r.URL.Query().Get("agent_id"),
		EventType: r.URL.Query().Get("event_type"),
		Limit:     limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	exports := make([]clarionEventExport, 0, len(events))
	for _, event := range events {
		exports = append(exports, event.toClarionEventExport())
	}

	writeJSON(w, http.StatusOK, clarionEventExportResponse{
		OK:              true,
		ContractVersion: clarionExportContractVersion,
		Count:           len(exports),
		Events:          exports,
	})
}

func (e visibilityEvent) toClarionEventExport() clarionEventExport {
	return clarionEventExport{
		ContractVersion:       clarionExportContractVersion,
		SchemaVersion:         e.SchemaVersion,
		EventID:               e.EventID,
		EventType:             e.EventType,
		TimestampMS:           e.TimestampMS,
		Source:                e.Source,
		TenantID:              e.TenantID,
		DeviceID:              e.DeviceID,
		AgentID:               e.AgentID,
		SensorVersion:         e.SensorVersion,
		Sequence:              e.Sequence,
		ClarionContextObjects: clarionContextObjectsForEventType(e.EventType),
		Payload:               e.Payload,
		ReceivedAtMS:          e.ReceivedAtMS,
	}
}

func clarionContextObjectsForEventType(eventType string) []string {
	switch eventType {
	case "aegis.agent.registered", "aegis.agent.heartbeat":
		return []string{"Device", "Agent"}
	case "aegis.process.started", "aegis.process.exited":
		return []string{"Device", "Agent", "User", "Session", "Process", "Process lineage edge"}
	case "aegis.flow.started", "aegis.flow.ended":
		return []string{"Device", "Agent", "User", "Process", "Flow", "Destination"}
	case "aegis.dns.observed":
		return []string{"Device", "Agent", "Process", "DNS observation", "Destination"}
	case "aegis.application.classified":
		return []string{"Device", "Agent", "Process", "Application"}
	case "aegis.agent.detected", "aegis.risk_finding.created":
		return []string{"Device", "Agent", "Process", "Flow", "Destination", "AI-agent or automation finding", "Evidence bundle"}
	default:
		return []string{"Device", "Agent", "Evidence bundle"}
	}
}
