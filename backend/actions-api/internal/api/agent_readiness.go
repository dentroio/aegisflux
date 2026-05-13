package api

// Agent readiness scoring (WO-GROWTH-004).
//
// Readiness answers a single operator question: "Can I trust evidence and
// decisions for this agent right now?" Scoring is a composition of small,
// explainable dimensions so the UI can surface "what to fix first" rather
// than a single opaque grade.
//
// Buckets:
//   ready           - all critical dimensions healthy
//   needs_attention - at least one warn-state dimension
//   stale           - heartbeat too old (no agent data trusted)
//   degraded        - heartbeat ok but multiple problems
//   unknown         - missing data, treat with caution
//
// Posture is observe-only: no enforcement claim is implied by readiness.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	readinessBucketReady          = "ready"
	readinessBucketNeedsAttention = "needs_attention"
	readinessBucketStale          = "stale"
	readinessBucketDegraded       = "degraded"
	readinessBucketUnknown        = "unknown"

	readinessStateGood    = "good"
	readinessStateWarn    = "warn"
	readinessStateBad     = "bad"
	readinessStateUnknown = "unknown"
)

// AgentReadiness is the per-agent readiness object.
type AgentReadiness struct {
	Bucket     string                    `json:"bucket"`
	Score      int                       `json:"score"`
	Summary    string                    `json:"summary"`
	FixFirst   string                    `json:"fix_first,omitempty"`
	Dimensions []AgentReadinessDimension `json:"dimensions"`
}

// AgentReadinessDimension explains a single signal and its state.
type AgentReadinessDimension struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	State  string `json:"state"`
	Value  string `json:"value,omitempty"`
	Detail string `json:"detail,omitempty"`
	Weight int    `json:"weight,omitempty"`
}

// FleetReadinessSummary is the response from the fleet endpoint.
type FleetReadinessSummary struct {
	OK            bool                       `json:"ok"`
	GeneratedAtMS int64                      `json:"generated_at_ms"`
	Total         int                        `json:"total"`
	Buckets       map[string]int             `json:"buckets"`
	AverageScore  int                        `json:"average_score"`
	Agents        []FleetReadinessAgent      `json:"agents"`
	FixFirst      []FleetReadinessHotspot    `json:"fix_first"`
}

type FleetReadinessAgent struct {
	AgentUID string         `json:"agent_uid"`
	HostID   string         `json:"host_id"`
	Hostname string         `json:"hostname,omitempty"`
	Status   string         `json:"status"`
	LastSeen time.Time      `json:"last_seen"`
	Readiness AgentReadiness `json:"readiness"`
}

type FleetReadinessHotspot struct {
	AgentUID string `json:"agent_uid"`
	HostID   string `json:"host_id"`
	Reason   string `json:"reason"`
	Bucket   string `json:"bucket"`
}

// computeAgentReadiness builds a readiness object for an agent. Inputs come
// from the existing AgentInfo merged record so callers do not need to fetch
// extra data.
func computeAgentReadiness(agent AgentInfo, currentAgentVersion string, now time.Time) AgentReadiness {
	dims := []AgentReadinessDimension{
		dimensionHeartbeat(agent, now),
		dimensionEventIngestion(agent, now),
		dimensionDetectionPack(agent, now),
		dimensionAgentVersion(agent, currentAgentVersion),
		dimensionCollectorHealth(agent),
		dimensionConnectivity(agent),
	}

	score, bucket := bucketize(dims, agent, now)
	summary := summarizeReadiness(bucket, agent)
	fixFirst := pickFixFirst(dims, bucket)

	return AgentReadiness{
		Bucket:     bucket,
		Score:      score,
		Dimensions: dims,
		Summary:    summary,
		FixFirst:   fixFirst,
	}
}

func dimensionHeartbeat(agent AgentInfo, now time.Time) AgentReadinessDimension {
	if agent.LastSeen.IsZero() {
		return AgentReadinessDimension{
			ID:     "heartbeat",
			Label:  "Heartbeat freshness",
			State:  readinessStateUnknown,
			Value:  "never reported",
			Detail: "Agent has no recorded heartbeat. It may not have completed registration.",
			Weight: 25,
		}
	}
	age := now.Sub(agent.LastSeen)
	value := humanizeDuration(age) + " ago"
	switch {
	case age < 5*time.Minute:
		return AgentReadinessDimension{ID: "heartbeat", Label: "Heartbeat freshness", State: readinessStateGood, Value: value, Detail: "Heartbeat is recent — agent is alive.", Weight: 25}
	case age < 30*time.Minute:
		return AgentReadinessDimension{ID: "heartbeat", Label: "Heartbeat freshness", State: readinessStateWarn, Value: value, Detail: "Heartbeat is delayed. Check the agent or network path before trusting decisions.", Weight: 25}
	default:
		return AgentReadinessDimension{ID: "heartbeat", Label: "Heartbeat freshness", State: readinessStateBad, Value: value, Detail: "Heartbeat is stale. The agent is likely offline; treat the device as untrusted for new evidence.", Weight: 25}
	}
}

