package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// handleServiceReadyz reports process readiness and critical lab dependencies (WO-OPS-003).
// Returns HTTP 503 when NATS is required but unavailable. Ingest or detection-pipeline problems
// yield HTTP 200 with state "degraded" so operators can still inspect cached agent rows.
func (s *Server) handleServiceReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checks := map[string]string{}
	state := "ready"

	if s.nc == nil || !s.nc.IsConnected() {
		checks["nats"] = "unavailable"
		state = "unhealthy"
	} else {
		checks["nats"] = "ok"
	}

	ingestURL := strings.TrimSpace(os.Getenv("INGEST_API_URL"))
	if ingestURL == "" {
		ingestURL = "http://localhost:9091"
	}
	{
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(ingestURL, "/")+"/healthz", nil)
		if err != nil {
			checks["ingest"] = "unavailable: " + err.Error()
			state = degradedIfNotUnhealthy(state)
		} else {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				checks["ingest"] = "unavailable: " + err.Error()
				state = degradedIfNotUnhealthy(state)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					checks["ingest"] = fmt.Sprintf("http_%d", resp.StatusCode)
					state = degradedIfNotUnhealthy(state)
				} else {
					checks["ingest"] = "ok"
				}
			}
		}
	}

	base := strings.TrimRight(strings.TrimSpace(os.Getenv("DETECTION_PIPELINE_URL")), "/")
	if base == "" {
		base = "http://localhost:8089"
	}
	{
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
		if err != nil {
			checks["detection_pipeline"] = "unavailable: " + err.Error()
			state = degradedIfNotUnhealthy(state)
		} else {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				checks["detection_pipeline"] = "unavailable: " + err.Error()
				state = degradedIfNotUnhealthy(state)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					checks["detection_pipeline"] = fmt.Sprintf("http_%d", resp.StatusCode)
					state = degradedIfNotUnhealthy(state)
				} else {
					checks["detection_pipeline"] = "ok"
				}
			}
		}
	}

	out := map[string]any{
		"service":           "actions-api",
		"state":             state,
		"checks":            checks,
		"uptime_hint":       "process",
		"heartbeat_total": actionsHeartbeatTotal(),
	}
	w.Header().Set("Content-Type", "application/json")
	if state == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(out)
}

func degradedIfNotUnhealthy(current string) string {
	if current == "unhealthy" {
		return current
	}
	return "degraded"
}
