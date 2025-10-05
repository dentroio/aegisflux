package gateway

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"golang.org/x/crypto/chacha20poly1305"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/auth"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

// WebSocketGateway handles WebSocket connections from agents
type WebSocketGateway struct {
	upgrader          websocket.Upgrader
	agentConnections  map[string]*types.AgentConnection
	connectionManager *types.ConnectionManager
	authService       *auth.AuthService
	messageRouter     *MessageRouter
	config            *types.Configuration
	metrics           *types.ConnectionMetrics
	httpServer        *http.Server
	httpHandlers      *HTTPHandlers
	nc                *nats.Conn // NATS connection for receiving messages from Actions API
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewWebSocketGateway creates a new WebSocket gateway instance
func NewWebSocketGateway(config *types.Configuration) (*WebSocketGateway, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize upgrader with configuration
	upgrader := websocket.Upgrader{
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// TODO: Implement proper origin checking for production
			return true
		},
	}

	// Initialize connection manager
	connectionManager := &types.ConnectionManager{
		Connections: make(map[string]*types.AgentConnection),
	}

	// Initialize metrics
	metrics := &types.ConnectionMetrics{
		LastReset: time.Now(),
	}

	// Initialize auth service
	authService, err := auth.NewAuthService(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create auth service: %w", err)
	}

	// Initialize message router
	messageRouter, err := NewMessageRouter(config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create message router: %w", err)
	}

	// Connect to NATS for receiving messages from Actions API
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to NATS: %v", err)
	} else {
		log.Printf("WebSocket Gateway connected to NATS at %s", natsURL)
	}

	gateway := &WebSocketGateway{
		upgrader:          upgrader,
		agentConnections:  make(map[string]*types.AgentConnection),
		connectionManager: connectionManager,
		authService:       authService,
		messageRouter:     messageRouter,
		config:            config,
		metrics:           metrics,
		nc:                nc, // NATS connection
		ctx:               ctx,
		cancel:            cancel,
	}

	// Initialize HTTP handlers
	gateway.httpHandlers = NewHTTPHandlers(gateway)
	gateway.httpHandlers.RegisterHTTPRoutes()

	// Register default message handlers
	gateway.registerDefaultHandlers()

	return gateway, nil
}

// Start starts the WebSocket gateway server
func (wsg *WebSocketGateway) Start() error {
	// Start NATS subscription for messages from Actions API
	if wsg.nc != nil {
		go wsg.startNATSSubscription()
	}

	// Create HTTP server
	wsg.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", wsg.config.Port),
		Handler: nil, // Use default mux with registered handlers
	}

	log.Printf("WebSocket Gateway starting on port %d", wsg.config.Port)
	return wsg.httpServer.ListenAndServe()
}

// Stop gracefully stops the WebSocket gateway
func (wsg *WebSocketGateway) Stop() error {
	log.Println("Stopping WebSocket Gateway...")
	
	// Cancel context to stop all goroutines
	wsg.cancel()

	// Close all active connections
	wsg.mu.Lock()
	for agentID, conn := range wsg.agentConnections {
		log.Printf("Closing connection for agent: %s", agentID)
		if conn.Connection != nil {
			conn.Connection.Close()
		}
	}
	wsg.mu.Unlock()

	// Shutdown HTTP server
	if wsg.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := wsg.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down HTTP server: %v", err)
		}
	}

	log.Println("WebSocket Gateway stopped")
	return nil
}

