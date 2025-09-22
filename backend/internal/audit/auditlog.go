package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sgerhart/aegisflux/backend/internal/db"
)

// AuditLogger handles audit logging operations
type AuditLogger struct {
	store *db.Store
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(store *db.Store) *AuditLogger {
	return &AuditLogger{
		store: store,
	}
}

// AuditEvent represents an audit event
type AuditEvent struct {
	ID        uuid.UUID       `json:"id"`
	Actor     string          `json:"actor"`
	Action    string          `json:"action"`
	Target    string          `json:"target,omitempty"`
	Details   json.RawMessage `json:"details"`
	Timestamp time.Time       `json:"timestamp"`
	IPAddress string          `json:"ip_address,omitempty"`
	UserAgent string          `json:"user_agent,omitempty"`
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(ctx context.Context, actor, action, target string, details json.RawMessage) (uuid.UUID, error) {
	return al.store.LogAuditEvent(ctx, actor, action, target, details)
}

// LogHTTPRequest logs an HTTP request as an audit event
func (al *AuditLogger) LogHTTPRequest(ctx context.Context, r *http.Request, action, target string, details json.RawMessage) (uuid.UUID, error) {
	// Extract actor information
	actor := al.extractActor(r)
	
	// Add HTTP-specific details
	httpDetails := map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"query":      r.URL.RawQuery,
		"user_agent": r.UserAgent(),
		"remote_addr": r.RemoteAddr,
	}
	
	// Add existing details
	if details != nil {
		var existingDetails map[string]interface{}
		if err := json.Unmarshal(details, &existingDetails); err == nil {
			for k, v := range existingDetails {
				httpDetails[k] = v
			}
		}
	}
	
	// Marshal combined details
	combinedDetails, err := json.Marshal(httpDetails)
	if err != nil {
		log.Printf("Warning: Failed to marshal audit details: %v", err)
		combinedDetails = details
	}
	
	return al.LogEvent(ctx, actor, action, target, combinedDetails)
}

// LogBundleCreate logs a bundle creation event
func (al *AuditLogger) LogBundleCreate(ctx context.Context, r *http.Request, bundleID uuid.UUID, bundleName string, createdBy string) (uuid.UUID, error) {
	details := map[string]interface{}{
		"bundle_id":   bundleID.String(),
		"bundle_name": bundleName,
		"created_by":  createdBy,
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "bundle_create", bundleID.String(), detailsBytes)
}

// LogBundleGet logs a bundle retrieval event
func (al *AuditLogger) LogBundleGet(ctx context.Context, r *http.Request, bundleID uuid.UUID) (uuid.UUID, error) {
	details := map[string]interface{}{
		"bundle_id": bundleID.String(),
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "bundle_get", bundleID.String(), detailsBytes)
}

// LogBundleVerify logs a bundle verification event
func (al *AuditLogger) LogBundleVerify(ctx context.Context, r *http.Request, bundleID uuid.UUID, verified bool, errorMsg string) (uuid.UUID, error) {
	details := map[string]interface{}{
		"bundle_id": bundleID.String(),
		"verified":  verified,
	}
	
	if errorMsg != "" {
		details["error"] = errorMsg
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "bundle_verify", bundleID.String(), detailsBytes)
}

// LogAssignmentCreate logs an assignment creation event
func (al *AuditLogger) LogAssignmentCreate(ctx context.Context, r *http.Request, assignmentID uuid.UUID, bundleID uuid.UUID, createdBy string, dryRun bool) (uuid.UUID, error) {
	details := map[string]interface{}{
		"assignment_id": assignmentID.String(),
		"bundle_id":     bundleID.String(),
		"created_by":    createdBy,
		"dry_run":       dryRun,
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "assignment_create", assignmentID.String(), detailsBytes)
}

// LogAssignmentGet logs an assignment retrieval event
func (al *AuditLogger) LogAssignmentGet(ctx context.Context, r *http.Request, assignmentID uuid.UUID) (uuid.UUID, error) {
	details := map[string]interface{}{
		"assignment_id": assignmentID.String(),
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "assignment_get", assignmentID.String(), detailsBytes)
}

// LogAssignmentCancel logs an assignment cancellation event
func (al *AuditLogger) LogAssignmentCancel(ctx context.Context, r *http.Request, assignmentID uuid.UUID, cancelledBy string) (uuid.UUID, error) {
	details := map[string]interface{}{
		"assignment_id": assignmentID.String(),
		"cancelled_by":  cancelledBy,
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "assignment_cancel", assignmentID.String(), detailsBytes)
}

// LogAgentRegister logs an agent registration event
func (al *AuditLogger) LogAgentRegister(ctx context.Context, r *http.Request, agentUID uuid.UUID, hostID string) (uuid.UUID, error) {
	details := map[string]interface{}{
		"agent_uid": agentUID.String(),
		"host_id":   hostID,
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "agent_register", agentUID.String(), detailsBytes)
}

// LogKeyRotate logs a key rotation event
func (al *AuditLogger) LogKeyRotate(ctx context.Context, r *http.Request, oldKid, newKid string, rotatedBy string) (uuid.UUID, error) {
	details := map[string]interface{}{
		"old_kid":     oldKid,
		"new_kid":     newKid,
		"rotated_by":  rotatedBy,
	}
	
	detailsBytes, _ := json.Marshal(details)
	return al.LogHTTPRequest(ctx, r, "key_rotate", newKid, detailsBytes)
}

// LogSystemEvent logs a system event
func (al *AuditLogger) LogSystemEvent(ctx context.Context, actor, action, target string, details map[string]interface{}) (uuid.UUID, error) {
	detailsBytes, _ := json.Marshal(details)
	return al.LogEvent(ctx, actor, action, target, detailsBytes)
}

// extractActor extracts the actor (user) from the HTTP request
func (al *AuditLogger) extractActor(r *http.Request) string {
	// Try to get from client certificate
	if cert, err := al.extractClientCert(r); err == nil {
		return cert.Subject.CommonName
	}
	
	// Try to get from header
	if user := r.Header.Get("X-User"); user != "" {
		return user
	}
	
	// Try to get from authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Simple extraction - in production, you'd want proper JWT parsing
		if len(auth) > 7 && auth[:7] == "Bearer " {
			return "bearer_token_user"
		}
	}
	
	// Fallback to remote address
	return r.RemoteAddr
}

