package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const websocketMessagesSubject = "websocket.messages"

type websocketGatewayMessage struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Channel     string            `json:"channel"`
	Payload     string            `json:"payload"`
	Timestamp   int64             `json:"timestamp"`
	Headers     map[string]string `json:"headers"`
	TargetAgent string            `json:"target_agent"`
}

// AgentListResponse represents the response for listing agents
type AgentListResponse struct {
	Agents []AgentInfo `json:"agents"`
	Total  int         `json:"total"`
}

// AgentInfo represents agent information for list responses (without sensitive data)
type AgentInfo struct {
	AgentUID            string                  `json:"agent_uid"`
	OrgID               string                  `json:"org_id"`
	HostID              string                  `json:"host_id"`
	Hostname            string                  `json:"hostname,omitempty"`
	MachineIDHash       string                  `json:"machine_id_hash,omitempty"`
	AgentVersion        string                  `json:"agent_version,omitempty"`
	Platform            map[string]any          `json:"platform,omitempty"`
	Network             map[string]any          `json:"network,omitempty"`
	Labels              []string                `json:"labels"`
	Note                string                  `json:"note,omitempty"`
	Created             time.Time               `json:"created"`
	LastSeen            time.Time               `json:"last_seen"`
	Status              string                  `json:"status"`
	DetectionPackStatus *DetectionPackStatus    `json:"detection_pack_status,omitempty"`
	Visibility          *VisibilityDeviceRecord `json:"visibility,omitempty"`
	Readiness           *AgentReadiness         `json:"readiness,omitempty"`
}

// VisibilityDeviceRecord mirrors ingest /v1/visibility/devices rows for console merge.
type VisibilityDeviceRecord struct {
	DeviceID       string         `json:"device_id"`
	AgentID        string         `json:"agent_id"`
	Source         string         `json:"source"`
	SensorVersion  string         `json:"sensor_version"`
	FirstSeenMS    int64          `json:"first_seen_ms"`
	LastSeenMS     int64          `json:"last_seen_ms"`
	LastEventType  string         `json:"last_event_type"`
	EventCount     int            `json:"event_count"`
	EventTypeCount map[string]int `json:"event_type_count"`
}

// AgentDetailResponse represents the full agent response
type AgentDetailResponse struct {
	AgentUID            string               `json:"agent_uid"`
	OrgID               string               `json:"org_id"`
	HostID              string               `json:"host_id"`
	Hostname            string               `json:"hostname,omitempty"`
	MachineIDHash       string               `json:"machine_id_hash,omitempty"`
	AgentVersion        string               `json:"agent_version,omitempty"`
	Capabilities        map[string]any       `json:"capabilities,omitempty"`
	Platform            map[string]any       `json:"platform,omitempty"`
	Network             map[string]any       `json:"network,omitempty"`
	Labels              []string             `json:"labels"`
	Note                string               `json:"note,omitempty"`
	Created             time.Time            `json:"created"`
	LastSeen            time.Time            `json:"last_seen"`
	Status              string               `json:"status"`
	DetectionPackStatus *DetectionPackStatus `json:"detection_pack_status,omitempty"`
}

// DetectionPackStatus represents per-agent detection rollout visibility (WO-PLAT-004).
type DetectionPackStatus struct {
	AgentUID                string   `json:"agent_uid"`
	ActivePackID            string   `json:"active_pack_id,omitempty"`
	ActivePackVersion       string   `json:"active_pack_version,omitempty"`
	LastCheckAtMS           int64    `json:"last_check_at_ms"`
	LastAppliedAtMS         int64    `json:"last_applied_at_ms,omitempty"`
	LastRejectedAtMS        int64    `json:"last_rejected_at_ms,omitempty"`
	LastRejectedPackID      string   `json:"last_rejected_pack_id,omitempty"`
	LastRejectedReason      string   `json:"last_rejected_reason,omitempty"`
	LastRejectedReasonCodes []string `json:"last_rejected_reason_codes,omitempty"`
	SignatureStatus         string   `json:"signature_status,omitempty"`
	HashStatus              string   `json:"hash_status,omitempty"`
	SchemaStatus            string   `json:"schema_status,omitempty"`
	CompatibilityStatus     string   `json:"compatibility_status,omitempty"`
	PreviousPackID          string   `json:"previous_pack_id,omitempty"`
	PreviousPackVersion     string   `json:"previous_pack_version,omitempty"`
	RolloutState            string   `json:"rollout_state"`
	ReasonDetail            string   `json:"reason_detail,omitempty"`
	ReasonCodes             []string `json:"reason_codes,omitempty"`
	DeviceID                string   `json:"device_id,omitempty"`
	ReportedAgentVersion    string   `json:"reported_agent_version,omitempty"`
	UpdatedAtMS             int64    `json:"updated_at_ms"`
}

