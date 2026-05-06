package store

import (
	"strings"

	"aegisflux/backend/detection-pipeline/internal/model"
)

func (s *Store) PutAgentPackStatus(st *model.AgentPackStatus) error {
	s.mu.Lock()
	s.AgentPackStatus[st.AgentUID] = st
	s.mu.Unlock()
	return s.persist()
}

func (s *Store) GetAgentPackStatus(agentUID string) (*model.AgentPackStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.AgentPackStatus[agentUID]
	return v, ok
}

func (s *Store) ListAgentPackStatuses() []*model.AgentPackStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.AgentPackStatus, 0, len(s.AgentPackStatus))
	for _, v := range s.AgentPackStatus {
		out = append(out, v)
	}
	return out
}

// ListAgentPackStatusesForPackID returns statuses where the agent reports this pack as active or rejected.
func (s *Store) ListAgentPackStatusesForPackID(packID string) []*model.AgentPackStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pid := strings.TrimSpace(packID)
	out := make([]*model.AgentPackStatus, 0)
	for _, v := range s.AgentPackStatus {
		if v == nil {
			continue
		}
		if v.ActivePackID == pid || v.LastRejectedPackID == pid {
			out = append(out, v)
		}
	}
	return out
}
