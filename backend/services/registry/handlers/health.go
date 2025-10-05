package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/sgerhart/aegisflux/backend/internal/db"
	"github.com/sgerhart/aegisflux/backend/services/registry/signing"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	store  *db.Store
	signer *signing.Signer
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(store *db.Store, signer *signing.Signer) *HealthHandler {
	return &HealthHandler{
		store:  store,
		signer: signer,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
	Uptime    string            `json:"uptime,omitempty"`
}

// ReadyResponse represents the readiness check response
type ReadyResponse struct {
	Ready     bool              `json:"ready"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// Health handles GET /healthz
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
		Services:  make(map[string]string),
	}

	// Check database health
	dbHealthy := "healthy"
	if err := h.store.HealthCheck(r.Context()); err != nil {
		dbHealthy = "unhealthy"
		response.Status = "degraded"
	}
	response.Services["database"] = dbHealthy

	// Check signer health
	signerHealthy := "healthy"
	if !h.signer.IsHealthy() {
		signerHealthy = "unhealthy"
		response.Status = "degraded"
	}
	response.Services["signer"] = signerHealthy

	// Set HTTP status code based on overall health
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Ready handles GET /readyz
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := ReadyResponse{
		Ready:     true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make(map[string]string),
	}

	// Check database readiness
	dbReady := "ready"
	if err := h.store.ReadyCheck(r.Context()); err != nil {
		dbReady = "not_ready"
		response.Ready = false
	}
	response.Checks["database"] = dbReady

	// Check signer readiness
	signerReady := "ready"
	if !h.signer.IsReady() {
		signerReady = "not_ready"
		response.Ready = false
	}
	response.Checks["signer"] = signerReady

	// Set HTTP status code based on readiness
	statusCode := http.StatusOK
	if !response.Ready {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Liveness handles GET /livez (alternative liveness endpoint)
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"alive":     true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Startup handles GET /startup (alternative startup endpoint)
func (h *HealthHandler) Startup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"started":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DetailedHealth handles GET /health/detailed
func (h *HealthHandler) DetailedHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"services": map[string]interface{}{
			"database": h.getDatabaseHealth(r.Context()),
			"signer":   h.getSignerHealth(),
		},
		"system": map[string]interface{}{
			"go_version": "1.21",
			"platform":   "linux/amd64",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getDatabaseHealth returns detailed database health information
func (h *HealthHandler) getDatabaseHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": make(map[string]interface{}),
	}

	// Basic connectivity check
	if err := h.store.HealthCheck(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	// Readiness check
	if err := h.store.ReadyCheck(ctx); err != nil {
		health["status"] = "degraded"
		health["warning"] = err.Error()
	}

	return health
}

// getSignerHealth returns detailed signer health information
func (h *HealthHandler) getSignerHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": make(map[string]interface{}),
	}

	if !h.signer.IsHealthy() {
		health["status"] = "unhealthy"
		health["error"] = "signer is not healthy"
		return health
	}

	if !h.signer.IsReady() {
		health["status"] = "degraded"
		health["warning"] = "signer is not ready"
	}

	// Get active key information
	publicKey, kid, err := h.signer.GetActivePublicKey()
	if err != nil {
		health["warning"] = "could not get active key information"
	} else {
		health["active_key"] = map[string]interface{}{
			"kid":        kid,
			"public_key": publicKey[:32] + "...", // Truncate for security
		}
	}

	return health
}