// extractClientCert extracts client certificate from request
func (al *AuditLogger) extractClientCert(r *http.Request) (*X509Certificate, error) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate found")
	}
	
	return r.TLS.PeerCertificates[0], nil
}

// GetAuditLog retrieves audit log entries
func (al *AuditLogger) GetAuditLog(ctx context.Context, limit, offset int) ([]*db.AuditLogEntry, error) {
	return al.store.GetAuditLog(ctx, limit, offset)
}

// AuditMiddleware creates middleware for automatic audit logging
func (al *AuditLogger) AuditMiddleware(action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log the request
			_, err := al.LogHTTPRequest(r.Context(), r, action, "", nil)
			if err != nil {
				log.Printf("Warning: Failed to log audit event: %v", err)
			}
			
			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// X509Certificate represents an X.509 certificate (simplified interface)
type X509Certificate interface {
	Subject() string
	Issuer() string
	SerialNumber() string
}

// AuditQuery represents a query for audit logs
type AuditQuery struct {
	Actor     string    `json:"actor,omitempty"`
	Action    string    `json:"action,omitempty"`
	Target    string    `json:"target,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// SearchAuditLog searches audit logs based on criteria
func (al *AuditLogger) SearchAuditLog(ctx context.Context, query AuditQuery) ([]*db.AuditLogEntry, error) {
	// This would require extending the store with search functionality
	// For now, we'll use the basic list function
	return al.store.GetAuditLog(ctx, query.Limit, query.Offset)
}

// AuditStats represents audit statistics
type AuditStats struct {
	TotalEvents    int                    `json:"total_events"`
	EventsByAction map[string]int         `json:"events_by_action"`
	EventsByActor  map[string]int         `json:"events_by_actor"`
	RecentEvents   []*db.AuditLogEntry    `json:"recent_events"`
}

// GetAuditStats retrieves audit statistics
func (al *AuditLogger) GetAuditStats(ctx context.Context) (*AuditStats, error) {
	// Get recent events
	recentEvents, err := al.store.GetAuditLog(ctx, 10, 0)
	if err != nil {
		return nil, err
	}
	
	// Calculate stats (simplified - would need more complex queries in production)
	stats := &AuditStats{
		TotalEvents:    len(recentEvents),
		EventsByAction: make(map[string]int),
		EventsByActor:  make(map[string]int),
		RecentEvents:   recentEvents,
	}
	
	for _, event := range recentEvents {
		stats.EventsByAction[event.Action]++
		stats.EventsByActor[event.Actor]++
	}
	
	return stats, nil
}