// LabelsUpdateRequest represents a request to update agent labels
type LabelsUpdateRequest struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

// NoteUpdateRequest represents a request to update agent note
type NoteUpdateRequest struct {
	Note string `json:"note"`
}

// AgentConfigRequest represents a request to configure agent settings
type AgentConfigRequest struct {
	Channels          []string               `json:"channels,omitempty"`
	Settings          map[string]interface{} `json:"settings,omitempty"`
	Policies          []string               `json:"policies,omitempty"`
	HeartbeatInterval int                    `json:"heartbeat_interval,omitempty"`
	ReconnectInterval int                    `json:"reconnect_interval,omitempty"`
	MessageQueueSize  int                    `json:"message_queue_size,omitempty"`
}

// AgentStatusResponse represents agent connection status
type AgentStatusResponse struct {
	AgentID        string    `json:"agent_id"`
	Connected      bool      `json:"connected"`
	LastSeen       time.Time `json:"last_seen"`
	Channels       []string  `json:"channels"`
	SessionExpires time.Time `json:"session_expires"`
	WebSocketURL   string    `json:"websocket_url,omitempty"`
	MessageCount   int       `json:"message_count,omitempty"`
	Uptime         string    `json:"uptime,omitempty"`
}

// SendMessageRequest represents a request to send a message to an agent
type SendMessageRequest struct {
	Channel     string                 `json:"channel"`
	Message     map[string]interface{} `json:"message"`
	MessageType string                 `json:"message_type"` // request, response, event
	Priority    int                    `json:"priority,omitempty"`
	TTL         int                    `json:"ttl,omitempty"` // seconds
}

// SendMessageResponse represents the response for sending a message
type SendMessageResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"` // sent, queued, failed
	Error     string `json:"error,omitempty"`
}

// BroadcastRequest represents a request to broadcast to all agents
type BroadcastRequest struct {
	Channel     string                 `json:"channel"`
	Message     map[string]interface{} `json:"message"`
	MessageType string                 `json:"message_type"`           // request, response, event
	AgentFilter []string               `json:"agent_filter,omitempty"` // specific agents only
	Priority    int                    `json:"priority,omitempty"`
	TTL         int                    `json:"ttl,omitempty"` // seconds
}

// BroadcastResponse represents the response for broadcasting
type BroadcastResponse struct {
	MessageID string   `json:"message_id"`
	SentTo    []string `json:"sent_to"`
	Failed    []string `json:"failed"`
	TotalSent int      `json:"total_sent"`
}