// handleWebSocketUpgrade handles HTTP to WebSocket upgrade requests
func (wsg *WebSocketGateway) handleWebSocketUpgrade(w http.ResponseWriter, r *http.Request) {
	// Extract agent information from headers
	agentID := r.Header.Get("X-Agent-ID")
	publicKeyStr := r.Header.Get("X-Agent-Public-Key")
	userAgent := r.Header.Get("User-Agent")

	// Validate required headers
	if agentID == "" || publicKeyStr == "" {
		log.Printf("Missing required headers: agentID=%s, publicKey=%s", agentID, publicKeyStr)
		http.Error(w, "Missing required headers: X-Agent-ID and X-Agent-Public-Key", http.StatusBadRequest)
		return
	}

	// Validate user agent
	if userAgent != "Aegis-Agent/1.0" {
		log.Printf("Invalid user agent: %s", userAgent)
		http.Error(w, "Invalid user agent", http.StatusBadRequest)
		return
	}

	// Decode public key
	publicKey, err := base64.StdEncoding.DecodeString(publicKeyStr)
	if err != nil {
		log.Printf("Failed to decode public key for agent %s: %v", agentID, err)
		http.Error(w, "Invalid public key format", http.StatusBadRequest)
		return
	}

	// Check if we have too many connections
	wsg.mu.RLock()
	connectionCount := len(wsg.agentConnections)
	wsg.mu.RUnlock()

	if connectionCount >= wsg.config.MaxConnections {
		log.Printf("Maximum connections reached (%d), rejecting agent %s", wsg.config.MaxConnections, agentID)
		http.Error(w, "Maximum connections reached", http.StatusServiceUnavailable)
		return
	}

	// Upgrade to WebSocket
	conn, err := wsg.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed for agent %s: %v", agentID, err)
		wsg.metrics.ConnectionErrors++
		return
	}

	// Create agent connection
	agentConn := &types.AgentConnection{
		AgentID:         agentID,
		Connection:      &WebSocketConnWrapper{conn: conn},
		PublicKey:       ed25519.PublicKey(publicKey),
		ConnectedAt:     time.Now(),
		LastSeen:        time.Now(),
		IsAuthenticated: false,
		Channels:        []string{},
		Metadata:        make(map[string]interface{}),
	}

	// Start connection handler
	go wsg.handleAgentConnection(agentConn)

	wsg.metrics.TotalConnections++
	log.Printf("WebSocket connection established for agent: %s", agentID)
}

// handleAgentConnection handles communication with a specific agent
func (wsg *WebSocketGateway) handleAgentConnection(conn *types.AgentConnection) {
	defer func() {
		// Cleanup connection
		wsg.mu.Lock()
		delete(wsg.agentConnections, conn.AgentID)
		wsg.mu.Unlock()
		
		if conn.Connection != nil {
			conn.Connection.Close()
		}
		
		wsg.metrics.ActiveConnections--
		log.Printf("Connection closed for agent: %s", conn.AgentID)
	}()

	// Add connection to manager
	wsg.mu.Lock()
	wsg.agentConnections[conn.AgentID] = conn
	wsg.metrics.ActiveConnections++
	wsg.mu.Unlock()

	// Set initial connection timeout (longer for first message)
	conn.Connection.SetReadDeadline(time.Now().Add(5 * time.Minute))
	log.Printf("Starting message reading loop for agent %s", conn.AgentID)

	for {
		select {
		case <-wsg.ctx.Done():
			log.Printf("Shutting down connection for agent: %s", conn.AgentID)
			return
		default:
			// Read message from agent
			log.Printf("Waiting for message from agent %s...", conn.AgentID)
			messageType, data, err := conn.Connection.ReadMessage()
			if err != nil {
				log.Printf("Error reading message from agent %s: %v (error type: %T)", conn.AgentID, err, err)
				wsg.metrics.ConnectionErrors++
				return
			}
			log.Printf("Successfully read message from agent %s: type=%d, size=%d", conn.AgentID, messageType, len(data))

		// Update last seen
		conn.Mu.Lock()
		conn.LastSeen = time.Now()
		conn.Mu.Unlock()

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			log.Printf("Received text message from agent %s (size: %d)", conn.AgentID, len(data))
			if err := wsg.handleTextMessage(conn, data); err != nil {
				log.Printf("Error handling text message from agent %s: %v", conn.AgentID, err)
				wsg.sendErrorResponse(conn, "message_processing_error", err.Error(), 4002, 1)
			} else {
				log.Printf("Successfully processed text message from agent %s", conn.AgentID)
			}
			case websocket.BinaryMessage:
				log.Printf("Received binary message from agent %s (size: %d)", conn.AgentID, len(data))
				// TODO: Handle binary messages if needed
			case websocket.PingMessage:
				conn.Connection.WriteMessage(websocket.PongMessage, data)
			case websocket.CloseMessage:
				log.Printf("Received close message from agent: %s", conn.AgentID)
				return
			}

			// Reset read deadline
			conn.Connection.SetReadDeadline(time.Now().Add(wsg.config.ConnectionTimeout))
		}
	}
}

