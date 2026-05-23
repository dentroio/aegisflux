package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetAgentsPagination(t *testing.T) {
	s := NewServer()
	now := time.Now().UTC()

	s.store.mu.Lock()
	for i := 0; i < 5; i++ {
		uid := fmt.Sprintf("agent-%d", i)
		s.store.agents[uid] = &Agent{
			AgentUID: uid,
			OrgID:    "default-org",
			HostID:   uid,
			LastSeen: now.Add(-time.Duration(i) * time.Minute),
			Created:  now,
		}
	}
	s.store.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/agents?limit=2&offset=1", nil)
	rec := httptest.NewRecorder()
	s.getAgents(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp AgentListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 5 {
		t.Fatalf("total=%d want 5", resp.Total)
	}
	if len(resp.Agents) != 2 {
		t.Fatalf("page len=%d want 2", len(resp.Agents))
	}
}

func TestHandleBatchHeartbeats(t *testing.T) {
	s := NewServer()
	body := []byte(`{
	  "heartbeats": [
	    {"agent_uid":"batch-1","host_id":"host-1","last_seen":"2026-05-17T12:00:00Z"},
	    {"agent_uid":"batch-2","host_id":"host-2","last_seen":"2026-05-17T12:00:00Z"}
	  ]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/agents/heartbeats", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.handleBatchHeartbeats(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	if len(s.store.agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(s.store.agents))
	}
}
