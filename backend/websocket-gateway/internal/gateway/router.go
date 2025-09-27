package gateway

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/sgerhart/aegisflux/websocket-gateway/internal/types"
)

// MessageRouter handles routing of messages between agents and backend services
type MessageRouter struct {
	handlers        map[string]types.MessageHandler
	channels        map[string]*types.Channel
	config          *types.Configuration
	mu              sync.RWMutex
}

// NewMessageRouter creates a new message router
func NewMessageRouter(config *types.Configuration) (*MessageRouter, error) {
	router := &MessageRouter{
		handlers: make(map[string]types.MessageHandler),
		channels: make(map[string]*types.Channel),
		config:   config,
	}

	// Register default handlers
	router.registerDefaultHandlers()

	return router, nil
}

// RouteMessage routes a message to the appropriate handler
func (mr *MessageRouter) RouteMessage(agentID string, message types.SecureMessage) error {
	// TODO: Implement proper message decryption
	// For now, assume message is already decrypted

	// Determine handler based on channel
	handler, exists := mr.getHandlerForChannel(message.Channel)
	if !exists {
		log.Printf("No handler found for channel: %s", message.Channel)
		return fmt.Errorf("no handler for channel: %s", message.Channel)
	}

	// Execute handler
	return handler(agentID, message)
}

// RegisterHandler registers a message handler for a specific channel
func (mr *MessageRouter) RegisterHandler(channel string, handler types.MessageHandler) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.handlers[channel] = handler
	log.Printf("Registered handler for channel: %s", channel)
}

// UnregisterHandler removes a message handler
func (mr *MessageRouter) UnregisterHandler(channel string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	delete(mr.handlers, channel)
	log.Printf("Unregistered handler for channel: %s", channel)
}

// CreateChannel creates a new communication channel
func (mr *MessageRouter) CreateChannel(name string) *types.Channel {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	channel := &types.Channel{
		Name:        name,
		Subscribers: make(map[string]*types.AgentConnection),
		MessageQueue: []types.QueuedMessage{},
	}

	mr.channels[name] = channel
	log.Printf("Created channel: %s", name)

	return channel
}

// SubscribeToChannel subscribes an agent to a channel
func (mr *MessageRouter) SubscribeToChannel(agentID string, channelName string, conn *types.AgentConnection) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	channel, exists := mr.channels[channelName]
	if !exists {
		// Create channel if it doesn't exist
		channel = mr.CreateChannel(channelName)
	}

	channel.Mu.Lock()
	channel.Subscribers[agentID] = conn
	channel.Mu.Unlock()

	log.Printf("Agent %s subscribed to channel: %s", agentID, channelName)
	return nil
}

// UnsubscribeFromChannel unsubscribes an agent from a channel
func (mr *MessageRouter) UnsubscribeFromChannel(agentID string, channelName string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	channel, exists := mr.channels[channelName]
	if !exists {
		return fmt.Errorf("channel %s does not exist", channelName)
	}

	channel.Mu.Lock()
	delete(channel.Subscribers, agentID)
	channel.Mu.Unlock()

	log.Printf("Agent %s unsubscribed from channel: %s", agentID, channelName)
	return nil
}

// BroadcastToChannel broadcasts a message to all subscribers of a channel
func (mr *MessageRouter) BroadcastToChannel(channelName string, message types.SecureMessage) error {
	mr.mu.RLock()
	channel, exists := mr.channels[channelName]
	mr.mu.RUnlock()

	if !exists {
		return fmt.Errorf("channel %s not found", channelName)
	}

	channel.Mu.RLock()
	subscribers := make([]*types.AgentConnection, 0, len(channel.Subscribers))
	for _, conn := range channel.Subscribers {
		subscribers = append(subscribers, conn)
	}
	channel.Mu.RUnlock()

	// Send message to all subscribers
	var errors []error
	for _, conn := range subscribers {
		// TODO: Implement proper message encryption
		// For now, send raw message
		if err := mr.sendToAgent(conn, message); err != nil {
			errors = append(errors, fmt.Errorf("failed to send to agent %s: %w", conn.AgentID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("broadcast failed: %v", errors)
	}

	return nil
}

// getHandlerForChannel returns the appropriate handler for a channel
func (mr *MessageRouter) getHandlerForChannel(channel string) (types.MessageHandler, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	// Direct channel match
	if handler, exists := mr.handlers[channel]; exists {
		return handler, true
	}

	// Pattern matching for agent-specific channels
	// e.g., "agent.001.policies" -> "agent.*.policies"
	parts := strings.Split(channel, ".")
	if len(parts) >= 3 {
		pattern := parts[0] + ".*." + parts[2]
		if handler, exists := mr.handlers[pattern]; exists {
			return handler, true
		}
	}

	return nil, false
}

// sendToAgent sends a message to a specific agent
func (mr *MessageRouter) sendToAgent(conn *types.AgentConnection, message types.SecureMessage) error {
	// TODO: Implement proper message encryption and sending
	// For now, just log the message
	log.Printf("Sending message to agent %s on channel %s: %s", 
		conn.AgentID, message.Channel, message.Type)
	
	// TODO: Actually send the message via WebSocket
	return nil
}

// registerDefaultHandlers registers default message handlers
func (mr *MessageRouter) registerDefaultHandlers() {
	// Note: Handlers are now registered by the gateway
	// This method is kept for future use
	log.Println("Message router initialized - handlers will be registered by gateway")
}

// GetChannelStats returns statistics about channels
func (mr *MessageRouter) GetChannelStats() map[string]interface{} {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	stats := map[string]interface{}{
		"total_channels": len(mr.channels),
		"total_handlers": len(mr.handlers),
		"channels":       make(map[string]interface{}),
	}

	for name, channel := range mr.channels {
		channel.Mu.RLock()
		channelStats := map[string]interface{}{
			"subscribers":    len(channel.Subscribers),
			"queued_messages": len(channel.MessageQueue),
		}
		channel.Mu.RUnlock()

		stats["channels"].(map[string]interface{})[name] = channelStats
	}

	return stats
}
