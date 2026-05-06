package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"aegisflux/backend/detection-pipeline/internal/model"
	"aegisflux/backend/detection-pipeline/internal/store"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}

func TestCreateResearchAndCandidate(t *testing.T) {
	st := store.New("")
	srv, err := NewServer(st, "http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	srv.Register(mux)

	rec := httptest.NewRecorder()
	body := `{"title":"R1","summary":"s"}`
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/detection/research-items", bytes.NewBufferString(body)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("research: %d %s", rec.Code, rec.Body.String())
	}
	var ri model.ResearchItem
	if err := json.Unmarshal(rec.Body.Bytes(), &ri); err != nil {
		t.Fatal(err)
	}

	root := repoRoot(t)
	candPath := filepath.Join(root, "schemas", "detection", "fixtures", "wo-det-002", "candidate_mcp_tool_bridge.example.json")
	candBytes, err := os.ReadFile(candPath)
	if err != nil {
		t.Skip("fixture missing:", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(candBytes, &payload); err != nil {
		t.Fatal(err)
	}
	payload["research_item_id"] = ri.ID
	out, _ := json.Marshal(payload)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/detection/candidates", bytes.NewReader(out))
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("candidate: %d %s", rec2.Code, rec2.Body.String())
	}
}
