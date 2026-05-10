package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerPlatformRoutes() {
	px := func(p string, h http.HandlerFunc) { s.mux.HandleFunc(p, h) }
	px("/platform/ai/providers", s.handlePlatformAIProviders)
	px("/platform/ai/providers/configure", s.handlePlatformAIProvidersConfigure)
	px("/platform/ai/providers/summary", s.handlePlatformAIProvidersSummary)
	px("/platform/ai/providers/", s.handlePlatformAIProviderSubpath)

	px("/platform/ai/privacy", s.handlePlatformAIPrivacy)
	px("/platform/ai/runs", s.handlePlatformAIRuns)
	px("/platform/ai/endpoint-evidence-analyst", s.handleEndpointEvidenceAnalyst)

	px("/platform/operational-events", s.handlePlatformOperationalEvents)

	px("/platform/draft-controls", s.handleDraftControlsCollection)
	px("/platform/draft-controls/", s.handleDraftControlsItem)

	px("/platform/research-feed", s.handleResearchFeedCollection)
	px("/platform/research-feed/", s.handleResearchFeedItem)

	px("/platform/detection-candidates", s.handleDetectionCandidatesCollection)
	px("/platform/detection-candidates/", s.handleDetectionCandidateItem)

	px("/platform/integration/devices/", s.handleIntegrationDeviceEvidence)
	px("/platform/redact/preview", s.handleRedactPreview)
}

func jsonWrite(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonRead(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return false
	}
	return true
}
