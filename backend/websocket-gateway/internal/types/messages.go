package types

import (
	"crypto/ed25519"
	"sync"
	"time"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeRequest   MessageType = "request"
	MessageTypeResponse  MessageType = "response"
	MessageTypeEvent     MessageType = "event"
	MessageTypeHeartbeat MessageType = "heartbeat"
	MessageTypeAck       MessageType = "ack"
)

// SecureMessage represents an encrypted and signed message
type SecureMessage struct {
	ID        string                 `json:"id"`
	Type      MessageType           `json:"type"`
	Channel   string                `json:"channel"`
	Payload   string                `json:"payload"`   // base64 encoded encrypted payload
	Timestamp int64                 `json:"timestamp"`
	Nonce     string                `json:"nonce"`     // base64 encoded nonce
	Signature string                `json:"signature"` // base64 encoded Ed25519 signature
	Headers   map[string]string     `json:"headers"`
}

// AuthenticationRequest represents the agent authentication request
type AuthenticationRequest struct {
	AgentID   string `json:"agent_id"`
	PublicKey string `json:"public_key"`   // base64 encoded Ed25519 public key
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`        // base64 encoded 16-byte nonce
	Signature string `json:"signature"`    // base64 encoded Ed25519 signature
}

// AuthenticationResponse represents the backend authentication response
type AuthenticationResponse struct {
	Success      bool   `json:"success"`
	BackendKey   string `json:"backend_key"`   // base64 encoded backend public key
	SessionToken string `json:"session_token"` // JWT session token
	ExpiresAt    int64  `json:"expires_at"`    // Unix timestamp
	Message      string `json:"message,omitempty"`
}

// AgentConnection represents an active agent WebSocket connection
type AgentConnection struct {
	AgentID          string
	Connection       WebSocketConn
	PublicKey        ed25519.PublicKey
	SessionToken     string
	LastSeen         time.Time
	ConnectedAt      time.Time
	IsAuthenticated  bool
	Channels         []string
	Metadata         map[string]interface{}
	Mu               sync.RWMutex
}

// WebSocketConn represents a WebSocket connection interface
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// ConnectionManager handles active agent connections
type ConnectionManager struct {
	Connections map[string]*AgentConnection
	Mu          sync.RWMutex
}

// MessageHandler represents a function that handles incoming messages
type MessageHandler func(agentID string, message SecureMessage) error

// ChannelManager manages communication channels
type ChannelManager struct {
	channels map[string]*Channel
	mu       sync.RWMutex
}

// Channel represents a communication channel
type Channel struct {
	Name        string
	Subscribers map[string]*AgentConnection
	MessageQueue []QueuedMessage
	Mu          sync.RWMutex
}

// QueuedMessage represents a message in the queue
type QueuedMessage struct {
	Message     SecureMessage
	AgentID     string
	Priority    int
	CreatedAt   time.Time
	RetryCount  int
	MaxRetries  int
}

// HealthCheck represents a health check message
type HealthCheck struct {
	AgentID     string            `json:"agent_id"`
	Status      string            `json:"status"`
	Uptime      int64             `json:"uptime"`
	LastSeen    int64             `json:"last_seen"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	Code       int    `json:"code"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// ConnectionMetrics represents connection statistics
type ConnectionMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	MessagesReceived    int64
	MessagesSent        int64
	AuthenticationFailures int64
	ConnectionErrors    int64
	LastReset           time.Time
}

// Configuration represents the WebSocket gateway configuration
type Configuration struct {
	Port                int           `json:"port"`
	ReadBufferSize      int           `json:"read_buffer_size"`
	WriteBufferSize     int           `json:"write_buffer_size"`
	MaxConnections      int           `json:"max_connections"`
	HeartbeatInterval   time.Duration `json:"heartbeat_interval"`
	ConnectionTimeout   time.Duration `json:"connection_timeout"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	PrivateKeyPath      string        `json:"private_key_path"`
	PublicKeyPath       string        `json:"public_key_path"`
	DatabaseURL         string        `json:"database_url"`
	LogLevel            string        `json:"log_level"`
}