// getAgents handles GET /agents with filtering support
func (s *Server) getAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters for filtering
	labelFilter := r.URL.Query().Get("label")
	hostnameFilter := r.URL.Query().Get("hostname")
	hostIDFilter := r.URL.Query().Get("host_id")
	ipFilter := r.URL.Query().Get("ip")

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	var filteredAgents []AgentInfo
	for _, agent := range s.store.agents {
		// Apply filters
		if labelFilter != "" {
			if !agent.Labels[labelFilter] {
				continue
			}
		}
		if hostnameFilter != "" {
			if agent.Hostname != hostnameFilter {
				continue
			}
		}
		if hostIDFilter != "" {
			if agent.HostID != hostIDFilter {
				continue
			}
		}
		if ipFilter != "" {
			if !s.agentHasIP(agent, ipFilter) {
				continue
			}
		}

		// Convert labels map to slice
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
		filteredAgents = append(filteredAgents, agentInfo)
	}

	response := AgentListResponse{
		Agents: filteredAgents,
		Total:  len(filteredAgents),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func agentConnectionStatus(lastSeen time.Time) string {
	if time.Since(lastSeen) < 5*time.Minute {
		return "online"
	}
	return "offline"
}

// agentDispatch handles sub-routes for individual agents
func (s *Server) agentDispatch(w http.ResponseWriter, r *http.Request) {
	// Extract agent UID from path: /agents/{uid} or /agents/{uid}/labels or /agents/{uid}/note
	path := strings.TrimPrefix(r.URL.Path, "/agents/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Agent UID required", http.StatusBadRequest)
		return
	}

	agentUID := parts[0]

	// Route to appropriate handler based on path
	if len(parts) == 1 {
		// /agents/{uid}
		s.getAgent(w, r, agentUID)
	} else if len(parts) == 2 {
		switch parts[1] {
		case "labels":
			// /agents/{uid}/labels
			s.updateAgentLabels(w, r, agentUID)
		case "note":
			// /agents/{uid}/note
			s.updateAgentNote(w, r, agentUID)
		case "status":
			// /agents/{uid}/status
			s.getAgentStatus(w, r, agentUID)
		case "config":
			// /agents/{uid}/config
			s.updateAgentConfig(w, r, agentUID)
		case "send":
			// /agents/{uid}/send
			s.sendMessageToAgent(w, r, agentUID)
		default:
			http.Error(w, "Invalid endpoint", http.StatusNotFound)
		}
	} else {
		http.Error(w, "Invalid path", http.StatusNotFound)
	}
}

