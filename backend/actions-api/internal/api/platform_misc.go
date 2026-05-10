package api

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func simulateMatches(deviceID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(deviceID))
	return int(h.Sum32()%127) + 3
}

// snapshotDraft returns the operator-editable subset of a draft for use in
// before/after history entries.
func snapshotDraft(d DraftControl) DraftSnapshot {
	var matches *int
	if d.ExpectedMatches != nil {
		v := *d.ExpectedMatches
		matches = &v
	}
	return DraftSnapshot{
		Status:           d.Status,
		ScopeSelectors:   append([]string(nil), d.ScopeSelectors...),
		BlastRadius:      d.BlastRadius,
		BlastRadiusNotes: append([]string(nil), d.BlastRadiusNotes...),
		RollbackPlan:     d.RollbackPlan,
		RollbackSteps:    append([]string(nil), d.RollbackSteps...),
		OperatorNotes:    d.OperatorNotes,
		ExpectedMatches:  matches,
		ExpectedBreakage: d.ExpectedBreakageRisk,
	}
}

// diffSnapshotKeys returns the list of operator-editable keys that changed
// between two snapshots. Used to record changed_keys on history entries.
func diffSnapshotKeys(before, after DraftSnapshot) []string {
	changes := []string{}
	if before.Status != after.Status {
		changes = append(changes, "status")
	}
	if !equalStringSlices(before.ScopeSelectors, after.ScopeSelectors) {
		changes = append(changes, "scope_selectors")
	}
	if before.BlastRadius != after.BlastRadius {
		changes = append(changes, "blast_radius")
	}
	if !equalStringSlices(before.BlastRadiusNotes, after.BlastRadiusNotes) {
		changes = append(changes, "blast_radius_notes")
	}
	if before.RollbackPlan != after.RollbackPlan {
		changes = append(changes, "rollback_plan")
	}
	if !equalStringSlices(before.RollbackSteps, after.RollbackSteps) {
		changes = append(changes, "rollback_steps")
	}
	if before.OperatorNotes != after.OperatorNotes {
		changes = append(changes, "operator_notes")
	}
	if !reflect.DeepEqual(before.ExpectedMatches, after.ExpectedMatches) {
		changes = append(changes, "expected_matches")
	}
	if before.ExpectedBreakage != after.ExpectedBreakage {
		changes = append(changes, "expected_breakage_risk")
	}
	sort.Strings(changes)
	return changes
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// appendDraftHistory appends a history entry to a draft (capped to keep the
// payload bounded). Status changes also bump the draft status if provided.
func appendDraftHistory(d *DraftControl, entry DraftDecisionEntry) {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.AtMS == 0 {
		entry.AtMS = time.Now().UnixMilli()
	}
	if entry.Actor == "" {
		entry.Actor = "operator"
	}
	d.History = append(d.History, entry)
	if len(d.History) > 50 {
		d.History = d.History[len(d.History)-50:]
	}
}

// buildSimulationResult produces a deterministic, operator-readable
// simulation result for a draft against an optional target device. The
// counts and lists are derived from a hash of the draft id + device id so
// repeated calls return the same projection. This keeps the lab demo stable
// without requiring access to the visibility store from actions-api.
func buildSimulationResult(d DraftControl, deviceID string, now int64) DraftSimulationResult {
	seed := deviceID + "::" + d.ID
	matches := simulateMatches(seed)
	devCount := derivedCount(seed+"::devices", 1, 6)
	userCount := derivedCount(seed+"::users", 0, 4)

	devices := make([]string, 0, devCount)
	if deviceID != "" {
		devices = append(devices, deviceID)
	}
	for i := 1; len(devices) < devCount; i++ {
		dev := fmt.Sprintf("lab-dev-%s-%d", shortHash(seed, i), i)
		if !containsStr(devices, dev) {
			devices = append(devices, dev)
		}
	}

	users := make([]string, 0, userCount)
	for i := 0; len(users) < userCount; i++ {
		users = append(users, fmt.Sprintf("user-%s", shortHash(seed+"users", i)))
	}

	procPool := []string{
		"/usr/local/bin/ollama",
		"/Applications/Cursor.app/Contents/MacOS/Cursor",
		"/usr/bin/python3",
		"/opt/claude/claude",
		"C:/Program Files/Cursor/cursor.exe",
		"/usr/bin/curl",
	}
	destPool := []string{
		"api.openai.com:443",
		"api.anthropic.com:443",
		"generativelanguage.googleapis.com:443",
		"api.together.xyz:443",
		"localhost:11434",
		"portkey.ai:443",
	}
	topProcesses := pickFromPool(seed+"procs", procPool, derivedCount(seed+"procs-n", 1, 3))
	topDestinations := pickFromPool(seed+"dests", destPool, derivedCount(seed+"dests-n", 1, 3))

	risk := d.ExpectedBreakageRisk
	if risk == "" {
		risk = "low (observe-only; no enforcement)"
	}
	confidence := d.Confidence
	if confidence == "" {
		confidence = "medium"
	}
	summary := fmt.Sprintf(
		"Observe-only projection: %d historical matches across %d device(s) over the last 24h. No enforcement happens — this counts what *would* have matched.",
		matches, len(devices),
	)

	return DraftSimulationResult{
		ID:               uuid.NewString(),
		AtMS:             now,
		DeviceID:         deviceID,
		Mode:             "observe_only",
		MatchCount:       matches,
		MatchedDeviceIDs: devices,
		MatchedUsers:     users,
		TopProcessPaths:  topProcesses,
		TopDestinations:  topDestinations,
		WindowStartMS:    now - 24*60*60*1000,
		WindowEndMS:      now,
		Confidence:       confidence,
		ExpectedBreakage: risk,
		Summary:          summary,
		ScopeSnapshot:    append([]string(nil), d.ScopeSelectors...),
	}
}

func derivedCount(seed string, min, max int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	if max <= min {
		return min
	}
	span := max - min + 1
	return min + int(h.Sum32()%uint32(span))
}

func shortHash(seed string, salt int) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%s:%d", seed, salt)))
	return fmt.Sprintf("%x", h.Sum32())[:6]
}

