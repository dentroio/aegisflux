package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSummaryDashboard_EmptyStore(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/vis.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := &IngestServer{store: store, logger: log}

	req := httptest.NewRequest(http.MethodGet, "/v1/visibility/summary/dashboard", nil)
	rec := httptest.NewRecorder()
	s.handleSummaryDashboard(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"totalDevices":0`) {
		t.Fatalf("expected empty totalDevices in body: %s", rec.Body.String())
	}
}

func TestSummaryDevice_MissingID(t *testing.T) {
	store, err := newFileVisibilityStore(t.TempDir() + "/vis.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := &IngestServer{store: store, logger: log}
	req := httptest.NewRequest(http.MethodGet, "/v1/visibility/summary/device", nil)
	rec := httptest.NewRecorder()
	s.handleSummaryDevice(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
