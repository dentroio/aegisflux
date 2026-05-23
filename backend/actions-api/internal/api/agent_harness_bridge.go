package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/actionsapi/internal/agentsharness"
)

// harnessBridge implements agentsharness.DataSource using platform state.
type harnessBridge struct {
	s *Server
}

func (b *harnessBridge) AllowExternalAI() bool {
	b.s.platform.mu.Lock()
	defer b.s.platform.mu.Unlock()
	return b.s.platform.Privacy.AllowExternalAI
}

func (b *harnessBridge) DefaultProviderKind() string {
	b.s.platform.mu.Lock()
	defer b.s.platform.mu.Unlock()
	return b.s.platform.DefaultProvider
}

func (b *harnessBridge) RedactJSONPreview(v any) string {
	buf, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	b.s.platform.mu.Lock()
	priv := b.s.platform.Privacy
	b.s.platform.mu.Unlock()
	return RedactLine(string(buf), priv)
}

func (b *harnessBridge) DeviceEvidenceSummary(deviceID string) map[string]any {
	return labDeviceEvidenceSummaryMap(deviceID)
}

func (b *harnessBridge) FindingsEvidencePaths(deviceID, findingID string) []map[string]string {
	return labFindingsEvidencePaths(deviceID, findingID)
}

func (b *harnessBridge) DetectionCandidateLookup(candidateID, deviceID string) map[string]any {
	b.s.platform.mu.Lock()
	defer b.s.platform.mu.Unlock()
	if candidateID != "" {
		for i := range b.s.platform.Candidates {
			if b.s.platform.Candidates[i].ID == candidateID {
				c := b.s.platform.Candidates[i]
				return map[string]any{
					"found":        true,
					"candidate_id": c.ID,
					"title":        c.Title,
					"status":       c.Status,
					"category":     c.Category,
				}
			}
		}
		return map[string]any{"found": false, "candidate_id": candidateID}
	}
	out := make([]map[string]any, 0, 5)
	for i := len(b.s.platform.Candidates) - 1; i >= 0 && len(out) < 5; i-- {
		c := b.s.platform.Candidates[i]
		out = append(out, map[string]any{"candidate_id": c.ID, "title": c.Title, "status": c.Status})
	}
	return map[string]any{"found": len(out) > 0, "recent": out, "device_id": deviceID}
}

func labDeviceEvidenceSummaryMap(deviceID string) map[string]any {
	sample := fmt.Sprintf(`{"windows":{"device_id":"win-lab-host","signals":["dns","collector"]},"linux":{"device_id":"lin-lab-host","signals":["flows","collector"]}}`)
	return map[string]any{
		"schema":                    "aegisflux.integration.evidence_summary.v1",
		"device_id":                 deviceID,
		"agent_id":                  deviceID,
		"os_or_source":              "visibility",
		"freshness_ms_hint":         time.Now().UnixMilli(),
		"ai_activity_summary":       map[string]any{"signals": []string{"dns", "process", "findings"}, "count_hint": simulateMatches(deviceID)},
		"inventory_summary":         map[string]any{"hints": []string{"browser_extensions", "sase"}},
		"finding_links":             labFindingsEvidencePaths(deviceID, ""),
		"integration_event_names":   []string{"aegis.device.observed", "aegis.ai_activity.summarized", "aegis.inventory.item_observed", "aegis.finding.created"},
		"sample_payload_windows_linux_json": sample,
	}
}

func labFindingsEvidencePaths(deviceID, findingID string) []map[string]string {
	out := []map[string]string{
		{"finding_id": "example-" + deviceID, "relative": fmt.Sprintf("/analyze/findings?device=%s", deviceID)},
	}
	if findingID != "" {
		out = append(out, map[string]string{
			"finding_id": findingID,
			"relative":   fmt.Sprintf("/analyze/findings?device=%s&finding=%s", deviceID, findingID),
		})
	}
	return out
}

func appendAgentHarnessJob(p *PlatformData, job *agentsharness.JobRecord) {
	p.AgentHarnessJobs = append(p.AgentHarnessJobs, *job)
	if len(p.AgentHarnessJobs) > 400 {
		p.AgentHarnessJobs = p.AgentHarnessJobs[len(p.AgentHarnessJobs)-400:]
	}
}

func appendAgentHarnessRun(p *PlatformData, run *agentsharness.RunRecord) {
	p.AgentHarnessRuns = append(p.AgentHarnessRuns, *run)
	if len(p.AgentHarnessRuns) > 400 {
		p.AgentHarnessRuns = p.AgentHarnessRuns[len(p.AgentHarnessRuns)-400:]
	}
}

func (s *Server) runAgentHarness(spec agentsharness.RunSpec) (*agentsharness.JobRecord, *agentsharness.RunRecord, error) {
	h := &agentsharness.Harness{Data: &harnessBridge{s: s}}
	job, run, err := h.Run(context.Background(), spec)
	if err != nil {
		return nil, nil, err
	}
	s.platform.mu.Lock()
	appendAgentHarnessJob(s.platform, job)
	appendAgentHarnessRun(s.platform, run)
	s.platform.mu.Unlock()
	return job, run, nil
}
