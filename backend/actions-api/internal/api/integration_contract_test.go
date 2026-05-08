package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegrationEvidenceSummarySchema(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/platform/integration/devices/lab-device-001", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"device_id", "agent_id", "ai_activity_summary", "inventory_summary", "finding_links", "integration_event_names"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("missing %s", key)
		}
	}
}

func TestDraftControlObserveOnlySimulation(t *testing.T) {
	s := NewServer()
	create := bytes.NewReader([]byte(`{"source_finding_id":"f1","proposed_action":"observe route x","scope_selectors":["device:abc"]}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/platform/draft-controls", create)
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status %d %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(body["id"])
	if id == "" {
		t.Fatal("missing id")
	}

	sim := bytes.NewReader([]byte(`{"device_id":"dev-xyz"}`))
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/platform/draft-controls/"+id+"/simulate", sim)
	req2.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("sim status %d %s", rec2.Code, rec2.Body.String())
	}
}
