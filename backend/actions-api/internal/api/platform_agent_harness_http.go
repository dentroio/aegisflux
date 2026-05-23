package api

import (
	"net/http"
	"sort"
	"strings"

	"backend/actionsapi/internal/agentsharness"
)

func (s *Server) handleAgentHarness(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/platform/ai/agent-harness/")
	rest = strings.Trim(rest, "/")
	switch {
	case rest == "agents" && r.Method == http.MethodGet:
		jsonWrite(w, http.StatusOK, map[string]any{"agents": agentsharness.SystemAgents})
	case rest == "tools" && r.Method == http.MethodGet:
		jsonWrite(w, http.StatusOK, map[string]any{"tools": agentsharness.ToolRegistry()})
	case rest == "runs" && r.Method == http.MethodGet:
		s.listAgentHarnessRuns(w, r)
	case rest == "jobs" && r.Method == http.MethodPost:
		s.postAgentHarnessJob(w, r)
	case strings.HasPrefix(rest, "runs/"):
		id := strings.TrimPrefix(rest, "runs/")
		id = strings.Trim(id, "/")
		if r.Method == http.MethodGet && id != "" {
			s.getAgentHarnessRun(w, r, id)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listAgentHarnessRuns(w http.ResponseWriter, r *http.Request) {
	agentID := strings.TrimSpace(r.URL.Query().Get("agent_id"))
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	findingID := strings.TrimSpace(r.URL.Query().Get("finding_id"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	s.platform.mu.Lock()
	src := append([]agentsharness.RunRecord(nil), s.platform.AgentHarnessRuns...)
	s.platform.mu.Unlock()

	out := make([]agentsharness.RunRecord, 0, len(src))
	for i := len(src) - 1; i >= 0; i-- {
		run := src[i]
		if agentID != "" && run.AgentID != agentID {
			continue
		}
		if deviceID != "" && run.DeviceID != deviceID {
			continue
		}
		if findingID != "" && run.FindingID != findingID {
			continue
		}
		if status != "" && run.Status != status {
			continue
		}
		out = append(out, run)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedMS > out[j].StartedMS })
	jsonWrite(w, http.StatusOK, map[string]any{"runs": out})
}

func (s *Server) getAgentHarnessRun(w http.ResponseWriter, r *http.Request, runID string) {
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.AgentHarnessRuns {
		if s.platform.AgentHarnessRuns[i].RunID == runID {
			jsonWrite(w, http.StatusOK, s.platform.AgentHarnessRuns[i])
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) postAgentHarnessJob(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AgentID          string         `json:"agent_id"`
		DeviceID         string         `json:"device_id"`
		FindingID        string         `json:"finding_id"`
		CandidateID      string         `json:"candidate_id"`
		Context          map[string]any `json:"context"`
		ProductImpacting *bool          `json:"product_impacting"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	agentID := strings.TrimSpace(body.AgentID)
	deviceID := strings.TrimSpace(body.DeviceID)
	if agentID == "" || deviceID == "" {
		http.Error(w, `{"error":"agent_id and device_id required"}`, http.StatusBadRequest)
		return
	}
	job, run, err := s.runAgentHarness(agentsharness.RunSpec{
		AgentID:          agentID,
		DeviceID:         deviceID,
		FindingID:        strings.TrimSpace(body.FindingID),
		CandidateID:      strings.TrimSpace(body.CandidateID),
		Context:          body.Context,
		ProductImpacting: body.ProductImpacting,
	})
	if err != nil {
		jsonWrite(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	jsonWrite(w, http.StatusOK, map[string]any{"job": job, "run": run})
}
