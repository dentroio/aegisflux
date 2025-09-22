package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Store handles database operations for the AegisFlux backend
type Store struct {
	db *sql.DB
}

// NewStore creates a new database store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Agent represents a registered agent
type Agent struct {
	AgentUID     uuid.UUID       `json:"agent_uid" db:"agent_uid"`
	HostID       string          `json:"host_id" db:"host_id"`
	Platform     json.RawMessage `json:"platform" db:"platform"`
	Labels       json.RawMessage `json:"labels" db:"labels"`
	Notes        string          `json:"notes" db:"notes"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	LastSeenAt   time.Time       `json:"last_seen_at" db:"last_seen_at"`
}

// Bundle represents an eBPF program bundle
type Bundle struct {
	BundleID   uuid.UUID       `json:"bundle_id" db:"bundle_id"`
	Name       string          `json:"name" db:"name"`
	Hash       string          `json:"hash" db:"hash"`
	Sig        string          `json:"sig" db:"sig"`
	Algo       string          `json:"algo" db:"algo"`
	Kid        string          `json:"kid" db:"kid"`
	Meta       json.RawMessage `json:"meta" db:"meta"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	CreatedBy  string          `json:"created_by" db:"created_by"`
}

// Assignment represents a bundle assignment to agents
type Assignment struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	HostSelector  json.RawMessage `json:"host_selector" db:"host_selector"`
	TTLTS         *time.Time      `json:"ttl_ts" db:"ttl_ts"`
	DryRun        bool            `json:"dry_run" db:"dry_run"`
	BundleID      uuid.UUID       `json:"bundle_id" db:"bundle_id"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	CreatedBy     string          `json:"created_by" db:"created_by"`
	Status        string          `json:"status" db:"status"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID      uuid.UUID       `json:"id" db:"id"`
	Actor   string          `json:"actor" db:"actor"`
	Action  string          `json:"action" db:"action"`
	Target  *string         `json:"target" db:"target"`
	Details json.RawMessage `json:"details" db:"details"`
	At      time.Time       `json:"at" db:"at"`
}

