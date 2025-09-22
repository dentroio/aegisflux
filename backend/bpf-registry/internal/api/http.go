package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"aegisflux/backend/bpf-registry/internal/model"
)

// ArtifactStore interface for artifact operations
type ArtifactStore interface {
	StoreArtifact(req *model.CreateArtifactRequest) (*model.Artifact, error)
	GetArtifact(id string) (*model.Artifact, error)
	GetArtifactBinary(id string) ([]byte, error)
	GetArtifactsForHost(hostID string) ([]*model.Artifact, error)
	ListArtifacts() ([]*model.Artifact, error)
	// Host assignment methods
	AssignArtifactToHost(artifactID, hostID string) error
	UnassignArtifactFromHost(artifactID, hostID string) error
	UpdateArtifactHosts(artifactID string, hosts []string) error
}

// HTTPAPI handles HTTP requests for the BPF registry
type HTTPAPI struct {
	store  ArtifactStore
	logger *slog.Logger
}

// NewHTTPAPI creates a new HTTP API handler
func NewHTTPAPI(store ArtifactStore, logger *slog.Logger) *HTTPAPI {
	return &HTTPAPI{
		store:  store,
		logger: logger,
	}
}

// SetupRoutes sets up all HTTP routes
func (api *HTTPAPI) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", api.handleHealthz)

	// Artifact endpoints - more specific routes to avoid conflicts
	mux.HandleFunc("/artifacts", api.handleArtifacts) // POST for creation, GET for listing
	
	// Specific routes for different artifact operations
	mux.HandleFunc("/artifacts/for-host/", api.handleGetArtifactsForHost)
	mux.HandleFunc("/artifacts/binary/", api.handleGetArtifactBinary)
	mux.HandleFunc("/artifacts/", api.handleArtifactWithPath) // Handle /artifacts/{id} and /artifacts/{id}/binary
	mux.HandleFunc("/assign/", api.handleAssignArtifact)
	mux.HandleFunc("/unassign/", api.handleUnassignArtifact)
	mux.HandleFunc("/hosts/", api.handleUpdateArtifactHosts)

	return mux
}

// handleHealthz handles health check requests
func (api *HTTPAPI) handleHealthz(w http.ResponseWriter, r *http.Request) {
	response := model.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

// handleArtifacts handles /artifacts endpoint (POST for creation, GET for listing)
func (api *HTTPAPI) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		api.handleCreateArtifact(w, r)
	case http.MethodGet:
		api.handleListArtifacts(w, r)
	default:
		api.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleArtifactWithPath handles /artifacts/{id} and /artifacts/{id}/binary
func (api *HTTPAPI) handleArtifactWithPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract path after /artifacts/
	path := strings.TrimPrefix(r.URL.Path, "/artifacts/")
	if path == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID")
		return
	}

	// Check if this is a binary request
	if strings.HasSuffix(path, "/binary") {
		// Extract artifact ID from /artifacts/{id}/binary
		artifactID := strings.TrimSuffix(path, "/binary")
		if artifactID == "" {
			api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID")
			return
		}
		api.handleGetArtifactBinaryByID(w, r, artifactID)
	} else {
		// Handle /artifacts/{id} for metadata
		api.handleGetArtifactByID(w, r, path)
	}
}

// handleGetArtifactByID handles GET /artifacts/{id} for metadata retrieval
func (api *HTTPAPI) handleGetArtifactByID(w http.ResponseWriter, r *http.Request, artifactID string) {
	api.logger.Info("Retrieving artifact metadata", "id", artifactID)

	artifact, err := api.store.GetArtifact(artifactID)
	if err != nil {
		api.logger.Error("Failed to retrieve artifact", "id", artifactID, "error", err)
		api.writeErrorResponse(w, http.StatusNotFound, "Artifact not found")
		return
	}

	api.logger.Info("Artifact retrieved successfully", "id", artifactID, "name", artifact.Name)
	api.writeJSONResponse(w, http.StatusOK, artifact)
}

