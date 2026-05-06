package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aegisflux/backend/detection-pipeline/internal/model"
)

// Store is an in-memory registry with optional JSON persistence.
type Store struct {
	mu sync.RWMutex

	path string

	Research       map[string]*model.ResearchItem
	Candidates     map[string]*model.Candidate
	Validations    map[string]*model.ValidationRun
	SignedPacks    map[string]*model.SignedPackArtifact
	AgentPackStatus map[string]*model.AgentPackStatus
}

func New(persistPath string) *Store {
	return &Store{
		path:             persistPath,
		Research:         make(map[string]*model.ResearchItem),
		Candidates:       make(map[string]*model.Candidate),
		Validations:      make(map[string]*model.ValidationRun),
		SignedPacks:      make(map[string]*model.SignedPackArtifact),
		AgentPackStatus:  make(map[string]*model.AgentPackStatus),
	}
}

func (s *Store) Load() error {
	if s.path == "" {
		return nil
	}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Research = snap.Research
	if s.Research == nil {
		s.Research = make(map[string]*model.ResearchItem)
	}
	s.Candidates = snap.Candidates
	if s.Candidates == nil {
		s.Candidates = make(map[string]*model.Candidate)
	}
	s.Validations = snap.Validations
	if s.Validations == nil {
		s.Validations = make(map[string]*model.ValidationRun)
	}
	s.SignedPacks = snap.SignedPacks
	if s.SignedPacks == nil {
		s.SignedPacks = make(map[string]*model.SignedPackArtifact)
	}
	s.AgentPackStatus = snap.AgentPackStatus
	if s.AgentPackStatus == nil {
		s.AgentPackStatus = make(map[string]*model.AgentPackStatus)
	}
	return nil
}

type snapshot struct {
	Research          map[string]*model.ResearchItem        `json:"research_items"`
	Candidates        map[string]*model.Candidate             `json:"candidates"`
	Validations       map[string]*model.ValidationRun       `json:"validations"`
	SignedPacks       map[string]*model.SignedPackArtifact    `json:"signed_packs"`
	AgentPackStatus   map[string]*model.AgentPackStatus     `json:"agent_pack_statuses"`
}

func (s *Store) persist() error {
	if s.path == "" {
		return nil
	}
	s.mu.RLock()
	snap := snapshot{
		Research:         s.Research,
		Candidates:       s.Candidates,
		Validations:      s.Validations,
		SignedPacks:      s.SignedPacks,
		AgentPackStatus:  s.AgentPackStatus,
	}
	s.mu.RUnlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// NowMS returns current Unix time in milliseconds.
func NowMS() int64 {
	return time.Now().UnixMilli()
}
