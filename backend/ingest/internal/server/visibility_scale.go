package server

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultIngestDedupeLimit       = 100_000
	defaultVisibilityRetentionHours = 168 // 7 days
	defaultJSONLMaxBytes           = 512 << 20 // 512 MiB
	fileStoreSyncEvery             = 64
)

func ingestDedupeLimitFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("AEGIS_INGEST_DEDUPE_LIMIT"))
	if raw == "" {
		return defaultIngestDedupeLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultIngestDedupeLimit
	}
	return n
}

func visibilityRetentionDuration() time.Duration {
	raw := strings.TrimSpace(os.Getenv("AEGIS_VISIBILITY_RETENTION_HOURS"))
	if raw == "" {
		return time.Duration(defaultVisibilityRetentionHours) * time.Hour
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return time.Duration(defaultVisibilityRetentionHours) * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

func jsonlMaxBytesFromEnv() int64 {
	raw := strings.TrimSpace(os.Getenv("AEGIS_VISIBILITY_JSONL_MAX_BYTES"))
	if raw == "" {
		return defaultJSONLMaxBytes
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return defaultJSONLMaxBytes
	}
	return n
}

func upsertDeviceRecord(byDevice map[string]*visibilityDeviceRecord, event visibilityEvent) {
	if event.DeviceID == "" {
		return
	}

	seenMS := event.ReceivedAtMS
	if seenMS == 0 {
		seenMS = event.TimestampMS
	}

	record, exists := byDevice[event.DeviceID]
	if !exists {
		record = &visibilityDeviceRecord{
			DeviceID:       event.DeviceID,
			AgentID:        event.AgentID,
			Source:         event.Source,
			TenantID:       event.TenantID,
			SensorVersion:  event.SensorVersion,
			FirstSeenMS:    seenMS,
			LastSeenMS:     seenMS,
			LastEventType:  event.EventType,
			LastSequence:   event.Sequence,
			EventTypeCount: make(map[string]int),
		}
		byDevice[event.DeviceID] = record
	}

	if seenMS < record.FirstSeenMS || record.FirstSeenMS == 0 {
		record.FirstSeenMS = seenMS
	}
	if seenMS >= record.LastSeenMS {
		record.LastSeenMS = seenMS
		record.LastEventType = event.EventType
		record.LastSequence = event.Sequence
		record.AgentID = event.AgentID
		record.Source = event.Source
		record.SensorVersion = event.SensorVersion
		record.TenantID = event.TenantID
	}
	record.EventCount++
	if event.EventType != "" {
		record.EventTypeCount[event.EventType]++
	}
}

func mergeDeviceTypeCounts(dst map[string]int, src map[string]int) {
	for k, v := range src {
		dst[k] += v
	}
}

func deviceTypeCountsJSON(counts map[string]int) string {
	if len(counts) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(counts)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func parseDeviceTypeCountsJSON(raw string) map[string]int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]int{}
	}
	out := make(map[string]int)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]int{}
	}
	return out
}

// HasMany returns a set of event IDs already present in the store.
func HasMany(ctx context.Context, store visibilityStore, eventIDs []string) (map[string]bool, error) {
	if store == nil || len(eventIDs) == 0 {
		return map[string]bool{}, nil
	}
	if batch, ok := store.(interface {
		HasMany(context.Context, []string) (map[string]bool, error)
	}); ok {
		return batch.HasMany(ctx, eventIDs)
	}

	seen := make(map[string]bool, len(eventIDs))
	for _, id := range eventIDs {
		if id == "" {
			continue
		}
		exists, err := store.Has(ctx, id)
		if err != nil {
			return nil, err
		}
		if exists {
			seen[id] = true
		}
	}
	return seen, nil
}
