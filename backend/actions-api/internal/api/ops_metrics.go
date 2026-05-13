package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var heartbeatAccepted atomic.Uint64

func incHeartbeatAccepted() {
	heartbeatAccepted.Add(1)
}

func actionsHeartbeatTotal() uint64 {
	return heartbeatAccepted.Load()
}

// handleOpsMetrics exposes a minimal Prometheus-style counter for lab observability (WO-OPS-005).
func (s *Server) handleOpsMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = fmt.Fprintf(w, "# HELP actions_api_heartbeat_accepted_total Accepted POST /agents/heartbeat requests\n# TYPE actions_api_heartbeat_accepted_total counter\nactions_api_heartbeat_accepted_total %d\n", actionsHeartbeatTotal())
}