// handleTextMessage processes text messages from agents
func (wsg *WebSocketGateway) handleTextMessage(conn *types.AgentConnection, data []byte) error {
	log.Printf("Processing text message from agent %s", conn.AgentID)
	
	// Parse message
	var message types.SecureMessage
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Failed to parse message from agent %s: %v", conn.AgentID, err)
		return fmt.Errorf("failed to parse message: %w", err)
	}

	log.Printf("Parsed message from agent %s: type=%s, channel=%s", conn.AgentID, message.Type, message.Channel)

	// Check if this is an authentication message
	if message.Type == types.MessageTypeRequest && message.Channel == "auth" {
		log.Printf("Processing authentication message from agent %s", conn.AgentID)
		return wsg.handleAuthentication(conn, message)
	}

	// Check if this is a registration complete message
	if message.Type == types.MessageTypeRequest && message.Channel == "registration.complete" {
		log.Printf("Processing registration complete message from agent %s", conn.AgentID)
		return wsg.handleAgentRegistrationComplete(conn.AgentID, message)
	}

	// Check if agent is authenticated
	if !conn.IsAuthenticated {
		log.Printf("Agent %s not authenticated, rejecting message", conn.AgentID)
		return fmt.Errorf("agent not authenticated")
	}

	// Route message through message router
	log.Printf("Routing message from agent %s through message router", conn.AgentID)
	return wsg.messageRouter.RouteMessage(conn.AgentID, message)
}

// handleAuthentication handles agent authentication
func (wsg *WebSocketGateway) handleAuthentication(conn *types.AgentConnection, message types.SecureMessage) error {
	// Decrypt and parse authentication request
	authReq, err := wsg.authService.DecryptAuthenticationRequest(message)
	if err != nil {
		wsg.metrics.AuthenticationFailures++
		return fmt.Errorf("failed to decrypt authentication request: %w", err)
	}

	// Authenticate agent
	authResp, err := wsg.authService.AuthenticateAgent(authReq, conn.PublicKey)
	if err != nil {
		wsg.metrics.AuthenticationFailures++
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Mark connection as authenticated
	conn.Mu.Lock()
	conn.IsAuthenticated = true
	conn.SessionToken = authResp.SessionToken
	conn.Mu.Unlock()

	// Send authentication response
	response := types.SecureMessage{
		ID:        fmt.Sprintf("auth_resp_%d", time.Now().UnixNano()),
		Type:      types.MessageTypeResponse,
		Channel:   "auth",
		Timestamp: time.Now().Unix(),
		Headers:   make(map[string]string),
	}

	// Encrypt response
	encryptedResp, err := wsg.authService.EncryptAuthenticationResponse(authResp)
	if err != nil {
		return fmt.Errorf("failed to encrypt authentication response: %w", err)
	}

	response.Payload = encryptedResp.Payload
	response.Nonce = encryptedResp.Nonce
	response.Signature = encryptedResp.Signature

	// Send response
	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal authentication response: %w", err)
	}

	if err := conn.Connection.WriteMessage(websocket.TextMessage, responseData); err != nil {
		return fmt.Errorf("failed to send authentication response: %w", err)
	}

	wsg.metrics.MessagesSent++
	log.Printf("Agent %s authenticated successfully", conn.AgentID)
	log.Printf("Authentication response sent to agent %s, connection should remain open", conn.AgentID)
	return nil
}

// sendErrorResponse sends an error response to an agent
func (wsg *WebSocketGateway) sendErrorResponse(conn *types.AgentConnection, errorType, message string, code, retryAfter int) {
	response := types.SecureMessage{
		ID:        fmt.Sprintf("error_%d", time.Now().UnixNano()),
		Type:      types.MessageTypeResponse,
		Channel:   "error",
		Timestamp: time.Now().Unix(),
		Payload:   fmt.Sprintf(`{"error":"%s","message":"%s","code":%d,"retry_after":%d}`, errorType, message, code, retryAfter),
		Headers:   make(map[string]string),
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}

	if err := conn.Connection.WriteMessage(websocket.TextMessage, responseData); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}

