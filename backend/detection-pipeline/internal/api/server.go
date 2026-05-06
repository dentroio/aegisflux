package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"aegisflux/backend/detection-pipeline/internal/eval"
	"aegisflux/backend/detection-pipeline/internal/ingestclient"
	"aegisflux/backend/detection-pipeline/internal/model"
	"aegisflux/backend/detection-pipeline/internal/pack"
	"aegisflux/backend/detection-pipeline/internal/rollout"
	"aegisflux/backend/detection-pipeline/internal/sign"
	"aegisflux/backend/detection-pipeline/internal/store"
)

// Server exposes WO-DET-002/003 HTTP APIs.
type Server struct {
	store     *store.Store
	ingest    *ingestclient.Client
	ingestURL string
	signer    *sign.Signer
	policy    rollout.Policy
}

func NewServer(st *store.Store, ingestURL string) (*Server, error) {
	sg, err := sign.NewSigner("detection-pipeline")
	if err != nil {
		return nil, err
	}
	return &Server{
		store:     st,
		ingest:    ingestclient.New(ingestURL),
		ingestURL: ingestURL,
		signer:    sg,
		policy:    rollout.LoadPolicyFromEnv(),
	}, nil
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/v1/detection/signer-info", s.handleSignerInfo)
	// Trailing slash pattern matches /signed-packs/{id}
	mux.HandleFunc("/v1/detection/signed-packs/", s.handleGetSignedPackByID)
	mux.HandleFunc("/v1/detection/signed-packs", s.handleListSignedPacksOnly)
	mux.HandleFunc("/v1/detection/research-items", s.handleResearchItemsRoute)
	mux.HandleFunc("/v1/detection/candidates", s.handleCandidatesCollectionRoute)
	mux.HandleFunc("/v1/detection/candidates/", s.handleCandidateSubroutes)
	s.registerRolloutRoutes(mux)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleListSignedPacksOnly(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items := s.store.ListSigned()
	out := make([]map[string]any, 0, len(items))
	for _, a := range items {
		row := map[string]any{
			"id":                  a.ID,
			"candidate_id":        a.CandidateID,
			"created_at_ms":       a.CreatedAtMS,
			"signature_algorithm": a.SignatureAlg,
			"key_id":              a.KeyID,
			"pack_bytes":          len(a.PackJSON),
		}
		if a.PackID != "" {
			row["pack_id"] = a.PackID
			row["pack_version"] = a.PackVersion
		}
		if a.SHA256Hex != "" {
			row["sha256"] = a.SHA256Hex
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleGetSignedPackByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/detection/signed-packs/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	art, ok := s.store.GetSigned(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(art.PackJSON)
}

func (s *Server) handleResearchItemsRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateResearch(w, r)
	case http.MethodGet:
		s.handleListResearch(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCandidatesCollectionRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateCandidate(w, r)
	case http.MethodGet:
		s.handleListCandidates(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSignerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"algorithm":       "ed25519",
		"key_id":          s.signer.KeyID(),
		"public_key_b64":  s.signer.PublicKeyBase64(),
		"sign_message":    "aegis.detection_pack.v1\\x00 || sha256(unsigned_pack_json_bytes)",
	})
}

func (s *Server) handleCreateResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Title     string `json:"title"`
		Summary   string `json:"summary,omitempty"`
		SourceURL string `json:"source_url,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, "title required", http.StatusBadRequest)
		return
	}
	item := &model.ResearchItem{
		ID:          newID("ri"),
		Title:       req.Title,
		Summary:     req.Summary,
		SourceURL:   req.SourceURL,
		CreatedAtMS: nowMS(),
	}
	if err := s.store.PutResearch(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) handleListResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.store.ListResearch()})
}

func (s *Server) handleCreateCandidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req createCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := req.validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.RequireResearch(req.ResearchItemID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	c := &model.Candidate{
		ID:              newID("cand"),
		ResearchItemID:  req.ResearchItemID,
		Title:           req.Title,
		Description:     req.Description,
		Status:          model.StatusDraft,
		CreatedAtMS:     nowMS(),
		UpdatedAtMS:     nowMS(),
		PackID:          req.PackID,
		PackVersion:     req.PackVersion,
		MinAgentVersion: req.MinAgentVersion,
		SupportedOS:     req.SupportedOS,
		Author:          req.Author,
		Source:            req.Source,
		EvaluatorLimits:   req.EvaluatorLimits,
		ProposedRules:     req.ProposedRules,
		References:        req.References,
	}
	if err := pack.ValidateProposedRules(c); err != nil {
		http.Error(w, "proposed_rules failed schema: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.PutCandidate(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

type createCandidateRequest struct {
	ResearchItemID  string          `json:"research_item_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description,omitempty"`
	PackID          string          `json:"pack_id"`
	PackVersion     string          `json:"pack_version"`
	MinAgentVersion string          `json:"min_agent_version"`
	SupportedOS     []string        `json:"supported_os"`
	Author          string          `json:"author,omitempty"`
	Source          string          `json:"source,omitempty"`
	EvaluatorLimits json.RawMessage `json:"evaluator_limits"`
	ProposedRules   json.RawMessage `json:"proposed_rules"`
	References      json.RawMessage `json:"references,omitempty"`
}

func (req *createCandidateRequest) validate() error {
	if strings.TrimSpace(req.ResearchItemID) == "" || strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("research_item_id and title required")
	}
	if strings.TrimSpace(req.PackID) == "" || strings.TrimSpace(req.PackVersion) == "" || strings.TrimSpace(req.MinAgentVersion) == "" {
		return fmt.Errorf("pack_id, pack_version, min_agent_version required")
	}
	if len(req.SupportedOS) == 0 {
		return fmt.Errorf("supported_os required")
	}
	if len(req.EvaluatorLimits) == 0 || len(req.ProposedRules) == 0 {
		return fmt.Errorf("evaluator_limits and proposed_rules required")
	}
	if strings.TrimSpace(req.Author) == "" && strings.TrimSpace(req.Source) == "" {
		return fmt.Errorf("author or source required")
	}
	return nil
}

func (s *Server) handleListCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := r.URL.Query().Get("status")
	items := s.store.ListCandidates()
	if status != "" {
		filtered := make([]*model.Candidate, 0)
		for _, c := range items {
			if string(c.Status) == status {
				filtered = append(filtered, c)
			}
		}
		items = filtered
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleCandidateSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/detection/candidates/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	id := parts[0]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 {
		s.handleGetCandidate(w, r, id)
		return
	}
	switch parts[1] {
	case "validate":
		if len(parts) == 2 {
			s.handleValidateCandidate(w, r, id)
			return
		}
	case "validations":
		if len(parts) == 2 {
			s.handleListValidations(w, r, id)
			return
		}
	case "approve":
		if len(parts) == 2 {
			s.handleApprove(w, r, id)
			return
		}
	case "reject":
		if len(parts) == 2 {
			s.handleReject(w, r, id)
			return
		}
	case "sign":
		if len(parts) == 2 {
			s.handleSign(w, r, id)
			return
		}
	case "signed-pack":
		if len(parts) == 2 {
			s.handleGetSignedPack(w, r, id)
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) handleGetCandidate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

type validateRequest struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	Limit    int    `json:"limit"`
}

func (s *Server) handleValidateCandidate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if c.Status != model.StatusDraft && c.Status != model.StatusValidationFailed && c.Status != model.StatusReadyForReview {
		http.Error(w, "candidate cannot be validated from state "+string(c.Status), http.StatusConflict)
		return
	}
	var req validateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Limit <= 0 {
		req.Limit = 500
	}

	c.Status = model.StatusValidating
	c.UpdatedAtMS = nowMS()
	_ = s.store.PutCandidate(c)

	vrun := &model.ValidationRun{
		ID:          newID("val"),
		CandidateID: id,
		StartedAtMS: nowMS(),
		Success:     false,
		IngestURL:   s.ingestURL,
		DeviceID:    req.DeviceID,
	}

	events, err := s.ingest.QueryEvents(context.Background(), req.DeviceID, req.TenantID, "", req.Limit)
	if err != nil {
		vrun.Errors = err.Error()
		vrun.EndedAtMS = nowMS()
		_ = s.store.PutValidation(vrun)
		c.Status = model.StatusValidationFailed
		c.LastValidationID = vrun.ID
		c.UpdatedAtMS = nowMS()
		_ = s.store.PutCandidate(c)
		writeJSON(w, http.StatusOK, vrun)
		return
	}
	vrun.EventsFetched = len(events)

	batch, err := eval.BuildBatch(events)
	if err != nil {
		vrun.Errors = err.Error()
		vrun.EndedAtMS = nowMS()
		_ = s.store.PutValidation(vrun)
		c.Status = model.StatusValidationFailed
		c.LastValidationID = vrun.ID
		c.UpdatedAtMS = nowMS()
		_ = s.store.PutCandidate(c)
		writeJSON(w, http.StatusOK, vrun)
		return
	}

	var rules []map[string]any
	if err := json.Unmarshal(c.ProposedRules, &rules); err != nil {
		vrun.Errors = "proposed_rules: " + err.Error()
		vrun.EndedAtMS = nowMS()
		_ = s.store.PutValidation(vrun)
		c.Status = model.StatusValidationFailed
		c.LastValidationID = vrun.ID
		c.UpdatedAtMS = nowMS()
		_ = s.store.PutCandidate(c)
		writeJSON(w, http.StatusOK, vrun)
		return
	}

	matched := 0
	var details []string
	for _, rule := range rules {
		ok, err := eval.RuleMatches(rule, batch)
		if err != nil {
			vrun.Errors = "rule " + fmt.Sprint(rule["rule_id"]) + ": " + err.Error()
			vrun.EndedAtMS = nowMS()
			_ = s.store.PutValidation(vrun)
			c.Status = model.StatusValidationFailed
			c.LastValidationID = vrun.ID
			c.UpdatedAtMS = nowMS()
			_ = s.store.PutCandidate(c)
			writeJSON(w, http.StatusOK, vrun)
			return
		}
		if ok {
			matched++
			if rid, ok := rule["rule_id"].(string); ok {
				details = append(details, "matched:"+rid)
			}
		}
	}
	vrun.MatchedRules = matched
	vrun.Success = true
	vrun.Details = strings.Join(details, "; ")
	vrun.EndedAtMS = nowMS()
	_ = s.store.PutValidation(vrun)

	c.Status = model.StatusReadyForReview
	c.LastValidationID = vrun.ID
	c.UpdatedAtMS = nowMS()
	_ = s.store.PutCandidate(c)
	writeJSON(w, http.StatusOK, vrun)
}

func (s *Server) handleListValidations(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.store.GetCandidate(id); !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.store.ListValidationsForCandidate(id)})
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if c.Status != model.StatusReadyForReview {
		http.Error(w, "approve only from ready_for_review", http.StatusConflict)
		return
	}
	c.Status = model.StatusApproved
	c.UpdatedAtMS = nowMS()
	if err := s.store.PutCandidate(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch c.Status {
	case model.StatusDraft, model.StatusValidating, model.StatusValidationFailed, model.StatusReadyForReview, model.StatusApproved:
	default:
		http.Error(w, "cannot reject from state "+string(c.Status), http.StatusConflict)
		return
	}
	c.Status = model.StatusRejected
	c.RejectReason = req.Reason
	c.UpdatedAtMS = nowMS()
	if err := s.store.PutCandidate(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleSign(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if c.Status != model.StatusApproved {
		http.Error(w, "sign only from approved", http.StatusConflict)
		return
	}
	doc, err := pack.AssemblePack(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unsigned, err := json.Marshal(cloneWithoutSignature(doc))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sigB64, err := s.signer.SignPackMessage(unsigned)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	finalBytes, err := sign.AttachSignature(doc, sigB64, s.signer.KeyID())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	comp, err := pack.NewSchemaCompiler()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var finalDoc any
	if err := json.Unmarshal(finalBytes, &finalDoc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := comp.Validate(finalDoc); err != nil {
		http.Error(w, "signed pack failed schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	docMap, _ := finalDoc.(map[string]any)
	art := &model.SignedPackArtifact{
		ID:           newID("sigpack"),
		CandidateID:  c.ID,
		CreatedAtMS:  nowMS(),
		PackJSON:     finalBytes,
		SignatureAlg: "ed25519",
		KeyID:        s.signer.KeyID(),
	}
	if docMap != nil {
		if pid, ok := docMap["pack_id"].(string); ok {
			art.PackID = pid
		}
		if pv, ok := docMap["pack_version"].(string); ok {
			art.PackVersion = pv
		}
	}
	sum := sha256.Sum256(finalBytes)
	art.SHA256Hex = hex.EncodeToString(sum[:])
	if err := s.store.PutSigned(art); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Status = model.StatusSigned
	c.SignedPackID = art.ID
	c.UpdatedAtMS = nowMS()
	if err := s.store.PutCandidate(c); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"candidate":   c,
		"signed_pack": art,
	})
}

func cloneWithoutSignature(p map[string]any) map[string]any {
	out := make(map[string]any, len(p))
	for k, v := range p {
		if k == "signature" {
			continue
		}
		out[k] = v
	}
	return out
}

func (s *Server) handleGetSignedPack(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	c, ok := s.store.GetCandidate(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if c.SignedPackID == "" {
		http.Error(w, "candidate has no signed pack", http.StatusNotFound)
		return
	}
	art, ok := s.store.GetSigned(c.SignedPackID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(art.PackJSON)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newID(prefix string) string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return prefix + "_" + hex.EncodeToString(b[:])
}

func nowMS() int64 {
	return store.NowMS()
}