func dimensionEventIngestion(agent AgentInfo, now time.Time) AgentReadinessDimension {
	if agent.Visibility == nil {
		return AgentReadinessDimension{
			ID:     "event_ingestion",
			Label:  "Event ingestion recency",
			State:  readinessStateUnknown,
			Value:  "no visibility record",
			Detail: "Ingest has no visibility record for this host. Confirm that the agent is sending telemetry.",
			Weight: 20,
		}
	}
	ts := time.UnixMilli(agent.Visibility.LastSeenMS)
	if agent.Visibility.LastSeenMS == 0 {
		return AgentReadinessDimension{ID: "event_ingestion", Label: "Event ingestion recency", State: readinessStateBad, Value: "no events", Detail: "Visibility record exists but no events have been ingested.", Weight: 20}
	}
	age := now.Sub(ts)
	value := humanizeDuration(age) + " ago"
	switch {
	case age < 10*time.Minute:
		return AgentReadinessDimension{ID: "event_ingestion", Label: "Event ingestion recency", State: readinessStateGood, Value: value, Detail: "Telemetry is fresh — evidence from this host is current.", Weight: 20}
	case age < 60*time.Minute:
		return AgentReadinessDimension{ID: "event_ingestion", Label: "Event ingestion recency", State: readinessStateWarn, Value: value, Detail: "Telemetry is slowing. Verify collectors are running.", Weight: 20}
	default:
		return AgentReadinessDimension{ID: "event_ingestion", Label: "Event ingestion recency", State: readinessStateBad, Value: value, Detail: "Telemetry has gone stale. Findings against this host should be treated as historical only.", Weight: 20}
	}
}

func dimensionDetectionPack(agent AgentInfo, now time.Time) AgentReadinessDimension {
	if agent.DetectionPackStatus == nil {
		return AgentReadinessDimension{
			ID:     "detection_pack",
			Label:  "Detection pack freshness",
			State:  readinessStateUnknown,
			Value:  "no rollout record",
			Detail: "No detection pack rollout is recorded for this agent yet.",
			Weight: 15,
		}
	}
	state := strings.ToLower(strings.TrimSpace(agent.DetectionPackStatus.RolloutState))
	switch state {
	case "applied", "active", "ok":
		// Check freshness too.
		if agent.DetectionPackStatus.LastAppliedAtMS == 0 {
			return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateWarn, Value: state, Detail: "Pack reported applied but no apply timestamp available.", Weight: 15}
		}
		age := now.Sub(time.UnixMilli(agent.DetectionPackStatus.LastAppliedAtMS))
		if age > 7*24*time.Hour {
			return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateWarn, Value: state, Detail: "Pack has not been refreshed in over a week. Check rollout pipeline.", Weight: 15}
		}
		return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateGood, Value: state, Detail: "Pack is applied and recent.", Weight: 15}
	case "rejected":
		return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateBad, Value: state, Detail: firstNonEmpty(agent.DetectionPackStatus.LastRejectedReason, "Pack was rejected — investigate signature/hash/schema."), Weight: 15}
	case "pending", "queued":
		return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateWarn, Value: state, Detail: "Pack rollout is in flight. Detection coverage may be incomplete.", Weight: 15}
	}
	return AgentReadinessDimension{ID: "detection_pack", Label: "Detection pack freshness", State: readinessStateWarn, Value: state, Detail: "Unrecognized rollout state. Confirm rollout pipeline health.", Weight: 15}
}

func dimensionAgentVersion(agent AgentInfo, currentVersion string) AgentReadinessDimension {
	current := strings.TrimSpace(currentVersion)
	have := strings.TrimSpace(agent.AgentVersion)
	if have == "" {
		return AgentReadinessDimension{ID: "agent_version", Label: "Agent version", State: readinessStateUnknown, Value: "unreported", Detail: "Agent did not report its version on heartbeat.", Weight: 10}
	}
	if current == "" {
		return AgentReadinessDimension{ID: "agent_version", Label: "Agent version", State: readinessStateGood, Value: have, Detail: "No fleet baseline configured — accepting reported version.", Weight: 10}
	}
	if have == current {
		return AgentReadinessDimension{ID: "agent_version", Label: "Agent version", State: readinessStateGood, Value: have, Detail: "Agent matches fleet baseline.", Weight: 10}
	}
	return AgentReadinessDimension{ID: "agent_version", Label: "Agent version", State: readinessStateWarn, Value: have, Detail: fmt.Sprintf("Agent drift detected: fleet baseline is %s.", current), Weight: 10}
}

