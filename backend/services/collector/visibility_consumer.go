package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/sgerhart/aegisflux/backend/internal/db"
)

// VisibilityConsumer handles consuming visibility data from NATS
type VisibilityConsumer struct {
	conn      *nats.Conn
	store     *db.Store
	ctx       context.Context
	cancel    context.CancelFunc
	subscriptions []*nats.Subscription
}

// NewVisibilityConsumer creates a new visibility consumer
func NewVisibilityConsumer(natsURL string, store *db.Store) (*VisibilityConsumer, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &VisibilityConsumer{
		conn:   conn,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start starts the visibility consumer
func (c *VisibilityConsumer) Start() error {
	// Subscribe to visibility events
	sub, err := c.conn.QueueSubscribe("aegis.visibility.host.*", "visibility-consumer", c.handleVisibilityEvent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to visibility events: %w", err)
	}
	c.subscriptions = append(c.subscriptions, sub)

	// Subscribe to enforcement decision events
	sub, err = c.conn.QueueSubscribe("aegis.enforcement.agent.*", "enforcement-consumer", c.handleEnforcementDecision)
	if err != nil {
		return fmt.Errorf("failed to subscribe to enforcement events: %w", err)
	}
	c.subscriptions = append(c.subscriptions, sub)

	// Subscribe to assignment events for tracking
	sub, err = c.conn.QueueSubscribe("aegis.assignments.*", "assignment-consumer", c.handleAssignmentEvent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to assignment events: %w", err)
	}
	c.subscriptions = append(c.subscriptions, sub)

	log.Println("Visibility consumer started successfully")
	return nil
}

// Stop stops the visibility consumer
func (c *VisibilityConsumer) Stop() error {
	c.cancel()

	// Unsubscribe from all subscriptions
	for _, sub := range c.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("Warning: Failed to unsubscribe: %v", err)
		}
	}

	// Close NATS connection
	if c.conn != nil {
		c.conn.Close()
	}

	log.Println("Visibility consumer stopped")
	return nil
}

// handleVisibilityEvent handles visibility data events
func (c *VisibilityConsumer) handleVisibilityEvent(msg *nats.Msg) {
	var event VisibilityEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling visibility event: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if err := c.processVisibilityEvent(ctx, event); err != nil {
		log.Printf("Error processing visibility event: %v", err)
		// Don't acknowledge the message so it can be retried
		return
	}

	// Acknowledge the message
	msg.Ack()
}

// handleEnforcementDecision handles enforcement decision events
func (c *VisibilityConsumer) handleEnforcementDecision(msg *nats.Msg) {
	var event EnforcementDecisionEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling enforcement decision event: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if err := c.processEnforcementDecision(ctx, event); err != nil {
		log.Printf("Error processing enforcement decision: %v", err)
		return
	}

	msg.Ack()
}

// handleAssignmentEvent handles assignment events for tracking
func (c *VisibilityConsumer) handleAssignmentEvent(msg *nats.Msg) {
	var event map[string]interface{}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Error unmarshaling assignment event: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	if err := c.processAssignmentEvent(ctx, event); err != nil {
		log.Printf("Error processing assignment event: %v", err)
		return
	}

	msg.Ack()
}

// VisibilityEvent represents a visibility data event
type VisibilityEvent struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	Timestamp  time.Time       `json:"timestamp"`
	AgentUID   string          `json:"agent_uid"`
	FrameID    string          `json:"frame_id"`
	Data       json.RawMessage `json:"data"`
	DataType   string          `json:"data_type"`
	SequenceID int64           `json:"sequence_id"`
}

