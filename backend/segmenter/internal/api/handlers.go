package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"aegisflux/backend/segmenter/internal/segmenter"
	"aegisflux/backend/segmenter/internal/types"
)

// Handler handles HTTP requests for the segmenter service
type Handler struct {
	segmenter *segmenter.Segmenter
	logger    *slog.Logger
}

// NewHandler creates a new handler instance
func NewHandler(segmenter *segmenter.Segmenter, logger *slog.Logger) *Handler {
	return &Handler{
		segmenter: segmenter,
		logger:    logger,
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "segmenter",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ProposeSegmentation handles segmentation proposal requests
func (h *Handler) ProposeSegmentation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.SegmentationProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Hosts) == 0 {
		http.Error(w, "At least one host is required", http.StatusBadRequest)
		return
	}

	if len(req.Goals) == 0 {
		http.Error(w, "At least one segmentation goal is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("Processing segmentation proposal request",
		"host_count", len(req.Hosts),
		"traffic_flows", len(req.TrafficData),
		"goals", req.Goals)

	// Process the request
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := h.segmenter.ProposeSegmentation(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to generate segmentation proposals", "error", err)
		http.Error(w, "Failed to generate proposals", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Generated segmentation proposals",
		"proposal_id", response.ProposalID,
		"proposal_count", len(response.Proposals))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreateSegmentationPlan handles segmentation plan creation requests
func (h *Handler) CreateSegmentationPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.SegmentationPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ProposalID == "" {
		http.Error(w, "Proposal ID is required", http.StatusBadRequest)
		return
	}

	if req.Proposal == nil {
		http.Error(w, "Proposal is required", http.StatusBadRequest)
		return
	}

	h.logger.Info("Creating segmentation plan",
		"proposal_id", req.ProposalID,
		"implementation_mode", req.ImplementationMode)

	// Process the request
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := h.segmenter.CreateSegmentationPlan(ctx, &req)
	if err != nil {
		h.logger.Error("Failed to create segmentation plan", "error", err)
		http.Error(w, "Failed to create plan", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Created segmentation plan",
		"plan_id", response.PlanID,
		"step_count", len(response.Steps))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetSegmentationStrategies returns available segmentation strategies
func (h *Handler) GetSegmentationStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	strategies := []map[string]interface{}{
		{
			"id":          string(types.StrategyMicrosegmentation),
			"name":        "Microsegmentation",
			"description": "Fine-grained segmentation based on host characteristics and traffic patterns",
			"complexity":  "high",
			"security":    "high",
		},
		{
			"id":          string(types.StrategyZeroTrust),
			"name":        "Zero Trust",
			"description": "Maximum isolation with individual segments for each host",
			"complexity":  "very_high",
			"security":    "very_high",
		},
		{
			"id":          string(types.StrategyTraditional),
			"name":        "Traditional",
			"description": "Simple segmentation based on network zones (DMZ, internal, etc.)",
			"complexity":  "low",
			"security":    "medium",
		},
		{
			"id":          string(types.StrategyHybrid),
			"name":        "Hybrid",
			"description": "Combination of multiple strategies based on requirements",
			"complexity":  "medium",
			"security":    "high",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"strategies": strategies,
	})
}

// GetSegmentationGoals returns available segmentation goals
func (h *Handler) GetSegmentationGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	goals := []map[string]interface{}{
		{
			"id":          string(types.GoalReduceLateralMovement),
			"name":        "Reduce Lateral Movement",
			"description": "Prevent attackers from moving laterally across the network",
			"priority":    "high",
		},
		{
			"id":          string(types.GoalCompliance),
			"name":        "Compliance",
			"description": "Meet regulatory compliance requirements",
			"priority":    "medium",
		},
		{
			"id":          string(types.GoalPerformance),
			"name":        "Performance",
			"description": "Optimize network performance while maintaining security",
			"priority":    "medium",
		},
		{
			"id":          string(types.GoalCost),
			"name":        "Cost Optimization",
			"description": "Minimize implementation and operational costs",
			"priority":    "low",
		},
		{
			"id":          string(types.GoalSecurity),
			"name":        "Enhanced Security",
			"description": "Maximize overall network security posture",
			"priority":    "high",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"goals": goals,
	})
}
