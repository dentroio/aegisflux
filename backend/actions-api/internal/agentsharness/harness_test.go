package agentsharness

import (
	"context"
	"encoding/json"
	"testing"
)

type mockDS struct {
	allowExt bool
	prov     string
}

func (m *mockDS) AllowExternalAI() bool          { return m.allowExt }
func (m *mockDS) DefaultProviderKind() string    { return m.prov }
func (m *mockDS) RedactJSONPreview(v any) string { b, _ := json.Marshal(v); return string(b) }
func (m *mockDS) DeviceEvidenceSummary(deviceID string) map[string]any {
	return map[string]any{"device_id": deviceID, "ok": true}
}
func (m *mockDS) FindingsEvidencePaths(deviceID, findingID string) []map[string]string {
	return []map[string]string{{"device_id": deviceID, "finding_id": findingID}}
}
func (m *mockDS) DetectionCandidateLookup(candidateID, deviceID string) map[string]any {
	return map[string]any{"device_id": deviceID, "candidate_id": candidateID, "stub": true}
}

func TestHarnessRunEndpointAnalystToolAudit(t *testing.T) {
	h := &Harness{Data: &mockDS{allowExt: false, prov: "local"}}
	job, run, err := h.Run(context.Background(), RunSpec{
		AgentID:  AgentEndpointAnalyst,
		DeviceID: "dev-1",
		Context:  map[string]any{"n": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != JobCompleted {
		t.Fatalf("job status %s", job.Status)
	}
	if run.Status != JobCompleted {
		t.Fatalf("run status %s", run.Status)
	}
	if len(run.ToolCalls) != 3 {
		t.Fatalf("tool calls %d", len(run.ToolCalls))
	}
	for _, tc := range run.ToolCalls {
		if tc.DurationMS < 0 {
			t.Fatalf("bad duration")
		}
		if tc.Error != "" {
			t.Fatalf("unexpected tool err: %s", tc.Error)
		}
	}
	if run.PromptRedactedPreview == "" {
		t.Fatal("expected redacted preview")
	}
}

func TestHarnessUnknownAgent(t *testing.T) {
	h := &Harness{Data: &mockDS{}}
	_, _, err := h.Run(context.Background(), RunSpec{AgentID: "nope", DeviceID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}
