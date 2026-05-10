package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteVisibilityStore struct {
	db   *sql.DB
	path string
}

func newSQLiteVisibilityStore(path string) (*sqliteVisibilityStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("AEGIS_VISIBILITY_SQLITE_PATH is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite visibility mkdir: %w", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("sqlite visibility abs: %w", err)
	}
	dsn := "file:" + strings.ReplaceAll(abs, "\\", "/") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(1)

	schema := `
CREATE TABLE IF NOT EXISTS visibility_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  envelope_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_visibility_events_event_id ON visibility_events(event_id);
`
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite schema: %w", err)
	}

	return &sqliteVisibilityStore{db: db, path: abs}, nil
}

func (s *sqliteVisibilityStore) Has(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM visibility_events WHERE event_id = ?`, eventID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *sqliteVisibilityStore) Append(ctx context.Context, event visibilityEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if event.ReceivedAtMS == 0 {
		event.ReceivedAtMS = time.Now().UnixMilli()
	}

	raw, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("sqlite marshal event: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO visibility_events (event_id, envelope_json) VALUES (?, ?)`, event.EventID, raw)
	if err != nil {
		return fmt.Errorf("sqlite insert: %w", err)
	}
	return nil
}

func (s *sqliteVisibilityStore) Query(ctx context.Context, filter visibilityQueryFilter) ([]visibilityEvent, error) {
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

	q := strings.Builder{}
	q.WriteString(`SELECT envelope_json FROM visibility_events WHERE 1=1`)
	args := make([]any, 0, 6)
	if filter.EventID != "" {
		q.WriteString(` AND json_extract(envelope_json, '$.event_id') = ?`)
		args = append(args, filter.EventID)
	}
	if filter.TenantID != "" {
		q.WriteString(` AND json_extract(envelope_json, '$.tenant_id') = ?`)
		args = append(args, filter.TenantID)
	}
	if filter.DeviceID != "" {
		q.WriteString(` AND json_extract(envelope_json, '$.device_id') = ?`)
		args = append(args, filter.DeviceID)
	}
	if filter.AgentID != "" {
		q.WriteString(` AND json_extract(envelope_json, '$.agent_id') = ?`)
		args = append(args, filter.AgentID)
	}
	if filter.EventType != "" {
		q.WriteString(` AND json_extract(envelope_json, '$.event_type') = ?`)
		args = append(args, filter.EventType)
	}
	q.WriteString(` ORDER BY id DESC LIMIT ?`)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite query: %w", err)
	}
	defer rows.Close()

	out := make([]visibilityEvent, 0, limit)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var ev visibilityEvent
		if err := json.Unmarshal(raw, &ev); err != nil {
			return nil, fmt.Errorf("sqlite unmarshal: %w", err)
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *sqliteVisibilityStore) ListDevices(ctx context.Context, filter visibilityDeviceFilter) ([]visibilityDeviceRecord, error) {
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

	rows, err := s.db.QueryContext(ctx, `SELECT envelope_json FROM visibility_events ORDER BY id DESC LIMIT ?`, 50000)
	if err != nil {
		return nil, fmt.Errorf("sqlite list devices scan: %w", err)
	}
	defer rows.Close()

	byDevice := make(map[string]*visibilityDeviceRecord)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var event visibilityEvent
		if err := json.Unmarshal(raw, &event); err != nil {
			continue
		}
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
	if err := rows.Err(); err != nil {
		return nil, err
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

func (s *sqliteVisibilityStore) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
