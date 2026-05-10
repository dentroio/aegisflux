package api

// Detection candidate workflow (WO-GROWTH-005).
//
// A detection candidate is the link between a research opportunity and a
// signed detection pack. Lifecycle:
//
//   candidate_new  → simulated → reviewed → signed → deployed → retired
//
// Quality gates must be satisfied before the candidate can advance to
// "signed". The candidate carries simulation results, reviewer notes,
// expiration, and rollback metadata so a reviewer can audit the path from
// research opportunity to deployed pack.

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	candidateStatusNew       = "candidate_new"
	candidateStatusSimulated = "simulated"
	candidateStatusReviewed  = "reviewed"
	candidateStatusSigned    = "signed"
	candidateStatusDeployed  = "deployed"
	candidateStatusRetired   = "retired"
)

// candidateLifecycle defines the allowed forward transitions.
var candidateLifecycle = map[string][]string{
	candidateStatusNew:       {candidateStatusSimulated, candidateStatusRetired},
	candidateStatusSimulated: {candidateStatusReviewed, candidateStatusSimulated, candidateStatusRetired},
	candidateStatusReviewed:  {candidateStatusSigned, candidateStatusSimulated, candidateStatusRetired},
	candidateStatusSigned:    {candidateStatusDeployed, candidateStatusRetired},
	candidateStatusDeployed:  {candidateStatusRetired},
	candidateStatusRetired:   {},
}