// handleHealthCheck handles health check requests
func (wsg *WebSocketGateway) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	metrics := wsg.GetMetrics()
	
	health := map[string]interface{}{
		"status":           "healthy",
		"timestamp":        time.Now().Unix(),
		"total_connections": metrics.TotalConnections,
		"active_connections": metrics.ActiveConnections,
		"messages_received": metrics.MessagesReceived,
		"messages_sent":    metrics.MessagesSent,
		"uptime":          time.Since(metrics.LastReset).Seconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(health); err != nil {
		log.Printf("Failed to encode health response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// startHealthMonitoring starts the health monitoring goroutine
func (wsg *WebSocketGateway) startHealthMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wsg.ctx.Done():
			return
		case <-ticker.C:
			wsg.performHealthCheck()
		}
	}
}

// performHealthCheck performs health checks on all connections
func (wsg *WebSocketGateway) performHealthCheck() {
	wsg.mu.RLock()
	connections := make([]*types.AgentConnection, 0, len(wsg.agentConnections))
	for _, conn := range wsg.agentConnections {
		connections = append(connections, conn)
	}
	wsg.mu.RUnlock()

	for _, conn := range connections {
		conn.Mu.RLock()
		lastSeen := conn.LastSeen
		conn.Mu.RUnlock()

		// Check if connection is stale
		if time.Since(lastSeen) > wsg.config.ConnectionTimeout {
			log.Printf("Connection timeout for agent: %s", conn.AgentID)
			conn.Connection.Close()
			continue
		}

		// Send ping to check connection health
		if err := conn.Connection.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
			log.Printf("Health check failed for agent %s: %v", conn.AgentID, err)
			conn.Connection.Close()
		}
	}
}

// startMessageProcessing starts the message processing goroutine
func (wsg *WebSocketGateway) startMessageProcessing() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-wsg.ctx.Done():
			return
		case <-ticker.C:
			// Process queued messages
			// TODO: Implement message queue processing
		}
	}
}

// GetMetrics returns connection metrics
func (wsg *WebSocketGateway) GetMetrics() *types.ConnectionMetrics {
	wsg.mu.RLock()
	defer wsg.mu.RUnlock()
	
	// Create a copy of metrics
	metrics := *wsg.metrics
	metrics.ActiveConnections = int64(len(wsg.agentConnections))
	
	return &metrics
}

// registerDefaultHandlers registers default message handlers with the gateway
func (wsg *WebSocketGateway) registerDefaultHandlers() {
	// Register heartbeat handler
	wsg.messageRouter.RegisterHandler("agent.*.heartbeat", wsg.handleAgentHeartbeat)
	
	// Register registration handlers
	wsg.messageRouter.RegisterHandler("agent.registration", wsg.handleAgentRegistration)
	wsg.messageRouter.RegisterHandler("agent.registration.complete", wsg.handleAgentRegistrationComplete)
	
	// Register other message handlers
	wsg.messageRouter.RegisterHandler("agent.*.policies", wsg.handleAgentPolicies)
	wsg.messageRouter.RegisterHandler("agent.*.anomalies", wsg.handleAgentAnomalies)
	wsg.messageRouter.RegisterHandler("agent.*.threats", wsg.handleAgentThreats)
	wsg.messageRouter.RegisterHandler("agent.*.processes", wsg.handleAgentProcesses)
	wsg.messageRouter.RegisterHandler("agent.*.status", wsg.handleAgentStatus)
	wsg.messageRouter.RegisterHandler("agent.*.logs", wsg.handleAgentLogs)
	
	log.Println("Default message handlers registered")
}

// Default message handlers for the gateway

func (wsg *WebSocketGateway) handleAgentHeartbeat(agentID string, message types.SecureMessage) error {
	log.Printf("Received heartbeat from agent: %s", agentID)
	
	// Update agent last seen timestamp
	wsg.mu.Lock()
	if conn, exists := wsg.agentConnections[agentID]; exists {
		conn.Mu.Lock()
		conn.LastSeen = time.Now()
		conn.Mu.Unlock()
	}
	wsg.mu.Unlock()
	
	// Update Actions API with heartbeat (async)
	go wsg.updateActionsAPIHeartbeat(agentID)
	
	return nil
}

