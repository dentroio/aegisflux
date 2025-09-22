package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sgerhart/aegisflux/backend/internal/db"
	"github.com/sgerhart/aegisflux/backend/services/registry/signing"
)

// AssignmentHandler handles assignment-related HTTP requests
type AssignmentHandler struct {
	store  *db.Store
	signer *signing.Signer
}

// NewAssignmentHandler creates a new assignment handler
func NewAssignmentHandler(store *db.Store, signer *signing.Signer) *AssignmentHandler {
	return &AssignmentHandler{
		store:  store,
		signer: signer,
	}
}

// PolicySnapshot represents the policy configuration for an assignment
type PolicySnapshot struct {
	AllowCIDRV4 []string `json:"allow_cidr_v4"`
	DenyCIDRV4  []string `json:"deny_cidr_v4"`
	AllowCIDRV6 []string `json:"allow_cidr_v6,omitempty"`
	DenyCIDRV6  []string `json:"deny_cidr_v6,omitempty"`
	Edges       []Edge   `json:"edges"`
	Rules       []Rule   `json:"rules,omitempty"`
}

// Edge represents a network flow edge between services
type Edge struct {
	Source      string `json:"src"`
	Destination string `json:"dst"`
	Port        int    `json:"port,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
}

// Rule represents a custom policy rule
type Rule struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Condition   map[string]interface{} `json:"condition"`
	Action      string                 `json:"action"`
	Priority    int                    `json:"priority,omitempty"`
}

// CreateAssignmentRequest represents a request to create a new assignment (Cap7.10 extended)
type CreateAssignmentRequest struct {
	BundleID     uuid.UUID       `json:"bundle_id"`
	Mode         string          `json:"mode"` // "observe" | "block"
	TTLSeconds   *int            `json:"ttl_seconds,omitempty"`
	Selector     json.RawMessage `json:"selector"` // Renamed from host_selector
	Snapshot     PolicySnapshot  `json:"snapshot"`
	DryRun       bool            `json:"dry_run,omitempty"`
	CreatedBy    string          `json:"created_by"`
}

// CreateAssignmentResponse represents the response after creating an assignment (Cap7.10 extended)
type CreateAssignmentResponse struct {
	ID           uuid.UUID       `json:"id"`
	BundleID     uuid.UUID       `json:"bundle_id"`
	Mode         string          `json:"mode"`
	TTLTS        *time.Time      `json:"ttl_ts"`
	Selector     json.RawMessage `json:"selector"`
	Snapshot     PolicySnapshot  `json:"snapshot"`
	SnapshotSig  string          `json:"snapshot_sig"`
	SnapshotKid  string          `json:"snapshot_kid"`
	DryRun       bool            `json:"dry_run"`
	CreatedAt    time.Time       `json:"created_at"`
	CreatedBy    string          `json:"created_by"`
	Status       string          `json:"status"`
}

// AssignmentResponse represents an assignment in API responses (Cap7.10 extended)
type AssignmentResponse struct {
	ID           uuid.UUID       `json:"id"`
	BundleID     uuid.UUID       `json:"bundle_id"`
	Mode         string          `json:"mode"`
	TTLTS        *time.Time      `json:"ttl_ts"`
	Selector     json.RawMessage `json:"selector"`
	Snapshot     PolicySnapshot  `json:"snapshot"`
	SnapshotSig  string          `json:"snapshot_sig"`
	SnapshotKid  string          `json:"snapshot_kid"`
	DryRun       bool            `json:"dry_run"`
	CreatedAt    time.Time       `json:"created_at"`
	CreatedBy    string          `json:"created_by"`
	Status       string          `json:"status"`
}

// ListAssignmentsResponse represents the response for listing assignments
type ListAssignmentsResponse struct {
	Assignments []AssignmentResponse `json:"assignments"`
	Total       int                  `json:"total"`
	Limit       int                  `json:"limit"`
	Offset      int                  `json:"offset"`
}

// AgentAssignmentResponse represents an assignment for a specific agent
type AgentAssignmentResponse struct {
	AgentUID     uuid.UUID       `json:"agent_uid"`
	HostID       string          `json:"host_id"`
	AssignmentID uuid.UUID       `json:"assignment_id"`
	BundleID     uuid.UUID       `json:"bundle_id"`
	BundleName   string          `json:"bundle_name"`
	BundleHash   string          `json:"bundle_hash"`
	BundleSig    string          `json:"bundle_sig"`
	BundleKid    string          `json:"bundle_kid"`
	BundleMeta   json.RawMessage `json:"bundle_meta"`
	TTLTS        *time.Time      `json:"ttl_ts"`
	DryRun       bool            `json:"dry_run"`
}

// CreateAssignment handles POST /assignments (Cap7.10 extended)
func (h *AssignmentHandler) CreateAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Selector == nil {
		http.Error(w, "Selector is required", http.StatusBadRequest)
		return
	}
	if req.BundleID == uuid.Nil {
		http.Error(w, "Bundle ID is required", http.StatusBadRequest)
		return
	}
	if req.Mode != "observe" && req.Mode != "block" {
		http.Error(w, "Mode must be 'observe' or 'block'", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "unknown"
	}

	// Validate selector format
	var selector map[string]interface{}
	if err := json.Unmarshal(req.Selector, &selector); err != nil {
		http.Error(w, fmt.Sprintf("Invalid selector format: %v", err), http.StatusBadRequest)
		return
	}

	// Validate policy snapshot schema
	if err := h.validatePolicySnapshot(req.Snapshot); err != nil {
		http.Error(w, fmt.Sprintf("Invalid policy snapshot: %v", err), http.StatusBadRequest)
		return
	}

	// Validate that bundle exists
	bundle, err := h.store.GetBundle(r.Context(), req.BundleID)
	if err != nil {
		http.Error(w, "Bundle not found", http.StatusNotFound)
		return
	}

	// Sign the policy snapshot
	snapshotBytes, err := json.Marshal(req.Snapshot)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	snapshotSig, snapshotKid, err := h.signer.Sign(snapshotBytes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate TTL timestamp if TTL seconds is provided
	var ttlTS *time.Time
	if req.TTLSeconds != nil && *req.TTLSeconds > 0 {
		ttl := time.Now().Add(time.Duration(*req.TTLSeconds) * time.Second)
		ttlTS = &ttl
	}

	// Create assignment in database (extend the db.Assignment struct if needed)
	assignmentID := uuid.New()
	assignment := &db.Assignment{
		ID:            assignmentID,
		HostSelector:  req.Selector, // Store as HostSelector for backward compatibility
		TTLTS:         ttlTS,
		DryRun:        req.DryRun,
		BundleID:      req.BundleID,
		CreatedBy:     req.CreatedBy,
		Status:        "active",
	}

	if err := h.store.CreateAssignment(r.Context(), assignment); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create assignment: %v", err), http.StatusInternalServerError)
		return
	}

	// Store signed snapshot (we'll need to extend the database schema)
	// For now, we'll store it in the assignment metadata or create a separate table
	
	// Log audit event
	auditDetails := map[string]interface{}{
		"selector":     req.Selector,
		"bundle_id":    req.BundleID.String(),
		"bundle_name":  bundle.Name,
		"mode":         req.Mode,
		"dry_run":      req.DryRun,
		"snapshot_sig": snapshotSig,
		"snapshot_kid": snapshotKid,
	}
	if ttlTS != nil {
		auditDetails["ttl_ts"] = ttlTS.Format(time.RFC3339)
	}
	auditDetailsBytes, _ := json.Marshal(auditDetails)
	
	_, err = h.store.LogAuditEvent(r.Context(), req.CreatedBy, "assignment_create", assignmentID.String(), auditDetailsBytes)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to log audit event: %v\n", err)
	}

	// TODO: Emit NATS event aegis.assignments.created
	// This will be implemented in the NATS publisher

	// Create response
	response := CreateAssignmentResponse{
		ID:           assignmentID,
		BundleID:     req.BundleID,
		Mode:         req.Mode,
		TTLTS:        ttlTS,
		Selector:     req.Selector,
		Snapshot:     req.Snapshot,
		SnapshotSig:  snapshotSig,
		SnapshotKid:  snapshotKid,
		DryRun:       req.DryRun,
		CreatedAt:    assignment.CreatedAt,
		CreatedBy:    req.CreatedBy,
		Status:       "active",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// validatePolicySnapshot validates the policy snapshot schema
func (h *AssignmentHandler) validatePolicySnapshot(snapshot PolicySnapshot) error {
	// Validate CIDR blocks if provided
	for _, cidr := range append(snapshot.AllowCIDRV4, snapshot.DenyCIDRV4...) {
		if cidr == "" {
			return fmt.Errorf("empty CIDR block not allowed")
		}
		// Additional CIDR validation could be added here
	}

	// Validate edges
	for i, edge := range snapshot.Edges {
		if edge.Source == "" {
			return fmt.Errorf("edge %d: source cannot be empty", i)
		}
		if edge.Destination == "" {
			return fmt.Errorf("edge %d: destination cannot be empty", i)
		}
		if edge.Port < 0 || edge.Port > 65535 {
			return fmt.Errorf("edge %d: invalid port %d", i, edge.Port)
		}
	}

	// Validate rules
	for i, rule := range snapshot.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule %d: ID cannot be empty", i)
		}
		if rule.Type == "" {
			return fmt.Errorf("rule %d: type cannot be empty", i)
		}
		if rule.Action == "" {
			return fmt.Errorf("rule %d: action cannot be empty", i)
		}
		if rule.Priority < 0 {
			return fmt.Errorf("rule %d: priority cannot be negative", i)
		}
	}

	return nil
}

// GetAssignment handles GET /assignments/{id}
func (h *AssignmentHandler) GetAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract assignment ID from URL path
	assignmentIDStr := r.URL.Path[len("/assignments/"):]
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		http.Error(w, "Invalid assignment ID", http.StatusBadRequest)
		return
	}

	// Get assignment from database
	assignment, err := h.store.GetAssignment(r.Context(), assignmentID)
	if err != nil {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// Create response
	response := AssignmentResponse{
		ID:           assignment.ID,
		HostSelector: assignment.HostSelector,
		TTLTS:        assignment.TTLTS,
		DryRun:       assignment.DryRun,
		BundleID:     assignment.BundleID,
		CreatedAt:    assignment.CreatedAt,
		CreatedBy:    assignment.CreatedBy,
		Status:       assignment.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListAssignments handles GET /assignments
func (h *AssignmentHandler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit := 50  // default limit
	offset := 0  // default offset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get assignments from database
	assignments, err := h.store.ListAssignments(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list assignments: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseAssignments := make([]AssignmentResponse, len(assignments))
	for i, assignment := range assignments {
		responseAssignments[i] = AssignmentResponse{
			ID:           assignment.ID,
			HostSelector: assignment.HostSelector,
			TTLTS:        assignment.TTLTS,
			DryRun:       assignment.DryRun,
			BundleID:     assignment.BundleID,
			CreatedAt:    assignment.CreatedAt,
			CreatedBy:    assignment.CreatedBy,
			Status:       assignment.Status,
		}
	}

	response := ListAssignmentsResponse{
		Assignments: responseAssignments,
		Total:       len(responseAssignments),
		Limit:       limit,
		Offset:      offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAgentAssignments handles GET /assignments/for-host/{host_id}
func (h *AssignmentHandler) GetAgentAssignments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract host ID from URL path
	hostID := r.URL.Path[len("/assignments/for-host/"):]
	if hostID == "" {
		http.Error(w, "Host ID is required", http.StatusBadRequest)
		return
	}

	// Get assignments for the agent from database
	assignments, err := h.store.GetAgentBundleAssignments(r.Context(), hostID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent assignments: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseAssignments := make([]AgentAssignmentResponse, len(assignments))
	for i, assignment := range assignments {
		responseAssignments[i] = AgentAssignmentResponse{
			AgentUID:     assignment.AgentUID,
			HostID:       assignment.HostID,
			AssignmentID: assignment.AssignmentID,
			BundleID:     assignment.BundleID,
			BundleName:   assignment.BundleName,
			BundleHash:   assignment.BundleHash,
			BundleSig:    assignment.BundleSig,
			BundleKid:    assignment.BundleKid,
			BundleMeta:   assignment.BundleMeta,
			TTLTS:        assignment.TTLTS,
			DryRun:       assignment.DryRun,
		}
	}

	response := map[string]interface{}{
		"assignments": responseAssignments,
		"total":       len(responseAssignments),
		"host_id":     hostID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelAssignment handles DELETE /assignments/{id}
func (h *AssignmentHandler) CancelAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract assignment ID from URL path
	assignmentIDStr := r.URL.Path[len("/assignments/"):]
	assignmentID, err := uuid.Parse(assignmentIDStr)
	if err != nil {
		http.Error(w, "Invalid assignment ID", http.StatusBadRequest)
		return
	}

	// Get current user from header or default
	user := r.Header.Get("X-User")
	if user == "" {
		user = "unknown"
	}

	// Update assignment status to cancelled
	query := `UPDATE assignments SET status = 'cancelled' WHERE id = $1`
	result, err := h.store.(*db.Store).(*sql.DB).ExecContext(r.Context(), query, assignmentID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to cancel assignment: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rows affected: %v", err), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	// Log audit event
	auditDetails := map[string]interface{}{
		"assignment_id": assignmentID.String(),
		"action":        "cancel",
	}
	auditDetailsBytes, _ := json.Marshal(auditDetails)
	
	_, err = h.store.LogAuditEvent(r.Context(), user, "assignment_cancel", assignmentID.String(), auditDetailsBytes)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to log audit event: %v\n", err)
	}

	// Return success response
	response := map[string]interface{}{
		"message":        "Assignment cancelled successfully",
		"assignment_id":  assignmentID,
		"cancelled_by":   user,
		"cancelled_at":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ValidateAssignment handles POST /assignments/validate
func (h *AssignmentHandler) ValidateAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		HostSelector json.RawMessage `json:"host_selector"`
		BundleID     uuid.UUID       `json:"bundle_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate bundle exists
	bundle, err := h.store.GetBundle(r.Context(), req.BundleID)
	if err != nil {
		http.Error(w, "Bundle not found", http.StatusNotFound)
		return
	}

	// Validate host selector format
	var selector map[string]interface{}
	if err := json.Unmarshal(req.HostSelector, &selector); err != nil {
		http.Error(w, fmt.Sprintf("Invalid host selector format: %v", err), http.StatusBadRequest)
		return
	}

	// Count matching agents (this would require a more complex query in a real implementation)
	// For now, we'll just validate the selector format
	matchingAgents := 0
	if hostID, ok := selector["host_id"].(string); ok && hostID != "" {
		// Check if specific agent exists
		_, err := h.store.GetAgent(r.Context(), hostID)
		if err == nil {
			matchingAgents = 1
		}
	} else {
		// For label-based selectors, we would need to query agents table
		// This is a simplified implementation
		matchingAgents = -1 // Unknown
	}

	response := map[string]interface{}{
		"valid":           true,
		"bundle_id":       req.BundleID,
		"bundle_name":     bundle.Name,
		"host_selector":   req.HostSelector,
		"matching_agents": matchingAgents,
		"message":         "Assignment validation successful",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
