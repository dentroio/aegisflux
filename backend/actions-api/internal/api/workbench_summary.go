package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func (s *Server) getAgentsWorkbenchSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ingestURL := os.Getenv("INGEST_API_URL")
	if ingestURL == "" {
		ingestURL = "http://localhost:9091"
	}

	byHost := map[string]*VisibilityDeviceRecord{}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Get(ingestURL + "/v1/visibility/devices?limit=120")
	if err != nil {
		log.Printf("workbench summary: ingest devices: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("workbench summary: ingest returned HTTP %d", resp.StatusCode)
		} else {
			var payload struct {
				Devices []VisibilityDeviceRecord `json:"devices"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				log.Printf("workbench summary: decode ingest: %v", err)
			} else {
				for i := range payload.Devices {
					d := &payload.Devices[i]
					byHost[d.DeviceID] = d
					if d.AgentID != "" {
						byHost[d.AgentID] = d
					}
				}
			}
		}
	}

	currentBaseline := os.Getenv("AEGISFLUX_AGENT_BASELINE_VERSION")
	now := time.Now()

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	filteredAgents := make([]AgentInfo, 0, len(s.store.agents))
	for _, agent := range s.store.agents {
		labels := make([]string, 0, len(agent.Labels))
		for label := range agent.Labels {
			labels = append(labels, label)
		}

		agentInfo := AgentInfo{
			AgentUID:            agent.AgentUID,
			OrgID:               agent.OrgID,
			HostID:              agent.HostID,
			Hostname:            agent.Hostname,
			MachineIDHash:       agent.MachineIDHash,
			AgentVersion:        agent.AgentVersion,
			Platform:            agent.Platform,
			Network:             agent.Network,
			Labels:              labels,
			Note:                agent.Note,
			Created:             agent.Created,
			LastSeen:            agent.LastSeen,
			Status:              agentConnectionStatus(agent.LastSeen),
			DetectionPackStatus: s.fetchDetectionPackStatus(agent.AgentUID),
		}
		if v := byHost[agent.HostID]; v != nil {
			agentInfo.Visibility = v
		} else if v := byHost[agent.AgentUID]; v != nil {
			agentInfo.Visibility = v
		}
		readiness := computeAgentReadiness(agentInfo, currentBaseline, now)
		agentInfo.Readiness = &readiness
		filteredAgents = append(filteredAgents, agentInfo)
	}

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(AgentListResponse{
		Agents: filteredAgents,
		Total:  len(filteredAgents),
	})
}
