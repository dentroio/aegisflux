package api

import (
	"testing"
	"time"
)

func TestComputeAgentReadiness_ReadyBucket(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := AgentInfo{
		AgentUID:     "agent-1",
		HostID:       "host-1",
		Hostname:     "lab-1",
		AgentVersion: "1.4.0",
		LastSeen:     now.Add(-30 * time.Second),
		Status:       "online",
		Visibility: &VisibilityDeviceRecord{
			LastSeenMS:     now.UnixMilli() - 60_000,
			EventCount:     500,
			EventTypeCount: map[string]int{"aegis.process.started": 100, "aegis.flow.started": 100, "aegis.dns.observed": 300},
		},
		DetectionPackStatus: &DetectionPackStatus{
			RolloutState:    "applied",
			LastAppliedAtMS: now.UnixMilli() - 60*60*1000,
		},
	}
	r := computeAgentReadiness(agent, "1.4.0", now)
	if r.Bucket != readinessBucketReady {
		t.Fatalf("expected ready bucket, got %s (score %d) — dims %+v", r.Bucket, r.Score, r.Dimensions)
	}
	if r.Score < 80 {
		t.Fatalf("expected high score, got %d", r.Score)
	}
}

func TestComputeAgentReadiness_StaleHeartbeat(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := AgentInfo{
		AgentUID: "agent-2",
		HostID:   "host-2",
		LastSeen: now.Add(-3 * time.Hour),
		Status:   "offline",
	}
	r := computeAgentReadiness(agent, "1.4.0", now)
	if r.Bucket != readinessBucketStale {
		t.Fatalf("expected stale bucket for old heartbeat, got %s", r.Bucket)
	}
	if r.FixFirst == "" {
		t.Fatalf("expected fix_first guidance, got empty string")
	}
}

func TestComputeAgentReadiness_NeedsAttentionOnVersionDrift(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := AgentInfo{
		AgentUID:     "agent-3",
		HostID:       "host-3",
		AgentVersion: "1.2.0",
		LastSeen:     now.Add(-30 * time.Second),
		Status:       "online",
		Visibility: &VisibilityDeviceRecord{
			LastSeenMS:     now.UnixMilli() - 60_000,
			EventCount:     50,
			EventTypeCount: map[string]int{"a": 20, "b": 30},
		},
		DetectionPackStatus: &DetectionPackStatus{
			RolloutState:    "applied",
			LastAppliedAtMS: now.UnixMilli() - 60*60*1000,
		},
	}
	r := computeAgentReadiness(agent, "1.4.0", now)
	if r.Bucket != readinessBucketNeedsAttention {
		t.Fatalf("expected needs_attention for version drift, got %s — %+v", r.Bucket, r.Dimensions)
	}
}

func TestComputeAgentReadiness_UnknownNoData(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := AgentInfo{AgentUID: "agent-4"}
	r := computeAgentReadiness(agent, "", now)
	if r.Bucket != readinessBucketStale && r.Bucket != readinessBucketUnknown && r.Bucket != readinessBucketDegraded {
		t.Fatalf("expected stale/unknown/degraded for empty inputs, got %s — %+v", r.Bucket, r.Dimensions)
	}
	if r.Summary == "" {
		t.Fatalf("expected summary string")
	}
}

func TestComputeAgentReadiness_DegradedWhenManyBad(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	agent := AgentInfo{
		AgentUID: "agent-5",
		HostID:   "host-5",
		LastSeen: now.Add(-15 * time.Minute), // warn (heartbeat 15m -> warn)
		Status:   "offline",
		Visibility: &VisibilityDeviceRecord{
			LastSeenMS:     0,
			EventCount:     0,
			EventTypeCount: map[string]int{},
		},
		DetectionPackStatus: &DetectionPackStatus{
			RolloutState:       "rejected",
			LastRejectedReason: "signature_invalid",
		},
	}
	r := computeAgentReadiness(agent, "1.4.0", now)
	if r.Bucket != readinessBucketDegraded {
		t.Fatalf("expected degraded for multiple bad signals, got %s — %+v", r.Bucket, r.Dimensions)
	}
}