// updateActionsAPIHeartbeat updates the Actions API with the agent's heartbeat
func (wsg *WebSocketGateway) updateActionsAPIHeartbeat(agentID string) {
	// Create heartbeat update request
	heartbeatData := map[string]interface{}{
		"agent_uid": agentID,
		"last_seen": time.Now().Format(time.RFC3339),
		"status":    "active",
	}
	
	jsonData, err := json.Marshal(heartbeatData)
	if err != nil {
		log.Printf("Failed to marshal heartbeat data for agent %s: %v", agentID, err)
		return
	}
	
	// Send to Actions API
	actionsAPIURL := "http://actions-api:8083/agents/heartbeat"
	resp, err := http.Post(actionsAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send heartbeat to Actions API for agent %s: %v", agentID, err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Actions API heartbeat update failed for agent %s: status %d", agentID, resp.StatusCode)
		return
	}
	
	log.Printf("Successfully updated Actions API heartbeat for agent %s", agentID)
}

func (wsg *WebSocketGateway) handleAgentPolicies(agentID string, message types.SecureMessage) error {
	log.Printf("Received policy message from agent: %s", agentID)
	// TODO: Process policy updates
	return nil
}

func (wsg *WebSocketGateway) handleAgentAnomalies(agentID string, message types.SecureMessage) error {
	log.Printf("Received anomaly message from agent: %s", agentID)
	// TODO: Process anomaly alerts
	return nil
}

func (wsg *WebSocketGateway) handleAgentThreats(agentID string, message types.SecureMessage) error {
	log.Printf("Received threat message from agent: %s", agentID)
	// TODO: Process threat intelligence
	return nil
}

func (wsg *WebSocketGateway) handleAgentProcesses(agentID string, message types.SecureMessage) error {
	log.Printf("Received process message from agent: %s", agentID)
	// TODO: Process process events
	return nil
}

func (wsg *WebSocketGateway) handleAgentStatus(agentID string, message types.SecureMessage) error {
	log.Printf("Received status message from agent: %s", agentID)
	// TODO: Update agent status
	return nil
}

func (wsg *WebSocketGateway) handleAgentLogs(agentID string, message types.SecureMessage) error {
	log.Printf("Received log message from agent: %s", agentID)
	// TODO: Process log messages
	return nil
}

