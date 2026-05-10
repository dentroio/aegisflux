package api

// Audit-mode bundle staging and endpoint status (WO-GROWTH-007).
//
// An audit-mode bundle is the foundation for safe enforcement. The bundle
// declares scope, expected match telemetry, expiration, and rollback notes.
// Endpoints accept the bundle and report status (pending, accepted,
// rejected, incompatible, stale) plus observe-only match telemetry.
//
// Audit-mode bundles never block, deny, or quarantine. The handlers enforce
// the observe-only contract by rejecting any mode other than "audit".

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	auditModeAudit = "audit"

	auditStatusDraft        = "draft"
	auditStatusStaged       = "staged"
	auditStatusExpired      = "expired"
	auditStatusRevoked      = "revoked"

	endpointStatusPending      = "pending"
	endpointStatusAccepted     = "accepted"
	endpointStatusRejected     = "rejected"
	endpointStatusIncompatible = "incompatible"
	endpointStatusStale        = "stale"
)

var endpointStatusValues = map[string]struct{}{
	endpointStatusPending:      {},
	endpointStatusAccepted:     {},
	endpointStatusRejected:     {},
	endpointStatusIncompatible: {},
	endpointStatusStale:        {},
}

func (s *Server) handleAuditBundlesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		s.platform.mu.Lock()
		out := append([]AuditBundle(nil), s.platform.AuditBundles...)
		s.platform.mu.Unlock()
		filtered := out[:0]
		statusCounts := map[string]int{}
		for _, b := range out {
			statusCounts[b.Status]++
			if status != "" && !strings.EqualFold(b.Status, status) {
				continue
			}
			filtered = append(filtered, b)
		}
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].UpdatedAtMS > filtered[j].UpdatedAtMS })
		jsonWrite(w, http.StatusOK, map[string]any{
			"bundles":       filtered,
			"total":         len(out),
			"status_counts": statusCounts,
		})
	case http.MethodPost:
		s.createAuditBundle(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

type auditBundleCreateInput struct {
	Title             string   `json:"title"`
	Description       string   `json:"description,omitempty"`
	Mode              string   `json:"mode,omitempty"`
	Version           string   `json:"version,omitempty"`
	Scope             []string `json:"scope,omitempty"`
	ExpectedTelemetry []string `json:"expected_match_telemetry,omitempty"`
	ApprovalRefs      []string `json:"approval_refs,omitempty"`
	RollbackNotes     string   `json:"rollback_notes,omitempty"`
	SourceCandidateID string   `json:"source_candidate_id,omitempty"`
	SourceDraftID     string   `json:"source_draft_id,omitempty"`
	ExpiresAtMS       int64    `json:"expires_at_ms,omitempty"`
}

func (s *Server) createAuditBundle(w http.ResponseWriter, r *http.Request) {
	var in auditBundleCreateInput
	if !jsonRead(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}
	mode := strings.ToLower(strings.TrimSpace(in.Mode))
	if mode == "" {
		mode = auditModeAudit
	}
	if mode != auditModeAudit {
		http.Error(w, `{"error":"only audit mode is supported"}`, http.StatusBadRequest)
		return
	}
	now := time.Now().UnixMilli()
	bundle := AuditBundle{
		ID:                uuid.NewString(),
		Version:           strings.TrimSpace(in.Version),
		Mode:              mode,
		Title:             in.Title,
		Description:       strings.TrimSpace(in.Description),
		Scope:             append([]string(nil), in.Scope...),
		ExpectedTelemetry: append([]string(nil), in.ExpectedTelemetry...),
		ApprovalRefs:      append([]string(nil), in.ApprovalRefs...),
		RollbackNotes:     strings.TrimSpace(in.RollbackNotes),
		SourceCandidateID: strings.TrimSpace(in.SourceCandidateID),
		SourceDraftID:     strings.TrimSpace(in.SourceDraftID),
		Status:            auditStatusDraft,
		ExpiresAtMS:       in.ExpiresAtMS,
		CreatedAtMS:       now,
		UpdatedAtMS:       now,
	}
	if bundle.Version == "" {
		bundle.Version = "v1"
	}
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.created",
		To:     auditStatusDraft,
	})
	s.platform.mu.Lock()
	s.platform.appendAuditBundle(bundle)
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.created",
		Status:      bundle.Status,
		Subject:     bundle.ID,
		Description: bundle.Title,
	})
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusCreated, bundle)
}

