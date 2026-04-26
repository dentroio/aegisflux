package server

import (
	"encoding/json"
	"net/http"
)

type flowQueryResponse struct {
	OK    bool         `json:"ok"`
	Count int          `json:"count"`
	Flows []flowRecord `json:"flows"`
}

type flowRecord struct {
	EventID               string          `json:"event_id"`
	EventType             string          `json:"event_type"`
	TimestampMS           int64           `json:"timestamp_ms"`
	DeviceID              string          `json:"device_id"`
	AgentID               string          `json:"agent_id"`
	FlowID                string          `json:"flow_id"`
	ProcessGUID           *string         `json:"process_guid,omitempty"`
	PID                   *int            `json:"pid,omitempty"`
	ProcessName           *string         `json:"process_name,omitempty"`
	User                  *string         `json:"user,omitempty"`
	Protocol              *string         `json:"protocol,omitempty"`
	Direction             *string         `json:"direction,omitempty"`
	LocalIP               *string         `json:"local_ip,omitempty"`
	LocalPort             *int            `json:"local_port,omitempty"`
	RemoteIP              *string         `json:"remote_ip,omitempty"`
	RemotePort            *int            `json:"remote_port,omitempty"`
	RemoteHostname        *string         `json:"remote_hostname,omitempty"`
	AttributionMethod     *string         `json:"attribution_method,omitempty"`
	AttributionConfidence *float64        `json:"attribution_confidence,omitempty"`
	BytesSent             *int64          `json:"bytes_sent,omitempty"`
	BytesReceived         *int64          `json:"bytes_received,omitempty"`
	DurationMS            *int64          `json:"duration_ms,omitempty"`
	ConnectionState       *string         `json:"connection_state,omitempty"`
	CollectionMethod      *string         `json:"collection_method,omitempty"`
	Payload               json.RawMessage `json:"payload"`
}

type flowPayload struct {
	FlowID                string   `json:"flow_id"`
	ProcessGUID           *string  `json:"process_guid"`
	PID                   *int     `json:"pid"`
	ProcessName           *string  `json:"process_name"`
	User                  *string  `json:"user"`
	Protocol              *string  `json:"protocol"`
	Direction             *string  `json:"direction"`
	LocalIP               *string  `json:"local_ip"`
	LocalPort             *int     `json:"local_port"`
	RemoteIP              *string  `json:"remote_ip"`
	RemotePort            *int     `json:"remote_port"`
	RemoteHostname        *string  `json:"remote_hostname"`
	AttributionMethod     *string  `json:"attribution_method"`
	AttributionConfidence *float64 `json:"attribution_confidence"`
	BytesSent             *int64   `json:"bytes_sent"`
	BytesReceived         *int64   `json:"bytes_received"`
	DurationMS            *int64   `json:"duration_ms"`
	ConnectionState       *string  `json:"connection_state"`
	CollectionMethod      *string  `json:"collection_method"`
}

func (s *IngestServer) handleVisibilityFlows(w http.ResponseWriter, r *http.Request) {
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
	pid, hasPID, err := parseOptionalInt(r.URL.Query().Get("pid"), "pid")
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

	flowID := r.URL.Query().Get("flow_id")
	processGUID := r.URL.Query().Get("process_guid")
	remoteIP := r.URL.Query().Get("remote_ip")
	remoteHostname := r.URL.Query().Get("remote_hostname")

	flows := make([]flowRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.flow.started" && event.EventType != "aegis.flow.ended" {
			continue
		}
		record, err := event.toFlowRecord()
		if err != nil {
			s.logger.Warn("Skipping malformed flow visibility event",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"error", err)
			continue
		}
		if flowID != "" && record.FlowID != flowID {
			continue
		}
		if processGUID != "" && stringValue(record.ProcessGUID) != processGUID {
			continue
		}
		if hasPID && (record.PID == nil || *record.PID != pid) {
			continue
		}
		if remoteIP != "" && stringValue(record.RemoteIP) != remoteIP {
			continue
		}
		if remoteHostname != "" && stringValue(record.RemoteHostname) != remoteHostname {
			continue
		}
		flows = append(flows, record)
		if len(flows) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, flowQueryResponse{
		OK:    true,
		Count: len(flows),
		Flows: flows,
	})
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (e visibilityEvent) toFlowRecord() (flowRecord, error) {
	var payload flowPayload
	if len(e.Payload) > 0 {
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return flowRecord{}, err
		}
	}

	record := flowRecord{
		EventID:               e.EventID,
		EventType:             e.EventType,
		TimestampMS:           e.TimestampMS,
		DeviceID:              e.DeviceID,
		AgentID:               e.AgentID,
		FlowID:                payload.FlowID,
		ProcessGUID:           payload.ProcessGUID,
		PID:                   payload.PID,
		ProcessName:           payload.ProcessName,
		User:                  payload.User,
		Protocol:              payload.Protocol,
		Direction:             payload.Direction,
		LocalIP:               payload.LocalIP,
		LocalPort:             payload.LocalPort,
		RemoteIP:              payload.RemoteIP,
		RemotePort:            payload.RemotePort,
		RemoteHostname:        payload.RemoteHostname,
		AttributionMethod:     payload.AttributionMethod,
		AttributionConfidence: payload.AttributionConfidence,
		BytesSent:             payload.BytesSent,
		BytesReceived:         payload.BytesReceived,
		DurationMS:            payload.DurationMS,
		ConnectionState:       payload.ConnectionState,
		CollectionMethod:      payload.CollectionMethod,
		Payload:               e.Payload,
	}
	return record, nil
}
