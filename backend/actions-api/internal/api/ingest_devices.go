package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SummaryDependencyStatus reports a single upstream dependency for console summaries (WO-OPS-003).
type SummaryDependencyStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok, degraded, unavailable
	Detail string `json:"detail,omitempty"`
}

// FetchIngestVisibilityDevices loads visibility device rows from ingest for merging with agents.
// The returned status always describes the ingest devices call (not per-device health).
func FetchIngestVisibilityDevices(ingestURL string, client *http.Client) (map[string]*VisibilityDeviceRecord, SummaryDependencyStatus) {
	probe := SummaryDependencyStatus{Name: "ingest", Status: "unavailable"}
	if client == nil {
		client = http.DefaultClient
	}
	base := strings.TrimRight(strings.TrimSpace(ingestURL), "/")
	if base == "" {
		probe.Detail = "INGEST_API_URL is empty"
		return nil, probe
	}

	byHost := map[string]*VisibilityDeviceRecord{}
	url := base + "/v1/visibility/devices?limit=120"
	resp, err := client.Get(url)
	if err != nil {
		probe.Detail = err.Error()
		return byHost, probe
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		probe.Status = "degraded"
		probe.Detail = fmt.Sprintf("ingest returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		return byHost, probe
	}

	var payload struct {
		Devices []VisibilityDeviceRecord `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		probe.Status = "degraded"
		probe.Detail = "decode ingest devices: " + err.Error()
		return byHost, probe
	}
	for i := range payload.Devices {
		d := &payload.Devices[i]
		byHost[d.DeviceID] = d
		if d.AgentID != "" {
			byHost[d.AgentID] = d
		}
	}
	probe.Status = "ok"
	return byHost, probe
}

// summaryHTTPClient returns the HTTP client used for ingest and dependency probes from actions-api.
func summaryHTTPClient() *http.Client {
	return &http.Client{Timeout: 12 * time.Second}
}