// handleAgentRegistration handles agent registration requests
func (wsg *WebSocketGateway) handleAgentRegistration(agentID string, message types.SecureMessage) error {
	log.Printf("Received registration request from agent: %s", agentID)
	
	// Decode the registration payload (it's base64-encoded JSON)
	log.Printf("Registration payload from agent %s: %s", agentID, message.Payload)
	
	// Decode base64 payload
	decodedPayload, err := base64.StdEncoding.DecodeString(message.Payload)
	if err != nil {
		log.Printf("Failed to decode base64 payload from agent %s: %v", agentID, err)
		return fmt.Errorf("invalid base64 payload: %w", err)
	}
	
	log.Printf("Decoded payload from agent %s: %s", agentID, string(decodedPayload))
	
	var registrationData map[string]interface{}
	if err := json.Unmarshal(decodedPayload, &registrationData); err != nil {
		log.Printf("Failed to decode JSON payload from agent %s: %v", agentID, err)
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	
	// Extract agent information from the registration data
	orgID, _ := registrationData["org_id"].(string)
	hostID, _ := registrationData["host_id"].(string)
	agentVersion, _ := registrationData["agent_version"].(string)
	
	// Set default values if not provided
	if orgID == "" {
		orgID = "default-org"
	}
	if hostID == "" {
		hostID = agentID
	}
	if agentVersion == "" {
		agentVersion = "1.0.0"
	}
	
	// Extract agent public key from registration data
	agentPubKey, _ := registrationData["agent_pubkey"].(string)
	if agentPubKey == "" {
		log.Printf("Missing agent_pubkey in registration data from agent %s", agentID)
		return fmt.Errorf("missing agent_pubkey in registration data")
	}
	
	// Create registration request for Actions API
	registrationRequest := map[string]interface{}{
		"org_id":         orgID,
		"host_id":        hostID,
		"agent_pubkey":   agentPubKey,
		"machine_id_hash": registrationData["machine_id_hash"],
		"agent_version":  agentVersion,
		"capabilities":   registrationData["capabilities"],
		"platform":       registrationData["platform"],
		"network":        registrationData["network"],
	}
	
	// Call Actions API to register the agent
	actionsAPIURL := "http://host.docker.internal:8083/agents/register/init"
	jsonData, err := json.Marshal(registrationRequest)
	if err != nil {
		log.Printf("Failed to marshal registration request for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}
	
	// Make HTTP request to Actions API
	log.Printf("Making HTTP request to Actions API: %s", actionsAPIURL)
	log.Printf("Request payload: %s", string(jsonData))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(actionsAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to call Actions API for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to call Actions API: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("Actions API response status: %d", resp.StatusCode)
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Actions API response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to read Actions API response: %w", err)
	}
	
	log.Printf("Actions API response body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Actions API returned error for agent %s: %d %s", agentID, resp.StatusCode, string(body))
		return fmt.Errorf("Actions API error: %d %s", resp.StatusCode, string(body))
	}
	
	// Parse the response
	var registrationResponse map[string]interface{}
	if err := json.Unmarshal(body, &registrationResponse); err != nil {
		log.Printf("Failed to parse Actions API response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to parse Actions API response: %w", err)
	}
	
	log.Printf("Successfully registered agent %s with Actions API", agentID)
	
	// Extract agent_uid from the registration response
	agentUID, _ := registrationResponse["agent_uid"].(string)
	if agentUID != "" && agentUID != agentID {
		// Update connection key from hostname to UID
		log.Printf("Updating connection key from hostname %s to UID %s", agentID, agentUID)
		wsg.mu.Lock()
		if conn, exists := wsg.agentConnections[agentID]; exists {
			// Remove old key (hostname)
			delete(wsg.agentConnections, agentID)
			// Update connection's AgentID to UID
			conn.AgentID = agentUID
			// Store with new key (UID)
			wsg.agentConnections[agentUID] = conn
			log.Printf("Successfully updated connection key from %s to %s", agentID, agentUID)
		}
		wsg.mu.Unlock()
		// Update agentID for subsequent operations
		agentID = agentUID
	}
	
	// Send success response back to agent
	response := types.SecureMessage{
		ID:        fmt.Sprintf("reg_resp_%d", time.Now().UnixNano()),
		Type:      types.MessageTypeResponse,
		Channel:   "agent.registration",
		Timestamp: time.Now().Unix(),
		Payload:   string(body),
		Headers:   make(map[string]string),
	}
	
	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal registration response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to marshal registration response: %w", err)
	}
	
	// Send response to agent
	if conn, exists := wsg.agentConnections[agentID]; exists {
		if err := conn.Connection.WriteMessage(websocket.TextMessage, responseData); err != nil {
			log.Printf("Failed to send registration response to agent %s: %v", agentID, err)
			return fmt.Errorf("failed to send registration response: %w", err)
		}
		log.Printf("Sent registration response to agent %s", agentID)
	}
	
	return nil
}

// handleAgentRegistrationComplete handles agent registration completion requests
func (wsg *WebSocketGateway) handleAgentRegistrationComplete(agentID string, message types.SecureMessage) error {
	log.Printf("Received registration complete request from agent: %s", agentID)
	
	// Decode the registration complete payload (it's base64-encoded JSON)
	log.Printf("Registration complete payload from agent %s: %s", agentID, message.Payload)
	
	// Decode base64 payload
	decodedPayload, err := base64.StdEncoding.DecodeString(message.Payload)
	if err != nil {
		log.Printf("Failed to decode base64 payload from agent %s: %v", agentID, err)
		return fmt.Errorf("invalid base64 payload: %w", err)
	}
	
	log.Printf("Decoded registration complete payload from agent %s: %s", agentID, string(decodedPayload))
	
	var completionData map[string]interface{}
	if err := json.Unmarshal(decodedPayload, &completionData); err != nil {
		log.Printf("Failed to decode JSON payload from agent %s: %v", agentID, err)
		return fmt.Errorf("invalid JSON payload: %w", err)
	}
	
	// Extract completion information
	registrationID, _ := completionData["registration_id"].(string)
	hostID, _ := completionData["host_id"].(string)
	signature, _ := completionData["signature"].(string)
	
	if registrationID == "" || hostID == "" || signature == "" {
		log.Printf("Missing required fields in registration complete request from agent %s", agentID)
		return fmt.Errorf("missing required fields: registration_id, host_id, or signature")
	}
	
	// Create registration complete request for Actions API
	completionRequest := map[string]interface{}{
		"registration_id": registrationID,
		"host_id":        hostID,
		"signature":      signature,
	}
	
	// Call Actions API to complete the agent registration
	actionsAPIURL := "http://host.docker.internal:8083/agents/register/complete"
	jsonData, err := json.Marshal(completionRequest)
	if err != nil {
		log.Printf("Failed to marshal registration complete request for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to marshal registration complete request: %w", err)
	}
	
	// Make HTTP request to Actions API
	log.Printf("Making HTTP request to Actions API: %s", actionsAPIURL)
	log.Printf("Request payload: %s", string(jsonData))
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(actionsAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to call Actions API for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to call Actions API: %w", err)
	}
	defer resp.Body.Close()
	
	log.Printf("Actions API response status: %d", resp.StatusCode)
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Actions API response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to read Actions API response: %w", err)
	}
	
	log.Printf("Actions API response body: %s", string(body))
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Actions API returned error for agent %s: %d %s", agentID, resp.StatusCode, string(body))
		return fmt.Errorf("Actions API error: %d %s", resp.StatusCode, string(body))
	}
	
	// Parse the response
	var completionResponse map[string]interface{}
	if err := json.Unmarshal(body, &completionResponse); err != nil {
		log.Printf("Failed to parse Actions API response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to parse Actions API response: %w", err)
	}
	
	log.Printf("Successfully completed agent registration for agent %s", agentID)
	
	// Extract agent_uid from the registration complete response
	agentUID, _ := completionResponse["agent_uid"].(string)
	if agentUID != "" && agentUID != agentID {
		// Update connection key from hostname to UID
		log.Printf("Updating connection key from hostname %s to UID %s", agentID, agentUID)
		wsg.mu.Lock()
		if conn, exists := wsg.agentConnections[agentID]; exists {
			// Remove old key (hostname)
			delete(wsg.agentConnections, agentID)
			// Update connection's AgentID to UID
			conn.AgentID = agentUID
			// Store with new key (UID)
			wsg.agentConnections[agentUID] = conn
			log.Printf("Successfully updated connection key from %s to %s", agentID, agentUID)
		}
		wsg.mu.Unlock()
		// Update agentID for subsequent operations
		agentID = agentUID
	}
	
	// Send success response back to agent
	response := types.SecureMessage{
		ID:        fmt.Sprintf("reg_complete_resp_%d", time.Now().UnixNano()),
		Type:      types.MessageTypeResponse,
		Channel:   "agent.registration.complete",
		Timestamp: time.Now().Unix(),
		Payload:   string(body),
		Headers:   make(map[string]string),
	}
	
	responseData, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal registration complete response for agent %s: %v", agentID, err)
		return fmt.Errorf("failed to marshal registration complete response: %w", err)
	}
	
	// Send response to agent
	if conn, exists := wsg.agentConnections[agentID]; exists {
		if err := conn.Connection.WriteMessage(websocket.TextMessage, responseData); err != nil {
			log.Printf("Failed to send registration complete response to agent %s: %v", agentID, err)
			return fmt.Errorf("failed to send registration complete response: %w", err)
		}
		log.Printf("Sent registration complete response to agent %s", agentID)
	}
	
	return nil
}


