package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sgerhart/aegisflux/backend/internal/db"
	"github.com/sgerhart/aegisflux/backend/services/registry/signing"
)

// BundleHandler handles bundle-related HTTP requests
type BundleHandler struct {
	store  *db.Store
	signer *signing.Signer
}

// NewBundleHandler creates a new bundle handler
func NewBundleHandler(store *db.Store, signer *signing.Signer) *BundleHandler {
	return &BundleHandler{
		store:  store,
		signer: signer,
	}
}

// CreateBundleRequest represents a request to create a new bundle
type CreateBundleRequest struct {
	Name        string          `json:"name"`
	Content     string          `json:"content"`     // Base64 encoded bundle content
	Description string          `json:"description,omitempty"`
	Version     string          `json:"version,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedBy   string          `json:"created_by"`
}

// CreateBundleResponse represents the response after creating a bundle
type CreateBundleResponse struct {
	BundleID   uuid.UUID       `json:"bundle_id"`
	Hash       string          `json:"hash"`
	Signature  string          `json:"signature"`
	KeyID      string          `json:"kid"`
	Algorithm  string          `json:"algorithm"`
	CreatedAt  time.Time       `json:"created_at"`
	Metadata   json.RawMessage `json:"metadata"`
}

// BundleResponse represents a bundle in API responses
type BundleResponse struct {
	BundleID   uuid.UUID       `json:"bundle_id"`
	Name       string          `json:"name"`
	Hash       string          `json:"hash"`
	Signature  string          `json:"signature"`
	Algorithm  string          `json:"algorithm"`
	KeyID      string          `json:"kid"`
	Metadata   json.RawMessage `json:"metadata"`
	CreatedAt  time.Time       `json:"created_at"`
	CreatedBy  string          `json:"created_by"`
}

// ListBundlesResponse represents the response for listing bundles
type ListBundlesResponse struct {
	Bundles []BundleResponse `json:"bundles"`
	Total   int              `json:"total"`
	Limit   int              `json:"limit"`
	Offset  int              `json:"offset"`
}

// CreateBundle handles POST /bundles
func (h *BundleHandler) CreateBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Bundle name is required", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, "Bundle content is required", http.StatusBadRequest)
		return
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "unknown"
	}

	// Decode base64 content
	content, err := decodeBase64Content(req.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid base64 content: %v", err), http.StatusBadRequest)
		return
	}

	// Calculate hash of the content
	hash := calculateContentHash(content)

	// Check if bundle with this hash already exists
	existingBundle, err := h.store.GetBundleByHash(r.Context(), hash)
	if err == nil && existingBundle != nil {
		// Bundle already exists, return existing bundle info
		response := CreateBundleResponse{
			BundleID:  existingBundle.BundleID,
			Hash:      existingBundle.Hash,
			Signature: existingBundle.Sig,
			KeyID:     existingBundle.Kid,
			Algorithm: existingBundle.Algo,
			CreatedAt: existingBundle.CreatedAt,
			Metadata:  existingBundle.Meta,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create metadata
	metadata := map[string]interface{}{
		"description": req.Description,
		"version":     req.Version,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	
	// Merge with provided metadata
	if req.Metadata != nil {
		var providedMeta map[string]interface{}
		if err := json.Unmarshal(req.Metadata, &providedMeta); err == nil {
			for k, v := range providedMeta {
				metadata[k] = v
			}
		}
	}
	
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create metadata: %v", err), http.StatusInternalServerError)
		return
	}

	// Sign the content
	signature, kid, err := h.signer.Sign(content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign bundle: %v", err), http.StatusInternalServerError)
		return
	}

	// Create bundle in database
	bundleID := uuid.New()
	bundle := &db.Bundle{
		BundleID:  bundleID,
		Name:      req.Name,
		Hash:      hash,
		Sig:       signature,
		Algo:      "Ed25519",
		Kid:       kid,
		Meta:      metadataBytes,
		CreatedBy: req.CreatedBy,
	}

	if err := h.store.CreateBundle(r.Context(), bundle); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create bundle: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit event
	_, err = h.store.LogAuditEvent(r.Context(), req.CreatedBy, "bundle_create", bundleID.String(), metadataBytes)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to log audit event: %v\n", err)
	}

	// Create response
	response := CreateBundleResponse{
		BundleID:  bundleID,
		Hash:      hash,
		Signature: signature,
		KeyID:     kid,
		Algorithm: "Ed25519",
		CreatedAt: bundle.CreatedAt,
		Metadata:  metadataBytes,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetBundle handles GET /bundles/{id}
func (h *BundleHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract bundle ID from URL path
	bundleIDStr := r.URL.Path[len("/bundles/"):]
	bundleID, err := uuid.Parse(bundleIDStr)
	if err != nil {
		http.Error(w, "Invalid bundle ID", http.StatusBadRequest)
		return
	}

	// Get bundle from database
	bundle, err := h.store.GetBundle(r.Context(), bundleID)
	if err != nil {
		http.Error(w, "Bundle not found", http.StatusNotFound)
		return
	}

	// Create response
	response := BundleResponse{
		BundleID:  bundle.BundleID,
		Name:      bundle.Name,
		Hash:      bundle.Hash,
		Signature: bundle.Sig,
		Algorithm: bundle.Algo,
		KeyID:     bundle.Kid,
		Metadata:  bundle.Meta,
		CreatedAt: bundle.CreatedAt,
		CreatedBy: bundle.CreatedBy,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListBundles handles GET /bundles
func (h *BundleHandler) ListBundles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	limit := 50  // default limit
	offset := 0  // default offset

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := parseIntParam(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := parseIntParam(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get bundles from database
	bundles, err := h.store.ListBundles(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list bundles: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	responseBundles := make([]BundleResponse, len(bundles))
	for i, bundle := range bundles {
		responseBundles[i] = BundleResponse{
			BundleID:  bundle.BundleID,
			Name:      bundle.Name,
			Hash:      bundle.Hash,
			Signature: bundle.Sig,
			Algorithm: bundle.Algo,
			KeyID:     bundle.Kid,
			Metadata:  bundle.Meta,
			CreatedAt: bundle.CreatedAt,
			CreatedBy: bundle.CreatedBy,
		}
	}

	response := ListBundlesResponse{
		Bundles: responseBundles,
		Total:   len(responseBundles),
		Limit:   limit,
		Offset:  offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// VerifyBundle handles POST /bundles/{id}/verify
func (h *BundleHandler) VerifyBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract bundle ID from URL path
	bundleIDStr := r.URL.Path[len("/bundles/"):]
	if len(bundleIDStr) > 0 && bundleIDStr[len(bundleIDStr)-len("/verify"):] == "/verify" {
		bundleIDStr = bundleIDStr[:len(bundleIDStr)-len("/verify")]
	}
	
	bundleID, err := uuid.Parse(bundleIDStr)
	if err != nil {
		http.Error(w, "Invalid bundle ID", http.StatusBadRequest)
		return
	}

	// Get bundle from database
	bundle, err := h.store.GetBundle(r.Context(), bundleID)
	if err != nil {
		http.Error(w, "Bundle not found", http.StatusNotFound)
		return
	}

	// Get bundle content from request
	var verifyReq struct {
		Content string `json:"content"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&verifyReq); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if verifyReq.Content == "" {
		http.Error(w, "Bundle content is required for verification", http.StatusBadRequest)
		return
	}

	// Decode and hash the provided content
	content, err := decodeBase64Content(verifyReq.Content)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid base64 content: %v", err), http.StatusBadRequest)
		return
	}

	providedHash := calculateContentHash(content)

	// Verify hash matches
	if providedHash != bundle.Hash {
		http.Error(w, "Bundle content hash mismatch", http.StatusBadRequest)
		return
	}

	// Verify signature
	err = h.signer.Verify(content, bundle.Sig, bundle.Kid)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bundle signature verification failed: %v", err), http.StatusBadRequest)
		return
	}

	// Return verification result
	response := map[string]interface{}{
		"verified":  true,
		"bundle_id": bundleID,
		"hash":      bundle.Hash,
		"kid":       bundle.Kid,
		"message":   "Bundle signature verified successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func decodeBase64Content(content string) ([]byte, error) {
	// Try standard base64 first
	if data, err := decodeBase64(content); err == nil {
		return data, nil
	}
	
	// Try URL-safe base64
	return decodeBase64URL(content)
}

func decodeBase64(s string) ([]byte, error) {
	import "encoding/base64"
	return base64.StdEncoding.DecodeString(s)
}

func decodeBase64URL(s string) ([]byte, error) {
	import "encoding/base64"
	return base64.URLEncoding.DecodeString(s)
}

func calculateContentHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}

func parseIntParam(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

