package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAgentListLimit = 100
	maxAgentListLimit     = 1000
	maxBatchHeartbeats    = 500
)

type heartbeatPayload struct {
	AgentUID      string         `json:"agent_uid"`
	OrgID         string         `json:"org_id"`
	HostID        string         `json:"host_id"`
	Hostname      string         `json:"hostname"`
	MachineIDHash string         `json:"machine_id_hash"`
	AgentVersion  string         `json:"agent_version"`
	LastSeen      string         `json:"last_seen"`
	Status        string         `json:"status"`
	Capabilities  map[string]any `json:"capabilities"`
	Platform      map[string]any `json:"platform"`
	Network       map[string]any `json:"network"`
	Labels        []string       `json:"labels"`
	Note          string         `json:"note"`
}

func parseAgentListLimit(raw string) int {
	if raw == "" {
		return defaultAgentListLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultAgentListLimit
	}
	if n > maxAgentListLimit {
		return maxAgentListLimit
	}
	return n
}

func parseAgentListOffset(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func queryBoolDefaultFalse(r *http.Request, name string) bool {
	v := strings.TrimSpace(r.URL.Query().Get(name))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func (s *Server) handleBatchHeartbeats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Heartbeats []heartbeatPayload `json:"heartbeats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Heartbeats) == 0 {
		http.Error(w, "heartbeats array required", http.StatusBadRequest)
		return
	}
	if len(req.Heartbeats) > maxBatchHeartbeats {
		http.Error(w, "too many heartbeats in one request", http.StatusBadRequest)
		return
	}

	updated := 0
	registered := 0
	failed := 0
	s.store.mu.Lock()
	for _, hb := range req.Heartbeats {
		if hb.AgentUID == "" {
			failed++
			continue
		}
		lastSeen, err := time.Parse(time.RFC3339, hb.LastSeen)
		if err != nil {
			failed++
			continue
		}
		if agent, exists := s.store.agents[hb.AgentUID]; exists {
			s.applyHeartbeat(agent, hb.OrgID, hb.HostID, hb.Hostname, hb.MachineIDHash, hb.AgentVersion, hb.Note, lastSeen, hb.Capabilities, hb.Platform, hb.Network, hb.Labels)
			updated++
			continue
		}
		agent := s.newAgentFromHeartbeat(hb.AgentUID, hb.OrgID, hb.HostID, hb.Hostname, hb.MachineIDHash, hb.AgentVersion, hb.Note, lastSeen, hb.Capabilities, hb.Platform, hb.Network, hb.Labels)
		s.store.agents[agent.AgentUID] = agent
		if agent.HostID != "" {
			s.store.byHost[agent.HostID] = agent
		}
		registered++
	}
	s.store.mu.Unlock()

	accepted := updated + registered
	for i := 0; i < accepted; i++ {
		incHeartbeatAccepted()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"updated":    updated,
		"registered": registered,
		"failed":     failed,
		"accepted":   accepted,
	})
}

func sortAgentsByLastSeen(agents []AgentInfo) {
	sort.Slice(agents, func(i, j int) bool {
		if agents[i].LastSeen.Equal(agents[j].LastSeen) {
			return agents[i].AgentUID < agents[j].AgentUID
		}
		return agents[i].LastSeen.After(agents[j].LastSeen)
	})
}

func paginateAgents(agents []AgentInfo, offset, limit int) []AgentInfo {
	if offset >= len(agents) {
		return []AgentInfo{}
	}
	end := offset + limit
	if end > len(agents) {
		end = len(agents)
	}
	return agents[offset:end]
}