// startNATSSubscription starts listening for messages from Actions API via NATS
func (wsg *WebSocketGateway) startNATSSubscription() {
	if wsg.nc == nil {
		log.Printf("NATS not connected, skipping subscription")
		return
	}

	subject := "websocket.messages"
	log.Printf("Subscribing to NATS subject: %s", subject)

	_, err := wsg.nc.Subscribe(subject, func(m *nats.Msg) {
		wsg.handleNATSMessage(m)
	})

	if err != nil {
		log.Printf("Failed to subscribe to NATS subject %s: %v", subject, err)
		return
	}

	log.Printf("Successfully subscribed to NATS subject: %s", subject)
	
	// Keep the subscription alive
	select {
	case <-wsg.ctx.Done():
		log.Printf("NATS subscription stopped due to context cancellation")
	}
}

// handleNATSMessage handles messages received from NATS (from Actions API)
func (wsg *WebSocketGateway) handleNATSMessage(m *nats.Msg) {
	log.Printf("Received message from NATS: %s", string(m.Data))

	// Parse the message
	var messageData map[string]interface{}
	if err := json.Unmarshal(m.Data, &messageData); err != nil {
		log.Printf("Failed to parse NATS message: %v", err)
		return
	}

	// Extract target agent and channel
	targetAgent, _ := messageData["target_agent"].(string)
	channel, _ := messageData["channel"].(string)
	messageType, _ := messageData["type"].(string)

	if targetAgent == "" {
		log.Printf("No target agent specified in NATS message")
		return
	}

	log.Printf("Routing NATS message to agent %s on channel %s", targetAgent, channel)

	// Send message to the target agent
	if err := wsg.sendMessageToAgent(targetAgent, channel, messageData, messageType); err != nil {
		log.Printf("Failed to send message to agent %s: %v", targetAgent, err)
	}
}

