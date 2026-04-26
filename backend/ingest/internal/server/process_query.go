package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type processQueryResponse struct {
	OK        bool            `json:"ok"`
	Count     int             `json:"count"`
	Processes []processRecord `json:"processes"`
}

type processRecord struct {
	EventID           string          `json:"event_id"`
	EventType         string          `json:"event_type"`
	TimestampMS       int64           `json:"timestamp_ms"`
	DeviceID          string          `json:"device_id"`
	AgentID           string          `json:"agent_id"`
	ProcessGUID       string          `json:"process_guid,omitempty"`
	ParentProcessGUID *string         `json:"parent_process_guid,omitempty"`
	PID               int             `json:"pid"`
	PPID              *int            `json:"ppid,omitempty"`
	Name              *string         `json:"name,omitempty"`
	Path              *string         `json:"path,omitempty"`
	CommandLine       *string         `json:"command_line,omitempty"`
	User              *string         `json:"user,omitempty"`
	CollectionMethod  *string         `json:"collection_method,omitempty"`
	ExitCode          *int            `json:"exit_code,omitempty"`
	DurationMS        *int64          `json:"duration_ms,omitempty"`
	Payload           json.RawMessage `json:"payload"`
}

type processPayload struct {
	ProcessGUID       string  `json:"process_guid"`
	ParentProcessGUID *string `json:"parent_process_guid"`
	PID               int     `json:"pid"`
	PPID              *int    `json:"ppid"`
	Name              *string `json:"name"`
	Path              *string `json:"path"`
	CommandLine       *string `json:"command_line"`
	User              *string `json:"user"`
	CollectionMethod  *string `json:"collection_method"`
	ExitCode          *int    `json:"exit_code"`
	DurationMS        *int64  `json:"duration_ms"`
}

func (s *IngestServer) handleVisibilityProcesses(w http.ResponseWriter, r *http.Request) {
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

	processGUID := r.URL.Query().Get("process_guid")
	processes := make([]processRecord, 0, limit)
	for _, event := range events {
		if event.EventType != "aegis.process.started" && event.EventType != "aegis.process.ended" {
			continue
		}
		record, err := event.toProcessRecord()
		if err != nil {
			s.logger.Warn("Skipping malformed process visibility event",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"error", err)
			continue
		}
		if processGUID != "" && record.ProcessGUID != processGUID {
			continue
		}
		if hasPID && record.PID != pid {
			continue
		}
		processes = append(processes, record)
		if len(processes) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, processQueryResponse{
		OK:        true,
		Count:     len(processes),
		Processes: processes,
	})
}

func parseOptionalInt(raw, name string) (int, bool, error) {
	if raw == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, fmt.Errorf("%s must be an integer", name)
	}
	return value, true, nil
}

func (e visibilityEvent) toProcessRecord() (processRecord, error) {
	var payload processPayload
	if len(e.Payload) > 0 {
		if err := json.Unmarshal(e.Payload, &payload); err != nil {
			return processRecord{}, err
		}
	}

	record := processRecord{
		EventID:           e.EventID,
		EventType:         e.EventType,
		TimestampMS:       e.TimestampMS,
		DeviceID:          e.DeviceID,
		AgentID:           e.AgentID,
		ProcessGUID:       payload.ProcessGUID,
		ParentProcessGUID: payload.ParentProcessGUID,
		PID:               payload.PID,
		PPID:              payload.PPID,
		Name:              payload.Name,
		Path:              payload.Path,
		CommandLine:       payload.CommandLine,
		User:              payload.User,
		CollectionMethod:  payload.CollectionMethod,
		ExitCode:          payload.ExitCode,
		DurationMS:        payload.DurationMS,
		Payload:           e.Payload,
	}
	return record, nil
}
