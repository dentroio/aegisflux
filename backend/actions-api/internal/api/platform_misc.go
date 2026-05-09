package api

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func simulateMatches(deviceID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(deviceID))
	return int(h.Sum32()%127) + 3
}

func (s *Server) handleDraftControlsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		out := append([]DraftControl(nil), s.platform.Drafts...)
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusOK, map[string]any{"drafts": out})
	case http.MethodPost:
		var d DraftControl
		if !jsonRead(w, r, &d) {
			return
		}
		if strings.TrimSpace(d.SourceFindingID) == "" || strings.TrimSpace(d.ProposedAction) == "" {
			http.Error(w, `{"error":"source_finding_id and proposed_action required"}`, http.StatusBadRequest)
			return
		}
		if d.ID == "" {
			d.ID = uuid.NewString()
		}
		if d.Status == "" {
			d.Status = "draft_observe_only"
		}
		now := time.Now().UnixMilli()
		d.CreatedMS = now
		d.UpdatedMS = now
		if len(d.ScopeSelectors) == 0 {
			if d.SourceDeviceID != "" {
				d.ScopeSelectors = []string{"device:" + d.SourceDeviceID}
			} else {
				d.ScopeSelectors = []string{"device:*"}
			}
		}
		if len(d.EvidenceRefs) == 0 {
			d.EvidenceRefs = []string{d.SourceFindingID}
		}
		if d.Confidence == "" {
			d.Confidence = "medium"
		}
		if d.ExpectedBreakageRisk == "" {
			d.ExpectedBreakageRisk = "low (observe-only; no enforcement)"
		}
		if d.BlastRadius == "" {
			d.BlastRadius = "Single-device observe-only projection. Counts historical matches before any restrict or deny is considered."
		}
		if len(d.BlastRadiusNotes) == 0 {
			d.BlastRadiusNotes = []string{
				"Counts historical matches for the same scope without enforcing.",
				"Does not block traffic, processes, or browser activity.",
				"Re-run simulate after evidence updates to refresh the match count.",
			}
		}
		if d.RollbackPlan == "" {
			d.RollbackPlan = "Disable the draft and retain evidence anchors."
		}
		if len(d.RollbackSteps) == 0 {
			d.RollbackSteps = []string{
				"Set status to draft_archived to stop simulation cycles.",
				"Keep the source finding link so the audit trail stays intact.",
				"If a follow-up control was generated from this draft, archive it as well.",
			}
		}
		s.platform.mu.Lock()
		s.platform.Drafts = append(s.platform.Drafts, d)
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.created",
			Status:      d.Status,
			Subject:     d.ID,
			DeviceID:    d.SourceDeviceID,
			Description: fmt.Sprintf("Observe-only draft control recorded for finding %s", d.SourceFindingID),
			CreatedMS:   now,
		})
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusCreated, map[string]any{"id": d.ID, "draft": d})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDraftControlsItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/platform/draft-controls/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(path, "/simulate") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSuffix(path, "/simulate")
		s.simulateDraft(w, r, id)
		return
	}

	id := strings.TrimRight(path, "/")
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		var found *DraftControl
		for i := range s.platform.Drafts {
			if s.platform.Drafts[i].ID == id {
				clone := s.platform.Drafts[i]
				found = &clone
				break
			}
		}
		s.platform.mu.Unlock()
		if found == nil {
			http.NotFound(w, r)
			return
		}
		jsonWrite(w, http.StatusOK, found)
	case http.MethodPatch:
		s.patchDraft(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) simulateDraft(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		DeviceID string `json:"device_id"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	deviceID := strings.TrimSpace(body.DeviceID)
	s.platform.mu.Lock()
	found := false
	var snapshot DraftControl
	for i := range s.platform.Drafts {
		if s.platform.Drafts[i].ID != id {
			continue
		}
		found = true
		matches := simulateMatches(deviceID + id)
		now := time.Now().UnixMilli()
		s.platform.Drafts[i].SimulationMatches = matches
		s.platform.Drafts[i].SimulationDeviceID = deviceID
		s.platform.Drafts[i].SimulationAtMS = now
		s.platform.Drafts[i].UpdatedMS = now
		matchesCopy := matches
		s.platform.Drafts[i].ExpectedMatches = &matchesCopy
		snapshot = s.platform.Drafts[i]
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.simulated",
			Status:      "observe_only",
			Subject:     id,
			DeviceID:    deviceID,
			Description: fmt.Sprintf("Historical window matched %d events (lab projection)", matches),
			CreatedMS:   now,
		})
		break
	}
	s.platform.mu.Unlock()
	if !found {
		http.NotFound(w, r)
		return
	}
	jsonWrite(w, http.StatusOK, map[string]any{
		"draft_id":       id,
		"matched_events": *snapshot.ExpectedMatches,
		"draft":          snapshot,
	})
}

func (s *Server) patchDraft(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		OperatorNotes        *string  `json:"operator_notes,omitempty"`
		Status               *string  `json:"status,omitempty"`
		ExpectedBreakageRisk *string  `json:"expected_breakage_risk,omitempty"`
		ScopeSelectors       []string `json:"scope_selectors,omitempty"`
		BlastRadius          *string  `json:"blast_radius,omitempty"`
		RollbackPlan         *string  `json:"rollback_plan,omitempty"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	found := false
	var snapshot DraftControl
	for i := range s.platform.Drafts {
		if s.platform.Drafts[i].ID != id {
			continue
		}
		found = true
		now := time.Now().UnixMilli()
		if body.OperatorNotes != nil {
			s.platform.Drafts[i].OperatorNotes = *body.OperatorNotes
		}
		if body.Status != nil && strings.TrimSpace(*body.Status) != "" {
			s.platform.Drafts[i].Status = *body.Status
		}
		if body.ExpectedBreakageRisk != nil {
			s.platform.Drafts[i].ExpectedBreakageRisk = *body.ExpectedBreakageRisk
		}
		if body.ScopeSelectors != nil {
			s.platform.Drafts[i].ScopeSelectors = body.ScopeSelectors
		}
		if body.BlastRadius != nil {
			s.platform.Drafts[i].BlastRadius = *body.BlastRadius
		}
		if body.RollbackPlan != nil {
			s.platform.Drafts[i].RollbackPlan = *body.RollbackPlan
		}
		s.platform.Drafts[i].UpdatedMS = now
		snapshot = s.platform.Drafts[i]
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.updated",
			Status:      s.platform.Drafts[i].Status,
			Subject:     id,
			DeviceID:    s.platform.Drafts[i].SourceDeviceID,
			Description: "Observe-only draft control updated",
			CreatedMS:   now,
		})
		break
	}
	s.platform.mu.Unlock()
	if !found {
		http.NotFound(w, r)
		return
	}
	jsonWrite(w, http.StatusOK, snapshot)
}

func (s *Server) handleIntegrationDeviceEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/platform/integration/devices/")
	deviceID := strings.Trim(rest, "/")
	if deviceID == "" {
		http.NotFound(w, r)
		return
	}
	sample := fmt.Sprintf(`{"windows":{"device_id":"win-lab-host","signals":["dns","collector"]},"linux":{"device_id":"lin-lab-host","signals":["flows","collector"]}}`)
	jsonWrite(w, http.StatusOK, map[string]any{
		"schema":                            "aegisflux.integration.evidence_summary.v1",
		"device_id":                         deviceID,
		"agent_id":                          deviceID,
		"os_or_source":                      "visibility",
		"freshness_ms_hint":                 time.Now().UnixMilli(),
		"ai_activity_summary":               map[string]any{"signals": []string{"dns", "process", "findings"}, "count_hint": simulateMatches(deviceID)},
		"inventory_summary":                 map[string]any{"hints": []string{"browser_extensions", "sase"}},
		"finding_links":                     []map[string]string{{"finding_id": "example-" + deviceID, "relative": fmt.Sprintf("/analyze/findings?device=%s", deviceID)}},
		"integration_event_names":           []string{"aegis.device.observed", "aegis.ai_activity.summarized", "aegis.inventory.item_observed", "aegis.finding.created"},
		"sample_payload_windows_linux_json": sample,
	})
}

func (s *Server) handleRedactPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	priv := s.platform.Privacy
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, map[string]any{"redacted": RedactLine(body.Text, priv)})
}
