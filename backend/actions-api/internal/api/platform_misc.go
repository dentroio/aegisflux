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
		d.CreatedMS = time.Now().UnixMilli()
		if len(d.ScopeSelectors) == 0 {
			d.ScopeSelectors = []string{"device:*"}
		}
		if len(d.EvidenceRefs) == 0 {
			d.EvidenceRefs = []string{d.SourceFindingID}
		}
		if d.BlastRadius == "" {
			d.BlastRadius = "single-device observe-only projection"
		}
		if d.RollbackPlan == "" {
			d.RollbackPlan = "disable draft simulation; retain evidence anchors"
		}
		s.platform.mu.Lock()
		s.platform.Drafts = append(s.platform.Drafts, d)
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.created",
			Status:      d.Status,
			Subject:     d.ID,
			Description: "Observe-only draft control recorded",
			CreatedMS:   time.Now().UnixMilli(),
		})
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusCreated, map[string]any{"id": d.ID})
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
	if !strings.HasSuffix(path, "/simulate") || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	id := strings.TrimSuffix(path, "/simulate")
	var body struct {
		DeviceID string `json:"device_id"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	deviceID := strings.TrimSpace(body.DeviceID)
	s.platform.mu.Lock()
	found := false
	for i := range s.platform.Drafts {
		if s.platform.Drafts[i].ID != id {
			continue
		}
		found = true
		matches := simulateMatches(deviceID + id)
		s.platform.Drafts[i].SimulationMatches = matches
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.simulated",
			Status:      "observe_only",
			Subject:     id,
			DeviceID:    deviceID,
			Description: fmt.Sprintf("Historical window matched %d events (lab projection)", matches),
			CreatedMS:   time.Now().UnixMilli(),
		})
		jsonWrite(w, http.StatusOK, map[string]any{"draft_id": id, "matched_events": matches})
		break
	}
	s.platform.mu.Unlock()
	if !found {
		http.NotFound(w, r)
	}
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
