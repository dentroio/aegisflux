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

const sqliteSchemaVersion = 2

type sqliteVisibilityStore struct {
	db        *sql.DB
	path      string
	retention time.Duration
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
	dsn := "file:" + strings.ReplaceAll(abs, "\\", "/") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	db.SetMaxOpenConns(4)

	store := &sqliteVisibilityStore{
		db:        db,
		path:      abs,
		retention: visibilityRetentionDuration(),
	}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *sqliteVisibilityStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS visibility_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  device_id TEXT NOT NULL DEFAULT '',
  agent_id TEXT NOT NULL DEFAULT '',
  tenant_id TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL DEFAULT '',
  received_at_ms INTEGER NOT NULL DEFAULT 0,
  timestamp_ms INTEGER NOT NULL DEFAULT 0,
  envelope_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_visibility_events_event_id ON visibility_events(event_id);
CREATE INDEX IF NOT EXISTS idx_visibility_events_device_received ON visibility_events(device_id, received_at_ms DESC);
CREATE INDEX IF NOT EXISTS idx_visibility_events_tenant ON visibility_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_visibility_events_type ON visibility_events(event_type);

CREATE TABLE IF NOT EXISTS visibility_devices (
  device_id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  tenant_id TEXT NOT NULL DEFAULT '',
  sensor_version TEXT NOT NULL DEFAULT '',
  first_seen_ms INTEGER NOT NULL DEFAULT 0,
  last_seen_ms INTEGER NOT NULL DEFAULT 0,
  last_event_type TEXT NOT NULL DEFAULT '',
  last_sequence INTEGER NOT NULL DEFAULT 0,
  event_count INTEGER NOT NULL DEFAULT 0,
  event_type_counts_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_visibility_devices_last_seen ON visibility_devices(last_seen_ms DESC);
CREATE INDEX IF NOT EXISTS idx_visibility_devices_tenant ON visibility_devices(tenant_id);
`
	if _, err := s.db.ExecContext(context.Background(), schema); err != nil {
		return fmt.Errorf("sqlite schema: %w", err)
	}

	var version int
	if err := s.db.QueryRowContext(context.Background(), `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("sqlite user_version: %w", err)
	}
	if version >= sqliteSchemaVersion {
		return nil
	}

	// Backfill denormalized columns for legacy rows (envelope-only schema).
	_, _ = s.db.ExecContext(context.Background(), `
UPDATE visibility_events SET
  device_id = COALESCE(NULLIF(device_id, ''), json_extract(envelope_json, '$.device_id')),
  agent_id = COALESCE(NULLIF(agent_id, ''), json_extract(envelope_json, '$.agent_id')),
  tenant_id = COALESCE(NULLIF(tenant_id, ''), json_extract(envelope_json, '$.tenant_id')),
  event_type = COALESCE(NULLIF(event_type, ''), json_extract(envelope_json, '$.event_type')),
  received_at_ms = CASE WHEN received_at_ms > 0 THEN received_at_ms ELSE COALESCE(json_extract(envelope_json, '$.received_at_ms'), json_extract(envelope_json, '$.timestamp_ms'), 0) END,
  timestamp_ms = CASE WHEN timestamp_ms > 0 THEN timestamp_ms ELSE COALESCE(json_extract(envelope_json, '$.timestamp_ms'), 0) END
WHERE device_id = '' OR event_type = '' OR received_at_ms = 0`)

	if err := s.rebuildDeviceIndex(context.Background()); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(context.Background(), fmt.Sprintf(`PRAGMA user_version = %d`, sqliteSchemaVersion)); err != nil {
		return fmt.Errorf("sqlite set user_version: %w", err)
	}
	return nil
}

