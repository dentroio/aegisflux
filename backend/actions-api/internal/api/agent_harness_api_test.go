package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backend/actionsapi/internal/agentsharness"
)

func TestAgentHarnessJobsAndRunDetail(t *testing.T) {
	s := NewServer()
	body := bytes.NewReader([]byte(`{"agent_id":"endpoint_analyst","device_id":"lab-1","context":{"k":1}}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/platform/ai/agent-harness/jobs", body)
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("jobs status %d %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		Run struct {
			RunID     string `json:"run_id"`
			ToolCalls []struct {
				ToolID     string `json:"tool_id"`
				DurationMS int64  `json:"duration_ms"`
			} `json:"tool_calls"`
		} `json:"run"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Run.RunID == "" || len(payload.Run.ToolCalls) != 3 {
		t.Fatalf("unexpected run payload: %+v", payload.Run)
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/platform/ai/agent-harness/runs/"+payload.Run.RunID, nil)
	s.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("detail status %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/platform/ai/agent-harness/runs?device_id=lab-1", nil)
	s.Handler().ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("list status %d", rec3.Code)
	}
}

func TestEndpointEvidenceAnalystUsesHarness(t *testing.T) {
	s := NewServer()
	reqBody := bytes.NewReader([]byte(`{"device_id":"d1","finding_id":"f1","context":{"x":1}}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/platform/ai/endpoint-evidence-analyst", reqBody)
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	s.platform.mu.Lock()
	n := len(s.platform.AgentHarnessRuns)
	s.platform.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 harness run, got %d", n)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	ebc, ok := resp["evidence_bound_conclusion"].(map[string]any)
	if !ok || ebc["conclusion"] == nil {
		t.Fatalf("missing evidence_bound_conclusion: %+v", resp)
	}
	if ebc["confidence_bucket"] == nil || ebc["confidence_rationale"] == nil {
		t.Fatalf("incomplete evidence bound fields: %+v", ebc)
	}
}

func TestAgentHarnessAgentsAndTools(t *testing.T) {
	s := NewServer()
	for _, path := range []string{"/platform/ai/agent-harness/agents", "/platform/ai/agent-harness/tools"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status %d", path, rec.Code)
		}
	}
}

func TestAgentHarnessInvalidJob(t *testing.T) {
	s := NewServer()
	body := bytes.NewReader([]byte(`{"agent_id":"unknown","device_id":"x"}`))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/platform/ai/agent-harness/jobs", body)
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rec.Code)
	}
}

func TestHarnessToolRegistryMutatesFalse(t *testing.T) {
	for _, tm := range agentsharness.ToolRegistry() {
		if tm.Mutates {
			t.Fatalf("tool %s must be read-only", tm.ID)
		}
	}
}

func TestHarnessAgentsListIncludesWOAgents(t *testing.T) {
	ids := map[string]bool{}
	for _, a := range agentsharness.SystemAgents {
		ids[a.ID] = true
	}
	for _, want := range []string{
		agentsharness.AgentEndpointAnalyst,
		agentsharness.AgentDetectionResearcher,
		agentsharness.AgentPackAuthor,
		agentsharness.AgentSimulationAgent,
		agentsharness.AgentControlDesigner,
		agentsharness.AgentGovernanceReviewer,
	} {
		if !ids[want] {
			t.Fatalf("missing agent %s", want)
		}
	}
}

func TestHarnessJobLifecycleCompleted(t *testing.T) {
	s := NewServer()
	body := strings.NewReader(`{"agent_id":"pack_author","device_id":"z","context":{}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/platform/ai/agent-harness/jobs", body)
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code, rec.Body.String())
	}
	var out struct {
		Job struct {
			Status string `json:"status"`
		} `json:"job"`
		Run struct {
			Status    string `json:"status"`
			ToolCalls []any  `json:"tool_calls"`
		} `json:"run"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Job.Status != agentsharness.JobCompleted || out.Run.Status != agentsharness.JobCompleted {
		t.Fatalf("status job=%s run=%s", out.Job.Status, out.Run.Status)
	}
	if len(out.Run.ToolCalls) != 1 {
		t.Fatalf("pack_author should run 1 tool, got %d", len(out.Run.ToolCalls))
	}
}
