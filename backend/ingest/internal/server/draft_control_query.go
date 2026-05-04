package server

import "net/http"

type draftControlQueryResponse struct {
	OK       bool                 `json:"ok"`
	Count    int                  `json:"count"`
	Controls []draftControlRecord `json:"draft_controls"`
}

func (s *IngestServer) handleVisibilityDraftControls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	limit, err := parseQueryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pid, hasPID, err := parseOptionalInt(r.URL.Query().Get("pid"), "pid")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	investigation, err := s.collectInvestigation(
		r,
		deviceID,
		r.URL.Query().Get("agent_id"),
		r.URL.Query().Get("process_guid"),
		pid,
		hasPID,
		limit,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, draftControlQueryResponse{
		OK:       true,
		Count:    len(investigation.Drafts),
		Controls: investigation.Drafts,
	})
}
