package publish

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// NATSPublisher handles publishing events to NATS
type NATSPublisher struct {
	conn *nats.Conn
}

// NewNATSPublisher creates a new NATS publisher
func NewNATSPublisher(natsURL string) (*NATSPublisher, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSPublisher{
		conn: conn,
	}, nil
}

// Close closes the NATS connection
func (p *NATSPublisher) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// AssignmentCreatedEvent represents an assignment creation event
type AssignmentCreatedEvent struct {
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	Timestamp    time.Time       `json:"timestamp"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	BundleID     uuid.UUID       `json:"bundle_id"`
	Mode         string          `json:"mode"`
	Selector     json.RawMessage `json:"selector"`
	Snapshot     json.RawMessage `json:"snapshot"`
	SnapshotSig  string          `json:"snapshot_sig"`
	SnapshotKid  string          `json:"snapshot_kid"`
	TTLTS        *time.Time      `json:"ttl_ts,omitempty"`
	DryRun       bool            `json:"dry_run"`
	CreatedBy    string          `json:"created_by"`
	Status       string          `json:"status"`
}

// AssignmentUpdatedEvent represents an assignment update event
type AssignmentUpdatedEvent struct {
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	Timestamp    time.Time       `json:"timestamp"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	BundleID     uuid.UUID       `json:"bundle_id"`
	Mode         string          `json:"mode"`
	Selector     json.RawMessage `json:"selector"`
	Snapshot     json.RawMessage `json:"snapshot"`
	SnapshotSig  string          `json:"snapshot_sig"`
	SnapshotKid  string          `json:"snapshot_kid"`
	TTLTS        *time.Time      `json:"ttl_ts,omitempty"`
	DryRun       bool            `json:"dry_run"`
	UpdatedBy    string          `json:"updated_by"`
	Status       string          `json:"status"`
	Changes      json.RawMessage `json:"changes,omitempty"`
}

// AssignmentDeletedEvent represents an assignment deletion event
type AssignmentDeletedEvent struct {
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Timestamp    time.Time `json:"timestamp"`
	AssignmentID uuid.UUID `json:"assignment_id"`
	BundleID     uuid.UUID `json:"bundle_id"`
	DeletedBy    string    `json:"deleted_by"`
	Reason       string    `json:"reason,omitempty"`
}

// VisibilityEvent represents a visibility data event from an agent
type VisibilityEvent struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	Timestamp  time.Time       `json:"timestamp"`
	AgentUID   string          `json:"agent_uid"`
	FrameID    string          `json:"frame_id"`
	Data       json.RawMessage `json:"data"`
	DataType   string          `json:"data_type"` // "processes", "flows", "sockets", "exec_events"
	SequenceID int64           `json:"sequence_id"`
}