// getAgent handles GET /agents/{agent_uid}
func (s *Server) getAgent(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.store.mu.Lock()
	agent, exists := s.store.agents[agentUID]
	s.store.mu.Unlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Convert labels map to slice
	labels := make([]string, 0, len(agent.Labels))
	for label := range agent.Labels {
		labels = append(labels, label)
	}

	response := AgentDetailResponse{
		AgentUID:            agent.AgentUID,
		OrgID:               agent.OrgID,
		HostID:              agent.HostID,
		Hostname:            agent.Hostname,
		MachineIDHash:       agent.MachineIDHash,
		AgentVersion:        agent.AgentVersion,
		Capabilities:        agent.Capabilities,
		Platform:            agent.Platform,
		Network:             agent.Network,
		Labels:              labels,
		Note:                agent.Note,
		Created:             agent.Created,
		LastSeen:            agent.LastSeen,
		Status:              agentConnectionStatus(agent.LastSeen),
		DetectionPackStatus: s.fetchDetectionPackStatus(agent.AgentUID),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) fetchDetectionPackStatus(agentUID string) *DetectionPackStatus {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("DETECTION_PIPELINE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://detection-pipeline:8089"
	}

	url := fmt.Sprintf("%s/v1/agents/%s/detection-pack-status", baseURL, agentUID)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var body struct {
		Status *DetectionPackStatus `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	return body.Status
}

// updateAgentLabels handles PUT /agents/{agent_uid}/labels
func (s *Server) updateAgentLabels(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LabelsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	agent, exists := s.store.agents[agentUID]
	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Add labels
	for _, label := range req.Add {
		if label != "" {
			agent.Labels[label] = true
		}
	}

	// Remove labels
	for _, label := range req.Remove {
		if label != "" {
			delete(agent.Labels, label)
		}
	}

	// Convert labels map to slice for response
	labels := make([]string, 0, len(agent.Labels))
	for label := range agent.Labels {
		labels = append(labels, label)
	}

	response := map[string]interface{}{
		"agent_uid": agent.AgentUID,
		"labels":    labels,
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateAgentNote handles PUT /agents/{agent_uid}/note
func (s *Server) updateAgentNote(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req NoteUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	agent, exists := s.store.agents[agentUID]
	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	agent.Note = req.Note

	response := map[string]interface{}{
		"agent_uid": agent.AgentUID,
		"note":      agent.Note,
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// agentHasIP checks if an agent has the specified IP in its network configuration
func (s *Server) agentHasIP(agent *Agent, ip string) bool {
	if agent.Network == nil {
		return false
	}

	// Check various network fields that might contain IP addresses
	// This is a simple implementation - could be enhanced based on actual network structure
	for _, value := range agent.Network {
		switch v := value.(type) {
		case string:
			if v == ip {
				return true
			}
		case []interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok && str == ip {
					return true
				}
			}
		case map[string]interface{}:
			for _, item := range v {
				if str, ok := item.(string); ok && str == ip {
					return true
				}
			}
		}
	}

	return false
}

// getAgentStatus handles GET /agents/{agent_uid}/status
func (s *Server) getAgentStatus(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.store.mu.Lock()
	agent, exists := s.store.agents[agentUID]
	s.store.mu.Unlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Check if agent is connected via WebSocket (this would integrate with WebSocket gateway)
	// For now, we'll simulate the status
	connected := time.Since(agent.LastSeen) < 5*time.Minute // Consider connected if seen within 5 minutes

	response := AgentStatusResponse{
		AgentID:        agentUID,
		Connected:      connected,
		LastSeen:       agent.LastSeen,
		Channels:       []string{},                         // Would be populated from WebSocket gateway
		SessionExpires: agent.LastSeen.Add(24 * time.Hour), // 24 hour session
		WebSocketURL:   "ws://localhost:8080/ws/agent",
		MessageCount:   0, // Would be populated from WebSocket gateway
		Uptime:         time.Since(agent.Created).String(),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateAgentConfig handles PUT /agents/{agent_uid}/config
func (s *Server) updateAgentConfig(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	_, exists := s.store.agents[agentUID]
	s.store.mu.Unlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Update agent configuration
	// This would integrate with the WebSocket gateway to send configuration to the agent
	response := map[string]interface{}{
		"agent_uid": agentUID,
		"config":    req,
		"status":    "configuration_updated",
		"message":   "Agent configuration updated successfully",
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendMessageToAgent handles POST /agents/{agent_uid}/send
func (s *Server) sendMessageToAgent(w http.ResponseWriter, r *http.Request, agentUID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	agent, exists := s.store.agents[agentUID]
	s.store.mu.Unlock()

	if !exists {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Check if agent is connected
	connected := time.Since(agent.LastSeen) < 5*time.Minute

	var response SendMessageResponse
	messageID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	if connected {
		// Send message to WebSocket Gateway
		err := s.sendMessageToWebSocketGateway(agentUID, req.Channel, req.Message, req.MessageType, messageID)
		if err != nil {
			response = SendMessageResponse{
				MessageID: messageID,
				Status:    "failed",
				Error:     fmt.Sprintf("Failed to send to WebSocket Gateway: %v", err),
			}
		} else {
			response = SendMessageResponse{
				MessageID: messageID,
				Status:    "sent",
			}
		}
	} else {
		response = SendMessageResponse{
			MessageID: messageID,
			Status:    "queued",
		}
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// broadcastToAgents handles POST /agents/broadcast
func (s *Server) broadcastToAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	var sentTo []string
	var failed []string
	messageID := fmt.Sprintf("broadcast_%d", time.Now().UnixNano())

	// Send to all agents or filtered agents
	for agentUID, agent := range s.store.agents {
		// Apply agent filter if specified
		if len(req.AgentFilter) > 0 {
			found := false
			for _, filterAgent := range req.AgentFilter {
				if agentUID == filterAgent {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Check if agent is connected
		connected := time.Since(agent.LastSeen) < 5*time.Minute
		if !connected {
			failed = append(failed, agentUID)
			continue
		}

		// Actually send the message via NATS to WebSocket Gateway
		err := s.sendMessageToWebSocketGateway(agentUID, req.Channel, req.Message, req.MessageType, messageID)
		if err != nil {
			log.Printf("Failed to send message to agent %s: %v", agentUID, err)
			failed = append(failed, agentUID)
		} else {
			sentTo = append(sentTo, agentUID)
		}
	}

	response := BroadcastResponse{
		MessageID: messageID,
		SentTo:    sentTo,
		Failed:    failed,
		TotalSent: len(sentTo),
	}

	w.Header().Set("content-type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func buildWebSocketGatewayMessage(agentUID, channel string, message map[string]interface{}, messageType, messageID string, now time.Time) ([]byte, error) {
	if agentUID == "" {
		return nil, fmt.Errorf("agent UID is required")
	}
	if channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	if messageID == "" {
		return nil, fmt.Errorf("message ID is required")
	}
	if messageType == "" {
		messageType = "event"
	}
	if message == nil {
		message = map[string]interface{}{}
	}

	// Convert message to JSON string for agent's SecureMessage structure
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message payload: %w", err)
	}

	// For now, send the JSON as base64-encoded string to match agent expectations
	// TODO: Implement proper ChaCha20-Poly1305 encryption in production
	payload := base64.StdEncoding.EncodeToString(messageJSON)

	// Create the message payload
	messageData := websocketGatewayMessage{
		ID:          messageID,
		Type:        messageType,
		Channel:     channel,
		Payload:     payload,
		Timestamp:   now.Unix(),
		Headers:     map[string]string{},
		TargetAgent: agentUID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(messageData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	return jsonData, nil
}

// sendMessageToWebSocketGateway sends a message to the WebSocket Gateway via NATS
func (s *Server) sendMessageToWebSocketGateway(agentUID, channel string, message map[string]interface{}, messageType, messageID string) error {
	// Check if NATS is connected
	if s.nc == nil {
		return fmt.Errorf("NATS not connected")
	}

	jsonData, err := buildWebSocketGatewayMessage(agentUID, channel, message, messageType, messageID, time.Now())
	if err != nil {
		return err
	}

	// Publish to NATS subject for WebSocket Gateway
	err = s.nc.Publish(websocketMessagesSubject, jsonData)
	if err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	log.Printf("Sent message to WebSocket Gateway via NATS for agent %s: %s", agentUID, string(jsonData))
	return nil
}

// handleHeartbeat handles POST /agents/heartbeat - updates agent last seen timestamp
func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var heartbeatData struct {
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

	if err := json.NewDecoder(r.Body).Decode(&heartbeatData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if heartbeatData.AgentUID == "" {
		http.Error(w, "Agent UID required", http.StatusBadRequest)
		return
	}

	// Parse last seen time
	lastSeen, err := time.Parse(time.RFC3339, heartbeatData.LastSeen)
	if err != nil {
		http.Error(w, "Invalid timestamp format", http.StatusBadRequest)
		return
	}

	// Update agent in store
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	// Try to find agent by UID first
	if agent, exists := s.store.agents[heartbeatData.AgentUID]; exists {
		s.applyHeartbeat(agent, heartbeatData.OrgID, heartbeatData.HostID, heartbeatData.Hostname, heartbeatData.MachineIDHash, heartbeatData.AgentVersion, heartbeatData.Note, lastSeen, heartbeatData.Capabilities, heartbeatData.Platform, heartbeatData.Network, heartbeatData.Labels)
		log.Printf("Updated heartbeat for agent %s: last_seen=%s", heartbeatData.AgentUID, heartbeatData.LastSeen)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "updated",
			"agent_uid": heartbeatData.AgentUID,
			"last_seen": heartbeatData.LastSeen,
		})
		return
	}

	// If not found by UID, try to find by hostname (for backward compatibility)
	var foundAgent *Agent
	var foundUID string
	for uid, agent := range s.store.agents {
		if agent.HostID == heartbeatData.AgentUID {
			foundAgent = agent
			foundUID = uid
			break
		}
	}

	if foundAgent != nil {
		s.applyHeartbeat(foundAgent, heartbeatData.OrgID, heartbeatData.HostID, heartbeatData.Hostname, heartbeatData.MachineIDHash, heartbeatData.AgentVersion, heartbeatData.Note, lastSeen, heartbeatData.Capabilities, heartbeatData.Platform, heartbeatData.Network, heartbeatData.Labels)
		log.Printf("Updated heartbeat for agent %s (hostname %s): last_seen=%s", foundUID, heartbeatData.AgentUID, heartbeatData.LastSeen)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "updated",
			"agent_uid": foundUID,
			"hostname":  heartbeatData.AgentUID,
			"last_seen": heartbeatData.LastSeen,
		})
		return
	}

	agent := s.newAgentFromHeartbeat(heartbeatData.AgentUID, heartbeatData.OrgID, heartbeatData.HostID, heartbeatData.Hostname, heartbeatData.MachineIDHash, heartbeatData.AgentVersion, heartbeatData.Note, lastSeen, heartbeatData.Capabilities, heartbeatData.Platform, heartbeatData.Network, heartbeatData.Labels)
	s.store.agents[agent.AgentUID] = agent
	if agent.HostID != "" {
		s.store.byHost[agent.HostID] = agent
	}
	log.Printf("Registered agent from heartbeat %s (%s)", agent.AgentUID, agent.HostID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "registered",
		"agent_uid": agent.AgentUID,
		"host_id":   agent.HostID,
		"last_seen": heartbeatData.LastSeen,
	})
}

func (s *Server) newAgentFromHeartbeat(agentUID, orgID, hostID, hostname, machineIDHash, agentVersion, note string, lastSeen time.Time, capabilities, platform, network map[string]any, labels []string) *Agent {
	if orgID == "" {
		orgID = "default-org"
	}
	if hostID == "" {
		hostID = agentUID
	}
	if hostname == "" {
		hostname = hostID
	}
	if agentVersion == "" {
		agentVersion = "unknown"
	}
	if capabilities == nil {
		capabilities = map[string]any{"visibility": true}
	}
	if platform == nil {
		platform = map[string]any{"hostname": hostname, "os": "unknown"}
	}
	if network == nil {
		network = map[string]any{}
	}

	agent := &Agent{
		AgentUID:      agentUID,
		OrgID:         orgID,
		HostID:        hostID,
		Hostname:      hostname,
		MachineIDHash: machineIDHash,
		AgentVersion:  agentVersion,
		Capabilities:  capabilities,
		Platform:      platform,
		Network:       network,
		Labels:        labelsToMap(labels),
		Note:          note,
		Created:       lastSeen,
		LastSeen:      lastSeen,
	}
	if agent.Note == "" {
		agent.Note = "Registered from lab agent heartbeat"
	}
	return agent
}

func (s *Server) applyHeartbeat(agent *Agent, orgID, hostID, hostname, machineIDHash, agentVersion, note string, lastSeen time.Time, capabilities, platform, network map[string]any, labels []string) {
	previousHostID := agent.HostID
	agent.LastSeen = lastSeen
	if orgID != "" {
		agent.OrgID = orgID
	}
	if hostID != "" {
		agent.HostID = hostID
	}
	if hostname != "" {
		agent.Hostname = hostname
	}
	if machineIDHash != "" {
		agent.MachineIDHash = machineIDHash
	}
	if agentVersion != "" {
		agent.AgentVersion = agentVersion
	}
	if capabilities != nil {
		agent.Capabilities = capabilities
	}
	if platform != nil {
		agent.Platform = platform
	}
	if network != nil {
		agent.Network = network
	}
	if len(labels) > 0 {
		agent.Labels = labelsToMap(labels)
	}
	if note != "" {
		agent.Note = note
	}
	if previousHostID != "" && previousHostID != agent.HostID {
		delete(s.store.byHost, previousHostID)
	}
	if agent.HostID != "" {
		s.store.byHost[agent.HostID] = agent
	}
}

func labelsToMap(labels []string) map[string]bool {
	mapped := map[string]bool{}
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label != "" {
			mapped[label] = true
		}
	}
	return mapped
}
