package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"aegisflux/backend/ingest/internal/health"
	"aegisflux/backend/ingest/internal/metrics"
	"aegisflux/backend/ingest/protos"
)

type mockValidator struct {
	err error
}

func (m mockValidator) ValidateEvent(ctx context.Context, e *protos.Event) error {
	return m.err
}

type mockPublisher struct {
	err    error
	events []*protos.Event
}

var sharedTestMetrics = metrics.NewMetrics()

func (m *mockPublisher) PublishEvent(ctx context.Context, e *protos.Event) error {
	if m.err != nil {
		return m.err
	}
	m.events = append(m.events, e)
	return nil
}

func TestReadVisibilityEventsJSONL(t *testing.T) {
	body := `{"schema_version":"visibility.v1","event_id":"evt-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}` + "\n" +
		`{"schema_version":"visibility.v1","event_id":"evt-2","event_type":"aegis.collector.status","timestamp_ms":1777075005617,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":4,"payload":{"collector":"windows.process","status":"healthy"}}`

	events, err := readVisibilityEvents(bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("readVisibilityEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].EventType != "aegis.process.started" {
		t.Fatalf("unexpected event type: %s", events[0].EventType)
	}
}

func TestVisibilityEventToProtoEvent(t *testing.T) {
	event := visibilityEvent{
		SchemaVersion: "visibility.v1",
		EventID:       "evt-1",
		EventType:     "aegis.process.started",
		TimestampMS:   1777075005616,
		Source:        "aegis-windows-agent",
		DeviceID:      "RMARTINEZ-WS",
		AgentID:       "windows-agent-dev",
		SensorVersion: "0.1.0",
		Sequence:      3,
		Payload:       []byte(`{"pid":3084,"name":"svchost.exe"}`),
	}

	protoEvent, err := event.toProtoEvent()
	if err != nil {
		t.Fatalf("toProtoEvent returned error: %v", err)
	}
	if protoEvent.Id != event.EventID {
		t.Fatalf("expected id %q, got %q", event.EventID, protoEvent.Id)
	}
	if protoEvent.Type != event.EventType {
		t.Fatalf("expected type %q, got %q", event.EventType, protoEvent.Type)
	}
	if protoEvent.Metadata["host_id"] != event.DeviceID {
		t.Fatalf("expected host_id %q, got %q", event.DeviceID, protoEvent.Metadata["host_id"])
	}
}

func TestHandleVisibilityEvents(t *testing.T) {
	publisher := &mockPublisher{}
	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: publisher,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	server.handleVisibilityEvents(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, rec.Code, rec.Body.String())
	}
	if len(publisher.events) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(publisher.events))
	}
}

func TestHandleVisibilityEventsRejectsValidationFailure(t *testing.T) {
	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{err: errors.New("invalid event")},
		publisher: &mockPublisher{},
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	server.handleVisibilityEvents(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