func (s *sqliteVisibilityStore) rebuildDeviceIndex(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM visibility_devices`)
	if err != nil {
		return fmt.Errorf("sqlite clear devices: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT device_id, agent_id, source, tenant_id, sensor_version, event_type,
       COALESCE(received_at_ms, timestamp_ms, 0), COALESCE(timestamp_ms, 0),
       COALESCE(json_extract(envelope_json, '$.sequence'), 0)
FROM visibility_events
WHERE device_id != ''
ORDER BY id ASC`)
	if err != nil {
		return fmt.Errorf("sqlite rebuild devices scan: %w", err)
	}
	defer rows.Close()

	byDevice := make(map[string]*visibilityDeviceRecord)
	for rows.Next() {
		var (
			deviceID, agentID, source, tenantID, sensorVersion, eventType string
			receivedMS, timestampMS, sequence                           int64
		)
		if err := rows.Scan(&deviceID, &agentID, &source, &tenantID, &sensorVersion, &eventType, &receivedMS, &timestampMS, &sequence); err != nil {
			return err
		}
		seenMS := receivedMS
		if seenMS == 0 {
			seenMS = timestampMS
		}
		upsertDeviceRecord(byDevice, visibilityEvent{
			DeviceID:      deviceID,
			AgentID:       agentID,
			Source:        source,
			TenantID:      tenantID,
			SensorVersion: sensorVersion,
			EventType:     eventType,
			ReceivedAtMS:  seenMS,
			TimestampMS:   timestampMS,
			Sequence:      sequence,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, record := range byDevice {
		if err := s.upsertDeviceTx(ctx, tx, *record); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteVisibilityStore) Has(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil
	}
	seen, err := s.HasMany(ctx, []string{eventID})
	if err != nil {
		return false, err
	}
	return seen[eventID], nil
}

func (s *sqliteVisibilityStore) HasMany(ctx context.Context, eventIDs []string) (map[string]bool, error) {
	ids := make([]string, 0, len(eventIDs))
	for _, id := range eventIDs {
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return map[string]bool{}, nil
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	q := fmt.Sprintf(`SELECT event_id FROM visibility_events WHERE event_id IN (%s)`, placeholders)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]bool, len(ids))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		seen[id] = true
	}
	return seen, rows.Err()
}

func (s *sqliteVisibilityStore) Append(ctx context.Context, event visibilityEvent) error {
	return s.AppendBatch(ctx, []visibilityEvent{event})
}

func (s *sqliteVisibilityStore) AppendBatch(ctx context.Context, events []visibilityEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite begin: %w", err)
	}

	nowMS := time.Now().UnixMilli()
	stmt, err := tx.PrepareContext(ctx, `
INSERT OR IGNORE INTO visibility_events (
  event_id, device_id, agent_id, tenant_id, event_type, received_at_ms, timestamp_ms, envelope_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("sqlite prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, event := range events {
		if event.ReceivedAtMS == 0 {
			event.ReceivedAtMS = nowMS
		}
		raw, err := json.Marshal(event)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqlite marshal event: %w", err)
		}
		res, err := stmt.ExecContext(
			ctx,
			event.EventID,
			event.DeviceID,
			event.AgentID,
			event.TenantID,
			event.EventType,
			event.ReceivedAtMS,
			event.TimestampMS,
			raw,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("sqlite insert: %w", err)
		}
		inserted, _ := res.RowsAffected()
		if inserted == 0 {
			continue
		}
		if err := s.upsertDeviceTx(ctx, tx, deviceRecordFromEvent(event)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite commit: %w", err)
	}
	return s.maybePrune(ctx)
}

func deviceRecordFromEvent(event visibilityEvent) visibilityDeviceRecord {
	seenMS := event.ReceivedAtMS
	if seenMS == 0 {
		seenMS = event.TimestampMS
	}
	counts := map[string]int{}
	if event.EventType != "" {
		counts[event.EventType] = 1
	}
	return visibilityDeviceRecord{
		DeviceID:       event.DeviceID,
		AgentID:        event.AgentID,
		Source:         event.Source,
		TenantID:       event.TenantID,
		SensorVersion:  event.SensorVersion,
		FirstSeenMS:    seenMS,
		LastSeenMS:     seenMS,
		LastEventType:  event.EventType,
		LastSequence:   event.Sequence,
		EventCount:     1,
		EventTypeCount: counts,
	}
}

func (s *sqliteVisibilityStore) upsertDeviceTx(ctx context.Context, tx *sql.Tx, incoming visibilityDeviceRecord) error {
	if incoming.DeviceID == "" {
		return nil
	}

	var (
		agentID, source, tenantID, sensorVersion, lastEventType, countsJSON string
		firstSeenMS, lastSeenMS, lastSequence                              int64
		eventCount                                                         int
	)
	err := tx.QueryRowContext(ctx, `
SELECT agent_id, source, tenant_id, sensor_version, first_seen_ms, last_seen_ms,
       last_event_type, last_sequence, event_count, event_type_counts_json
FROM visibility_devices WHERE device_id = ?`, incoming.DeviceID).Scan(
		&agentID, &source, &tenantID, &sensorVersion, &firstSeenMS, &lastSeenMS,
		&lastEventType, &lastSequence, &eventCount, &countsJSON,
	)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, `
INSERT INTO visibility_devices (
  device_id, agent_id, source, tenant_id, sensor_version, first_seen_ms, last_seen_ms,
  last_event_type, last_sequence, event_count, event_type_counts_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			incoming.DeviceID, incoming.AgentID, incoming.Source, incoming.TenantID, incoming.SensorVersion,
			incoming.FirstSeenMS, incoming.LastSeenMS, incoming.LastEventType, incoming.LastSequence,
			incoming.EventCount, deviceTypeCountsJSON(incoming.EventTypeCount),
		)
		return err
	}

	counts := parseDeviceTypeCountsJSON(countsJSON)
	mergeDeviceTypeCounts(counts, incoming.EventTypeCount)

	if incoming.FirstSeenMS > 0 && (firstSeenMS == 0 || incoming.FirstSeenMS < firstSeenMS) {
		firstSeenMS = incoming.FirstSeenMS
	}
	if incoming.LastSeenMS >= lastSeenMS {
		lastSeenMS = incoming.LastSeenMS
		lastEventType = incoming.LastEventType
		lastSequence = incoming.LastSequence
		agentID = incoming.AgentID
		source = incoming.Source
		tenantID = incoming.TenantID
		sensorVersion = incoming.SensorVersion
	}
	eventCount += incoming.EventCount

	_, err = tx.ExecContext(ctx, `
UPDATE visibility_devices SET
  agent_id = ?, source = ?, tenant_id = ?, sensor_version = ?,
  first_seen_ms = ?, last_seen_ms = ?, last_event_type = ?, last_sequence = ?,
  event_count = ?, event_type_counts_json = ?
WHERE device_id = ?`,
		agentID, source, tenantID, sensorVersion, firstSeenMS, lastSeenMS, lastEventType, lastSequence,
		eventCount, deviceTypeCountsJSON(counts), incoming.DeviceID,
	)
	return err
}

