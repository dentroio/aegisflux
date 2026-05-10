package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Lifecycle for a research item:
//
//	new          → just ingested, no review yet
//	scoped       → operator has confirmed scope and indicators
//	ready_for_pack→ ready to be promoted into a draft detection pack
//	promoted     → has been promoted into a pack (terminal for this view)
//	declined     → operator decided not to act (terminal)
const (
	researchStatusNew         = "new"
	researchStatusScoped      = "scoped"
	researchStatusReadyForPack = "ready_for_pack"
	researchStatusPromoted    = "promoted"
	researchStatusDeclined    = "declined"
)

func (s *Server) handleResearchFeedCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		out := append([]ResearchItem(nil), s.platform.Research...)
		s.platform.mu.Unlock()

		category := strings.TrimSpace(r.URL.Query().Get("category"))
		status := strings.TrimSpace(r.URL.Query().Get("status"))
		filtered := make([]ResearchItem, 0, len(out))
		for _, item := range out {
			if category != "" && !strings.EqualFold(item.Category, category) {
				continue
			}
			if status != "" && !strings.EqualFold(item.Status, status) {
				continue
			}
			filtered = append(filtered, item)
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			return filtered[i].IngestedMS > filtered[j].IngestedMS
		})

		statusCounts := map[string]int{}
		categoryCounts := map[string]int{}
		for _, item := range out {
			statusCounts[item.Status]++
			categoryCounts[item.Category]++
		}

		jsonWrite(w, http.StatusOK, map[string]any{
			"items":           filtered,
			"total":           len(out),
			"status_counts":   statusCounts,
			"category_counts": categoryCounts,
			"generated_at_ms": time.Now().UnixMilli(),
		})
	case http.MethodPost:
		var item ResearchItem
		if !jsonRead(w, r, &item) {
			return
		}
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Source) == "" {
			http.Error(w, `{"error":"title and source are required"}`, http.StatusBadRequest)
			return
		}
		now := time.Now().UnixMilli()
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		if item.Status == "" {
			item.Status = researchStatusNew
		}
		if item.Category == "" {
			item.Category = "ai_general"
		}
		if item.IngestedMS == 0 {
			item.IngestedMS = now
		}
		item.UpdatedMS = now

		s.platform.mu.Lock()
		s.platform.appendResearch(item)
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "research.ingested",
			Status:      item.Status,
			Subject:     item.ID,
			Description: fmt.Sprintf("Research item ingested: %s", item.Title),
			CreatedMS:   now,
		})
		s.platform.mu.Unlock()
		jsonWrite(w, http.StatusCreated, map[string]any{"id": item.ID, "item": item})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleResearchFeedItem(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/platform/research-feed/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(path, "/promote") {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.promoteResearchItem(w, r, strings.TrimSuffix(path, "/promote"))
		return
	}

	id := strings.TrimRight(path, "/")
	switch r.Method {
	case http.MethodGet:
		s.platform.mu.Lock()
		var item *ResearchItem
		for i := range s.platform.Research {
			if s.platform.Research[i].ID == id {
				clone := s.platform.Research[i]
				item = &clone
				break
			}
		}
		s.platform.mu.Unlock()
		if item == nil {
			http.NotFound(w, r)
			return
		}
		jsonWrite(w, http.StatusOK, item)
	case http.MethodPatch:
		s.patchResearchItem(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) patchResearchItem(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Status         *string                `json:"status,omitempty"`
		OperatorNotes  *string                `json:"operator_notes,omitempty"`
		Indicators     []ResearchIndicator    `json:"indicators,omitempty"`
		Suggestion     *ResearchSuggestedRule `json:"suggested_detection,omitempty"`
		ProposedPackID *string                `json:"proposed_pack_id,omitempty"`
		RiskScore      *int                   `json:"risk_score,omitempty"`
	}
	if !jsonRead(w, r, &body) {
		return
	}
	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.Research {
		if s.platform.Research[i].ID != id {
			continue
		}
		now := time.Now().UnixMilli()
		if body.Status != nil && strings.TrimSpace(*body.Status) != "" {
			s.platform.Research[i].Status = *body.Status
		}
		if body.OperatorNotes != nil {
			s.platform.Research[i].OperatorNotes = *body.OperatorNotes
		}
		if body.Indicators != nil {
			s.platform.Research[i].Indicators = body.Indicators
		}
		if body.Suggestion != nil {
			s.platform.Research[i].SuggestedDetection = *body.Suggestion
		}
		if body.ProposedPackID != nil {
			s.platform.Research[i].ProposedPackID = *body.ProposedPackID
		}
		if body.RiskScore != nil {
			s.platform.Research[i].RiskScore = *body.RiskScore
		}
		s.platform.Research[i].UpdatedMS = now
		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "research.updated",
			Status:      s.platform.Research[i].Status,
			Subject:     id,
			Description: "Research item updated",
			CreatedMS:   now,
		})
		jsonWrite(w, http.StatusOK, s.platform.Research[i])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) promoteResearchItem(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		PackName     string   `json:"pack_name,omitempty"`
		PackScope    string   `json:"pack_scope,omitempty"`
		PackPolicies []string `json:"pack_policies,omitempty"`
	}
	_ = jsonRead(w, r, &body)

	s.platform.mu.Lock()
	defer s.platform.mu.Unlock()
	for i := range s.platform.Research {
		if s.platform.Research[i].ID != id {
			continue
		}
		now := time.Now().UnixMilli()
		packID := s.platform.Research[i].ProposedPackID
		if packID == "" {
			packID = "pack-" + strings.Split(uuid.NewString(), "-")[0]
		}
		s.platform.Research[i].ProposedPackID = packID
		s.platform.Research[i].Status = researchStatusPromoted
		s.platform.Research[i].UpdatedMS = now

		// Seed a detection candidate so the rest of the workflow (simulate /
		// review / sign / deploy / retire) is wired up. This becomes the
		// linked candidate for the research item.
		candidate := DetectionCandidate{
			ID:                uuid.NewString(),
			SourceResearchID:  s.platform.Research[i].ID,
			Title:             s.platform.Research[i].Title,
			Category:          s.platform.Research[i].Category,
			Status:            candidateStatusNew,
			Rule:              s.platform.Research[i].SuggestedDetection,
			OperatorNotes:     s.platform.Research[i].OperatorNotes,
			QualityGate:       DetectionCandidateGate{RequiredEvidence: append([]string(nil), s.platform.Research[i].EvidenceRequired...)},
			CreatedMS:         now,
			UpdatedMS:         now,
		}
		candidate.History = append(candidate.History, DetectionCandidateEvent{
			ID: uuid.NewString(), AtMS: now, Action: "created", To: candidateStatusNew, Actor: "system",
			Note: fmt.Sprintf("Seeded from research item %s on promotion.", id),
		})
		candidate = recomputeCandidateGate(candidate)
		s.platform.appendCandidate(candidate)
		s.platform.Research[i].LinkedCandidateID = candidate.ID

		s.platform.appendOp(OperationalEvent{
			ID:          uuid.NewString(),
			EventType:   "research.promoted",
			Status:      researchStatusPromoted,
			Subject:     id,
			Description: fmt.Sprintf("Research item promoted to pack %s and seeded candidate %s (governed; observe-only)", packID, candidate.ID),
			CreatedMS:   now,
		})
		jsonWrite(w, http.StatusOK, map[string]any{
			"id":              id,
			"item":            s.platform.Research[i],
			"proposed_pack":   packID,
			"candidate":       candidate,
			"candidate_id":    candidate.ID,
			"observe_only":    true,
			"governance_note": "Promoted as governed observe-only opportunity. The seeded candidate must pass quality gates (simulation, reviewer notes, expiration, rollback) before signing.",
		})
		return
	}
	http.NotFound(w, r)
}