// sendMessageToAgent sends a message to a specific agent via WebSocket
func (wsg *WebSocketGateway) sendMessageToAgent(agentID, channel string, message map[string]interface{}, messageType string) error {
	wsg.mu.Lock()
	conn, exists := wsg.agentConnections[agentID]
	wsg.mu.Unlock()

	if !exists {
		log.Printf("Agent %s not connected", agentID)
		return fmt.Errorf("agent %s not connected", agentID)
	}

	// Generate nonce that matches agent expectations and ChaCha20-Poly1305 requirements
	// Agent expects: base64.StdEncoding.EncodeToString([]byte("message_nonce"))
	// But ChaCha20-Poly1305 requires exactly 12 bytes, so we'll use first 12 bytes of "message_nonce"
	nonceBytes := []byte("message_nonc") // 12 bytes exactly
	nonceStr := base64.StdEncoding.EncodeToString(nonceBytes)

	// Derive shared key using both public keys (Option 3)
	// Agent will use: SHA256(agent_public_key + backend_public_key)
	// Backend uses: SHA256(agent_public_key + backend_public_key)
	// Both sides derive the same shared secret using public keys only
	backendPublicKey := wsg.authService.GetBackendPublicKey()
	
	// Use agent's public key first, then backend's public key (matches agent's order)
	combined := append(conn.PublicKey, backendPublicKey...)
	sharedKeyHash := sha256.Sum256(combined)
	sharedKey := sharedKeyHash[:]

	// Create ChaCha20-Poly1305 cipher
	cipher, err := chacha20poly1305.New(sharedKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Encrypt the payload
	payloadBytes := []byte(message["payload"].(string))
	encryptedPayload := cipher.Seal(nil, nonceBytes, payloadBytes, nil)
	encryptedPayloadStr := base64.StdEncoding.EncodeToString(encryptedPayload)

	// Create the message to send with proper SecureMessage format
	messageToSend := map[string]interface{}{
		"id":        message["id"],
		"type":      messageType,
		"channel":   channel,
		"payload":   encryptedPayloadStr, // Encrypted payload
		"timestamp": message["timestamp"],
		"nonce":     nonceStr, // Agent-expected nonce format
		"signature": "",       // Empty signature for now - agent team needs to handle this
		"headers":   message["headers"],
	}

	messageData, err := json.Marshal(messageToSend)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send the message
	if err := conn.Connection.WriteMessage(websocket.TextMessage, messageData); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("Successfully sent message to agent %s on channel %s", agentID, channel)
	return nil
}
