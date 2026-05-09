package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aegisflux/backend/ingest/protos"
)

const (
	maxVisibilityRequestBytes = 10 << 20
	maxVisibilityLineBytes    = 1 << 20
)

var (
	errEventValidation = errors.New("event validation failed")
	errEventPublish    = errors.New("event publish failed")
)

type visibilityEvent struct {
	SchemaVersion string          `json:"schema_version"`
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	TimestampMS   int64           `json:"timestamp_ms"`
	Source        string          `json:"source"`
	TenantID      string          `json:"tenant_id,omitempty"`
	DeviceID      string          `json:"device_id"`
	AgentID       string          `json:"agent_id"`
	SensorVersion string          `json:"sensor_version"`
	Sequence      int64           `json:"sequence"`
	Payload       json.RawMessage `json:"payload"`
	ReceivedAtMS  int64           `json:"received_at_ms,omitempty"`
}

type visibilityIngestResponse struct {
	OK       bool   `json:"ok"`
	Accepted int    `json:"accepted"`
	Message  string `json:"message"`
}

type visibilityQueryResponse struct {
	OK     bool              `json:"ok"`
	Count  int               `json:"count"`
	Events []visibilityEvent `json:"events"`
}

// RegisterHTTPRoutes registers HTTP endpoints owned by the ingest service.
func (s *IngestServer) RegisterHTTPRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/visibility/events", s.handleVisibilityEvents)
	mux.HandleFunc("/v1/visibility/devices", s.handleVisibilityDevices)
	mux.HandleFunc("/v1/visibility/processes", s.handleVisibilityProcesses)
	mux.HandleFunc("/v1/visibility/flows", s.handleVisibilityFlows)
	mux.HandleFunc("/v1/visibility/dns", s.handleVisibilityDNS)
	mux.HandleFunc("/v1/visibility/findings", s.handleVisibilityFindings)
	mux.HandleFunc("/v1/visibility/investigation", s.handleVisibilityInvestigation)
	mux.HandleFunc("/v1/visibility/draft-controls", s.handleVisibilityDraftControls)
	mux.HandleFunc("/v1/visibility/summary/dashboard", s.handleSummaryDashboard)
	mux.HandleFunc("/v1/visibility/summary/device", s.handleSummaryDevice)
	mux.HandleFunc("/v1/visibility/summary/inventory", s.handleSummaryInventory)
	mux.HandleFunc("/v1/clarion/events", s.handleClarionEventExport)
}

func (s *IngestServer) handleVisibilityEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
	case http.MethodGet:
		s.handleVisibilityEventQuery(w, r)
		return
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	events, err := readVisibilityEvents(http.MaxBytesReader(w, r.Body, maxVisibilityRequestBytes))
	if err != nil {
		s.metrics.IncrementEventsInvalid()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(events) == 0 {
		s.metrics.IncrementEventsInvalid()
		http.Error(w, "request did not contain any visibility events", http.StatusBadRequest)
		return
	}

	for _, event := range events {
		if err := s.processVisibilityEvent(r.Context(), event); err != nil {
			if errors.Is(err, errEventValidation) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}

	writeJSON(w, http.StatusAccepted, visibilityIngestResponse{
		OK:       true,
		Accepted: len(events),
		Message:  "visibility events accepted",
	})
}

func (s *IngestServer) handleVisibilityEventQuery(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, visibilityQueryResponse{
		OK:     true,
		Count:  len(events),
		Events: events,
	})
}

func parseQueryLimit(raw string) (int, error) {
	if raw == "" {
		return defaultVisibilityQueryLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("limit must be an integer")
	}
	if limit <= 0 {
		return 0, fmt.Errorf("limit must be positive")
	}
	if limit > maxVisibilityQueryLimit {
		return maxVisibilityQueryLimit, nil
	}
	return limit, nil
}

