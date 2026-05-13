package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgentsWorkbenchSummary_IngestFailureSurfacesDependency(t *testing.T) {
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/visibility/devices" {
			http.Error(w, "ingest unavailable", http.StatusBadGateway)
			return
		}
		http.NotFound(w, r)
	}))
	defer ingest.Close()

	t.Setenv("INGEST_API_URL", ingest.URL)
	s := NewServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/console/summary/agents-workbench", nil)
	s.getAgentsWorkbenchSummary(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body AgentListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency probe, got %d", len(body.Dependencies))
	}
	if body.Dependencies[0].Name != "ingest" || body.Dependencies[0].Status == "ok" {
		t.Fatalf("expected ingest dependency not ok, got %#v", body.Dependencies[0])
	}
}

func TestAgentsWorkbenchSummary_IngestOKOmitsDegradedDeps(t *testing.T) {
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/visibility/devices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"devices":[]}`))
	}))
	defer ingest.Close()

	t.Setenv("INGEST_API_URL", ingest.URL)
	s := NewServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/console/summary/agents-workbench", nil)
	s.getAgentsWorkbenchSummary(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body AgentListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Dependencies) != 1 || body.Dependencies[0].Status != "ok" {
		t.Fatalf("expected single ok ingest probe, got %#v", body.Dependencies)
	}
}