func (s *Server) handleAuditBundleItem(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/platform/audit-bundles/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(rest, "/", 2)
	id := parts[0]
	sub := ""
	if len(parts) == 2 {
		sub = parts[1]
	}
	switch sub {
	case "":
		switch r.Method {
		case http.MethodGet:
			bundle, ok := s.findAuditBundle(id)
			if !ok {
				http.NotFound(w, r)
				return
			}
			jsonWrite(w, http.StatusOK, bundle)
		case http.MethodPatch:
			s.patchAuditBundle(w, r, id)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	case "stage":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		s.stageAuditBundle(w, r, id)
	case "status":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		s.reportAuditBundleStatus(w, r, id)
	case "match":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		s.reportAuditBundleMatch(w, r, id)
	case "revoke":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		s.revokeAuditBundle(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) findAuditBundle(id string) (AuditBundle, bool) {
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for _, b := range s.platform.AuditBundles {
		if b.ID == id {
			return b, true
		}
	}
	return AuditBundle{}, false
}

type auditBundlePatch struct {
	Title             *string  `json:"title,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Scope             []string `json:"scope,omitempty"`
	ExpectedTelemetry []string `json:"expected_match_telemetry,omitempty"`
	ApprovalRefs      []string `json:"approval_refs,omitempty"`
	RollbackNotes     *string  `json:"rollback_notes,omitempty"`
	ExpiresAtMS       *int64   `json:"expires_at_ms,omitempty"`
	Note              string   `json:"note,omitempty"`
}

func (s *Server) patchAuditBundle(w http.ResponseWriter, r *http.Request, id string) {
	var in auditBundlePatch
	if !jsonRead(w, r, &in) {
		return
	}
	now := time.Now().UnixMilli()
	s.platform.mu.Lock()
	idx := -1
	for i := range s.platform.AuditBundles {
		if s.platform.AuditBundles[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.platform.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	bundle := &s.platform.AuditBundles[idx]
	if bundle.Status != auditStatusDraft {
		s.platform.mu.Unlock()
		http.Error(w, `{"error":"only draft bundles can be edited"}`, http.StatusConflict)
		return
	}
	if in.Title != nil {
		bundle.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		bundle.Description = strings.TrimSpace(*in.Description)
	}
	if in.Scope != nil {
		bundle.Scope = append([]string(nil), in.Scope...)
	}
	if in.ExpectedTelemetry != nil {
		bundle.ExpectedTelemetry = append([]string(nil), in.ExpectedTelemetry...)
	}
	if in.ApprovalRefs != nil {
		bundle.ApprovalRefs = append([]string(nil), in.ApprovalRefs...)
	}
	if in.RollbackNotes != nil {
		bundle.RollbackNotes = strings.TrimSpace(*in.RollbackNotes)
	}
	if in.ExpiresAtMS != nil {
		bundle.ExpiresAtMS = *in.ExpiresAtMS
	}
	bundle.UpdatedAtMS = now
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.updated",
		Note:   strings.TrimSpace(in.Note),
	})
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.updated",
		Status:      bundle.Status,
		Subject:     bundle.ID,
		Description: bundle.Title,
	})
	out := *bundle
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, out)
}

type auditBundleStageInput struct {
	DeviceIDs []string `json:"device_ids,omitempty"`
	Note      string   `json:"note,omitempty"`
}

func (s *Server) stageAuditBundle(w http.ResponseWriter, r *http.Request, id string) {
	var in auditBundleStageInput
	if r.ContentLength > 0 {
		if !jsonRead(w, r, &in) {
			return
		}
	}
	now := time.Now().UnixMilli()
	s.platform.mu.Lock()
	idx := -1
	for i := range s.platform.AuditBundles {
		if s.platform.AuditBundles[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.platform.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	bundle := &s.platform.AuditBundles[idx]
	if bundle.Mode != auditModeAudit {
		s.platform.mu.Unlock()
		http.Error(w, `{"error":"only audit mode bundles can be staged"}`, http.StatusBadRequest)
		return
	}
	if bundle.Status != auditStatusDraft && bundle.Status != auditStatusStaged {
		s.platform.mu.Unlock()
		http.Error(w, `{"error":"bundle is not stageable"}`, http.StatusConflict)
		return
	}
	from := bundle.Status
	bundle.Status = auditStatusStaged
	if bundle.StagedAtMS == 0 {
		bundle.StagedAtMS = now
	}
	bundle.UpdatedAtMS = now
	if len(in.DeviceIDs) > 0 {
		known := map[string]int{}
		for i, st := range bundle.EndpointStatuses {
			known[st.DeviceID] = i
		}
		for _, d := range in.DeviceIDs {
			d = strings.TrimSpace(d)
			if d == "" {
				continue
			}
			if i, ok := known[d]; ok {
				bundle.EndpointStatuses[i].Status = endpointStatusPending
				bundle.EndpointStatuses[i].ReportedAtMS = now
				continue
			}
			bundle.EndpointStatuses = append(bundle.EndpointStatuses, AuditBundleStatus{
				DeviceID:     d,
				Status:       endpointStatusPending,
				ReportedAtMS: now,
			})
		}
	}
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.staged",
		From:   from,
		To:     auditStatusStaged,
		Note:   strings.TrimSpace(in.Note),
	})
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.staged",
		Status:      bundle.Status,
		Subject:     bundle.ID,
		Description: bundle.Title,
	})
	out := *bundle
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, out)
}

type auditBundleStatusReport struct {
	DeviceID     string `json:"device_id"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	AgentVersion string `json:"agent_version,omitempty"`
}

func (s *Server) reportAuditBundleStatus(w http.ResponseWriter, r *http.Request, id string) {
	var in auditBundleStatusReport
	if !jsonRead(w, r, &in) {
		return
	}
	in.DeviceID = strings.TrimSpace(in.DeviceID)
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	if in.DeviceID == "" {
		http.Error(w, `{"error":"device_id is required"}`, http.StatusBadRequest)
		return
	}
	if _, ok := endpointStatusValues[in.Status]; !ok {
		http.Error(w, `{"error":"unsupported endpoint status"}`, http.StatusBadRequest)
		return
	}
	now := time.Now().UnixMilli()
	s.platform.mu.Lock()
	idx := -1
	for i := range s.platform.AuditBundles {
		if s.platform.AuditBundles[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.platform.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	bundle := &s.platform.AuditBundles[idx]
	found := false
	prev := ""
	for i := range bundle.EndpointStatuses {
		if bundle.EndpointStatuses[i].DeviceID == in.DeviceID {
			prev = bundle.EndpointStatuses[i].Status
			bundle.EndpointStatuses[i].Status = in.Status
			bundle.EndpointStatuses[i].Reason = strings.TrimSpace(in.Reason)
			bundle.EndpointStatuses[i].AgentVersion = strings.TrimSpace(in.AgentVersion)
			bundle.EndpointStatuses[i].ReportedAtMS = now
			found = true
			break
		}
	}
	if !found {
		bundle.EndpointStatuses = append(bundle.EndpointStatuses, AuditBundleStatus{
			DeviceID:     in.DeviceID,
			Status:       in.Status,
			Reason:       strings.TrimSpace(in.Reason),
			AgentVersion: strings.TrimSpace(in.AgentVersion),
			ReportedAtMS: now,
		})
	}
	bundle.UpdatedAtMS = now
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.endpoint_status",
		From:   prev,
		To:     in.Status,
		Note:   strings.TrimSpace(in.Reason),
	})
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.endpoint_status",
		Status:      in.Status,
		Subject:     bundle.ID,
		DeviceID:    in.DeviceID,
		Description: bundle.Title,
	})
	out := *bundle
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, out)
}