// EnforcementDecisionEvent represents an enforcement decision event
type EnforcementDecisionEvent struct {
	EventID      string          `json:"event_id"`
	EventType    string          `json:"event_type"`
	Timestamp    time.Time       `json:"timestamp"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	AgentUID     string          `json:"agent_uid"`
	Verdict      string          `json:"verdict"`
	Reason       string          `json:"reason,omitempty"`
	RuleID       string          `json:"rule_id,omitempty"`
	FlowData     json.RawMessage `json:"flow_data,omitempty"`
	ProcessData  json.RawMessage `json:"process_data,omitempty"`
	Mode         string          `json:"mode"`
}

// processVisibilityEvent processes a visibility event and stores it in the database
func (c *VisibilityConsumer) processVisibilityEvent(ctx context.Context, event VisibilityEvent) error {
	// Parse the data based on data type
	switch event.DataType {
	case "processes":
		return c.processProcessData(ctx, event)
	case "flows":
		return c.processFlowData(ctx, event)
	case "sockets":
		return c.processSocketData(ctx, event)
	case "exec_events":
		return c.processExecEventData(ctx, event)
	case "frame":
		return c.processFrameData(ctx, event)
	default:
		return fmt.Errorf("unknown data type: %s", event.DataType)
	}
}

// processProcessData processes process information
func (c *VisibilityConsumer) processProcessData(ctx context.Context, event VisibilityEvent) error {
	var processes []ProcessInfo
	if err := json.Unmarshal(event.Data, &processes); err != nil {
		return fmt.Errorf("failed to unmarshal process data: %w", err)
	}

	for _, proc := range processes {
		// Convert to database model and store
		dbProcess := &db.Process{
			ID:               uuid.New(),
			AgentUID:         event.AgentUID,
			PID:              proc.PID,
			PPID:             proc.PPID,
			Name:             proc.Name,
			Cmdline:          proc.Cmdline,
			ExecutablePath:   proc.ExecutablePath,
			WorkingDirectory: proc.WorkingDirectory,
			UserID:           proc.UserID,
			GroupID:          proc.GroupID,
			StartTime:        proc.StartTime,
			EndTime:          proc.EndTime,
			Status:           proc.Status,
			MemoryUsage:      proc.MemoryUsage,
			CPUUsage:         proc.CPUUsage,
			NetworkConnections: proc.NetworkConnections,
			CreatedAt:        time.Now(),
		}

		if err := c.store.CreateProcess(ctx, dbProcess); err != nil {
			log.Printf("Warning: Failed to create process record: %v", err)
		}
	}

	return nil
}

// processFlowData processes network flow data
func (c *VisibilityConsumer) processFlowData(ctx context.Context, event VisibilityEvent) error {
	var flows []NetworkFlowInfo
	if err := json.Unmarshal(event.Data, &flows); err != nil {
		return fmt.Errorf("failed to unmarshal flow data: %w", err)
	}

	for _, flow := range flows {
		// Convert to database model and store
		dbFlow := &db.NetworkFlow{
			ID:             uuid.New(),
			AgentUID:       event.AgentUID,
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
			CreatedAt:      time.Now(),
		}

		if err := c.store.CreateNetworkFlow(ctx, dbFlow); err != nil {
			log.Printf("Warning: Failed to create network flow record: %v", err)
		}
	}

	return nil
}

// processSocketData processes socket information
func (c *VisibilityConsumer) processSocketData(ctx context.Context, event VisibilityEvent) error {
	var sockets []SocketInfo
	if err := json.Unmarshal(event.Data, &sockets); err != nil {
		return fmt.Errorf("failed to unmarshal socket data: %w", err)
	}

	for _, socket := range sockets {
		// Convert to database model and store
		dbSocket := &db.Socket{
			ID:            uuid.New(),
			AgentUID:      event.AgentUID,
			FD:            socket.FD,
			PID:           socket.PID,
			Family:        socket.Family,
			Type:          socket.Type,
			Protocol:      socket.Protocol,
			LocalAddress:  socket.LocalAddress,
			LocalPort:     socket.LocalPort,
			RemoteAddress: socket.RemoteAddress,
			RemotePort:    socket.RemotePort,
			State:         socket.State,
			CreatedAt:     time.Now(),
		}

		if err := c.store.CreateSocket(ctx, dbSocket); err != nil {
			log.Printf("Warning: Failed to create socket record: %v", err)
		}
	}

	return nil
}

// processExecEventData processes execution event data
func (c *VisibilityConsumer) processExecEventData(ctx context.Context, event VisibilityEvent) error {
	var execEvents []ExecEventInfo
	if err := json.Unmarshal(event.Data, &execEvents); err != nil {
		return fmt.Errorf("failed to unmarshal exec event data: %w", err)
	}

	for _, execEvent := range execEvents {
		// Convert to database model and store
		dbExecEvent := &db.ExecEvent{
			ID:              uuid.New(),
			AgentUID:        event.AgentUID,
			PID:             execEvent.PID,
			PPID:            execEvent.PPID,
			ExecutablePath:  execEvent.ExecutablePath,
			Cmdline:         execEvent.Cmdline,
			WorkingDirectory: execEvent.WorkingDirectory,
			UserID:          execEvent.UserID,
			GroupID:         execEvent.GroupID,
			ExitCode:        execEvent.ExitCode,
			Timestamp:       execEvent.Timestamp,
			DurationMs:      execEvent.DurationMs,
		}

		if err := c.store.CreateExecEvent(ctx, dbExecEvent); err != nil {
			log.Printf("Warning: Failed to create exec event record: %v", err)
		}
	}

	return nil
}

// processFrameData processes complete visibility frame data
func (c *VisibilityConsumer) processFrameData(ctx context.Context, event VisibilityEvent) error {
	var frame VisibilityFrame
	if err := json.Unmarshal(event.Data, &frame); err != nil {
		return fmt.Errorf("failed to unmarshal frame data: %w", err)
	}

	// Store the complete frame
	dbFrame := &db.VisibilityFrame{
		ID:         uuid.New(),
		AgentUID:   event.AgentUID,
		Timestamp:  event.Timestamp,
		Processes:  frame.Processes,
		Flows:      frame.Flows,
		Sockets:    frame.Sockets,
		ExecEvents: frame.ExecEvents,
		CreatedAt:  time.Now(),
	}

	if err := c.store.CreateVisibilityFrame(ctx, dbFrame); err != nil {
		return fmt.Errorf("failed to create visibility frame: %w", err)
	}

	return nil
}

// processEnforcementDecision processes enforcement decision events
func (c *VisibilityConsumer) processEnforcementDecision(ctx context.Context, event EnforcementDecisionEvent) error {
	// Convert to database model and store
	dbDecision := &db.EnforcementDecision{
		ID:           uuid.New(),
		AssignmentID: event.AssignmentID,
		AgentUID:     event.AgentUID,
		Verdict:      event.Verdict,
		Reason:       event.Reason,
		RuleID:       event.RuleID,
		FlowData:     event.FlowData,
		ProcessData:  event.ProcessData,
		Timestamp:    event.Timestamp,
		Mode:         event.Mode,
	}

	if err := c.store.CreateEnforcementDecision(ctx, dbDecision); err != nil {
		return fmt.Errorf("failed to create enforcement decision: %w", err)
	}

	return nil
}

// processAssignmentEvent processes assignment events for tracking
func (c *VisibilityConsumer) processAssignmentEvent(ctx context.Context, event map[string]interface{}) error {
	// Log assignment events for audit purposes
	eventType, ok := event["event_type"].(string)
	if !ok {
		return fmt.Errorf("missing event_type in assignment event")
	}

	assignmentID, ok := event["assignment_id"].(string)
	if !ok {
		return fmt.Errorf("missing assignment_id in assignment event")
	}

	// Log the assignment event
	details := map[string]interface{}{
		"event_type":     eventType,
		"assignment_id":  assignmentID,
		"timestamp":      event["timestamp"],
	}

	detailsBytes, _ := json.Marshal(details)
	_, err := c.store.LogAuditEvent(ctx, "system", "assignment_event_received", assignmentID, detailsBytes)
	
	return err
}

// Data structures for parsing visibility data

type ProcessInfo struct {
	PID                int             `json:"pid"`
	PPID               int             `json:"ppid"`
	Name               string          `json:"name"`
	Cmdline            string          `json:"cmdline"`
	ExecutablePath     string          `json:"executable_path"`
	WorkingDirectory   string          `json:"working_directory"`
	UserID             int             `json:"user_id"`
	GroupID            int             `json:"group_id"`
	StartTime          *time.Time      `json:"start_time"`
	EndTime            *time.Time      `json:"end_time"`
	Status             string          `json:"status"`
	MemoryUsage        int64           `json:"memory_usage"`
	CPUUsage           float64         `json:"cpu_usage"`
	NetworkConnections json.RawMessage `json:"network_connections"`
}

type NetworkFlowInfo struct {
	SrcIP             string     `json:"src_ip"`
	DstIP             string     `json:"dst_ip"`
	SrcPort           int        `json:"src_port"`
	DstPort           int        `json:"dst_port"`
	Protocol          string     `json:"protocol"`
	BytesSent         int64      `json:"bytes_sent"`
	BytesReceived     int64      `json:"bytes_received"`
	PacketsSent       int64      `json:"packets_sent"`
	PacketsReceived   int64      `json:"packets_received"`
	StartTime         *time.Time `json:"start_time"`
	EndTime           *time.Time `json:"end_time"`
	Status            string     `json:"status"`
	ProcessID         int        `json:"process_id"`
	ProcessName       string     `json:"process_name"`
}

type SocketInfo struct {
	FD             int     `json:"fd"`
	PID            int     `json:"pid"`
	Family         int     `json:"family"`
	Type           int     `json:"type"`
	Protocol       int     `json:"protocol"`
	LocalAddress   string  `json:"local_address"`
	LocalPort      int     `json:"local_port"`
	RemoteAddress  string  `json:"remote_address"`
	RemotePort     int     `json:"remote_port"`
	State          string  `json:"state"`
}

type ExecEventInfo struct {
	PID              int        `json:"pid"`
	PPID             int        `json:"ppid"`
	ExecutablePath   string     `json:"executable_path"`
	Cmdline          string     `json:"cmdline"`
	WorkingDirectory string     `json:"working_directory"`
	UserID           int        `json:"user_id"`
	GroupID          int        `json:"group_id"`
	ExitCode         int        `json:"exit_code"`
	Timestamp        time.Time  `json:"timestamp"`
	DurationMs       int        `json:"duration_ms"`
}

type VisibilityFrame struct {
	Processes  json.RawMessage `json:"processes"`
	Flows      json.RawMessage `json:"flows"`
	Sockets    json.RawMessage `json:"sockets"`
	ExecEvents json.RawMessage `json:"exec_events"`
}