func containsStr(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func pickFromPool(seed string, pool []string, n int) []string {
	if n <= 0 || len(pool) == 0 {
		return nil
	}
	if n > len(pool) {
		n = len(pool)
	}
	out := make([]string, 0, n)
	for i := 0; len(out) < n; i++ {
		idx := derivedCount(seed, 0, len(pool)-1)
		seed = fmt.Sprintf("%s:%d:%d", seed, idx, i)
		v := pool[idx]
		if !containsStr(out, v) {
			out = append(out, v)
		}
	}
	return out
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
		initialSnapshot := snapshotDraft(d)
		appendDraftHistory(&d, DraftDecisionEntry{
			AtMS:        now,
			Action:      "created",
			Note:        fmt.Sprintf("Draft control seeded from finding %s.", d.SourceFindingID),
			Status:      d.Status,
			ChangedKeys: []string{"status", "scope_selectors", "blast_radius", "rollback_plan"},
			After:       &initialSnapshot,
		})

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
		Note     string `json:"note"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	deviceID := strings.TrimSpace(body.DeviceID)
	s.platform.mu.Lock()
	found := false
	var snapshot DraftControl
	var sim DraftSimulationResult
	for i := range s.platform.Drafts {
		if s.platform.Drafts[i].ID != id {
			continue
		}
		found = true
		now := time.Now().UnixMilli()
		before := snapshotDraft(s.platform.Drafts[i])
		sim = buildSimulationResult(s.platform.Drafts[i], deviceID, now)

		s.platform.Drafts[i].SimulationMatches = sim.MatchCount
		s.platform.Drafts[i].SimulationDeviceID = deviceID
		s.platform.Drafts[i].SimulationAtMS = now
		s.platform.Drafts[i].UpdatedMS = now
		matchesCopy := sim.MatchCount
		s.platform.Drafts[i].ExpectedMatches = &matchesCopy
		s.platform.Drafts[i].Simulations = append(s.platform.Drafts[i].Simulations, sim)
		if len(s.platform.Drafts[i].Simulations) > 20 {
			s.platform.Drafts[i].Simulations = s.platform.Drafts[i].Simulations[len(s.platform.Drafts[i].Simulations)-20:]
		}

		after := snapshotDraft(s.platform.Drafts[i])
		appendDraftHistory(&s.platform.Drafts[i], DraftDecisionEntry{
			AtMS:         now,
			Action:       "simulated",
			Status:       s.platform.Drafts[i].Status,
			Note:         body.Note,
			ChangedKeys:  diffSnapshotKeys(before, after),
			Before:       &before,
			After:        &after,
			SimulationID: sim.ID,
		})
		snapshot = s.platform.Drafts[i]
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control.simulated",
			Status:      "observe_only",
			Subject:     id,
			DeviceID:    deviceID,
			Description: fmt.Sprintf("Lab simulation matched %d events across %d device(s)", sim.MatchCount, len(sim.MatchedDeviceIDs)),
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
		"matched_events": sim.MatchCount,
		"simulation":     sim,
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
		BlastRadiusNotes     []string `json:"blast_radius_notes,omitempty"`
		RollbackPlan         *string  `json:"rollback_plan,omitempty"`
		RollbackSteps        []string `json:"rollback_steps,omitempty"`
		DecisionNote         string   `json:"decision_note,omitempty"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	found := false
	var snapshot DraftControl
	var changedKeys []string
	var action string
	for i := range s.platform.Drafts {
		if s.platform.Drafts[i].ID != id {
			continue
		}
		found = true
		now := time.Now().UnixMilli()
		before := snapshotDraft(s.platform.Drafts[i])
		previousStatus := s.platform.Drafts[i].Status
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
		if body.BlastRadiusNotes != nil {
			s.platform.Drafts[i].BlastRadiusNotes = body.BlastRadiusNotes
		}
		if body.RollbackPlan != nil {
			s.platform.Drafts[i].RollbackPlan = *body.RollbackPlan
		}
		if body.RollbackSteps != nil {
			s.platform.Drafts[i].RollbackSteps = body.RollbackSteps
		}
		s.platform.Drafts[i].UpdatedMS = now
		after := snapshotDraft(s.platform.Drafts[i])
		changedKeys = diffSnapshotKeys(before, after)
		action = decideHistoryAction(changedKeys, previousStatus, s.platform.Drafts[i].Status)
		appendDraftHistory(&s.platform.Drafts[i], DraftDecisionEntry{
			AtMS:        now,
			Action:      action,
			Note:        body.DecisionNote,
			Status:      s.platform.Drafts[i].Status,
			ChangedKeys: changedKeys,
			Before:      &before,
			After:       &after,
		})
		snapshot = s.platform.Drafts[i]
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "draft_control." + action,
			Status:      s.platform.Drafts[i].Status,
			Subject:     id,
			DeviceID:    s.platform.Drafts[i].SourceDeviceID,
			Description: fmt.Sprintf("Observe-only draft control %s (%s)", action, strings.Join(changedKeys, ", ")),
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
		"draft":        snapshot,
		"action":       action,
		"changed_keys": changedKeys,
	})
}

func decideHistoryAction(changedKeys []string, previousStatus, currentStatus string) string {
	for _, k := range changedKeys {
		if k == "status" {
			return "status_changed"
		}
	}
	hasScopeChange := false
	hasNotesChange := false
	for _, k := range changedKeys {
		switch k {
		case "scope_selectors", "blast_radius", "blast_radius_notes", "rollback_plan", "rollback_steps":
			hasScopeChange = true
		case "operator_notes":
			hasNotesChange = true
		}
	}
	switch {
	case hasScopeChange:
		return "scope_edited"
	case hasNotesChange:
		return "note_added"
	}
	if previousStatus != currentStatus {
		return "status_changed"
	}
	return "updated"
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
