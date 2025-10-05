package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Connect to test database
	db, err := sql.Open("postgres", "postgres://testuser:testpass@localhost:5432/aegisflux_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM audit_log")
		db.Exec("DELETE FROM assignments")
		db.Exec("DELETE FROM bundles")
		db.Exec("DELETE FROM agents")
		db.Exec("DELETE FROM signing_keys")
		db.Close()
	}

	return db, cleanup
}

func runMigrations(db *sql.DB) error {
	// Read and execute migration file
	migrationSQL, err := ioutil.ReadFile("../migrate/001_init.sql")
	if err != nil {
		return err
	}
	
	_, err = db.Exec(string(migrationSQL))
	return err
}

func TestStore_CreateAgent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	agent := &Agent{
		AgentUID: uuid.New(),
		HostID:   "test-host-1",
		Platform: json.RawMessage(`{"os": "linux", "arch": "amd64"}`),
		Labels:   json.RawMessage(`["production", "web"]`),
		Notes:    "Test agent",
	}

	err := store.CreateAgent(ctx, agent)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Verify agent was created
	retrieved, err := store.GetAgent(ctx, "test-host-1")
	if err != nil {
		t.Fatalf("Failed to retrieve agent: %v", err)
	}

	if retrieved.HostID != agent.HostID {
		t.Errorf("Expected host_id %s, got %s", agent.HostID, retrieved.HostID)
	}
}

func TestStore_CreateBundle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	bundle := &Bundle{
		BundleID:  uuid.New(),
		Name:      "test-bundle",
		Hash:      "abcd1234",
		Sig:       "signature123",
		Algo:      "Ed25519",
		Kid:       "key-1",
		Meta:      json.RawMessage(`{"version": "1.0.0"}`),
		CreatedBy: "test-user",
	}

	err := store.CreateBundle(ctx, bundle)
	if err != nil {
		t.Fatalf("Failed to create bundle: %v", err)
	}

	// Verify bundle was created
	retrieved, err := store.GetBundle(ctx, bundle.BundleID)
	if err != nil {
		t.Fatalf("Failed to retrieve bundle: %v", err)
	}

	if retrieved.Name != bundle.Name {
		t.Errorf("Expected name %s, got %s", bundle.Name, retrieved.Name)
	}
}

func TestStore_CreateAssignment(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	// First create a bundle
	bundle := &Bundle{
		BundleID:  uuid.New(),
		Name:      "test-bundle",
		Hash:      "abcd1234",
		Sig:       "signature123",
		Algo:      "Ed25519",
		Kid:       "key-1",
		Meta:      json.RawMessage(`{"version": "1.0.0"}`),
		CreatedBy: "test-user",
	}
	store.CreateBundle(ctx, bundle)

	// Create assignment
	assignment := &Assignment{
		ID:           uuid.New(),
		HostSelector: json.RawMessage(`{"host_id": "test-host-1"}`),
		TTLTS:        timePtr(time.Now().Add(time.Hour)),
		DryRun:       true,
		BundleID:     bundle.BundleID,
		CreatedBy:    "test-user",
		Status:       "active",
	}

	err := store.CreateAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("Failed to create assignment: %v", err)
	}

	// Verify assignment was created
	retrieved, err := store.GetAssignment(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve assignment: %v", err)
	}

	if retrieved.Status != assignment.Status {
		t.Errorf("Expected status %s, got %s", assignment.Status, retrieved.Status)
	}
}

func TestStore_LogAuditEvent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	details := json.RawMessage(`{"action": "test", "target": "test-target"}`)
	auditID, err := store.LogAuditEvent(ctx, "test-user", "test-action", "test-target", details)
	if err != nil {
		t.Fatalf("Failed to log audit event: %v", err)
	}

	if auditID == uuid.Nil {
		t.Error("Expected non-nil audit ID")
	}
}

func TestStore_GetAgentBundleAssignments(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	// Create agent
	agent := &Agent{
		AgentUID: uuid.New(),
		HostID:   "test-host-1",
		Platform: json.RawMessage(`{"os": "linux", "arch": "amd64"}`),
		Labels:   json.RawMessage(`["production"]`),
	}
	store.CreateAgent(ctx, agent)

	// Create bundle
	bundle := &Bundle{
		BundleID:  uuid.New(),
		Name:      "test-bundle",
		Hash:      "abcd1234",
		Sig:       "signature123",
		Algo:      "Ed25519",
		Kid:       "key-1",
		CreatedBy: "test-user",
	}
	store.CreateBundle(ctx, bundle)

	// Create assignment
	assignment := &Assignment{
		ID:           uuid.New(),
		HostSelector: json.RawMessage(`{"host_id": "test-host-1"}`),
		BundleID:     bundle.BundleID,
		CreatedBy:    "test-user",
		Status:       "active",
	}
	store.CreateAssignment(ctx, assignment)

	// Get assignments for agent
	assignments, err := store.GetAgentBundleAssignments(ctx, "test-host-1")
	if err != nil {
		t.Fatalf("Failed to get agent assignments: %v", err)
	}

	if len(assignments) != 1 {
		t.Errorf("Expected 1 assignment, got %d", len(assignments))
	}

	if assignments[0].HostID != "test-host-1" {
		t.Errorf("Expected host_id test-host-1, got %s", assignments[0].HostID)
	}
}

func TestStore_HealthCheck(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	err := store.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestStore_ReadyCheck(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewStore(db)
	ctx := context.Background()

	err := store.ReadyCheck(ctx)
	if err != nil {
		t.Errorf("Ready check failed: %v", err)
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}





