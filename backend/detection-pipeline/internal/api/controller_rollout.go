package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"aegisflux/backend/detection-pipeline/internal/model"
	"aegisflux/backend/detection-pipeline/internal/rollout"
)

func (s *Server) registerRolloutRoutes(mux *http.ServeMux) {
	// Prefix handlers cover /.../latest and /.../{pack_id}/artifact (exact /latest must not be shadowed).
	mux.HandleFunc("/v1/detection-packs/", s.handleDetectionPacksPrefix)
	mux.HandleFunc("/detection-packs/", s.handleDetectionPacksPrefix)

	mux.HandleFunc("/v1/agents/", s.handleAgentsSubpath)
	mux.HandleFunc("/agents/", s.handleAgentsSubpath)
}

func (s *Server) rolloutAllowed(w http.ResponseWriter) bool {
	if !s.policy.LabOnlyEnabled {
		http.Error(w, "detection pack rollout controller is disabled outside lab (set DETECTION_ROLLOUT_LAB_ONLY=true)", http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handleLatestPack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.rolloutAllowed(w) {
		return
	}
	osName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("os")))
	agentVer := strings.TrimSpace(r.URL.Query().Get("agent_version"))
	packIDFilter := strings.TrimSpace(r.URL.Query().Get("pack_id"))
	if osName == "" || agentVer == "" {
		http.Error(w, "query os and agent_version are required", http.StatusBadRequest)
		return
	}
	arts := s.store.ListSigned()
	art, meta, err := rollout.SelectLatestCompatible(arts, osName, agentVer, packIDFilter, s.policy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	uq := url.Values{}
	uq.Set("os", osName)
	uq.Set("agent_version", agentVer)
	writeJSON(w, http.StatusOK, map[string]any{
		"pack_id":           meta.PackID,
		"pack_version":      meta.PackVersion,
		"artifact_id":       art.ID,
		"sha256":            meta.SHA256Hex,
		"created_at_ms":     art.CreatedAtMS,
		"expires_at":        optionalRFC3339(meta.ExpiresAt),
		"min_agent_version": meta.MinAgentVersion,
		"supported_os":      meta.SupportedOS,
		"mode":              meta.Mode,
		"artifact_url":      fmt.Sprintf("/v1/detection-packs/%s/artifact?%s", meta.PackID, uq.Encode()),
	})
}

func optionalRFC3339(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func (s *Server) handleDetectionPacksPrefix(w http.ResponseWriter, r *http.Request) {
	if !s.rolloutAllowed(w) {
		return
	}
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/v1/detection-packs/")
	path = strings.TrimPrefix(path, "/detection-packs/")
	path = strings.Trim(path, "/")
	if path == "latest" {
		s.handleLatestPack(w, r)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	packID := parts[0]
	switch parts[1] {
	case "artifact":
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		s.handlePackArtifact(w, r, packID)
	case "rollout-status":
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		s.handlePackRolloutStatus(w, r, packID)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handlePackArtifact(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	osName := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("os")))
	agentVer := strings.TrimSpace(r.URL.Query().Get("agent_version"))
	verFilter := strings.TrimSpace(r.URL.Query().Get("version"))
	if osName == "" || agentVer == "" {
		http.Error(w, "query os and agent_version are required", http.StatusBadRequest)
		return
	}
	arts := s.store.ListSigned()
	var art *model.SignedPackArtifact
	var meta *rollout.PackMeta
	var err error
	if verFilter != "" {
		art, meta, err = rollout.FindArtifactByPackAndVersion(arts, packID, verFilter)
	} else {
		art, meta, err = rollout.FindArtifactByPackAndVersion(arts, packID, "")
	}
	if err != nil || art == nil || meta == nil {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	if !strings.EqualFold(meta.PackID, packID) {
		http.Error(w, "artifact not found", http.StatusNotFound)
		return
	}
	if !s.policy.AllowPack(meta.PackID) {
		http.Error(w, "pack not allowed by rollout policy", http.StatusForbidden)
		return
	}
	if !meta.Compatible(osName, agentVer) {
		http.Error(w, "pack incompatible with os/agent_version", http.StatusNotAcceptable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-SHA256", meta.SHA256Hex)
	w.Header().Set("X-Signature-Key-Id", art.KeyID)
	w.Header().Set("X-Signature-Algorithm", art.SignatureAlg)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(art.PackJSON)
}

func (s *Server) handlePackRolloutStatus(w http.ResponseWriter, r *http.Request, packID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.rolloutAllowed(w) {
		return
	}
	now := time.Now().UnixMilli()
	staleMS := s.policy.StaleAfterMS
	list := s.store.ListAgentPackStatusesForPackID(packID)
	type row struct {
		*model.AgentPackStatus
		Stale bool `json:"computed_stale"`
	}
	out := make([]row, 0, len(list))
	applied, rejected, incompatible, expired := 0, 0, 0, 0
	for _, st := range list {
		stale := st.LastCheckAtMS > 0 && now-st.LastCheckAtMS > staleMS
		out = append(out, row{AgentPackStatus: st, Stale: stale})
		switch st.RolloutState {
		case model.RolloutApplied:
			applied++
		case model.RolloutRejected:
			rejected++
		case model.RolloutIncompatible:
			incompatible++
		case model.RolloutExpired:
			expired++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pack_id":             packID,
		"agents_reported":     len(out),
		"count_applied":       applied,
		"count_rejected":      rejected,
		"count_incompatible":  incompatible,
		"count_expired":       expired,
		"stale_threshold_ms": staleMS,
		"agents":              out,
	})
}

func (s *Server) handleAgentsSubpath(w http.ResponseWriter, r *http.Request) {
	if !s.rolloutAllowed(w) {
		return
	}
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/v1/agents/")
	path = strings.TrimPrefix(path, "/agents/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "detection-pack-status" {
		http.NotFound(w, r)
		return
	}
	uid := parts[0]
	switch r.Method {
	case http.MethodGet:
		st, ok := s.store.GetAgentPackStatus(uid)
		if !ok {
			writeJSON(w, http.StatusOK, map[string]any{"status": nil})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": st})
	case http.MethodPost:
		s.handlePostAgentPackStatus(w, r, uid)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type postAgentPackStatusRequest struct {
	DeviceID               string   `json:"device_id"`
	ReportedAgentVersion   string   `json:"reported_agent_version"`
	RolloutState           string   `json:"rollout_state"`
	ReasonCodes            []string `json:"reason_codes"`
	ReasonDetail           string   `json:"reason_detail"`
	ActivePackID           string   `json:"active_pack_id"`
	ActivePackVersion      string   `json:"active_pack_version"`
	PreviousPackID         string   `json:"previous_pack_id"`
	PreviousPackVersion    string   `json:"previous_pack_version"`
	SignatureStatus        string   `json:"signature_status"`
	HashStatus             string   `json:"hash_status"`
	SchemaStatus           string   `json:"schema_status"`
	CompatibilityStatus    string   `json:"compatibility_status"`
	LastRejectedPackID     string   `json:"last_rejected_pack_id"`
	LastRejectedReason     string   `json:"last_rejected_reason"`
	LastRejectedReasonCodes []string `json:"last_rejected_reason_codes"`
	EmitVisibility         bool     `json:"emit_visibility"`
}

func (s *Server) handlePostAgentPackStatus(w http.ResponseWriter, r *http.Request, agentUID string) {
	var req postAgentPackStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rs := model.RolloutState(strings.TrimSpace(req.RolloutState))
	switch rs {
	case model.RolloutNotChecked, model.RolloutApplied, model.RolloutRejected, model.RolloutStale,
		model.RolloutIncompatible, model.RolloutExpired, model.RolloutRollback:
	default:
		http.Error(w, "invalid rollout_state", http.StatusBadRequest)
		return
	}
	now := nowMS()
	prev, _ := s.store.GetAgentPackStatus(agentUID)
	st := &model.AgentPackStatus{
		AgentUID:                 agentUID,
		DeviceID:                 strings.TrimSpace(req.DeviceID),
		ReportedAgentVersion:     strings.TrimSpace(req.ReportedAgentVersion),
		RolloutState:             rs,
		ReasonCodes:              req.ReasonCodes,
		ReasonDetail:             strings.TrimSpace(req.ReasonDetail),
		ActivePackID:             strings.TrimSpace(req.ActivePackID),
		ActivePackVersion:        strings.TrimSpace(req.ActivePackVersion),
		PreviousPackID:           strings.TrimSpace(req.PreviousPackID),
		PreviousPackVersion:      strings.TrimSpace(req.PreviousPackVersion),
		SignatureStatus:          strings.TrimSpace(req.SignatureStatus),
		HashStatus:               strings.TrimSpace(req.HashStatus),
		SchemaStatus:             strings.TrimSpace(req.SchemaStatus),
		CompatibilityStatus:      strings.TrimSpace(req.CompatibilityStatus),
		LastRejectedPackID:       strings.TrimSpace(req.LastRejectedPackID),
		LastRejectedReason:       strings.TrimSpace(req.LastRejectedReason),
		LastRejectedReasonCodes: req.LastRejectedReasonCodes,
		LastCheckAtMS:            now,
		UpdatedAtMS:              now,
	}
	if st.ReasonCodes == nil {
		st.ReasonCodes = []string{}
	}
	if rs == model.RolloutApplied {
		st.LastAppliedAtMS = now
	}
	if rs == model.RolloutRejected {
		st.LastRejectedAtMS = now
		if st.LastRejectedPackID == "" {
			st.LastRejectedPackID = st.ActivePackID
		}
		if st.LastRejectedReason == "" && st.ReasonDetail != "" {
			st.LastRejectedReason = st.ReasonDetail
		}
		if len(st.LastRejectedReasonCodes) == 0 && len(st.ReasonCodes) > 0 {
			st.LastRejectedReasonCodes = append([]string(nil), st.ReasonCodes...)
		}
	}
	if prev != nil {
		if st.PreviousPackID == "" {
			st.PreviousPackID = prev.ActivePackID
			st.PreviousPackVersion = prev.ActivePackVersion
		}
		if st.LastAppliedAtMS == 0 && prev.LastAppliedAtMS != 0 {
			st.LastAppliedAtMS = prev.LastAppliedAtMS
		}
	}
	if err := s.store.PutAgentPackStatus(st); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.EmitVisibility {
		base := strings.TrimRight(os.Getenv("DETECTION_PIPELINE_PUBLIC_BASE_URL"), "/")
		if base == "" {
			base = "http://127.0.0.1:8089"
		}
		if err := s.maybeEmitDetectionPackStatus(context.Background(), st, base); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"status":           st,
				"visibility_error": err.Error(),
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": st})
}