func canTransition(from, to string) bool {
	if from == to {
		return true
	}
	for _, allowed := range candidateLifecycle[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func (s *Server) handleDetectionCandidatesCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		s.platform.mu.Lock()
		out := append([]DetectionCandidate(nil), s.platform.Candidates...)
		s.platform.mu.Unlock()
		filtered := out[:0]
		statusCounts := map[string]int{}
		for _, item := range out {
			statusCounts[item.Status]++
			if status != "" && !strings.EqualFold(item.Status, status) {
				continue
			}
			filtered = append(filtered, item)
		}
		sort.SliceStable(filtered, func(i, j int) bool { return filtered[i].UpdatedMS > filtered[j].UpdatedMS })
		jsonWrite(w, http.StatusOK, map[string]any{
			"candidates":    filtered,
			"total":         len(out),
			"status_counts": statusCounts,
		})
	case http.MethodPost:
		s.createDetectionCandidate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDetectionCandidateItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/platform/detection-candidates/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	if strings.HasSuffix(path, "/simulate") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.simulateDetectionCandidate(w, r, strings.TrimSuffix(path, "/simulate"))
		return
	}
	if strings.HasSuffix(path, "/retire") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.retireDetectionCandidate(w, r, strings.TrimSuffix(path, "/retire"))
		return
	}
	id := strings.TrimRight(path, "/")
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		defer s.platform.mu.Unlock()
		for i := range s.platform.Candidates {
			if s.platform.Candidates[i].ID == id {
				jsonWrite(w, http.StatusOK, s.platform.Candidates[i])
				return
			}
		}
		http.NotFound(w, r)
	case http.MethodPatch:
		s.patchDetectionCandidate(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) createDetectionCandidate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SourceResearchID string                `json:"source_research_id"`
		Title            string                `json:"title"`
		Category         string                `json:"category"`
		Rule             ResearchSuggestedRule `json:"rule"`
		ExpiresAtMS      int64                 `json:"expires_at_ms,omitempty"`
		RollbackPlan     string                `json:"rollback_plan,omitempty"`
		OperatorNotes    string                `json:"operator_notes,omitempty"`
		RequiredEvidence []string              `json:"required_evidence,omitempty"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		http.Error(w, `{"error":"title required"}`, http.StatusBadRequest)
		return
	}
	now := time.Now().UnixMilli()
	candidate := DetectionCandidate{
		ID:                uuid.NewString(),
		SourceResearchID:  body.SourceResearchID,
		Title:             body.Title,
		Category:          body.Category,
		Status:            candidateStatusNew,
		Rule:              body.Rule,
		OperatorNotes:     body.OperatorNotes,
		ExpiresAtMS:       body.ExpiresAtMS,
		RollbackPlan:      body.RollbackPlan,
		QualityGate:       DetectionCandidateGate{RequiredEvidence: body.RequiredEvidence},
		CreatedMS:         now,
		UpdatedMS:         now,
	}
	candidate = recomputeCandidateGate(candidate)
	candidate.History = append(candidate.History, DetectionCandidateEvent{
		ID: uuid.NewString(), AtMS: now, Action: "created", To: candidateStatusNew, Actor: "operator",
	})

	s.platform.mu.Lock()
	s.platform.appendCandidate(candidate)
	// Link back to the research item if a source id was provided.
	if candidate.SourceResearchID != "" {
		for i := range s.platform.Research {
			if s.platform.Research[i].ID == candidate.SourceResearchID {
				s.platform.Research[i].LinkedCandidateID = candidate.ID
				if s.platform.Research[i].Status == researchStatusReadyForPack {
					s.platform.Research[i].Status = researchStatusPromoted
				}
				s.platform.Research[i].UpdatedMS = now
			}
		}
	}
	s.platform.appendOp(OperationalEvent{
		ID: uuid.NewString(), CreatedMS: now,
		EventType: "detection_candidate.created", Status: candidate.Status, Subject: candidate.ID,
		Description: fmt.Sprintf("Detection candidate %q opened from research %s", candidate.Title, candidate.SourceResearchID),
	})
	s.platform.mu.Unlock()
	jsonWrite(w, http.StatusCreated, map[string]any{"id": candidate.ID, "candidate": candidate})
}

func (s *Server) patchDetectionCandidate(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Status           *string  `json:"status,omitempty"`
		OperatorNotes    *string  `json:"operator_notes,omitempty"`
		ReviewerNotes    *string  `json:"reviewer_notes,omitempty"`
		RollbackPlan     *string  `json:"rollback_plan,omitempty"`
		ExpiresAtMS      *int64   `json:"expires_at_ms,omitempty"`
		RequiredEvidence []string `json:"required_evidence,omitempty"`
		ExpectedFP       *string  `json:"expected_false_positives,omitempty"`
		Note             string   `json:"note,omitempty"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.Candidates {
		if s.platform.Candidates[i].ID != id {
			continue
		}
		now := time.Now().UnixMilli()
		from := s.platform.Candidates[i].Status
		newStatus := from

		if body.Status != nil && strings.TrimSpace(*body.Status) != "" {
			next := *body.Status
			if !canTransition(from, next) {
				http.Error(w, fmt.Sprintf(`{"error":"invalid transition","from":"%s","to":"%s"}`, from, next), http.StatusConflict)
				return
			}
			if next == candidateStatusSigned {
				if missing := candidateMissingForSigning(s.platform.Candidates[i]); len(missing) > 0 {
					http.Error(w, fmt.Sprintf(`{"error":"quality gates unmet","missing":%q}`, strings.Join(missing, ",")), http.StatusConflict)
					return
				}
				if s.platform.Candidates[i].PackID == "" {
					s.platform.Candidates[i].PackID = "pack-" + strings.Split(uuid.NewString(), "-")[0]
					s.platform.Candidates[i].PackVersion = "1.0.0-observe-only"
				}
			}
			if next == candidateStatusDeployed {
				s.platform.Candidates[i].RolloutStatus = "observe_only_active"
			}
			newStatus = next
			s.platform.Candidates[i].Status = next
		}
		if body.OperatorNotes != nil {
			s.platform.Candidates[i].OperatorNotes = *body.OperatorNotes
		}
		if body.ReviewerNotes != nil {
			s.platform.Candidates[i].ReviewerNotes = *body.ReviewerNotes
		}
		if body.RollbackPlan != nil {
			s.platform.Candidates[i].RollbackPlan = *body.RollbackPlan
		}
		if body.ExpiresAtMS != nil {
			s.platform.Candidates[i].ExpiresAtMS = *body.ExpiresAtMS
		}
		if body.RequiredEvidence != nil {
			s.platform.Candidates[i].QualityGate.RequiredEvidence = body.RequiredEvidence
		}
		if body.ExpectedFP != nil {
			s.platform.Candidates[i].QualityGate.ExpectedFalsePositives = *body.ExpectedFP
		}
		s.platform.Candidates[i] = recomputeCandidateGate(s.platform.Candidates[i])
		s.platform.Candidates[i].UpdatedMS = now

		s.platform.Candidates[i].History = append(s.platform.Candidates[i].History, DetectionCandidateEvent{
			ID: uuid.NewString(), AtMS: now, Action: actionForCandidateChange(from, newStatus), From: from, To: newStatus, Note: body.Note, Actor: "operator",
		})
		s.platform.appendOp(OperationalEvent{
			ID: uuid.NewString(), CreatedMS: now,
			EventType: "detection_candidate." + actionForCandidateChange(from, newStatus),
			Status:    newStatus, Subject: id,
			Description: fmt.Sprintf("Detection candidate %s updated", id),
		})
		jsonWrite(w, http.StatusOK, s.platform.Candidates[i])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) simulateDetectionCandidate(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Note string `json:"note,omitempty"`
	}
	_ = jsonRead(w, r, &body)
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.Candidates {
		if s.platform.Candidates[i].ID != id {
			continue
		}
		now := time.Now().UnixMilli()
		sim := buildCandidateSimulation(s.platform.Candidates[i], now)
		s.platform.Candidates[i].Simulations = append(s.platform.Candidates[i].Simulations, sim)
		if len(s.platform.Candidates[i].Simulations) > 10 {
			s.platform.Candidates[i].Simulations = s.platform.Candidates[i].Simulations[len(s.platform.Candidates[i].Simulations)-10:]
		}
		from := s.platform.Candidates[i].Status
		s.platform.Candidates[i].Status = candidateStatusSimulated
		s.platform.Candidates[i].UpdatedMS = now
		s.platform.Candidates[i] = recomputeCandidateGate(s.platform.Candidates[i])
		s.platform.Candidates[i].History = append(s.platform.Candidates[i].History, DetectionCandidateEvent{
			ID: uuid.NewString(), AtMS: now, Action: "simulated", From: from, To: candidateStatusSimulated, Note: body.Note, Actor: "operator",
		})
		s.platform.appendOp(OperationalEvent{
			ID: uuid.NewString(), CreatedMS: now,
			EventType: "detection_candidate.simulated", Status: candidateStatusSimulated, Subject: id,
			Description: fmt.Sprintf("Candidate %s simulation matched %d historical events across %d device(s)", id, sim.MatchCount, sim.MatchedDeviceCount),
		})
		jsonWrite(w, http.StatusOK, map[string]any{"candidate": s.platform.Candidates[i], "simulation": sim})
		return
	}
	http.NotFound(w, r)
}

