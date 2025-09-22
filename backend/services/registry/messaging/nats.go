package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// NATSClient handles NATS messaging operations
type NATSClient struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATSClient creates a new NATS client
func NewNATSClient(url string) (*NATSClient, error) {
	// Connect to NATS server
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &NATSClient{
		conn: conn,
		js:   js,
	}

	// Initialize streams and subjects
	if err := client.initializeStreams(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return client, nil
}

// initializeStreams creates required NATS streams
func (nc *NATSClient) initializeStreams() error {
	// Define streams
	streams := []struct {
		name     string
		subjects []string
	}{
		{
			name:     "BUNDLES",
			subjects: []string{"bundles.>", "assignments.>"},
		},
		{
			name:     "AUDIT",
			subjects: []string{"audit.>"},
		},
	}

	for _, stream := range streams {
		// Check if stream already exists
		_, err := nc.js.StreamInfo(stream.name)
		if err == nil {
			continue // Stream already exists
		}

		// Create stream
		_, err = nc.js.AddStream(&nats.StreamConfig{
			Name:     stream.name,
			Subjects: stream.subjects,
			Storage:  nats.FileStorage,
			MaxAge:   24 * time.Hour, // Keep messages for 24 hours
		})
		if err != nil {
			return fmt.Errorf("failed to create stream %s: %w", stream.name, err)
		}

		log.Printf("Created NATS stream: %s", stream.name)
	}

	return nil
}

// BundlePublishedMessage represents a bundle published event
type BundlePublishedMessage struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	BundleID   uuid.UUID       `json:"bundle_id"`
	BundleName string          `json:"bundle_name"`
	Hash       string          `json:"hash"`
	Signature  string          `json:"signature"`
	KeyID      string          `json:"kid"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedBy  string          `json:"created_by"`
	Timestamp  time.Time       `json:"timestamp"`
}

// AssignmentCreatedMessage represents an assignment created event
type AssignmentCreatedMessage struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	AssignmentID  uuid.UUID       `json:"assignment_id"`
	BundleID      uuid.UUID       `json:"bundle_id"`
	BundleName    string          `json:"bundle_name"`
	HostSelector  json.RawMessage `json:"host_selector"`
	TTLSeconds    *int            `json:"ttl_seconds,omitempty"`
	DryRun        bool            `json:"dry_run"`
	CreatedBy     string          `json:"created_by"`
	Timestamp     time.Time       `json:"timestamp"`
}

// KeyRotatedMessage represents a key rotation event
type KeyRotatedMessage struct {
	EventID    string    `json:"event_id"`
	EventType  string    `json:"event_type"`
	OldKeyID   string    `json:"old_kid"`
	NewKeyID   string    `json:"new_kid"`
	RotatedBy  string    `json:"rotated_by"`
	Timestamp  time.Time `json:"timestamp"`
}

// AuditMessage represents an audit event
type AuditMessage struct {
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	Actor     string          `json:"actor"`
	Action    string          `json:"action"`
	Target    string          `json:"target,omitempty"`
	Details   json.RawMessage `json:"details"`
	Timestamp time.Time       `json:"timestamp"`
}

// PublishBundlePublished publishes a bundle published event
func (nc *NATSClient) PublishBundlePublished(ctx context.Context, bundleID uuid.UUID, bundleName, hash, signature, keyID string, metadata json.RawMessage, createdBy string) error {
	message := BundlePublishedMessage{
		EventID:    uuid.New().String(),
		EventType:  "bundle.published",
		BundleID:   bundleID,
		BundleName: bundleName,
		Hash:       hash,
		Signature:  signature,
		KeyID:      keyID,
		Metadata:   metadata,
		CreatedBy:  createdBy,
		Timestamp:  time.Now().UTC(),
	}

	return nc.publishMessage(ctx, "bundles.published", message)
}

// PublishAssignmentCreated publishes an assignment created event
func (nc *NATSClient) PublishAssignmentCreated(ctx context.Context, assignmentID, bundleID uuid.UUID, bundleName string, hostSelector json.RawMessage, ttlSeconds *int, dryRun bool, createdBy string) error {
	message := AssignmentCreatedMessage{
		EventID:      uuid.New().String(),
		EventType:    "assignment.created",
		AssignmentID: assignmentID,
		BundleID:     bundleID,
		BundleName:   bundleName,
		HostSelector: hostSelector,
		TTLSeconds:   ttlSeconds,
		DryRun:       dryRun,
		CreatedBy:    createdBy,
		Timestamp:    time.Now().UTC(),
	}

	return nc.publishMessage(ctx, "assignments.created", message)
}

// PublishKeyRotated publishes a key rotation event
func (nc *NATSClient) PublishKeyRotated(ctx context.Context, oldKeyID, newKeyID, rotatedBy string) error {
	message := KeyRotatedMessage{
		EventID:   uuid.New().String(),
		EventType: "key.rotated",
		OldKeyID:  oldKeyID,
		NewKeyID:  newKeyID,
		RotatedBy: rotatedBy,
		Timestamp: time.Now().UTC(),
	}

	return nc.publishMessage(ctx, "keys.rotated", message)
}

// PublishAuditEvent publishes an audit event
func (nc *NATSClient) PublishAuditEvent(ctx context.Context, actor, action, target string, details json.RawMessage) error {
	message := AuditMessage{
		EventID:   uuid.New().String(),
		EventType: "audit.event",
		Actor:     actor,
		Action:    action,
		Target:    target,
		Details:   details,
		Timestamp: time.Now().UTC(),
	}

	return nc.publishMessage(ctx, "audit.events", message)
}

// publishMessage publishes a message to NATS
func (nc *NATSClient) publishMessage(ctx context.Context, subject string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Publish with context timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		_, err = nc.js.PublishAsync(subject, data)
		if err != nil {
			return fmt.Errorf("failed to publish message to %s: %w", subject, err)
		}
	}

	log.Printf("Published message to %s: %s", subject, string(data))
	return nil
}

// SubscribeToBundleEvents subscribes to bundle-related events
func (nc *NATSClient) SubscribeToBundleEvents(subject string, handler func([]byte) error) (nats.Subscription, error) {
	return nc.js.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			log.Printf("Error handling bundle event: %v", err)
			// Don't acknowledge the message so it can be retried
			return
		}
		msg.Ack()
	})
}

// SubscribeToAuditEvents subscribes to audit events
func (nc *NATSClient) SubscribeToAuditEvents(handler func([]byte) error) (nats.Subscription, error) {
	return nc.js.Subscribe("audit.events", func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			log.Printf("Error handling audit event: %v", err)
			// Don't acknowledge the message so it can be retried
			return
		}
		msg.Ack()
	})
}

// Close closes the NATS connection
func (nc *NATSClient) Close() error {
	if nc.conn != nil {
		return nc.conn.Close()
	}
	return nil
}

// IsConnected checks if the NATS client is connected
func (nc *NATSClient) IsConnected() bool {
	return nc.conn != nil && nc.conn.IsConnected()
}

// GetConnectionStatus returns the connection status
func (nc *NATSClient) GetConnectionStatus() string {
	if nc.conn == nil {
		return "disconnected"
	}
	
	if nc.conn.IsConnected() {
		return "connected"
	}
	
	return "disconnected"
}

// GetStreamInfo returns information about NATS streams
func (nc *NATSClient) GetStreamInfo(streamName string) (*nats.StreamInfo, error) {
	return nc.js.StreamInfo(streamName)
}

// ListStreams returns a list of all streams
func (nc *NATSClient) ListStreams() ([]string, error) {
	streams := nc.js.StreamNames()
	return streams, nil
}

// CreateConsumer creates a consumer for a stream
func (nc *NATSClient) CreateConsumer(streamName string, consumerConfig *nats.ConsumerConfig) (*nats.ConsumerInfo, error) {
	return nc.js.AddConsumer(streamName, consumerConfig)
}

// GetConsumerInfo returns information about a consumer
func (nc *NATSClient) GetConsumerInfo(streamName, consumerName string) (*nats.ConsumerInfo, error) {
	return nc.js.ConsumerInfo(streamName, consumerName)
}

// DeleteConsumer deletes a consumer
func (nc *NATSClient) DeleteConsumer(streamName, consumerName string) error {
	return nc.js.DeleteConsumer(streamName, consumerName)
}

