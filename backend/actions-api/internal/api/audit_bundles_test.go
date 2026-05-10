package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServerForAudit(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		mux:      http.NewServeMux(),
		platform: newPlatformData(),
	}
	s.registerPlatformRoutes()
	return s
}

func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json: %v / body=%s", err, string(body))
	}
	return out
}

func doJSON(t *testing.T, s *Server, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	return rec
}

func TestAuditBundle_CreateRejectsNonAuditMode(t *testing.T) {
	s := newTestServerForAudit(t)
	rec := doJSON(t, s, http.MethodPost, "/platform/audit-bundles", map[string]any{
		"title": "Block all egress",
		"mode":  "enforce",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "audit") {
		t.Fatalf("expected mode error, got %s", rec.Body.String())
	}
}

func TestAuditBundle_CreateRequiresTitle(t *testing.T) {
	s := newTestServerForAudit(t)
	rec := doJSON(t, s, http.MethodPost, "/platform/audit-bundles", map[string]any{
		"title": "",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAuditBundle_StagingAndStatusReporting(t *testing.T) {
	s := newTestServerForAudit(t)
	rec := doJSON(t, s, http.MethodPost, "/platform/audit-bundles", map[string]any{
		"title":             "Audit Ollama listen ports",
		"description":       "Observe-only audit for ollama serve on 0.0.0.0",
		"scope":             []string{"device:linux-lab-1"},
		"expected_match_telemetry": []string{"process.listen_port==11434"},
		"rollback_notes":    "Revoke bundle id; agents return to pending.",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d body=%s", rec.Code, rec.Body.String())
	}
	created := decode[AuditBundle](t, rec.Body.Bytes())
	if created.Status != auditStatusDraft {
		t.Fatalf("expected draft status, got %q", created.Status)
	}
	if created.Mode != auditModeAudit {
		t.Fatalf("expected audit mode, got %q", created.Mode)
	}
	if len(created.History) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(created.History))
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/stage", map[string]any{
		"device_ids": []string{"linux-lab-1", "linux-lab-2"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("stage: %d body=%s", rec.Code, rec.Body.String())
	}
	staged := decode[AuditBundle](t, rec.Body.Bytes())
	if staged.Status != auditStatusStaged {
		t.Fatalf("expected staged, got %q", staged.Status)
	}
	if len(staged.EndpointStatuses) != 2 {
		t.Fatalf("expected 2 endpoint statuses, got %d", len(staged.EndpointStatuses))
	}
	for _, st := range staged.EndpointStatuses {
		if st.Status != endpointStatusPending {
			t.Fatalf("expected pending, got %q for %s", st.Status, st.DeviceID)
		}
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/status", map[string]any{
		"device_id":     "linux-lab-1",
		"status":        endpointStatusAccepted,
		"agent_version": "1.4.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
	updated := decode[AuditBundle](t, rec.Body.Bytes())
	var lab1 AuditBundleStatus
	for _, st := range updated.EndpointStatuses {
		if st.DeviceID == "linux-lab-1" {
			lab1 = st
		}
	}
	if lab1.Status != endpointStatusAccepted {
		t.Fatalf("expected accepted, got %q", lab1.Status)
	}
	if lab1.AgentVersion != "1.4.0" {
		t.Fatalf("agent_version not recorded: %q", lab1.AgentVersion)
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/status", map[string]any{
		"device_id": "linux-lab-2",
		"status":    "blocking",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported status, got %d", rec.Code)
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/match", map[string]any{
		"device_id": "linux-lab-1",
		"process":   "ollama",
		"indicator": "listen_port==11434",
		"detail":    "ollama serve --host 0.0.0.0",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("match: %d body=%s", rec.Code, rec.Body.String())
	}
	matched := decode[AuditBundle](t, rec.Body.Bytes())
	if len(matched.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matched.Matches))
	}
	if matched.Matches[0].DeviceID != "linux-lab-1" {
		t.Fatalf("match device_id wrong: %q", matched.Matches[0].DeviceID)
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/revoke", map[string]any{
		"note": "lab teardown",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke: %d body=%s", rec.Code, rec.Body.String())
	}
	revoked := decode[AuditBundle](t, rec.Body.Bytes())
	if revoked.Status != auditStatusRevoked {
		t.Fatalf("expected revoked, got %q", revoked.Status)
	}

	rec = doJSON(t, s, http.MethodPost, "/platform/audit-bundles/"+created.ID+"/match", map[string]any{
		"device_id": "linux-lab-1",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 after revoke, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSummarizeEndpointStatuses(t *testing.T) {
	out := summarizeEndpointStatuses([]AuditBundleStatus{
		{DeviceID: "a", Status: endpointStatusAccepted, LastMatchAtMS: 1},
		{DeviceID: "b", Status: endpointStatusPending},
		{DeviceID: "c", Status: endpointStatusRejected},
		{DeviceID: "d", Status: endpointStatusIncompatible},
		{DeviceID: "e", Status: endpointStatusStale},
		{DeviceID: "f", Status: endpointStatusAccepted, LastMatchAtMS: 2},
	})
	if out.Total != 6 {
		t.Fatalf("total: %d", out.Total)
	}
	if out.Accepted != 2 || out.Pending != 1 || out.Rejected != 1 || out.Incompatible != 1 || out.Stale != 1 {
		t.Fatalf("status counts wrong: %#v", out)
	}
	if out.WithMatches != 2 {
		t.Fatalf("with matches wrong: %d", out.WithMatches)
	}
}