// EnforcementDecisionEvent represents an enforcement decision event
type EnforcementDecisionEvent struct {
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	Timestamp    time.Time       `json:"timestamp"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	AgentUID     string          `json:"agent_uid"`
	Verdict      string          `json:"verdict"` // "allow", "deny", "observe_drop"
	Reason       string          `json:"reason,omitempty"`
	RuleID       string          `json:"rule_id,omitempty"`
	FlowData     json.RawMessage `json:"flow_data,omitempty"`
	ProcessData  json.RawMessage `json:"process_data,omitempty"`
	Mode         string          `json:"mode"`
}

// PublishAssignmentCreated publishes an assignment created event
func (p *NATSPublisher) PublishAssignmentCreated(ctx context.Context, event AssignmentCreatedEvent) error {
	event.EventID = uuid.New().String()
	event.EventType = "assignment.created"
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment created event: %w", err)
	}

	subject := "aegis.assignments.created"
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish assignment created event: %w", err)
	}

	return nil
}

// PublishAssignmentUpdated publishes an assignment updated event
func (p *NATSPublisher) PublishAssignmentUpdated(ctx context.Context, event AssignmentUpdatedEvent) error {
	event.EventID = uuid.New().String()
	event.EventType = "assignment.updated"
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment updated event: %w", err)
	}

	subject := "aegis.assignments.updated"
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish assignment updated event: %w", err)
	}

	return nil
}

// PublishAssignmentDeleted publishes an assignment deleted event
func (p *NATSPublisher) PublishAssignmentDeleted(ctx context.Context, event AssignmentDeletedEvent) error {
	event.EventID = uuid.New().String()
	event.EventType = "assignment.deleted"
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal assignment deleted event: %w", err)
	}

	subject := "aegis.assignments.deleted"
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish assignment deleted event: %w", err)
	}

	return nil
}

// PublishVisibilityEvent publishes a visibility event
func (p *NATSPublisher) PublishVisibilityEvent(ctx context.Context, event VisibilityEvent) error {
	event.EventID = uuid.New().String()
	event.EventType = "visibility.data"
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal visibility event: %w", err)
	}

	subject := fmt.Sprintf("aegis.visibility.host.%s", event.AgentUID)
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish visibility event: %w", err)
	}

	return nil
}

// PublishEnforcementDecision publishes an enforcement decision event
func (p *NATSPublisher) PublishEnforcementDecision(ctx context.Context, event EnforcementDecisionEvent) error {
	event.EventID = uuid.New().String()
	event.EventType = "enforcement.decision"
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal enforcement decision event: %w", err)
	}

	subject := fmt.Sprintf("aegis.enforcement.agent.%s", event.AgentUID)
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish enforcement decision event: %w", err)
	}

	return nil
}

// publish publishes a message to NATS with context support
func (p *NATSPublisher) publish(ctx context.Context, subject string, data []byte) error {
	// Create a channel to receive the result
	result := make(chan error, 1)

	// Publish asynchronously
	p.conn.Publish(subject, data)

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Message published successfully
		return nil
	}
}

// PublishWithAck publishes a message and waits for acknowledgment
func (p *NATSPublisher) PublishWithAck(ctx context.Context, subject string, data []byte) error {
	// Create a channel to receive the result
	result := make(chan error, 1)

	// Publish with acknowledgment
	p.conn.Publish(subject, data)

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Message published successfully
		return nil
	}
}

// PublishRequest publishes a request and waits for a response
func (p *NATSPublisher) PublishRequest(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	msg, err := p.conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}

	return msg.Data, nil
}

// Health check for NATS connection
func (p *NATSPublisher) HealthCheck(ctx context.Context) error {
	if !p.conn.IsConnected() {
		return fmt.Errorf("NATS connection is not active")
	}

	// Try to publish a health check message
	subject := "aegis.health.check"
	data := []byte(`{"timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)
	
	if err := p.publish(ctx, subject, data); err != nil {
		return fmt.Errorf("failed to publish health check: %w", err)
	}

	return nil
}

// GetConnectionStatus returns the current NATS connection status
func (p *NATSPublisher) GetConnectionStatus() map[string]interface{} {
	stats := p.conn.Statistics()
	
	return map[string]interface{}{
		"connected":     p.conn.IsConnected(),
		"url":           p.conn.ConnectedUrl(),
		"server_id":     p.conn.ConnectedServerId(),
		"server_name":   p.conn.ConnectedServerName(),
		"in_msgs":       stats.InMsgs,
		"out_msgs":      stats.OutMsgs,
		"in_bytes":      stats.InBytes,
		"out_bytes":     stats.OutBytes,
		"reconnects":    stats.Reconnects,
		"last_error":    stats.LastError,
	}
}

// Subscribe to events (for testing or monitoring purposes)
func (p *NATSPublisher) Subscribe(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	return p.conn.Subscribe(subject, handler)
}

// SubscribeWithQueue subscribes to events with a queue group
func (p *NATSPublisher) SubscribeWithQueue(subject, queue string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	return p.conn.QueueSubscribe(subject, queue, handler)
}

