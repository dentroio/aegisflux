package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSQLiteVisibilityStoreAppendQuery(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "vis.db")
	store, err := newSQLiteVisibilityStore(path)
	if err != nil {
		t.Fatalf("newSQLiteVisibilityStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	ev := visibilityEvent{
		SchemaVersion: "visibility.v1",
		EventID:       "e-sqlite-1",
		EventType:     "aegis.process.started",
		TimestampMS:   100,
		Source:        "test",
		DeviceID:      "dev-1",
		AgentID:       "ag-1",
		SensorVersion: "0.1.0",
		Sequence:      1,
		Payload:       json.RawMessage(`{"process_guid":"p1","pid":42,"name":"notepad.exe","collection_method":"test"}`),
	}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatalf("duplicate append: %v", err)
	}
	got, err := store.Query(ctx, visibilityQueryFilter{DeviceID: "dev-1", Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].EventID != "e-sqlite-1" {
		t.Fatalf("event id mismatch")
	}
}

func TestSQLiteVisibilityStoreListDevicesFromRegistry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "vis-devices.db")
	store, err := newSQLiteVisibilityStore(path)
	if err != nil {
		t.Fatalf("newSQLiteVisibilityStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	ctx := context.Background()
	events := []visibilityEvent{
		{
			SchemaVersion: "visibility.v1",
			EventID:       "e-dev-1",
			EventType:     "aegis.process.started",
			TimestampMS:   100,
			ReceivedAtMS:  100,
			Source:        "test",
			DeviceID:      "dev-a",
			AgentID:       "ag-a",
			SensorVersion: "0.1.0",
			Sequence:      1,
			Payload:       json.RawMessage(`{}`),
		},
		{
			SchemaVersion: "visibility.v1",
			EventID:       "e-dev-2",
			EventType:     "aegis.network.flow",
			TimestampMS:   200,
			ReceivedAtMS:  200,
			Source:        "test",
			DeviceID:      "dev-b",
			AgentID:       "ag-b",
			SensorVersion: "0.1.0",
			Sequence:      2,
			Payload:       json.RawMessage(`{}`),
		},
	}
	if err := store.AppendBatch(ctx, events); err != nil {
		t.Fatalf("append batch: %v", err)
	}

	devices, err := store.ListDevices(ctx, visibilityDeviceFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].DeviceID != "dev-b" {
		t.Fatalf("expected most recent device dev-b first, got %s", devices[0].DeviceID)
	}
}

func TestNewVisibilityStoreFromEnvSQLite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.db")
	t.Setenv("AEGIS_VISIBILITY_SQLITE_PATH", path)
	t.Setenv("AEGIS_VISIBILITY_STORE_PATH", "")

	st, err := newVisibilityStoreFromEnv()
	if err != nil {
		t.Fatalf("newVisibilityStoreFromEnv: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, ok := st.(*sqliteVisibilityStore); !ok {
		t.Fatalf("expected sqlite store, got %T", st)
	}
}

func TestNewVisibilityStoreFromEnvJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	_ = os.Unsetenv("AEGIS_VISIBILITY_SQLITE_PATH")
	t.Setenv("AEGIS_VISIBILITY_STORE_PATH", path)

	st, err := newVisibilityStoreFromEnv()
	if err != nil {
		t.Fatalf("newVisibilityStoreFromEnv: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, ok := st.(*fileVisibilityStore); !ok {
		t.Fatalf("expected file store, got %T", st)
	}
}