type auditBundleMatchReport struct {
	DeviceID  string `json:"device_id,omitempty"`
	Process   string `json:"process,omitempty"`
	Indicator string `json:"indicator,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

func (s *Server) reportAuditBundleMatch(w http.ResponseWriter, r *http.Request, id string) {
	var in auditBundleMatchReport
	if !jsonRead(w, r, &in) {
		return
	}
	now := time.Now().UnixMilli()
	s.platform.mu.Lock()
	idx := -1
	for i := range s.platform.AuditBundles {
		if s.platform.AuditBundles[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.platform.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	bundle := &s.platform.AuditBundles[idx]
	if bundle.Status != auditStatusStaged {
		s.platform.mu.Unlock()
		http.Error(w, `{"error":"bundle is not staged"}`, http.StatusConflict)
		return
	}
	match := AuditBundleMatch{
		ID:        uuid.NewString(),
		DeviceID:  strings.TrimSpace(in.DeviceID),
		Process:   strings.TrimSpace(in.Process),
		Indicator: strings.TrimSpace(in.Indicator),
		Detail:    strings.TrimSpace(in.Detail),
		AtMS:      now,
	}
	bundle.Matches = append(bundle.Matches, match)
	if len(bundle.Matches) > 200 {
		bundle.Matches = bundle.Matches[len(bundle.Matches)-200:]
	}
	for i := range bundle.EndpointStatuses {
		if bundle.EndpointStatuses[i].DeviceID == match.DeviceID {
			bundle.EndpointStatuses[i].LastMatchAtMS = now
			break
		}
	}
	bundle.UpdatedAtMS = now
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.match",
		Note:   match.Indicator,
	})
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.match",
		Status:      bundle.Status,
		Subject:     bundle.ID,
		DeviceID:    match.DeviceID,
		Description: bundle.Title,
	})
	out := *bundle
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, out)
}

func (s *Server) revokeAuditBundle(w http.ResponseWriter, r *http.Request, id string) {
	var in struct {
		Note string `json:"note,omitempty"`
	}
	if r.ContentLength > 0 {
		if !jsonRead(w, r, &in) {
			return
		}
	}
	now := time.Now().UnixMilli()
	s.platform.mu.Lock()
	idx := -1
	for i := range s.platform.AuditBundles {
		if s.platform.AuditBundles[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.platform.mu.Unlock()
		http.NotFound(w, r)
		return
	}
	bundle := &s.platform.AuditBundles[idx]
	from := bundle.Status
	bundle.Status = auditStatusRevoked
	bundle.UpdatedAtMS = now
	bundle.History = append(bundle.History, AuditBundleEvent{
		ID:     uuid.NewString(),
		AtMS:   now,
		Action: "audit_bundle.revoked",
		From:   from,
		To:     auditStatusRevoked,
		Note:   strings.TrimSpace(in.Note),
	})
	s.platform.appendOp(OperationalEvent{
		ID:          uuid.NewString(),
		CreatedMS:   now,
		EventType:   "audit_bundle.revoked",
		Status:      bundle.Status,
		Subject:     bundle.ID,
		Description: bundle.Title,
	})
	out := *bundle
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusOK, out)
}

// auditBundleEndpointSummary is a small aggregate that the UI can render
// without walking every endpoint status row.
type auditBundleEndpointSummary struct {
	Total        int `json:"total"`
	Pending      int `json:"pending"`
	Accepted     int `json:"accepted"`
	Rejected     int `json:"rejected"`
	Incompatible int `json:"incompatible"`
	Stale        int `json:"stale"`
	WithMatches  int `json:"with_matches"`
}

func summarizeEndpointStatuses(statuses []AuditBundleStatus) auditBundleEndpointSummary {
	out := auditBundleEndpointSummary{Total: len(statuses)}
	for _, st := range statuses {
		switch st.Status {
		case endpointStatusPending:
			out.Pending++
		case endpointStatusAccepted:
			out.Accepted++
		case endpointStatusRejected:
			out.Rejected++
		case endpointStatusIncompatible:
			out.Incompatible++
		case endpointStatusStale:
			out.Stale++
		}
		if st.LastMatchAtMS > 0 {
			out.WithMatches++
		}
	}
	return out
}
