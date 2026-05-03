package server

import (
	"net/http"
)

type visibilityDevicesResponse struct {
	OK      bool                     `json:"ok"`
	Count   int                      `json:"count"`
	Devices []visibilityDeviceRecord `json:"devices"`
}

func (s *IngestServer) handleVisibilityDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		http.Error(w, "visibility store is not configured", http.StatusServiceUnavailable)
		return
	}

	limit, err := parseQueryLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	devices, err := s.store.ListDevices(r.Context(), visibilityDeviceFilter{
		TenantID: r.URL.Query().Get("tenant_id"),
		Limit:    limit,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, http.StatusOK, visibilityDevicesResponse{
		OK:      true,
		Count:   len(devices),
		Devices: devices,
	})
}
