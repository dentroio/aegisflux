package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/actionsapi/internal/agentsharness"

	"github.com/google/uuid"
)

func (s *Server) mergeProviderSecretsLocked() []AIProviderDTO {
	out := make([]AIProviderDTO, len(s.platform.Providers))
	copy(out, s.platform.Providers)
	for i := range out {
		out[i].SecretConfigured = strings.TrimSpace(s.platform.providerSecrets[out[i].ID]) != ""
	}
	return out
}

func (s *Server) handlePlatformAIProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		list := s.mergeProviderSecretsLocked()
		def := s.platform.DefaultProvider
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusOK, map[string]any{"providers": list, "default_provider_id": def})
	case http.MethodPost:
		var body struct {
			Kind    string `json:"kind"`
			Name    string `json:"name"`
			Enabled *bool  `json:"enabled"`
			Secret  string `json:"secret"`
		}
		if !jsonRead(w, r, &body) {
			return
		}
		id := strings.ToLower(strings.TrimSpace(body.Kind))
		if id == "" {
			http.Error(w, `{"error":"kind required"}`, http.StatusBadRequest)
			return
		}
		s.platform.mu.Lock()
		found := -1
		for i := range s.platform.Providers {
			if s.platform.Providers[i].ID == id || s.platform.Providers[i].Kind == id {
				found = i
				break
			}
		}
		if found < 0 {
			s.platform.Providers = append(s.platform.Providers, AIProviderDTO{ID: id, Kind: id, Name: body.Name, Enabled: true})
			found = len(s.platform.Providers) - 1
		}
		if body.Name != "" {
			s.platform.Providers[found].Name = body.Name
		}
		if body.Enabled != nil {
			s.platform.Providers[found].Enabled = *body.Enabled
		}
		if strings.TrimSpace(body.Secret) != "" {
			s.platform.providerSecrets[s.platform.Providers[found].ID] = body.Secret
		}
		s.platform.appendAudit("ai_provider_upsert", "", id, false)
		s.platform.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePlatformAIProvidersConfigure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DefaultProviderID string `json:"default_provider_id"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	s.platform.DefaultProvider = strings.TrimSpace(body.DefaultProviderID)
	s.platform.appendAudit("ai_default_provider", "", s.platform.DefaultProvider, false)
	s.platform.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePlatformAIProvidersSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.platform.mu.Lock()
	list := s.mergeProviderSecretsLocked()
	s.platform.mu.Unlock()
	ok := 0
	for _, p := range list {
		if p.Enabled && p.LastHealthOK {
			ok++
		}
	}
	jsonWrite(w, http.StatusOK, map[string]any{
		"healthy_providers": ok,
		"total_providers":   len(list),
		"summary":           fmt.Sprintf("%d/%d healthy", ok, len(list)),
	})
}

func (s *Server) handlePlatformAIProviderSubpath(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/platform/ai/providers/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "test":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.platform.mu.Lock()
		s.platform.touchProviderHealth(id, true, "mock connectivity ok (lab)")
		s.platform.appendAudit("ai_provider_test", "", id, false)
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusOK, map[string]any{"ok": true, "latency_ms": 12, "message": "mock test (no external call)"})
	case "health":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.platform.mu.Lock()
		var dto *AIProviderDTO
		for i := range s.platform.Providers {
			if s.platform.Providers[i].ID == id {
				d := s.platform.Providers[i]
				dto = &d
				break
			}
		}
		s.platform.mu.Unlock()
		if dto == nil {
			http.NotFound(w, r)
			return
		}
		jsonWrite(w, http.StatusOK, map[string]any{
			"id":         dto.ID,
			"ok":         dto.LastHealthOK,
			"checked_ms": dto.LastHealthMS,
			"message":    dto.LastHealthMsg,
		})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handlePlatformAIPrivacy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		p := s.platform.Privacy
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusOK, p)
	case http.MethodPut:
		var body PrivacySettings
		if !jsonRead(w, r, &body) {
			return
		}
		s.platform.mu.Lock()
		s.platform.Privacy = body
		s.platform.appendAudit("privacy_settings_update", "", "privacy", false)
		s.platform.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePlatformAIRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	s.platform.mu.Lock()
	out := make([]AIRunRecord, 0, len(s.platform.Runs))
	for _, run := range s.platform.Runs {
		if deviceID != "" && run.DeviceID != deviceID {
			continue
		}
		out = append(out, run)
	}
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, map[string]any{"runs": out})
}

func (s *Server) handleEndpointEvidenceAnalyst(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DeviceID  string         `json:"device_id"`
		FindingID string         `json:"finding_id"`
		Context   map[string]any `json:"context"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	deviceID := strings.TrimSpace(body.DeviceID)
	if deviceID == "" {
		http.Error(w, `{"error":"device_id required"}`, http.StatusBadRequest)
		return
	}
	_, runDetail, err := s.runAgentHarness(agentsharness.RunSpec{
		AgentID:   agentsharness.AgentEndpointAnalyst,
		DeviceID:  deviceID,
		FindingID: strings.TrimSpace(body.FindingID),
		Context:   body.Context,
	})
	if err != nil {
		http.Error(w, `{"error":"harness failed"}`, http.StatusInternalServerError)
		return
	}
	if runDetail.Status != agentsharness.JobCompleted {
		s.platform.mu.Lock()
		s.platform.appendAudit("endpoint_evidence_analyst_failed", deviceID, runDetail.RunID, true)
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusUnprocessableEntity, map[string]any{
			"error":                            "evidence_bound_or_harness_failed",
			"run_id":                           runDetail.RunID,
			"status":                           runDetail.Status,
			"evidence_bound_validation_errors": runDetail.EvidenceBoundValidationErrors,
			"evidence_bound_conclusion":        runDetail.EvidenceBoundConclusion,
		})
		return
	}

	run := AIRunRecord{
		RunID:          runDetail.RunID,
		AgentID:        "endpoint_evidence_analyst",
		DeviceID:       deviceID,
		CreatedMS:      runDetail.StartedMS,
		ProviderKind:   runDetail.ProviderKind,
		Status:         runDetail.Status,
		Redacted:       true,
		PrivacyApplied: runDetail.PrivacyApplied,
		Assessment:     runDetail.Assessment,
		Evidence:       runDetail.EvidenceSummary,
		Confidence:     runDetail.Confidence,
		NextAction:     runDetail.RecommendedNextAction,
	}
	s.platform.mu.Lock()
	s.platform.appendRun(run)
	s.platform.appendAudit("endpoint_evidence_analyst", deviceID, run.RunID, true)
	s.platform.Audit = append(s.platform.Audit, PrivacyAuditRecord{
		ID:        uuid.NewString(),
		CreatedMS: time.Now().UnixMilli(),
		Action:    "ai_run_completed",
		DeviceID:  deviceID,
		Detail:    run.RunID,
		Redacted:  true,
	})
	s.platform.mu.Unlock()

	jsonWrite(w, http.StatusOK, map[string]any{
		"run_id":                    run.RunID,
		"agent_id":                  "endpoint_evidence_analyst",
		"assessment":                runDetail.Assessment,
		"evidence":                  runDetail.EvidenceSummary,
		"confidence":                runDetail.Confidence,
		"recommended_next_action":   runDetail.RecommendedNextAction,
		"evidence_bound_conclusion": runDetail.EvidenceBoundConclusion,
	})
}

func (s *Server) handlePlatformOperationalEvents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filterType := r.URL.Query().Get("event_type")
		deviceID := r.URL.Query().Get("device_id")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 || limit > 500 {
			limit = 200
		}
		s.platform.mu.Lock()
		var out []OperationalEvent
		for i := len(s.platform.Events) - 1; i >= 0 && len(out) < limit; i-- {
			ev := s.platform.Events[i]
			if filterType != "" && ev.EventType != filterType {
				continue
			}
			if deviceID != "" && ev.DeviceID != deviceID {
				continue
			}
			out = append(out, ev)
		}
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusOK, map[string]any{"events": out})
	case http.MethodPost:
		var ev OperationalEvent
		if !jsonRead(w, r, &ev) {
			return
		}
		if strings.TrimSpace(ev.EventType) == "" {
			http.Error(w, `{"error":"event_type required"}`, http.StatusBadRequest)
			return
		}
		if ev.ID == "" {
			ev.ID = uuid.NewString()
		}
		if ev.CreatedMS == 0 {
			ev.CreatedMS = time.Now().UnixMilli()
		}
		s.platform.mu.Lock()
		s.platform.appendOp(ev)
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusCreated, map[string]any{"id": ev.ID})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