func dimensionCollectorHealth(agent AgentInfo) AgentReadinessDimension {
	if agent.Visibility == nil {
		return AgentReadinessDimension{ID: "collectors", Label: "Collector coverage", State: readinessStateUnknown, Value: "no visibility record", Detail: "Cannot estimate collector coverage without ingest data.", Weight: 15}
	}
	count := agent.Visibility.EventCount
	types := len(agent.Visibility.EventTypeCount)
	switch {
	case count == 0:
		return AgentReadinessDimension{ID: "collectors", Label: "Collector coverage", State: readinessStateBad, Value: "no events", Detail: "Ingest sees this agent but has not received any events yet.", Weight: 15}
	case types <= 1:
		return AgentReadinessDimension{ID: "collectors", Label: "Collector coverage", State: readinessStateWarn, Value: fmt.Sprintf("%d type(s)", types), Detail: "Only a single event type is reported. Collector(s) likely partial.", Weight: 15}
	default:
		return AgentReadinessDimension{ID: "collectors", Label: "Collector coverage", State: readinessStateGood, Value: fmt.Sprintf("%d type(s)", types), Detail: fmt.Sprintf("%d events across %d event types.", count, types), Weight: 15}
	}
}

func dimensionConnectivity(agent AgentInfo) AgentReadinessDimension {
	switch strings.ToLower(strings.TrimSpace(agent.Status)) {
	case "online":
		return AgentReadinessDimension{ID: "connectivity", Label: "Tunnel/connectivity", State: readinessStateGood, Value: "online", Detail: "Agent is actively connected.", Weight: 15}
	case "stale":
		return AgentReadinessDimension{ID: "connectivity", Label: "Tunnel/connectivity", State: readinessStateWarn, Value: "stale", Detail: "Agent heartbeat is delayed. Check tunnel, agent process, and network path before trusting new evidence.", Weight: 15}
	case "offline":
		return AgentReadinessDimension{ID: "connectivity", Label: "Tunnel/connectivity", State: readinessStateBad, Value: "offline", Detail: "Agent is offline. Treat as untrusted for new evidence.", Weight: 15}
	}
	return AgentReadinessDimension{ID: "connectivity", Label: "Tunnel/connectivity", State: readinessStateUnknown, Value: "unknown", Detail: "Connection status is unknown.", Weight: 15}
}

func bucketize(dims []AgentReadinessDimension, agent AgentInfo, now time.Time) (int, string) {
	weightTotal := 0
	weightGood := 0
	bad := 0
	warn := 0
	unknown := 0
	staleHeartbeat := false
	for _, d := range dims {
		w := d.Weight
		if w == 0 {
			w = 10
		}
		weightTotal += w
		switch d.State {
		case readinessStateGood:
			weightGood += w
		case readinessStateWarn:
			warn++
			weightGood += w / 2
		case readinessStateBad:
			bad++
		case readinessStateUnknown:
			unknown++
		}
		if d.ID == "heartbeat" && d.State == readinessStateBad {
			staleHeartbeat = true
		}
	}
	score := 0
	if weightTotal > 0 {
		score = (weightGood * 100) / weightTotal
	}
	bucket := readinessBucketUnknown
	switch {
	case staleHeartbeat:
		bucket = readinessBucketStale
	case bad >= 2:
		bucket = readinessBucketDegraded
	case bad == 1 || warn >= 2:
		bucket = readinessBucketNeedsAttention
	case warn == 1:
		bucket = readinessBucketNeedsAttention
	case unknown == len(dims):
		bucket = readinessBucketUnknown
	default:
		bucket = readinessBucketReady
	}
	if bucket == readinessBucketReady && score < 80 {
		bucket = readinessBucketNeedsAttention
	}
	return score, bucket
}

func summarizeReadiness(bucket string, agent AgentInfo) string {
	host := firstNonEmpty(agent.Hostname, agent.HostID, agent.AgentUID)
	switch bucket {
	case readinessBucketReady:
		return fmt.Sprintf("%s is ready: heartbeat, telemetry, detection pack, and connectivity all green.", host)
	case readinessBucketNeedsAttention:
		return fmt.Sprintf("%s needs attention before relying on its evidence for new control decisions.", host)
	case readinessBucketStale:
		return fmt.Sprintf("%s is stale. Heartbeat is too old to trust live evidence from this endpoint.", host)
	case readinessBucketDegraded:
		return fmt.Sprintf("%s is degraded across multiple readiness dimensions.", host)
	default:
		return fmt.Sprintf("%s readiness is unknown — confirm registration and telemetry before relying on its data.", host)
	}
}

