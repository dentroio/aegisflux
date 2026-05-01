package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// HTTPHandlers handles HTTP endpoints for the WebSocket Gateway
type HTTPHandlers struct {
	wsg *WebSocketGateway
}

// NewHTTPHandlers creates a new HTTP handlers instance
func NewHTTPHandlers(wsg *WebSocketGateway) *HTTPHandlers {
	return &HTTPHandlers{wsg: wsg}
}

// RegisterHTTPRoutes registers HTTP routes with the gateway
func (h *HTTPHandlers) RegisterHTTPRoutes() {
	log.Println("Registering HTTP routes...")

	// Health check endpoint
	http.HandleFunc("/health", h.handleHealth)
	log.Println("Registered /health endpoint")

	// Agent registration endpoints
	http.HandleFunc("/agents/register/init", h.handleRegisterInit)
	log.Println("Registered /agents/register/init endpoint")

	http.HandleFunc("/agents/register/complete", h.handleRegisterComplete)
	log.Println("Registered /agents/register/complete endpoint")

	// WebSocket endpoint
	http.HandleFunc("/ws/agent", h.handleWebSocketUpgrade)
	log.Println("Registered /ws/agent endpoint")

	log.Println("HTTP routes registered successfully")
}

// handleHealth handles health check requests
func (h *HTTPHandlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current metrics
	h.wsg.mu.RLock()
	activeConnections := len(h.wsg.agentConnections)
	h.wsg.mu.RUnlock()

	response := map[string]interface{}{
		"status":             "healthy",
		"active_connections": activeConnections,
		"timestamp":          time.Now().Unix(),
		"uptime":             time.Since(h.wsg.metrics.LastReset).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRegisterInit handles agent registration initialization
func (h *HTTPHandlers) handleRegisterInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse registration request
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received registration init request: %+v", req)

	// Extract required fields
	orgID, _ := req["org_id"].(string)
	hostID, _ := req["host_id"].(string)
	agentPubKey, _ := req["agent_pubkey"].(string)
	agentVersion, _ := req["agent_version"].(string)

	// Set defaults
	if orgID == "" {
		orgID = "default-org"
	}
	if hostID == "" {
		hostID = "unknown-host"
	}
	if agentVersion == "" {
		agentVersion = "1.0.0"
	}

	// Create registration request for Actions API
	registrationRequest := map[string]interface{}{
		"org_id":          orgID,
		"host_id":         hostID,
		"agent_pubkey":    agentPubKey,
		"machine_id_hash": req["machine_id_hash"],
		"agent_version":   agentVersion,
		"capabilities":    req["capabilities"],
		"platform":        map[string]interface{}{"os": "linux", "arch": "arm64"},
		"network":         map[string]interface{}{"interface": "eth0"},
	}

	// Call Actions API to register the agent
	actionsAPIURL := h.wsg.actionsAPIEndpoint("/agents/register/init")
	jsonData, err := json.Marshal(registrationRequest)
	if err != nil {
		log.Printf("Failed to marshal registration request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Making HTTP request to Actions API: %s", actionsAPIURL)
	log.Printf("Request payload: %s", string(jsonData))

	// Make HTTP request to Actions API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(actionsAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to call Actions API: %v", err)
		http.Error(w, "Failed to register with backend", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Printf("Actions API response status: %d", resp.StatusCode)

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Actions API response: %v", err)
		http.Error(w, "Failed to read backend response", http.StatusInternalServerError)
		return
	}

	log.Printf("Actions API response body: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		log.Printf("Actions API returned error: %d %s", resp.StatusCode, string(body))
		http.Error(w, fmt.Sprintf("Backend registration failed: %d", resp.StatusCode), http.StatusBadRequest)
		return
	}

	// Parse the response
	var registrationResponse map[string]interface{}
	if err := json.Unmarshal(body, &registrationResponse); err != nil {
		log.Printf("Failed to parse Actions API response: %v", err)
		http.Error(w, "Invalid backend response", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully registered agent with Actions API")

	// Return the response to the agent
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registrationResponse)
}

// handleRegisterComplete handles agent registration completion
func (h *HTTPHandlers) handleRegisterComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse completion request
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Received registration complete request: %+v", req)

	// Forward to Actions API
	actionsAPIURL := h.wsg.actionsAPIEndpoint("/agents/register/complete")
	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Failed to marshal completion request: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Make HTTP request to Actions API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(actionsAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to call Actions API: %v", err)
		http.Error(w, "Failed to complete registration with backend", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Actions API response: %v", err)
		http.Error(w, "Failed to read backend response", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Actions API returned error: %d %s", resp.StatusCode, string(body))
		http.Error(w, fmt.Sprintf("Backend completion failed: %d", resp.StatusCode), http.StatusBadRequest)
		return
	}

	// Return the response to the agent
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// handleWebSocketUpgrade handles WebSocket upgrade requests
func (h *HTTPHandlers) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// Delegate to the existing WebSocket handler
	h.wsg.handleWebSocketUpgrade(w, r)
}