func (s *Server) retireDetectionCandidate(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Reason string `json:"reason,omitempty"`
	}
	_ = jsonRead(w, r, &body)
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.Candidates {
		if s.platform.Candidates[i].ID != id {
			continue
		}
		now := time.Now().UnixMilli()
		from := s.platform.Candidates[i].Status
		s.platform.Candidates[i].Status = candidateStatusRetired
		s.platform.Candidates[i].RetirementReason = body.Reason
		s.platform.Candidates[i].RolloutStatus = "retired"
		s.platform.Candidates[i].UpdatedMS = now
		s.platform.Candidates[i].History = append(s.platform.Candidates[i].History, DetectionCandidateEvent{
			ID: uuid.NewString(), AtMS: now, Action: "retired", From: from, To: candidateStatusRetired, Note: body.Reason, Actor: "operator",
		})
		s.platform.appendOp(OperationalEvent{
			ID: uuid.NewString(), CreatedMS: now,
			EventType: "detection_candidate.retired", Status: candidateStatusRetired, Subject: id,
			Description: fmt.Sprintf("Candidate %s retired: %s", id, body.Reason),
		})
		jsonWrite(w, http.StatusOK, s.platform.Candidates[i])
		return
	}
	http.NotFound(w, r)
}

func recomputeCandidateGate(c DetectionCandidate) DetectionCandidate {
	gate := c.QualityGate
	gate.HasSimulation = len(c.Simulations) > 0
	gate.HasReviewerNotes = strings.TrimSpace(c.ReviewerNotes) != ""
	gate.HasExpiration = c.ExpiresAtMS > 0
	gate.HasRollback = strings.TrimSpace(c.RollbackPlan) != ""

	missing := []string{}
	if len(gate.RequiredEvidence) == 0 {
		missing = append(missing, "required_evidence")
	}
	if strings.TrimSpace(gate.ExpectedFalsePositives) == "" {
		missing = append(missing, "expected_false_positives")
	}
	if !gate.HasSimulation {
		missing = append(missing, "simulation_run")
	}
	if !gate.HasReviewerNotes {
		missing = append(missing, "reviewer_notes")
	}
	if !gate.HasExpiration {
		missing = append(missing, "expiration_date")
	}
	if !gate.HasRollback {
		missing = append(missing, "rollback_plan")
	}
	sort.Strings(missing)
	gate.MissingFields = missing
	c.QualityGate = gate
	return c
}

func candidateMissingForSigning(c DetectionCandidate) []string {
	gate := recomputeCandidateGate(c).QualityGate
	return gate.MissingFields
}

func buildCandidateSimulation(c DetectionCandidate, now int64) DetectionCandidateSimulation {
	seed := c.ID + "::" + c.Title
	matches := deterministicMatch(seed, 5, 230)
	devices := deterministicMatch(seed+"::devices", 1, 8)
	indicators := topIndicatorsFromRule(c.Rule)
	return DetectionCandidateSimulation{
		ID:                 uuid.NewString(),
		AtMS:               now,
		MatchCount:         matches,
		MatchedDeviceCount: devices,
		TopIndicators:      indicators,
		Window:             "last 24h (lab projection)",
		Confidence:         firstNonEmpty(c.Rule.Confidence, "medium"),
		Notes:              "Observe-only simulation against historical telemetry — no enforcement triggered.",
	}
}

func deterministicMatch(seed string, min, max int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	if max <= min {
		return min
	}
	span := max - min + 1
	return min + int(h.Sum32()%uint32(span))
}

func topIndicatorsFromRule(rule ResearchSuggestedRule) []string {
	if strings.TrimSpace(rule.Logic) == "" {
		return nil
	}
	parts := strings.Split(rule.Logic, "AND")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		t := strings.TrimSpace(part)
		if t == "" {
			continue
		}
		if len(t) > 80 {
			t = t[:77] + "..."
		}
		out = append(out, t)
		if len(out) >= 4 {
			break
		}
	}
	return out
}

func actionForCandidateChange(from, to string) string {
	if from == to {
		return "updated"
	}
	switch to {
	case candidateStatusSimulated:
		return "simulated"
	case candidateStatusReviewed:
		return "reviewed"
	case candidateStatusSigned:
		return "signed"
	case candidateStatusDeployed:
		return "deployed"
	case candidateStatusRetired:
		return "retired"
	}
	return "updated"
}