// handleGetArtifactBinaryByID handles GET /artifacts/{id}/binary for binary download
func (api *HTTPAPI) handleGetArtifactBinaryByID(w http.ResponseWriter, r *http.Request, artifactID string) {
	api.logger.Info("Retrieving artifact binary", "id", artifactID)

	binaryData, err := api.store.GetArtifactBinary(artifactID)
	if err != nil {
		api.logger.Error("Failed to retrieve artifact binary", "id", artifactID, "error", err)
		api.writeErrorResponse(w, http.StatusNotFound, "Artifact binary not found")
		return
	}

	// Set appropriate headers for binary download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.zst\"", artifactID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryData)))

	// Write binary data
	if _, err := w.Write(binaryData); err != nil {
		api.logger.Error("Failed to write binary data", "id", artifactID, "error", err)
		return
	}

	api.logger.Info("Artifact binary retrieved successfully", "id", artifactID, "size", len(binaryData))
}

// handleCreateArtifact handles artifact creation requests
func (api *HTTPAPI) handleCreateArtifact(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Creating artifact", "method", r.Method, "url", r.URL.Path)

	var req model.CreateArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.logger.Error("Failed to decode request body", "error", err)
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate required fields
	if req.Name == "" || req.Version == "" || req.Type == "" || req.Data == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: name, version, type, data")
		return
	}

	// Store artifact
	artifact, err := api.store.StoreArtifact(&req)
	if err != nil {
		api.logger.Error("Failed to store artifact", "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to store artifact")
		return
	}

	api.logger.Info("Artifact created successfully", "id", artifact.ID, "name", artifact.Name)
	api.writeJSONResponse(w, http.StatusCreated, artifact)
}

// handleGetArtifact handles artifact metadata retrieval requests
func (api *HTTPAPI) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/artifacts/")
	if id == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID")
		return
	}

	api.logger.Info("Retrieving artifact metadata", "id", id)

	artifact, err := api.store.GetArtifact(id)
	if err != nil {
		api.logger.Error("Failed to retrieve artifact", "id", id, "error", err)
		api.writeErrorResponse(w, http.StatusNotFound, "Artifact not found")
		return
	}

	api.writeJSONResponse(w, http.StatusOK, artifact)
}

// handleGetArtifactBinary handles artifact binary retrieval requests
func (api *HTTPAPI) handleGetArtifactBinary(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	id := strings.TrimPrefix(path, "/artifacts/binary/")
	if id == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID")
		return
	}

	api.logger.Info("Retrieving artifact binary", "id", id)

	binaryData, err := api.store.GetArtifactBinary(id)
	if err != nil {
		api.logger.Error("Failed to retrieve artifact binary", "id", id, "error", err)
		api.writeErrorResponse(w, http.StatusNotFound, "Artifact binary not found")
		return
	}

	// Set appropriate headers for binary download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.tar.zst\"", id))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryData)))

	_, err = w.Write(binaryData)
	if err != nil {
		api.logger.Error("Failed to write binary data", "id", id, "error", err)
	}
}

// handleGetArtifactsForHost handles requests for artifacts associated with a host
func (api *HTTPAPI) handleGetArtifactsForHost(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	hostID := strings.TrimPrefix(path, "/artifacts/for-host/")
	if hostID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing host ID")
		return
	}

	api.logger.Info("Retrieving artifacts for host", "host_id", hostID)

	artifacts, err := api.store.GetArtifactsForHost(hostID)
	if err != nil {
		api.logger.Error("Failed to retrieve artifacts for host", "host_id", hostID, "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve artifacts")
		return
	}

	response := model.ArtifactListResponse{
		Artifacts: make([]model.Artifact, len(artifacts)),
		Total:     len(artifacts),
	}

	for i, artifact := range artifacts {
		response.Artifacts[i] = *artifact
	}

	api.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (api *HTTPAPI) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		api.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// handleListArtifacts handles GET /artifacts for listing all artifacts
