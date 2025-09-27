package gateway

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

// WebSocketConnWrapper wraps gorilla websocket connection to implement our interface
type WebSocketConnWrapper struct {
	conn *websocket.Conn
	mu   sync.RWMutex
}

// ReadMessage reads a message from the WebSocket connection
func (w *WebSocketConnWrapper) ReadMessage() (messageType int, p []byte, err error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.conn.ReadMessage()
}

// WriteMessage writes a message to the WebSocket connection
func (w *WebSocketConnWrapper) WriteMessage(messageType int, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(messageType, data)
}

// Close closes the WebSocket connection
func (w *WebSocketConnWrapper) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.Close()
}

// SetReadDeadline sets the read deadline for the WebSocket connection
func (w *WebSocketConnWrapper) SetReadDeadline(t time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline for the WebSocket connection
func (w *WebSocketConnWrapper) SetWriteDeadline(t time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.SetWriteDeadline(t)
}

// ConnectionManager manages active agent connections
type ConnectionManager struct {
	Connections map[string]*types.AgentConnection
	Mu          sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		Connections: make(map[string]*types.AgentConnection),
	}
}

// RegisterConnection registers a new agent connection
func (cm *ConnectionManager) RegisterConnection(agentID string, conn *types.AgentConnection) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()

	// Close existing connection if any
	if existing, exists := cm.Connections[agentID]; exists {
		if existing.Connection != nil {
			existing.Connection.Close()
		}
	}

	cm.Connections[agentID] = conn
}

// UnregisterConnection removes an agent connection
func (cm *ConnectionManager) UnregisterConnection(agentID string) {
	cm.Mu.Lock()
	defer cm.Mu.Unlock()

	if conn, exists := cm.Connections[agentID]; exists {
		if conn.Connection != nil {
			conn.Connection.Close()
		}
		delete(cm.Connections, agentID)
	}
}

// GetConnection returns a connection by agent ID
func (cm *ConnectionManager) GetConnection(agentID string) (*types.AgentConnection, bool) {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()

	conn, exists := cm.Connections[agentID]
	return conn, exists
}

// GetAllConnections returns all active connections
func (cm *ConnectionManager) GetAllConnections() map[string]*types.AgentConnection {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()

	// Return a copy to avoid race conditions
	connections := make(map[string]*types.AgentConnection)
	for agentID, conn := range cm.Connections {
		connections[agentID] = conn
	}

	return connections
}

// SendToAgent sends a message to a specific agent
func (cm *ConnectionManager) SendToAgent(agentID string, message []byte) error {
	conn, exists := cm.GetConnection(agentID)
	if !exists {
		return fmt.Errorf("agent %s not connected", agentID)
	}

	if conn.Connection == nil {
		return fmt.Errorf("agent %s connection is nil", agentID)
	}

	return conn.Connection.WriteMessage(websocket.TextMessage, message)
}

// BroadcastToAllAgents sends a message to all connected agents
func (cm *ConnectionManager) BroadcastToAllAgents(message []byte) error {
	connections := cm.GetAllConnections()
	
	var errors []error
	for agentID, conn := range connections {
		if conn.Connection != nil {
			if err := conn.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				errors = append(errors, fmt.Errorf("failed to send to agent %s: %w", agentID, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed: %v", errors)
	}

	return nil
}

// GetConnectionCount returns the number of active connections
func (cm *ConnectionManager) GetConnectionCount() int {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()
	return len(cm.Connections)
}

// GetConnectionStats returns statistics about connections
func (cm *ConnectionManager) GetConnectionStats() map[string]interface{} {
	cm.Mu.RLock()
	defer cm.Mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections": len(cm.Connections),
		"authenticated":     0,
		"unauthenticated":   0,
	}

	for _, conn := range cm.Connections {
		if conn.IsAuthenticated {
			stats["authenticated"] = stats["authenticated"].(int) + 1
		} else {
			stats["unauthenticated"] = stats["unauthenticated"].(int) + 1
		}
	}

	return stats
}