func (s *sqliteVisibilityStore) maybePrune(ctx context.Context) error {
	if s.retention <= 0 {
		return nil
	}
	cutoff := time.Now().Add(-s.retention).UnixMilli()
	_, err := s.db.ExecContext(ctx, `DELETE FROM visibility_events WHERE received_at_ms > 0 AND received_at_ms < ?`, cutoff)
	return err
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
	args := make([]any, 0, 8)
	if filter.EventID != "" {
		q.WriteString(` AND event_id = ?`)
		args = append(args, filter.EventID)
	}
	if filter.TenantID != "" {
		q.WriteString(` AND tenant_id = ?`)
		args = append(args, filter.TenantID)
	}
	if filter.DeviceID != "" {
		q.WriteString(` AND device_id = ?`)
		args = append(args, filter.DeviceID)
	}
	if filter.AgentID != "" {
		q.WriteString(` AND agent_id = ?`)
		args = append(args, filter.AgentID)
	}
	if filter.EventType != "" {
		q.WriteString(` AND event_type = ?`)
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

	q := `
SELECT device_id, agent_id, source, tenant_id, sensor_version, first_seen_ms, last_seen_ms,
       last_event_type, last_sequence, event_count, event_type_counts_json
FROM visibility_devices WHERE 1=1`
	args := make([]any, 0, 2)
	if filter.TenantID != "" {
		q += ` AND tenant_id = ?`
		args = append(args, filter.TenantID)
	}
	q += ` ORDER BY last_seen_ms DESC, device_id ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite list devices: %w", err)
	}
	defer rows.Close()

	devices := make([]visibilityDeviceRecord, 0, limit)
	for rows.Next() {
		var record visibilityDeviceRecord
		var countsJSON string
		if err := rows.Scan(
			&record.DeviceID, &record.AgentID, &record.Source, &record.TenantID, &record.SensorVersion,
			&record.FirstSeenMS, &record.LastSeenMS, &record.LastEventType, &record.LastSequence,
			&record.EventCount, &countsJSON,
		); err != nil {
			return nil, err
		}
		record.EventTypeCount = parseDeviceTypeCountsJSON(countsJSON)
		devices = append(devices, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].LastSeenMS == devices[j].LastSeenMS {
			return devices[i].DeviceID < devices[j].DeviceID
		}
		return devices[i].LastSeenMS > devices[j].LastSeenMS
	})
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