func (api *HTTPAPI) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	api.logger.Info("Listing artifacts")

	artifacts, err := api.store.ListArtifacts()
	if err != nil {
		api.logger.Error("Failed to list artifacts", "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list artifacts")
		return
	}

	response := model.ArtifactListResponse{
		Artifacts: make([]model.Artifact, len(artifacts)),
		Total:     len(artifacts),
	}

	for i, artifact := range artifacts {
		response.Artifacts[i] = *artifact
	}

	api.logger.Info("Artifacts listed successfully", "count", len(artifacts))
	api.writeJSONResponse(w, http.StatusOK, response)
}

// writeErrorResponse writes an error response
func (api *HTTPAPI) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	errorResponse := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now(),
	}

	api.writeJSONResponse(w, statusCode, errorResponse)
}

// handleAssignArtifact handles POST /assign/{artifact_id}/{host_id}
func (api *HTTPAPI) handleAssignArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract artifact ID and host ID from path
	path := strings.TrimPrefix(r.URL.Path, "/assign/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid path format. Expected: /assign/{artifact_id}/{host_id}")
		return
	}

	artifactID := parts[0]
	hostID := parts[1]

	if artifactID == "" || hostID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID or host ID")
		return
	}

	api.logger.Info("Assigning artifact to host", "artifact_id", artifactID, "host_id", hostID)

	err := api.store.AssignArtifactToHost(artifactID, hostID)
	if err != nil {
		api.logger.Error("Failed to assign artifact to host", "artifact_id", artifactID, "host_id", hostID, "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to assign artifact to host")
		return
	}

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Artifact assigned to host successfully",
		"artifact_id": artifactID,
		"host_id":     hostID,
		"timestamp":   time.Now(),
	}

	api.logger.Info("Artifact assigned successfully", "artifact_id", artifactID, "host_id", hostID)
	api.writeJSONResponse(w, http.StatusOK, response)
}

// handleUnassignArtifact handles DELETE /unassign/{artifact_id}/{host_id}
func (api *HTTPAPI) handleUnassignArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract artifact ID and host ID from path
	path := strings.TrimPrefix(r.URL.Path, "/unassign/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid path format. Expected: /unassign/{artifact_id}/{host_id}")
		return
	}

	artifactID := parts[0]
	hostID := parts[1]

	if artifactID == "" || hostID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID or host ID")
		return
	}

	api.logger.Info("Unassigning artifact from host", "artifact_id", artifactID, "host_id", hostID)

	err := api.store.UnassignArtifactFromHost(artifactID, hostID)
	if err != nil {
		api.logger.Error("Failed to unassign artifact from host", "artifact_id", artifactID, "host_id", hostID, "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to unassign artifact from host")
		return
	}

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Artifact unassigned from host successfully",
		"artifact_id": artifactID,
		"host_id":     hostID,
		"timestamp":   time.Now(),
	}

	api.logger.Info("Artifact unassigned successfully", "artifact_id", artifactID, "host_id", hostID)
	api.writeJSONResponse(w, http.StatusOK, response)
}

// handleUpdateArtifactHosts handles PUT /hosts/{artifact_id}
func (api *HTTPAPI) handleUpdateArtifactHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		api.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract artifact ID from path
	artifactID := strings.TrimPrefix(r.URL.Path, "/hosts/")
	if artifactID == "" {
		api.writeErrorResponse(w, http.StatusBadRequest, "Missing artifact ID")
		return
	}

	// Parse request body
	var req struct {
		Hosts []string `json:"hosts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.logger.Error("Failed to decode request body", "error", err)
		api.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	api.logger.Info("Updating artifact hosts", "artifact_id", artifactID, "hosts", req.Hosts)

	err := api.store.UpdateArtifactHosts(artifactID, req.Hosts)
	if err != nil {
		api.logger.Error("Failed to update artifact hosts", "artifact_id", artifactID, "error", err)
		api.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update artifact hosts")
		return
	}

	response := map[string]interface{}{
		"status":      "success",
		"message":     "Artifact hosts updated successfully",
		"artifact_id": artifactID,
		"hosts":       req.Hosts,
		"timestamp":   time.Now(),
	}

	api.logger.Info("Artifact hosts updated successfully", "artifact_id", artifactID, "host_count", len(req.Hosts))
	api.writeJSONResponse(w, http.StatusOK, response)
}
