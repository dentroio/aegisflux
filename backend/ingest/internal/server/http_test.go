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
		dedupe:    newDuplicateTracker(100),
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

func TestHandleVisibilityEventsDeduplicatesEventID(t *testing.T) {
	publisher := &mockPublisher{}
	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: publisher,
		dedupe:    newDuplicateTracker(100),
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}`
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		server.handleVisibilityEvents(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: expected status %d, got %d: %s", i+1, http.StatusAccepted, rec.Code, rec.Body.String())
		}
	}

	if len(publisher.events) != 1 {
		t.Fatalf("expected duplicate event to be published once, got %d", len(publisher.events))
	}
}

func TestHandleVisibilityEventsStoresAndQueriesEvents(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	publisher := &mockPublisher{}
	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: publisher,
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/events?device_id=RMARTINEZ-WS&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityEvents(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	if !bytes.Contains(getRec.Body.Bytes(), []byte(`"event_id":"evt-1"`)) {
		t.Fatalf("expected query response to include stored event, got %s", getRec.Body.String())
	}
}

func TestHandleVisibilityProcessesReturnsNormalizedProcessEvents(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-process-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"process_guid":"proc-abc","parent_process_guid":"proc-parent","pid":3084,"ppid":200,"name":"python.exe","path":"C:\\AegisLab\\scripts\\agent_runner.py","command_line":"python.exe agent_runner.py","user":"RMARTINEZ\\tester","collection_method":"snapshot"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/processes?device_id=RMARTINEZ-WS&pid=3084&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityProcesses(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":1`),
		[]byte(`"process_guid":"proc-abc"`),
		[]byte(`"pid":3084`),
		[]byte(`"name":"python.exe"`),
		[]byte(`"command_line":"python.exe agent_runner.py"`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected process response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityFlowsReturnsNormalizedFlowEvents(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-flow-1","event_type":"aegis.flow.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":4,"payload":{"flow_id":"flow-abc","process_guid":"proc-abc","pid":3084,"process_name":"python.exe","user":"RMARTINEZ\\tester","protocol":"tcp","direction":"outbound","local_ip":"10.10.20.55","local_port":52944,"remote_ip":"203.0.113.10","remote_port":443,"remote_hostname":"api.model-gateway.lab","attribution_method":"fixture.pid","attribution_confidence":0.98}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/flows?device_id=RMARTINEZ-WS&pid=3084&remote_hostname=api.model-gateway.lab&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityFlows(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":1`),
		[]byte(`"flow_id":"flow-abc"`),
		[]byte(`"process_guid":"proc-abc"`),
		[]byte(`"pid":3084`),
		[]byte(`"remote_hostname":"api.model-gateway.lab"`),
		[]byte(`"attribution_confidence":0.98`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected flow response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityDNSReturnsNormalizedObservations(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-dns-1","event_type":"aegis.dns.observed","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":5,"payload":{"query":"api.model-gateway.lab","query_type":"A","answers":["203.0.113.10"],"resolver":"10.10.20.1","process_guid":"proc-abc","pid":3084,"correlation_method":"fixture.pid","correlation_confidence":0.91}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/dns?device_id=RMARTINEZ-WS&pid=3084&query=api.model-gateway.lab&answer=203.0.113.10&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityDNS(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":1`),
		[]byte(`"query":"api.model-gateway.lab"`),
		[]byte(`"answers":["203.0.113.10"]`),
		[]byte(`"process_guid":"proc-abc"`),
		[]byte(`"pid":3084`),
		[]byte(`"correlation_confidence":0.91`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected DNS response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityInvestigationReturnsProcessFlowAndDNSPath(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-process-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"process_guid":"proc-abc","parent_process_guid":"proc-parent","pid":3084,"ppid":200,"name":"python.exe","path":"C:\\AegisLab\\scripts\\agent_runner.py","command_line":"python.exe agent_runner.py","user":"RMARTINEZ\\tester","collection_method":"snapshot"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-flow-1","event_type":"aegis.flow.started","timestamp_ms":1777075005617,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":4,"payload":{"flow_id":"flow-abc","process_guid":"proc-abc","pid":3084,"process_name":"python.exe","user":"RMARTINEZ\\tester","protocol":"tcp","direction":"outbound","local_ip":"10.10.20.55","local_port":52944,"remote_ip":"203.0.113.10","remote_port":443,"remote_hostname":"api.model-gateway.lab","attribution_method":"fixture.pid","attribution_confidence":0.98}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-dns-1","event_type":"aegis.dns.observed","timestamp_ms":1777075005618,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":5,"payload":{"query":"api.model-gateway.lab","query_type":"A","answers":["203.0.113.10"],"resolver":"10.10.20.1","process_guid":"proc-abc","pid":3084,"correlation_method":"fixture.pid","correlation_confidence":0.91}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/investigation?device_id=RMARTINEZ-WS&process_guid=proc-abc&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityInvestigation(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"device_id":"RMARTINEZ-WS"`),
		[]byte(`"processes":1`),
		[]byte(`"flows":1`),
		[]byte(`"dns":1`),
		[]byte(`"process_guid":"proc-abc"`),
		[]byte(`"flow_id":"flow-abc"`),
		[]byte(`"query":"api.model-gateway.lab"`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected investigation response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityFindingsReturnsDetectionsAndRiskFindings(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-detection-1","event_type":"aegis.agent.detected","timestamp_ms":1777075005619,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":6,"payload":{"detection_id":"det-abc","process_guid":"proc-abc","flow_id":"flow-abc","classification":"script_or_agent_runtime","agent_likelihood":0.87,"confidence":0.82,"risk_score":42,"detected_patterns":["python_runtime","model_api_destination"],"evidence":[{"type":"command_line","value":"python.exe agent_runner.py","confidence":0.9}],"recommended_action":"review"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-finding-1","event_type":"aegis.risk_finding.created","timestamp_ms":1777075005620,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":7,"payload":{"finding_id":"finding-abc","severity":"medium","risk_score":42,"title":"Likely local AI agent script observed","description":"Python process launched from developer tooling made an outbound model gateway connection.","process_guid":"proc-abc","flow_id":"flow-abc","detection_id":"det-abc","evidence":[{"type":"classification","value":"script_or_agent_runtime","confidence":0.82}],"recommended_action":"review"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/findings?device_id=RMARTINEZ-WS&process_guid=proc-abc&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityFindings(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":2`),
		[]byte(`"detection_id":"det-abc"`),
		[]byte(`"finding_id":"finding-abc"`),
		[]byte(`"classification":"script_or_agent_runtime"`),
		[]byte(`"severity":"medium"`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected findings response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityInvestigationIncludesFindings(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-process-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"process_guid":"proc-abc","pid":3084,"name":"python.exe","collection_method":"snapshot"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-detection-1","event_type":"aegis.agent.detected","timestamp_ms":1777075005619,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":6,"payload":{"detection_id":"det-abc","process_guid":"proc-abc","flow_id":null,"classification":"script_or_agent_runtime","agent_likelihood":0.87,"confidence":0.82,"risk_score":42,"detected_patterns":["python_runtime"],"evidence":[{"type":"command_line","value":"python.exe agent_runner.py","confidence":0.9}],"recommended_action":"review"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/investigation?device_id=RMARTINEZ-WS&process_guid=proc-abc&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityInvestigation(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"findings":1`),
		[]byte(`"detection_id":"det-abc"`),
		[]byte(`"classification":"script_or_agent_runtime"`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected investigation response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleVisibilityDraftControlsReturnsObserveOnlyControl(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-process-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":3,"payload":{"process_guid":"proc-abc","pid":3084,"name":"python.exe","path":"C:\\AegisLab\\scripts\\agent_runner.py","command_line":"python.exe agent_runner.py"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-flow-1","event_type":"aegis.flow.started","timestamp_ms":1777075005617,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":4,"payload":{"flow_id":"flow-abc","process_guid":"proc-abc","pid":3084,"process_name":"python.exe","protocol":"tcp","direction":"outbound","remote_ip":"203.0.113.10","remote_port":443}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-dns-1","event_type":"aegis.dns.observed","timestamp_ms":1777075005618,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":5,"payload":{"query":"api.model-gateway.lab","answers":["203.0.113.10"],"process_guid":"proc-abc","pid":3084}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-finding-1","event_type":"aegis.risk_finding.created","timestamp_ms":1777075005619,"source":"aegis-windows-agent","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":6,"payload":{"finding_id":"finding-abc","severity":"medium","risk_score":42,"title":"Likely local AI agent script observed","process_guid":"proc-abc","flow_id":"flow-abc","recommended_action":"review"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/draft-controls?device_id=RMARTINEZ-WS&process_guid=proc-abc&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityDraftControls(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":1`),
		[]byte(`"mode":"observe-only"`),
		[]byte(`"status":"draft"`),
		[]byte(`"target":"203.0.113.10:443"`),
		[]byte(`"blast_radius"`),
		[]byte(`Likely local AI agent script observed`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected draft controls response to include %s, got %s", expected, getRec.Body.String())
		}
	}
}

func TestHandleClarionEventExportReturnsContractEvents(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-flow-1","event_type":"aegis.flow.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","tenant_id":"tenant-a","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":4,"payload":{"flow_id":"flow-abc","process_guid":"proc-abc","pid":3084,"remote_hostname":"api.model-gateway.lab"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-dns-1","event_type":"aegis.dns.observed","timestamp_ms":1777075005617,"source":"aegis-windows-agent","tenant_id":"tenant-b","device_id":"RMARTINEZ-WS","agent_id":"windows-agent-dev","sensor_version":"0.1.0","sequence":5,"payload":{"query":"api.model-gateway.lab","answers":["203.0.113.10"]}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/clarion/events?tenant_id=tenant-a&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleClarionEventExport(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"contract_version":"aegis-clarion.export.v1"`),
		[]byte(`"count":1`),
		[]byte(`"tenant_id":"tenant-a"`),
		[]byte(`"event_id":"evt-flow-1"`),
		[]byte(`"clarion_context_objects":["Device","Agent","User","Process","Flow","Destination"]`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected Clarion export response to include %s, got %s", expected, getRec.Body.String())
		}
	}
	if bytes.Contains(getRec.Body.Bytes(), []byte(`evt-dns-1`)) {
		t.Fatalf("expected tenant filter to exclude tenant-b event, got %s", getRec.Body.String())
	}
}

func TestHandleVisibilityDevicesReturnsDeviceSummaries(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/visibility-events.jsonl")
	if err != nil {
		t.Fatalf("newFileVisibilityStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &IngestServer{
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})),
		validator: mockValidator{},
		publisher: &mockPublisher{},
		dedupe:    newDuplicateTracker(100),
		store:     store,
		metrics:   sharedTestMetrics,
		checker:   health.NewServiceChecker(slog.Default()),
	}

	body := `{"schema_version":"visibility.v1","event_id":"evt-win-1","event_type":"aegis.process.started","timestamp_ms":1777075005616,"source":"aegis-windows-agent","tenant_id":"tenant-a","device_id":"windows-dev-agent-01","agent_id":"windows-dev-agent-01","sensor_version":"0.1.0","sequence":3,"payload":{"pid":3084,"name":"svchost.exe"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-linux-1","event_type":"aegis.flow.started","timestamp_ms":1777075005617,"source":"aegis-linux-agent","tenant_id":"tenant-a","device_id":"linux-dev-agent-01","agent_id":"linux-dev-agent-01","sensor_version":"0.1.0","sequence":4,"payload":{"flow_id":"flow-abc","remote_ip":"203.0.113.10"}}
` +
		`{"schema_version":"visibility.v1","event_id":"evt-win-2","event_type":"aegis.dns.observed","timestamp_ms":1777075005618,"source":"aegis-windows-agent","tenant_id":"tenant-a","device_id":"windows-dev-agent-01","agent_id":"windows-dev-agent-01","sensor_version":"0.1.0","sequence":5,"payload":{"query":"api.model-gateway.lab"}}`
	postReq := httptest.NewRequest(http.MethodPost, "/v1/visibility/events", bytes.NewBufferString(body))
	postRec := httptest.NewRecorder()
	server.handleVisibilityEvents(postRec, postReq)
	if postRec.Code != http.StatusAccepted {
		t.Fatalf("expected POST status %d, got %d: %s", http.StatusAccepted, postRec.Code, postRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/visibility/devices?tenant_id=tenant-a&limit=10", nil)
	getRec := httptest.NewRecorder()
	server.handleVisibilityDevices(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected GET status %d, got %d: %s", http.StatusOK, getRec.Code, getRec.Body.String())
	}
	for _, expected := range [][]byte{
		[]byte(`"count":2`),
		[]byte(`"device_id":"windows-dev-agent-01"`),
		[]byte(`"device_id":"linux-dev-agent-01"`),
		[]byte(`"source":"aegis-linux-agent"`),
		[]byte(`"aegis.dns.observed":1`),
	} {
		if !bytes.Contains(getRec.Body.Bytes(), expected) {
			t.Fatalf("expected devices response to include %s, got %s", expected, getRec.Body.String())
		}
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
