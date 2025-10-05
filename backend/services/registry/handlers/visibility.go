package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sgerhart/aegisflux/backend/internal/db"
)

// VisibilityHandler handles visibility-related HTTP requests
type VisibilityHandler struct {
	store *db.Store
}

// NewVisibilityHandler creates a new visibility handler
func NewVisibilityHandler(store *db.Store) *VisibilityHandler {
	return &VisibilityHandler{
		store: store,
	}
}

// GetLatestVisibility handles GET /agents/{id}/visibility/latest
func (h *VisibilityHandler) GetLatestVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/visibility/latest"); idx >= 0 && agentID[idx:] == "/visibility/latest" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get latest visibility frame
	frame, err := h.store.GetLatestVisibilityFrame(r.Context(), agentID)
	if err != nil {
		http.Error(w, "Failed to get visibility frame", http.StatusInternalServerError)
		return
	}

	if frame == nil {
		http.Error(w, "No visibility data found for agent", http.StatusNotFound)
		return
	}

	// Create response
	response := VisibilityResponse{
		AgentUID:   frame.AgentUID,
		Timestamp:  frame.Timestamp,
		Processes:  frame.Processes,
		Flows:      frame.Flows,
		Sockets:    frame.Sockets,
		ExecEvents: frame.ExecEvents,
		CreatedAt:  frame.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetVisibilityHistory handles GET /agents/{id}/visibility/history
func (h *VisibilityHandler) GetVisibilityHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/visibility/history"); idx >= 0 && agentID[idx:] == "/visibility/history" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get visibility history
	frames, err := h.store.GetVisibilityHistory(r.Context(), agentID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to get visibility history", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseFrames := make([]VisibilityResponse, len(frames))
	for i, frame := range frames {
		responseFrames[i] = VisibilityResponse{
			AgentUID:   frame.AgentUID,
			Timestamp:  frame.Timestamp,
			Processes:  frame.Processes,
			Flows:      frame.Flows,
			Sockets:    frame.Sockets,
			ExecEvents: frame.ExecEvents,
			CreatedAt:  frame.CreatedAt,
		}
	}

	response := VisibilityHistoryResponse{
		AgentUID: agentID,
		Frames:   responseFrames,
		Total:    len(responseFrames),
		Limit:    limit,
		Offset:   offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEnforcementDecisions handles GET /agents/{id}/enforcement/decisions
func (h *VisibilityHandler) GetEnforcementDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/enforcement/decisions"); idx >= 0 && agentID[idx:] == "/enforcement/decisions" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	verdict := r.URL.Query().Get("verdict")
	mode := r.URL.Query().Get("mode")

	// Get enforcement decisions
	decisions, err := h.store.GetEnforcementDecisions(r.Context(), agentID, verdict, mode, limit)
	if err != nil {
		http.Error(w, "Failed to get enforcement decisions", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseDecisions := make([]EnforcementDecisionResponse, len(decisions))
	for i, decision := range decisions {
		responseDecisions[i] = EnforcementDecisionResponse{
			ID:           decision.ID,
			AssignmentID: decision.AssignmentID,
			AgentUID:     decision.AgentUID,
			Verdict:      decision.Verdict,
			Reason:       decision.Reason,
			RuleID:       decision.RuleID,
			FlowData:     decision.FlowData,
			ProcessData:  decision.ProcessData,
			Timestamp:    decision.Timestamp,
			Mode:         decision.Mode,
		}
	}

	response := EnforcementDecisionsResponse{
		AgentUID:   agentID,
		Decisions:  responseDecisions,
		Total:      len(responseDecisions),
		Limit:      limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetNetworkFlows handles GET /agents/{id}/flows
func (h *VisibilityHandler) GetNetworkFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/flows"); idx >= 0 && agentID[idx:] == "/flows" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	protocol := r.URL.Query().Get("protocol")
	status := r.URL.Query().Get("status")

	// Get network flows
	flows, err := h.store.GetNetworkFlows(r.Context(), agentID, protocol, status, limit)
	if err != nil {
		http.Error(w, "Failed to get network flows", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseFlows := make([]NetworkFlowResponse, len(flows))
	for i, flow := range flows {
		responseFlows[i] = NetworkFlowResponse{
			ID:             flow.ID,
			AgentUID:       flow.AgentUID,
			SrcIP:          flow.SrcIP,
			DstIP:          flow.DstIP,
			SrcPort:        flow.SrcPort,
			DstPort:        flow.DstPort,
			Protocol:       flow.Protocol,
			BytesSent:      flow.BytesSent,
			BytesReceived:  flow.BytesReceived,
			PacketsSent:    flow.PacketsSent,
			PacketsReceived: flow.PacketsReceived,
			StartTime:      flow.StartTime,
			EndTime:        flow.EndTime,
			Status:         flow.Status,
			ProcessID:      flow.ProcessID,
			ProcessName:    flow.ProcessName,
			CreatedAt:      flow.CreatedAt,
		}
	}

	response := NetworkFlowsResponse{
		AgentUID: agentID,
		Flows:    responseFlows,
		Total:    len(responseFlows),
		Limit:    limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProcesses handles GET /agents/{id}/processes
func (h *VisibilityHandler) GetProcesses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/processes"); idx >= 0 && agentID[idx:] == "/processes" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	status := r.URL.Query().Get("status")
	name := r.URL.Query().Get("name")

	// Get processes
	processes, err := h.store.GetProcesses(r.Context(), agentID, status, name, limit)
	if err != nil {
		http.Error(w, "Failed to get processes", http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseProcesses := make([]ProcessResponse, len(processes))
	for i, process := range processes {
		responseProcesses[i] = ProcessResponse{
			ID:               process.ID,
			AgentUID:         process.AgentUID,
			PID:              process.PID,
			PPID:             process.PPID,
			Name:             process.Name,
			Cmdline:          process.Cmdline,
			ExecutablePath:   process.ExecutablePath,
			WorkingDirectory: process.WorkingDirectory,
			UserID:           process.UserID,
			GroupID:          process.GroupID,
			StartTime:        process.StartTime,
			EndTime:          process.EndTime,
			Status:           process.Status,
			MemoryUsage:      process.MemoryUsage,
			CPUUsage:         process.CPUUsage,
			NetworkConnections: process.NetworkConnections,
			CreatedAt:        process.CreatedAt,
		}
	}

	response := ProcessesResponse{
		AgentUID:  agentID,
		Processes: responseProcesses,
		Total:     len(responseProcesses),
		Limit:     limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetVisibilitySummary handles GET /agents/{id}/visibility/summary
func (h *VisibilityHandler) GetVisibilitySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Path[len("/agents/"):]
	if idx := len(agentID) - len("/visibility/summary"); idx >= 0 && agentID[idx:] == "/visibility/summary" {
		agentID = agentID[:idx]
	}

	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Get visibility summary using database function
	summary, err := h.store.GetVisibilitySummary(r.Context(), agentID)
	if err != nil {
		http.Error(w, "Failed to get visibility summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// Response types

type VisibilityResponse struct {
	AgentUID   string          `json:"agent_uid"`
	Timestamp  time.Time       `json:"timestamp"`
	Processes  json.RawMessage `json:"processes"`
	Flows      json.RawMessage `json:"flows"`
	Sockets    json.RawMessage `json:"sockets"`
	ExecEvents json.RawMessage `json:"exec_events"`
	CreatedAt  time.Time       `json:"created_at"`
}

type VisibilityHistoryResponse struct {
	AgentUID string               `json:"agent_uid"`
	Frames   []VisibilityResponse `json:"frames"`
	Total    int                  `json:"total"`
	Limit    int                  `json:"limit"`
	Offset   int                  `json:"offset"`
}

type EnforcementDecisionResponse struct {
	ID           uuid.UUID       `json:"id"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	AgentUID     string          `json:"agent_uid"`
	Verdict      string          `json:"verdict"`
	Reason       string          `json:"reason"`
	RuleID       string          `json:"rule_id"`
	FlowData     json.RawMessage `json:"flow_data"`
	ProcessData  json.RawMessage `json:"process_data"`
	Timestamp    time.Time       `json:"timestamp"`
	Mode         string          `json:"mode"`
}

type EnforcementDecisionsResponse struct {
	AgentUID  string                        `json:"agent_uid"`
	Decisions []EnforcementDecisionResponse `json:"decisions"`
	Total     int                           `json:"total"`
	Limit     int                           `json:"limit"`
}

type NetworkFlowResponse struct {
	ID             uuid.UUID  `json:"id"`
	AgentUID       string     `json:"agent_uid"`
	SrcIP          string     `json:"src_ip"`
	DstIP          string     `json:"dst_ip"`
	SrcPort        int        `json:"src_port"`
	DstPort        int        `json:"dst_port"`
	Protocol       string     `json:"protocol"`
	BytesSent      int64      `json:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received"`
	PacketsSent    int64      `json:"packets_sent"`
	PacketsReceived int64     `json:"packets_received"`
	StartTime      *time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time"`
	Status         string     `json:"status"`
	ProcessID      int        `json:"process_id"`
	ProcessName    string     `json:"process_name"`
	CreatedAt      time.Time  `json:"created_at"`
}

type NetworkFlowsResponse struct {
	AgentUID string                `json:"agent_uid"`
	Flows    []NetworkFlowResponse `json:"flows"`
	Total    int                   `json:"total"`
	Limit    int                   `json:"limit"`
}

type ProcessResponse struct {
	ID               uuid.UUID       `json:"id"`
	AgentUID         string          `json:"agent_uid"`
	PID              int             `json:"pid"`
	PPID             int             `json:"ppid"`
	Name             string          `json:"name"`
	Cmdline          string          `json:"cmdline"`
	ExecutablePath   string          `json:"executable_path"`
	WorkingDirectory string          `json:"working_directory"`
	UserID           int             `json:"user_id"`
	GroupID          int             `json:"group_id"`
	StartTime        *time.Time      `json:"start_time"`
	EndTime          *time.Time      `json:"end_time"`
	Status           string          `json:"status"`
	MemoryUsage      int64           `json:"memory_usage"`
	CPUUsage         float64         `json:"cpu_usage"`
	NetworkConnections json.RawMessage `json:"network_connections"`
	CreatedAt        time.Time       `json:"created_at"`
}

type ProcessesResponse struct {
	AgentUID  string            `json:"agent_uid"`
	Processes []ProcessResponse `json:"processes"`
	Total     int               `json:"total"`
	Limit     int               `json:"limit"`
}

type VisibilitySummaryResponse struct {
	AgentUID          string    `json:"agent_uid"`
	LatestFrameTS     time.Time `json:"latest_frame_ts"`
	TotalProcesses    int64     `json:"total_processes"`
	TotalFlows        int64     `json:"total_flows"`
	TotalSockets      int64     `json:"total_sockets"`
	TotalExecEvents   int64     `json:"total_exec_events"`
	ActiveProcesses   int64     `json:"active_processes"`
}





