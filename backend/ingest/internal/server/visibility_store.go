package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultVisibilityQueryLimit = 100
const maxVisibilityQueryLimit = 1000

type visibilityStore interface {
	Has(ctx context.Context, eventID string) (bool, error)
	Append(ctx context.Context, event visibilityEvent) error
	Query(ctx context.Context, filter visibilityQueryFilter) ([]visibilityEvent, error)
	ListDevices(ctx context.Context, filter visibilityDeviceFilter) ([]visibilityDeviceRecord, error)
	Close() error
}

type visibilityQueryFilter struct {
	EventID   string
	TenantID  string
	DeviceID  string
	AgentID   string
	EventType string
	Limit     int
}

type visibilityDeviceFilter struct {
	TenantID string
	Limit    int
}

type visibilityDeviceRecord struct {
	DeviceID       string         `json:"device_id"`
	AgentID        string         `json:"agent_id"`
	Source         string         `json:"source"`
	TenantID       string         `json:"tenant_id,omitempty"`
	SensorVersion  string         `json:"sensor_version"`
	FirstSeenMS    int64          `json:"first_seen_ms"`
	LastSeenMS     int64          `json:"last_seen_ms"`
	LastEventType  string         `json:"last_event_type"`
	LastSequence   int64          `json:"last_sequence"`
	EventCount     int            `json:"event_count"`
	EventTypeCount map[string]int `json:"event_type_count"`
}

type fileVisibilityStore struct {
	mu     sync.Mutex
	path   string
	file   *os.File
	events []visibilityEvent
	seen   map[string]struct{}
}

// newVisibilityStoreFromEnv selects SQLite (durable lab/CI) or JSONL file backing.
func newVisibilityStoreFromEnv() (visibilityStore, error) {
	if p := strings.TrimSpace(os.Getenv("AEGIS_VISIBILITY_SQLITE_PATH")); p != "" {
		return newSQLiteVisibilityStore(p)
	}
	return newFileVisibilityStore(os.Getenv("AEGIS_VISIBILITY_STORE_PATH"))
}

func newFileVisibilityStore(path string) (*fileVisibilityStore, error) {
	if path == "" {
		path = filepath.Join("data", "visibility-events.jsonl")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create visibility store directory: %w", err)
	}

	store := &fileVisibilityStore{
		path: path,
		seen: make(map[string]struct{}),
	}
	if err := store.load(); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open visibility store: %w", err)
	}
	store.file = file
	return store, nil
}

func (s *fileVisibilityStore) load() error {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read visibility store: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxVisibilityLineBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var event visibilityEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("failed to parse visibility store record: %w", err)
		}
		s.events = append(s.events, event)
		if event.EventID != "" {
			s.seen[event.EventID] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan visibility store: %w", err)
	}
	return nil
}

func (s *fileVisibilityStore) Has(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.seen[eventID]
	return exists, nil
}

func (s *fileVisibilityStore) Append(ctx context.Context, event visibilityEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if event.EventID != "" {
		if _, exists := s.seen[event.EventID]; exists {
			return nil
		}
	}
	if event.ReceivedAtMS == 0 {
		event.ReceivedAtMS = time.Now().UnixMilli()
	}

	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to encode visibility event: %w", err)
	}
	if _, err := s.file.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write visibility event: %w", err)
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync visibility event: %w", err)
	}

	s.events = append(s.events, event)
	if event.EventID != "" {
		s.seen[event.EventID] = struct{}{}
	}
	return nil
}

func (s *fileVisibilityStore) Query(ctx context.Context, filter visibilityQueryFilter) ([]visibilityEvent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultVisibilityQueryLimit
	}
	if limit > maxVisibilityQueryLimit {
		limit = maxVisibilityQueryLimit
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	results := make([]visibilityEvent, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(results) < limit; i-- {
		event := s.events[i]
		if filter.EventID != "" && event.EventID != filter.EventID {
			continue
		}
		if filter.TenantID != "" && event.TenantID != filter.TenantID {
			continue
		}
		if filter.DeviceID != "" && event.DeviceID != filter.DeviceID {
			continue
		}
		if filter.AgentID != "" && event.AgentID != filter.AgentID {
			continue
		}
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}
		results = append(results, event)
	}
	return results, nil
}

func (s *fileVisibilityStore) ListDevices(ctx context.Context, filter visibilityDeviceFilter) ([]visibilityDeviceRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultVisibilityQueryLimit
	}
	if limit > maxVisibilityQueryLimit {
		limit = maxVisibilityQueryLimit
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byDevice := make(map[string]*visibilityDeviceRecord)
	for _, event := range s.events {
		if event.DeviceID == "" {
			continue
		}
		if filter.TenantID != "" && event.TenantID != filter.TenantID {
			continue
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
		record.EventTypeCount[event.EventType]++
	}

	devices := make([]visibilityDeviceRecord, 0, len(byDevice))
	for _, record := range byDevice {
		devices = append(devices, *record)
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].LastSeenMS == devices[j].LastSeenMS {
			return devices[i].DeviceID < devices[j].DeviceID
		}
		return devices[i].LastSeenMS > devices[j].LastSeenMS
	})

	if len(devices) > limit {
		devices = devices[:limit]
	}
	return devices, nil
}

func (s *fileVisibilityStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}
