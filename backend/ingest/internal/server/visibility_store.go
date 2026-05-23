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
	AppendBatch(ctx context.Context, events []visibilityEvent) error
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
	mu              sync.Mutex
	path            string
	devicesPath     string
	file            *os.File
	seen            map[string]struct{}
	devices         map[string]*visibilityDeviceRecord
	writesSinceSync int
	maxBytes        int64
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
		path:        path,
		devicesPath: path + ".devices.json",
		seen:        make(map[string]struct{}),
		devices:     make(map[string]*visibilityDeviceRecord),
		maxBytes:    jsonlMaxBytesFromEnv(),
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
	if err := s.loadSeenIDs(); err != nil {
		return err
	}
	return s.loadDevicesIndex()
}

func (s *fileVisibilityStore) loadSeenIDs() error {
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
		if event.EventID != "" {
			s.seen[event.EventID] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan visibility store: %w", err)
	}
	return nil
}

func (s *fileVisibilityStore) loadDevicesIndex() error {
	raw, err := os.ReadFile(s.devicesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read visibility devices index: %w", err)
	}
	var records []visibilityDeviceRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		return fmt.Errorf("failed to parse visibility devices index: %w", err)
	}
	for i := range records {
		record := records[i]
		if record.DeviceID == "" {
			continue
		}
		if record.EventTypeCount == nil {
			record.EventTypeCount = make(map[string]int)
		}
		copy := record
		s.devices[record.DeviceID] = &copy
	}
	return nil
}

func (s *fileVisibilityStore) persistDevicesIndexLocked() error {
	records := make([]visibilityDeviceRecord, 0, len(s.devices))
	for _, record := range s.devices {
		records = append(records, *record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].LastSeenMS == records[j].LastSeenMS {
			return records[i].DeviceID < records[j].DeviceID
		}
		return records[i].LastSeenMS > records[j].LastSeenMS
	})
	raw, err := json.Marshal(records)
	if err != nil {
		return fmt.Errorf("failed to encode visibility devices index: %w", err)
	}
	tmp := s.devicesPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("failed to write visibility devices index: %w", err)
	}
	return os.Rename(tmp, s.devicesPath)
}

func (s *fileVisibilityStore) maybeRotateLocked() error {
	if s.maxBytes <= 0 {
		return nil
	}
	info, err := s.file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < s.maxBytes {
		return nil
	}
	if err := s.file.Close(); err != nil {
		return err
	}
	rotated := s.path + "." + time.Now().UTC().Format("20060102-150405") + ".jsonl"
	if err := os.Rename(s.path, rotated); err != nil {
		return fmt.Errorf("failed to rotate visibility store: %w", err)
	}
	s.seen = make(map[string]struct{})
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to reopen visibility store: %w", err)
	}
	s.file = file
	s.writesSinceSync = 0
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
	if err := s.maybeRotateLocked(); err != nil {
		return err
	}
	if _, err := s.file.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write visibility event: %w", err)
	}
	s.writesSinceSync++
	if s.writesSinceSync >= fileStoreSyncEvery {
		if err := s.file.Sync(); err != nil {
			return fmt.Errorf("failed to sync visibility event: %w", err)
		}
		s.writesSinceSync = 0
	}

	if event.EventID != "" {
		s.seen[event.EventID] = struct{}{}
	}
	upsertDeviceRecord(s.devices, event)
	return s.persistDevicesIndexLocked()
}

func (s *fileVisibilityStore) AppendBatch(ctx context.Context, events []visibilityEvent) error {
	for _, event := range events {
		if err := s.Append(ctx, event); err != nil {
			return err
		}
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

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read visibility store: %w", err)
	}
	defer file.Close()

	lines, err := readTailLines(file, limit*8)
	if err != nil {
		return nil, err
	}

	results := make([]visibilityEvent, 0, limit)
	for i := len(lines) - 1; i >= 0 && len(results) < limit; i-- {
		line := lines[i]
		if len(line) == 0 {
			continue
		}
		var event visibilityEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
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

func readTailLines(file *os.File, maxLines int) ([][]byte, error) {
	if maxLines <= 0 {
		maxLines = defaultVisibilityQueryLimit
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxVisibilityLineBytes)
	lines := make([][]byte, 0, maxLines)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if len(line) == 0 {
			continue
		}
		if len(lines) >= maxLines {
			copy(lines, lines[1:])
			lines[len(lines)-1] = line
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan visibility store: %w", err)
	}
	return lines, nil
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

	devices := make([]visibilityDeviceRecord, 0, len(s.devices))
	for _, record := range s.devices {
		if filter.TenantID != "" && record.TenantID != filter.TenantID {
			continue
		}
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
	if err := s.file.Sync(); err != nil {
		return err
	}
	err := s.file.Close()
	s.file = nil
	return err
}