func (s *IngestServer) processVisibilityEvent(ctx context.Context, event visibilityEvent) error {
	protoEvent, err := event.toProtoEvent()
	if err != nil {
		s.metrics.IncrementEventsInvalid()
		return err
	}

	if s.store != nil {
		exists, err := s.store.Has(ctx, event.EventID)
		if err != nil {
			return err
		}
		if exists {
			s.logger.Info("Duplicate persisted event ignored",
				"event_id", event.EventID,
				"event_type", event.EventType,
				"host_id", event.DeviceID)
			return nil
		}
	}

	if err := s.processEvent(ctx, protoEvent); err != nil {
		return err
	}

	if s.store != nil {
		if err := s.store.Append(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (s *IngestServer) processEvent(ctx context.Context, event *protos.Event) error {
	hostID := "unknown"
	if h, exists := event.Metadata["host_id"]; exists {
		hostID = h
	}

	s.logger.Info("Processing event",
		"event_id", event.Id,
		"event_type", event.Type,
		"host_id", hostID)

	if err := s.validator.ValidateEvent(ctx, event); err != nil {
		s.logger.Warn("Event validation failed",
			"event_id", event.Id,
			"event_type", event.Type,
			"host_id", hostID,
			"error", err)
		s.metrics.IncrementEventsInvalid()
		return fmt.Errorf("%w: %v", errEventValidation, err)
	}

	if s.dedupe != nil && s.dedupe.has(event.Id) {
		s.logger.Info("Duplicate event ignored",
			"event_id", event.Id,
			"event_type", event.Type,
			"host_id", hostID)
		return nil
	}

	publishCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := s.publisher.PublishEvent(publishCtx, event); err != nil {
		s.logger.Error("Failed to publish event",
			"event_id", event.Id,
			"event_type", event.Type,
			"host_id", hostID,
			"error", err)
		s.metrics.IncrementNatsPublishErrors()
		return fmt.Errorf("%w: %v", errEventPublish, err)
	}

	if s.dedupe != nil {
		s.dedupe.add(event.Id)
	}

	s.metrics.IncrementEventsTotal()
	s.logger.Info("Event processed successfully",
		"event_id", event.Id,
		"event_type", event.Type,
		"host_id", hostID)

	return nil
}

func readVisibilityEvents(r io.Reader) ([]visibilityEvent, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var events []visibilityEvent
		if err := json.Unmarshal([]byte(trimmed), &events); err != nil {
			return nil, fmt.Errorf("invalid visibility event array: %w", err)
		}
		return events, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(trimmed))
	scanner.Buffer(make([]byte, 0, 64*1024), maxVisibilityLineBytes)

	var events []visibilityEvent
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event visibilityEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("invalid visibility event JSONL record: %w", err)
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan visibility event JSONL: %w", err)
	}

	return events, nil
}

func (e visibilityEvent) toProtoEvent() (*protos.Event, error) {
	if e.EventID == "" {
		return nil, errors.New("visibility event missing event_id")
	}
	if e.EventType == "" {
		return nil, errors.New("visibility event missing event_type")
	}
	if e.Source == "" {
		return nil, errors.New("visibility event missing source")
	}
	if e.TimestampMS <= 0 {
		return nil, errors.New("visibility event timestamp_ms must be positive")
	}
	if len(e.Payload) == 0 {
		e.Payload = []byte("{}")
	}

	metadata := map[string]string{
		"schema_version": e.SchemaVersion,
		"tenant_id":      e.TenantID,
		"device_id":      e.DeviceID,
		"agent_id":       e.AgentID,
		"sensor_version": e.SensorVersion,
		"sequence":       fmt.Sprintf("%d", e.Sequence),
	}
	if e.DeviceID != "" {
		metadata["host_id"] = e.DeviceID
	}

	return &protos.Event{
		Id:        e.EventID,
		Type:      e.EventType,
		Source:    e.Source,
		Timestamp: e.TimestampMS,
		Metadata:  metadata,
		Payload:   []byte(e.Payload),
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