func pickFixFirst(dims []AgentReadinessDimension, bucket string) string {
	if bucket == readinessBucketReady {
		return ""
	}
	for _, d := range dims {
		if d.State == readinessStateBad {
			return fmt.Sprintf("%s: %s", d.Label, d.Detail)
		}
	}
	for _, d := range dims {
		if d.State == readinessStateWarn {
			return fmt.Sprintf("%s: %s", d.Label, d.Detail)
		}
	}
	for _, d := range dims {
		if d.State == readinessStateUnknown {
			return fmt.Sprintf("%s: %s", d.Label, d.Detail)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func humanizeDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// handleAgentReadinessFleet returns fleet-wide readiness.
func (s *Server) handleAgentReadinessFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	currentVersion := strings.TrimSpace(os.Getenv("AEGISFLUX_AGENT_BASELINE_VERSION"))

	agents := s.collectAgentInfos()
	now := time.Now()

	out := FleetReadinessSummary{
		OK:            true,
		GeneratedAtMS: now.UnixMilli(),
		Total:         len(agents),
		Buckets:       map[string]int{},
		Agents:        make([]FleetReadinessAgent, 0, len(agents)),
	}
	scoreSum := 0
	hotspotCandidates := []FleetReadinessHotspot{}
	for _, agent := range agents {
		readiness := computeAgentReadiness(agent, currentVersion, now)
		out.Buckets[readiness.Bucket]++
		scoreSum += readiness.Score
		out.Agents = append(out.Agents, FleetReadinessAgent{
			AgentUID:  agent.AgentUID,
			HostID:    agent.HostID,
			Hostname:  agent.Hostname,
			Status:    agent.Status,
			LastSeen:  agent.LastSeen,
			Readiness: readiness,
		})
		if readiness.Bucket != readinessBucketReady && readiness.FixFirst != "" {
			hotspotCandidates = append(hotspotCandidates, FleetReadinessHotspot{
				AgentUID: agent.AgentUID,
				HostID:   agent.HostID,
				Reason:   readiness.FixFirst,
				Bucket:   readiness.Bucket,
			})
		}
	}
	if out.Total > 0 {
		out.AverageScore = scoreSum / out.Total
	}
	sort.SliceStable(out.Agents, func(i, j int) bool {
		return out.Agents[i].Readiness.Score < out.Agents[j].Readiness.Score
	})
	sort.SliceStable(hotspotCandidates, func(i, j int) bool {
		// Stale > degraded > needs_attention > unknown
		order := map[string]int{readinessBucketStale: 0, readinessBucketDegraded: 1, readinessBucketNeedsAttention: 2, readinessBucketUnknown: 3, readinessBucketReady: 4}
		return order[hotspotCandidates[i].Bucket] < order[hotspotCandidates[j].Bucket]
	})
	if len(hotspotCandidates) > 8 {
		hotspotCandidates = hotspotCandidates[:8]
	}
	out.FixFirst = hotspotCandidates

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// collectAgentInfos returns the AgentInfo slice the workbench endpoint uses,
// without writing to the response. It mirrors the existing
// getAgentsWorkbenchSummary logic so readiness computation has the same data.
func (s *Server) collectAgentInfos() []AgentInfo {
	ingestURL := os.Getenv("INGEST_API_URL")
	if ingestURL == "" {
		ingestURL = "http://localhost:9091"
	}

	byHost, _ := FetchIngestVisibilityDevices(ingestURL, summaryHTTPClient())

	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	out := make([]AgentInfo, 0, len(s.store.agents))
	for _, agent := range s.store.agents {
		labels := make([]string, 0, len(agent.Labels))
		for label := range agent.Labels {
			labels = append(labels, label)
		}
		info := AgentInfo{
			AgentUID:            agent.AgentUID,
			OrgID:               agent.OrgID,
			HostID:              agent.HostID,
			Hostname:            agent.Hostname,
			MachineIDHash:       agent.MachineIDHash,
			AgentVersion:        agent.AgentVersion,
			Platform:            agent.Platform,
			Network:             agent.Network,
			Labels:              labels,
			Note:                agent.Note,
			Created:             agent.Created,
			LastSeen:            agent.LastSeen,
			Status:              agentConnectionStatus(agent.LastSeen),
			DetectionPackStatus: s.fetchDetectionPackStatus(agent.AgentUID),
		}
		if v := byHost[agent.HostID]; v != nil {
			info.Visibility = v
		} else if v := byHost[agent.AgentUID]; v != nil {
			info.Visibility = v
		}
		out = append(out, info)
	}
	return out
}
