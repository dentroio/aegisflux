package store

import (
	"fmt"

	"aegisflux/backend/detection-pipeline/internal/model"
)

func (s *Store) PutResearch(r *model.ResearchItem) error {
	s.mu.Lock()
	s.Research[r.ID] = r
	s.mu.Unlock()
	return s.persist()
}

func (s *Store) ListResearch() []*model.ResearchItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.ResearchItem, 0, len(s.Research))
	for _, v := range s.Research {
		out = append(out, v)
	}
	return out
}

func (s *Store) GetResearch(id string) (*model.ResearchItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.Research[id]
	return v, ok
}

func (s *Store) PutCandidate(c *model.Candidate) error {
	s.mu.Lock()
	s.Candidates[c.ID] = c
	s.mu.Unlock()
	return s.persist()
}

func (s *Store) GetCandidate(id string) (*model.Candidate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.Candidates[id]
	return v, ok
}

func (s *Store) ListCandidates() []*model.Candidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.Candidate, 0, len(s.Candidates))
	for _, v := range s.Candidates {
		out = append(out, v)
	}
	return out
}

func (s *Store) PutValidation(v *model.ValidationRun) error {
	s.mu.Lock()
	s.Validations[v.ID] = v
	s.mu.Unlock()
	return s.persist()
}

func (s *Store) ListValidationsForCandidate(candidateID string) []*model.ValidationRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.ValidationRun, 0)
	for _, v := range s.Validations {
		if v.CandidateID == candidateID {
			out = append(out, v)
		}
	}
	return out
}

func (s *Store) PutSigned(a *model.SignedPackArtifact) error {
	s.mu.Lock()
	s.SignedPacks[a.ID] = a
	s.mu.Unlock()
	return s.persist()
}

func (s *Store) GetSigned(id string) (*model.SignedPackArtifact, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.SignedPacks[id]
	return v, ok
}

func (s *Store) ListSigned() []*model.SignedPackArtifact {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.SignedPackArtifact, 0, len(s.SignedPacks))
	for _, v := range s.SignedPacks {
		out = append(out, v)
	}
	return out
}

func (s *Store) RequireResearch(id string) error {
	if _, ok := s.GetResearch(id); !ok {
		return fmt.Errorf("research_item %q not found", id)
	}
	return nil
}
