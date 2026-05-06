package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"aegisflux/backend/detection-pipeline/internal/model"
	"aegisflux/backend/detection-pipeline/internal/store"
)

func TestControllerLatest_NoSignedPacks(t *testing.T) {
	t.Setenv("DETECTION_ROLLOUT_LAB_ONLY", "true")
	st := store.New("")
	srv, err := NewServer(st, "http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/detection-packs/latest?os=linux&agent_version=1.0.0", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestControllerLatest_WithSignedPack(t *testing.T) {
	t.Setenv("DETECTION_ROLLOUT_LAB_ONLY", "true")
	st := store.New("")
	raw := []byte(`{
		"schema_version":"detection_pack.v1",
		"pack_id":"lab.pack",
		"pack_version":"1.0.0",
		"created_at":"2026-06-01T00:00:00Z",
		"author":"t",
		"min_agent_version":"0.1.0",
		"supported_os":["linux","windows"],
		"mode":"observe",
		"signature":{"algorithm":"ed25519","key_id":"k","value_b64":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="},
		"evaluator_limits":{"max_wall_time_ms_per_batch":50,"max_heap_bytes":65536,"max_rules_evaluated_per_batch":1,"max_cpu_percent_soft":1,"max_string_comparisons_per_rule":16,"max_clause_depth":2,"max_clauses_per_rule":4},
		"rules":[{"rule_id":"r1","priority":1,"title":"t","description":"d","classification":"c","pattern_tags":["p"],"agent_likelihood":0.1,"confidence":0.2,"risk_score":1,"recommended_action":"monitor","required_evidence":["process"],"match":{"process":{"command_line_contains_any":["x"]}}}]
	}`)
	_ = st.PutSigned(&model.SignedPackArtifact{
		ID:           "sig1",
		CandidateID:  "c1",
		CreatedAtMS:  1,
		PackJSON:     raw,
		SignatureAlg: "ed25519",
		KeyID:        "k",
		PackID:       "lab.pack",
		PackVersion:  "1.0.0",
		SHA256Hex:    "abc",
	})
	srv, err := NewServer(st, "http://127.0.0.1:9")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	srv.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/detection-packs/latest?os=linux&agent_version=1.0.0", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d: %s", rec.Code, rec.Body.String())
	}
}
