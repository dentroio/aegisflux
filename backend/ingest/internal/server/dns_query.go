package server

import (
	"encoding/json"
	"net/http"
)

type dnsQueryResponse struct {
	OK           bool        `json:"ok"`
	Count        int         `json:"count"`
	Observations []dnsRecord `json:"observations"`
}

type dnsRecord struct {
	EventID               string          `json:"event_id"`
	EventType             string          `json:"event_type"`
	TimestampMS           int64           `json:"timestamp_ms"`
	DeviceID              string          `json:"device_id"`
	AgentID               string          `json:"agent_id"`
	Query                 string          `json:"query"`
	QueryType             *string         `json:"query_type,omitempty"`
	Answers               []string        `json:"answers"`
	Resolver              *string         `json:"resolver,omitempty"`
	ProcessGUID           *string         `json:"process_guid,omitempty"`
	PID                   *int            `json:"pid,omitempty"`
	CorrelationMethod     string          `json:"correlation_method"`
	CorrelationConfidence float64         `json:"correlation_confidence"`
	Payload               json.RawMessage `json:"payload"`
}

type dnsPayload struct {
	Query                 string   `json:"query"`
	QueryType             *string  `json:"query_type"`
	Answers               []string `json:"answers"`
	Resolver              *string  `json:"resolver"`
	ProcessGUID           *string  `json:"process_guid"`
	PID                   *int     `json:"pid"`
	CorrelationMethod     string   `json:"correlation_method"`
	CorrelationConfidence float64  `json:"correlation_confidence"`
}

func (s *IngestServer) handleVisibilityDNS(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query().Get("query")
	answer := r.URL.Query().Get("answer")
	processGUID := r.URL.Query().Get("process_guid")

	observations := make([]dnsRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.dns.observed" {
			continue
		}
		record, err := event.toDNSRecord()
		if err != nil {
			s.logger.Warn("Skipping malformed DNS visibility event",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"error", err)
			continue
		}
		if query != "" && record.Query != query {
			continue
		}
		if answer != "" && !containsString(record.Answers, answer) {
			continue
		}
		if processGUID != "" && stringValue(record.ProcessGUID) != processGUID {
			continue
		}
		if hasPID && (record.PID == nil || *record.PID != pid) {
			continue
		}
		observations = append(observations, record)
		if len(observations) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, dnsQueryResponse{
		OK:           true,
		Count:        len(observations),
		Observations: observations,
	})
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (e visibilityEvent) toDNSRecord() (dnsRecord, error) {
	var payload dnsPayload
	if len(e.Payload) > 0 {
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return dnsRecord{}, err
		}
	}

	record := dnsRecord{
		EventID:               e.EventID,
		EventType:             e.EventType,
		TimestampMS:           e.TimestampMS,
		DeviceID:              e.DeviceID,
		AgentID:               e.AgentID,
		Query:                 payload.Query,
		QueryType:             payload.QueryType,
		Answers:               payload.Answers,
		Resolver:              payload.Resolver,
		ProcessGUID:           payload.ProcessGUID,
		PID:                   payload.PID,
		CorrelationMethod:     payload.CorrelationMethod,
		CorrelationConfidence: payload.CorrelationConfidence,
		Payload:               e.Payload,
	}
	return record, nil
}