// SigningKey represents a signing key
type SigningKey struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	Kid                 string     `json:"kid" db:"kid"`
	PublicKey           string     `json:"public_key" db:"public_key"`
	PrivateKeyEncrypted *string    `json:"private_key_encrypted" db:"private_key_encrypted"`
	Algorithm           string     `json:"algorithm" db:"algorithm"`
	Status              string     `json:"status" db:"status"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	RotatedAt           *time.Time `json:"rotated_at" db:"rotated_at"`
	ExpiresAt           *time.Time `json:"expires_at" db:"expires_at"`
}

// AgentBundleAssignment represents an agent's bundle assignment
type AgentBundleAssignment struct {
	AgentUID      uuid.UUID       `json:"agent_uid" db:"agent_uid"`
	HostID        string          `json:"host_id" db:"host_id"`
	Platform      json.RawMessage `json:"platform" db:"platform"`
	AgentLabels   json.RawMessage `json:"agent_labels" db:"agent_labels"`
	AssignmentID  uuid.UUID       `json:"assignment_id" db:"assignment_id"`
	BundleID      uuid.UUID       `json:"bundle_id" db:"bundle_id"`
	BundleName    string          `json:"bundle_name" db:"bundle_name"`
	BundleHash    string          `json:"bundle_hash" db:"bundle_hash"`
	BundleSig     string          `json:"bundle_sig" db:"bundle_sig"`
	BundleKid     string          `json:"bundle_kid" db:"bundle_kid"`
	BundleMeta    json.RawMessage `json:"bundle_meta" db:"bundle_meta"`
	TTLTS         *time.Time      `json:"ttl_ts" db:"ttl_ts"`
	DryRun        bool            `json:"dry_run" db:"dry_run"`
}

// Agent operations

// CreateAgent creates a new agent
func (s *Store) CreateAgent(ctx context.Context, agent *Agent) error {
	query := `
		INSERT INTO agents (agent_uid, host_id, platform, labels, notes)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (host_id) DO UPDATE SET
			platform = EXCLUDED.platform,
			labels = EXCLUDED.labels,
			notes = EXCLUDED.notes,
			updated_at = NOW(),
			last_seen_at = NOW()
	`
	_, err := s.db.ExecContext(ctx, query, agent.AgentUID, agent.HostID, agent.Platform, agent.Labels, agent.Notes)
	return err
}

// GetAgent retrieves an agent by host_id
func (s *Store) GetAgent(ctx context.Context, hostID string) (*Agent, error) {
	query := `SELECT agent_uid, host_id, platform, labels, notes, created_at, updated_at, last_seen_at FROM agents WHERE host_id = $1`
	
	agent := &Agent{}
	err := s.db.QueryRowContext(ctx, query, hostID).Scan(
		&agent.AgentUID, &agent.HostID, &agent.Platform, &agent.Labels,
		&agent.Notes, &agent.CreatedAt, &agent.UpdatedAt, &agent.LastSeenAt,
	)
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// ListAgents retrieves all agents with pagination
func (s *Store) ListAgents(ctx context.Context, limit, offset int) ([]*Agent, error) {
	query := `
		SELECT agent_uid, host_id, platform, labels, notes, created_at, updated_at, last_seen_at 
		FROM agents 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var agents []*Agent
	for rows.Next() {
		agent := &Agent{}
		err := rows.Scan(
			&agent.AgentUID, &agent.HostID, &agent.Platform, &agent.Labels,
			&agent.Notes, &agent.CreatedAt, &agent.UpdatedAt, &agent.LastSeenAt,
		)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// UpdateAgentLastSeen updates the last seen timestamp for an agent
func (s *Store) UpdateAgentLastSeen(ctx context.Context, hostID string) error {
	query := `UPDATE agents SET last_seen_at = NOW() WHERE host_id = $1`
	_, err := s.db.ExecContext(ctx, query, hostID)
	return err
}

// Bundle operations

// CreateBundle creates a new bundle
func (s *Store) CreateBundle(ctx context.Context, bundle *Bundle) error {
	query := `
		INSERT INTO bundles (bundle_id, name, hash, sig, algo, kid, meta, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := s.db.ExecContext(ctx, query, 
		bundle.BundleID, bundle.Name, bundle.Hash, bundle.Sig, 
		bundle.Algo, bundle.Kid, bundle.Meta, bundle.CreatedBy)
	return err
}

// GetBundle retrieves a bundle by ID
func (s *Store) GetBundle(ctx context.Context, bundleID uuid.UUID) (*Bundle, error) {
	query := `
		SELECT bundle_id, name, hash, sig, algo, kid, meta, created_at, created_by 
		FROM bundles WHERE bundle_id = $1
	`
	
	bundle := &Bundle{}
	err := s.db.QueryRowContext(ctx, query, bundleID).Scan(
		&bundle.BundleID, &bundle.Name, &bundle.Hash, &bundle.Sig,
		&bundle.Algo, &bundle.Kid, &bundle.Meta, &bundle.CreatedAt, &bundle.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

// GetBundleByHash retrieves a bundle by its hash
func (s *Store) GetBundleByHash(ctx context.Context, hash string) (*Bundle, error) {
	query := `
		SELECT bundle_id, name, hash, sig, algo, kid, meta, created_at, created_by 
		FROM bundles WHERE hash = $1
	`
	
	bundle := &Bundle{}
	err := s.db.QueryRowContext(ctx, query, hash).Scan(
		&bundle.BundleID, &bundle.Name, &bundle.Hash, &bundle.Sig,
		&bundle.Algo, &bundle.Kid, &bundle.Meta, &bundle.CreatedAt, &bundle.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return bundle, nil
}

// ListBundles retrieves all bundles with pagination
func (s *Store) ListBundles(ctx context.Context, limit, offset int) ([]*Bundle, error) {
	query := `
		SELECT bundle_id, name, hash, sig, algo, kid, meta, created_at, created_by 
		FROM bundles 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var bundles []*Bundle
	for rows.Next() {
		bundle := &Bundle{}
		err := rows.Scan(
			&bundle.BundleID, &bundle.Name, &bundle.Hash, &bundle.Sig,
			&bundle.Algo, &bundle.Kid, &bundle.Meta, &bundle.CreatedAt, &bundle.CreatedBy,
		)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	
	return bundles, nil
}

// Assignment operations

// CreateAssignment creates a new bundle assignment
func (s *Store) CreateAssignment(ctx context.Context, assignment *Assignment) error {
	query := `
		INSERT INTO assignments (id, host_selector, ttl_ts, dry_run, bundle_id, created_by, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		assignment.ID, assignment.HostSelector, assignment.TTLTS, assignment.DryRun,
		assignment.BundleID, assignment.CreatedBy, assignment.Status)
	return err
}

// GetAssignment retrieves an assignment by ID
func (s *Store) GetAssignment(ctx context.Context, assignmentID uuid.UUID) (*Assignment, error) {
	query := `
		SELECT id, host_selector, ttl_ts, dry_run, bundle_id, created_at, created_by, status 
		FROM assignments WHERE id = $1
	`
	
	assignment := &Assignment{}
	err := s.db.QueryRowContext(ctx, query, assignmentID).Scan(
		&assignment.ID, &assignment.HostSelector, &assignment.TTLTS, &assignment.DryRun,
		&assignment.BundleID, &assignment.CreatedAt, &assignment.CreatedBy, &assignment.Status,
	)
	if err != nil {
		return nil, err
	}
	return assignment, nil
}

// ListAssignments retrieves all assignments with pagination
func (s *Store) ListAssignments(ctx context.Context, limit, offset int) ([]*Assignment, error) {
	query := `
		SELECT id, host_selector, ttl_ts, dry_run, bundle_id, created_at, created_by, status 
		FROM assignments 
		ORDER BY created_at DESC 
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assignments []*Assignment
	for rows.Next() {
		assignment := &Assignment{}
		err := rows.Scan(
			&assignment.ID, &assignment.HostSelector, &assignment.TTLTS, &assignment.DryRun,
			&assignment.BundleID, &assignment.CreatedAt, &assignment.CreatedBy, &assignment.Status,
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	
	return assignments, nil
}

// GetAgentBundleAssignments retrieves bundle assignments for a specific agent
func (s *Store) GetAgentBundleAssignments(ctx context.Context, hostID string) ([]*AgentBundleAssignment, error) {
	query := `
		SELECT agent_uid, host_id, platform, agent_labels, assignment_id, 
		       bundle_id, bundle_name, bundle_hash, bundle_sig, bundle_kid, 
		       bundle_meta, ttl_ts, dry_run
		FROM agent_bundle_assignments 
		WHERE host_id = $1
	`
	
	rows, err := s.db.QueryContext(ctx, query, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var assignments []*AgentBundleAssignment
	for rows.Next() {
		assignment := &AgentBundleAssignment{}
		err := rows.Scan(
			&assignment.AgentUID, &assignment.HostID, &assignment.Platform, &assignment.AgentLabels,
			&assignment.AssignmentID, &assignment.BundleID, &assignment.BundleName, &assignment.BundleHash,
			&assignment.BundleSig, &assignment.BundleKid, &assignment.BundleMeta, &assignment.TTLTS, &assignment.DryRun,
		)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	
	return assignments, nil
}

// Audit operations

// LogAuditEvent logs an audit event
func (s *Store) LogAuditEvent(ctx context.Context, actor, action, target string, details json.RawMessage) (uuid.UUID, error) {
	query := `
		SELECT log_audit_event($1, $2, $3, $4)
	`
	
	var auditID uuid.UUID
	err := s.db.QueryRowContext(ctx, query, actor, action, target, details).Scan(&auditID)
	return auditID, err
}

// GetAuditLog retrieves audit log entries with pagination
func (s *Store) GetAuditLog(ctx context.Context, limit, offset int) ([]*AuditLogEntry, error) {
	query := `
		SELECT id, actor, action, target, details, at 
		FROM audit_log 
		ORDER BY at DESC 
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var entries []*AuditLogEntry
	for rows.Next() {
		entry := &AuditLogEntry{}
		err := rows.Scan(&entry.ID, &entry.Actor, &entry.Action, &entry.Target, &entry.Details, &entry.At)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	
	return entries, nil
}

// Signing key operations

// GetActiveSigningKey retrieves the currently active signing key
func (s *Store) GetActiveSigningKey(ctx context.Context) (*SigningKey, error) {
	query := `
		SELECT id, kid, public_key, private_key_encrypted, algorithm, status, 
		       created_at, rotated_at, expires_at
		FROM signing_keys 
		WHERE status = 'active' 
		ORDER BY created_at DESC 
		LIMIT 1
	`
	
	key := &SigningKey{}
	err := s.db.QueryRowContext(ctx, query).Scan(
		&key.ID, &key.Kid, &key.PublicKey, &key.PrivateKeyEncrypted, &key.Algorithm,
		&key.Status, &key.CreatedAt, &key.RotatedAt, &key.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// CreateSigningKey creates a new signing key
func (s *Store) CreateSigningKey(ctx context.Context, key *SigningKey) error {
	query := `
		INSERT INTO signing_keys (id, kid, public_key, private_key_encrypted, algorithm, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Kid, key.PublicKey, key.PrivateKeyEncrypted, key.Algorithm, key.Status, key.ExpiresAt)
	return err
}

// RotateSigningKey rotates the signing key by marking the current one as rotated and setting the new one as active
func (s *Store) RotateSigningKey(ctx context.Context, oldKid, newKid string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// Mark old key as rotated
	_, err = tx.ExecContext(ctx, `UPDATE signing_keys SET status = 'rotated', rotated_at = NOW() WHERE kid = $1`, oldKid)
	if err != nil {
		return err
	}
	
	// Set new key as active
	_, err = tx.ExecContext(ctx, `UPDATE signing_keys SET status = 'active' WHERE kid = $1`, newKid)
	if err != nil {
		return err
	}
	
	return tx.Commit()
}

// Health check operations

// HealthCheck checks if the database is healthy
func (s *Store) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// ReadyCheck checks if the database is ready (has required tables)
func (s *Store) ReadyCheck(ctx context.Context) error {
	query := `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('agents', 'bundles', 'assignments', 'audit_log', 'signing_keys')
	`
	
	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return err
	}
	
	if count < 5 {
		return fmt.Errorf("not all required tables exist: found %d of 5", count)
	}
	
	return nil
}

// VisibilityFrame represents a visibility data frame from an agent
type VisibilityFrame struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	AgentUID   string          `json:"agent_uid" db:"agent_uid"`
	Timestamp  time.Time       `json:"ts" db:"ts"`
	Processes  json.RawMessage `json:"procs" db:"procs"`
	Flows      json.RawMessage `json:"flows" db:"flows"`
	Sockets    json.RawMessage `json:"sockets" db:"sockets"`
	ExecEvents json.RawMessage `json:"exec_events" db:"exec_events"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// NetworkFlow represents a network flow record
type NetworkFlow struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	AgentUID       string     `json:"agent_uid" db:"agent_uid"`
	SrcIP          string     `json:"src_ip" db:"src_ip"`
	DstIP          string     `json:"dst_ip" db:"dst_ip"`
	SrcPort        int        `json:"src_port" db:"src_port"`
	DstPort        int        `json:"dst_port" db:"dst_port"`
	Protocol       string     `json:"protocol" db:"protocol"`
	BytesSent      int64      `json:"bytes_sent" db:"bytes_sent"`
	BytesReceived  int64      `json:"bytes_received" db:"bytes_received"`
	PacketsSent    int64      `json:"packets_sent" db:"packets_sent"`
	PacketsReceived int64     `json:"packets_received" db:"packets_received"`
	StartTime      *time.Time `json:"start_time" db:"start_time"`
	EndTime        *time.Time `json:"end_time" db:"end_time"`
	Status         string     `json:"status" db:"status"`
	ProcessID      int        `json:"process_id" db:"process_id"`
	ProcessName    string     `json:"process_name" db:"process_name"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// Process represents a process record
type Process struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	AgentUID         string          `json:"agent_uid" db:"agent_uid"`
	PID              int             `json:"pid" db:"pid"`
	PPID             int             `json:"ppid" db:"ppid"`
	Name             string          `json:"name" db:"name"`
	Cmdline          string          `json:"cmdline" db:"cmdline"`
	ExecutablePath   string          `json:"executable_path" db:"executable_path"`
	WorkingDirectory string          `json:"working_directory" db:"working_directory"`
	UserID           int             `json:"user_id" db:"user_id"`
	GroupID          int             `json:"group_id" db:"group_id"`
	StartTime        *time.Time      `json:"start_time" db:"start_time"`
	EndTime          *time.Time      `json:"end_time" db:"end_time"`
	Status           string          `json:"status" db:"status"`
	MemoryUsage      int64           `json:"memory_usage" db:"memory_usage"`
	CPUUsage         float64         `json:"cpu_usage" db:"cpu_usage"`
	NetworkConnections json.RawMessage `json:"network_connections" db:"network_connections"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// Socket represents a socket record
type Socket struct {
	ID            uuid.UUID `json:"id" db:"id"`
	AgentUID      string    `json:"agent_uid" db:"agent_uid"`
	FD            int       `json:"fd" db:"fd"`
	PID           int       `json:"pid" db:"pid"`
	Family        int       `json:"family" db:"family"`
	Type          int       `json:"type" db:"type"`
	Protocol      int       `json:"protocol" db:"protocol"`
	LocalAddress  string    `json:"local_address" db:"local_address"`
	LocalPort     int       `json:"local_port" db:"local_port"`
	RemoteAddress string    `json:"remote_address" db:"remote_address"`
	RemotePort    int       `json:"remote_port" db:"remote_port"`
	State         string    `json:"state" db:"state"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ExecEvent represents an execution event record
type ExecEvent struct {
	ID              uuid.UUID `json:"id" db:"id"`
	AgentUID        string    `json:"agent_uid" db:"agent_uid"`
	PID             int       `json:"pid" db:"pid"`
	PPID            int       `json:"ppid" db:"ppid"`
	ExecutablePath  string    `json:"executable_path" db:"executable_path"`
	Cmdline         string    `json:"cmdline" db:"cmdline"`
	WorkingDirectory string   `json:"working_directory" db:"working_directory"`
	UserID          int       `json:"user_id" db:"user_id"`
	GroupID         int       `json:"group_id" db:"group_id"`
	ExitCode        int       `json:"exit_code" db:"exit_code"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	DurationMs      int       `json:"duration_ms" db:"duration_ms"`
}

// EnforcementDecision represents an enforcement decision record
type EnforcementDecision struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	AssignmentID uuid.UUID       `json:"assignment_id" db:"assignment_id"`
	AgentUID     string          `json:"agent_uid" db:"agent_uid"`
	Verdict      string          `json:"verdict" db:"verdict"`
	Reason       string          `json:"reason" db:"reason"`
	RuleID       string          `json:"rule_id" db:"rule_id"`
	FlowData     json.RawMessage `json:"flow_data" db:"flow_data"`
	ProcessData  json.RawMessage `json:"process_data" db:"process_data"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
	Mode         string          `json:"mode" db:"mode"`
}

// Visibility operations

// CreateVisibilityFrame creates a new visibility frame
func (s *Store) CreateVisibilityFrame(ctx context.Context, frame *VisibilityFrame) error {
	query := `
		INSERT INTO visibility_frames (id, agent_uid, ts, procs, flows, sockets, exec_events)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := s.db.ExecContext(ctx, query,
		frame.ID, frame.AgentUID, frame.Timestamp, frame.Processes, frame.Flows, frame.Sockets, frame.ExecEvents)
	return err
}

// GetLatestVisibilityFrame gets the latest visibility frame for an agent
func (s *Store) GetLatestVisibilityFrame(ctx context.Context, agentUID string) (*VisibilityFrame, error) {
	query := `
		SELECT id, agent_uid, ts, procs, flows, sockets, exec_events, created_at
		FROM visibility_frames
		WHERE agent_uid = $1
		ORDER BY ts DESC
		LIMIT 1
	`
	
	frame := &VisibilityFrame{}
	err := s.db.QueryRowContext(ctx, query, agentUID).Scan(
		&frame.ID, &frame.AgentUID, &frame.Timestamp, &frame.Processes,
		&frame.Flows, &frame.Sockets, &frame.ExecEvents, &frame.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return frame, nil
}

// CreateNetworkFlow creates a new network flow record
func (s *Store) CreateNetworkFlow(ctx context.Context, flow *NetworkFlow) error {
	query := `
		INSERT INTO network_flows (id, agent_uid, src_ip, dst_ip, src_port, dst_port, protocol, 
			bytes_sent, bytes_received, packets_sent, packets_received, start_time, end_time, 
			status, process_id, process_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := s.db.ExecContext(ctx, query,
		flow.ID, flow.AgentUID, flow.SrcIP, flow.DstIP, flow.SrcPort, flow.DstPort, flow.Protocol,
		flow.BytesSent, flow.BytesReceived, flow.PacketsSent, flow.PacketsReceived,
		flow.StartTime, flow.EndTime, flow.Status, flow.ProcessID, flow.ProcessName)
	return err
}

// CreateProcess creates a new process record
func (s *Store) CreateProcess(ctx context.Context, process *Process) error {
	query := `
		INSERT INTO processes (id, agent_uid, pid, ppid, name, cmdline, executable_path, 
			working_directory, user_id, group_id, start_time, end_time, status, 
			memory_usage, cpu_usage, network_connections)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := s.db.ExecContext(ctx, query,
		process.ID, process.AgentUID, process.PID, process.PPID, process.Name, process.Cmdline,
		process.ExecutablePath, process.WorkingDirectory, process.UserID, process.GroupID,
		process.StartTime, process.EndTime, process.Status, process.MemoryUsage,
		process.CPUUsage, process.NetworkConnections)
	return err
}

// CreateSocket creates a new socket record
func (s *Store) CreateSocket(ctx context.Context, socket *Socket) error {
	query := `
		INSERT INTO sockets (id, agent_uid, fd, pid, family, type, protocol, 
			local_address, local_port, remote_address, remote_port, state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		socket.ID, socket.AgentUID, socket.FD, socket.PID, socket.Family, socket.Type,
		socket.Protocol, socket.LocalAddress, socket.LocalPort, socket.RemoteAddress,
		socket.RemotePort, socket.State)
	return err
}

// CreateExecEvent creates a new execution event record
func (s *Store) CreateExecEvent(ctx context.Context, event *ExecEvent) error {
	query := `
		INSERT INTO exec_events (id, agent_uid, pid, ppid, executable_path, cmdline, 
			working_directory, user_id, group_id, exit_code, timestamp, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.AgentUID, event.PID, event.PPID, event.ExecutablePath, event.Cmdline,
		event.WorkingDirectory, event.UserID, event.GroupID, event.ExitCode,
		event.Timestamp, event.DurationMs)
	return err
}

// CreateEnforcementDecision creates a new enforcement decision record
func (s *Store) CreateEnforcementDecision(ctx context.Context, decision *EnforcementDecision) error {
	query := `
		INSERT INTO enforcement_decisions (id, assignment_id, agent_uid, verdict, reason, 
			rule_id, flow_data, process_data, timestamp, mode)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := s.db.ExecContext(ctx, query,
		decision.ID, decision.AssignmentID, decision.AgentUID, decision.Verdict,
		decision.Reason, decision.RuleID, decision.FlowData, decision.ProcessData,
		decision.Timestamp, decision.Mode)
	return err
}

// GetVisibilityHistory gets visibility history for an agent
func (s *Store) GetVisibilityHistory(ctx context.Context, agentUID string, limit, offset int) ([]*VisibilityFrame, error) {
	query := `
		SELECT id, agent_uid, ts, procs, flows, sockets, exec_events, created_at
		FROM visibility_frames
		WHERE agent_uid = $1
		ORDER BY ts DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := s.db.QueryContext(ctx, query, agentUID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var frames []*VisibilityFrame
	for rows.Next() {
		frame := &VisibilityFrame{}
		err := rows.Scan(
			&frame.ID, &frame.AgentUID, &frame.Timestamp, &frame.Processes,
			&frame.Flows, &frame.Sockets, &frame.ExecEvents, &frame.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}

	return frames, nil
}

// GetEnforcementDecisions gets enforcement decisions for an agent
func (s *Store) GetEnforcementDecisions(ctx context.Context, agentUID, verdict, mode string, limit int) ([]*EnforcementDecision, error) {
	query := `
		SELECT id, assignment_id, agent_uid, verdict, reason, rule_id, 
			flow_data, process_data, timestamp, mode
		FROM enforcement_decisions
		WHERE agent_uid = $1
	`
	args := []interface{}{agentUID}
	argIndex := 2

	if verdict != "" {
		query += fmt.Sprintf(" AND verdict = $%d", argIndex)
		args = append(args, verdict)
		argIndex++
	}

	if mode != "" {
		query += fmt.Sprintf(" AND mode = $%d", argIndex)
		args = append(args, mode)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decisions []*EnforcementDecision
	for rows.Next() {
		decision := &EnforcementDecision{}
		err := rows.Scan(
			&decision.ID, &decision.AssignmentID, &decision.AgentUID, &decision.Verdict,
			&decision.Reason, &decision.RuleID, &decision.FlowData, &decision.ProcessData,
			&decision.Timestamp, &decision.Mode,
		)
		if err != nil {
			return nil, err
		}
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

// GetNetworkFlows gets network flows for an agent
func (s *Store) GetNetworkFlows(ctx context.Context, agentUID, protocol, status string, limit int) ([]*NetworkFlow, error) {
	query := `
		SELECT id, agent_uid, src_ip, dst_ip, src_port, dst_port, protocol,
			bytes_sent, bytes_received, packets_sent, packets_received,
			start_time, end_time, status, process_id, process_name, created_at
		FROM network_flows
		WHERE agent_uid = $1
	`
	args := []interface{}{agentUID}
	argIndex := 2

	if protocol != "" {
		query += fmt.Sprintf(" AND protocol = $%d", argIndex)
		args = append(args, protocol)
		argIndex++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY start_time DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flows []*NetworkFlow
	for rows.Next() {
		flow := &NetworkFlow{}
		err := rows.Scan(
			&flow.ID, &flow.AgentUID, &flow.SrcIP, &flow.DstIP, &flow.SrcPort, &flow.DstPort,
			&flow.Protocol, &flow.BytesSent, &flow.BytesReceived, &flow.PacketsSent,
			&flow.PacketsReceived, &flow.StartTime, &flow.EndTime, &flow.Status,
			&flow.ProcessID, &flow.ProcessName, &flow.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		flows = append(flows, flow)
	}

	return flows, nil
}

// GetProcesses gets processes for an agent
func (s *Store) GetProcesses(ctx context.Context, agentUID, status, name string, limit int) ([]*Process, error) {
	query := `
		SELECT id, agent_uid, pid, ppid, name, cmdline, executable_path,
			working_directory, user_id, group_id, start_time, end_time,
			status, memory_usage, cpu_usage, network_connections, created_at
		FROM processes
		WHERE agent_uid = $1
	`
	args := []interface{}{agentUID}
	argIndex := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if name != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", argIndex)
		args = append(args, "%"+name+"%")
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY start_time DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var processes []*Process
	for rows.Next() {
		process := &Process{}
		err := rows.Scan(
			&process.ID, &process.AgentUID, &process.PID, &process.PPID, &process.Name,
			&process.Cmdline, &process.ExecutablePath, &process.WorkingDirectory,
			&process.UserID, &process.GroupID, &process.StartTime, &process.EndTime,
			&process.Status, &process.MemoryUsage, &process.CPUUsage,
			&process.NetworkConnections, &process.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}

	return processes, nil
}

// GetVisibilitySummary gets visibility summary for an agent using the database function
func (s *Store) GetVisibilitySummary(ctx context.Context, agentUID string) (map[string]interface{}, error) {
	query := `SELECT * FROM get_agent_visibility_summary($1)`
	
	row := s.db.QueryRowContext(ctx, query, agentUID)
	
	var summary map[string]interface{}
	var latestFrameTS time.Time
	var totalProcesses, totalFlows, totalSockets, totalExecEvents, activeProcesses int64
	
	err := row.Scan(
		&summary, &latestFrameTS, &totalProcesses, &totalFlows, 
		&totalSockets, &totalExecEvents, &activeProcesses,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"agent_uid":           agentUID,
		"latest_frame_ts":     latestFrameTS,
		"total_processes":     totalProcesses,
		"total_flows":         totalFlows,
		"total_sockets":       totalSockets,
		"total_exec_events":   totalExecEvents,
		"active_processes":    activeProcesses,
	}, nil
}
