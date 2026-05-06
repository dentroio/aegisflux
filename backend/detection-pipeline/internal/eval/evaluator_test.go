package eval

import (
	"encoding/json"
	"testing"

	"aegisflux/backend/detection-pipeline/internal/ingestclient"
)

func TestRuleMatches_MCPCommandLine(t *testing.T) {
	raw := []byte(`{
		"rule_id": "test.mcp",
		"match": {
			"process": {
				"case_insensitive": true,
				"command_line_contains_any": ["@modelcontextprotocol", "mcp-server"]
			}
		}
	}`)
	var rule map[string]any
	if err := json.Unmarshal(raw, &rule); err != nil {
		t.Fatal(err)
	}
	ev := ingestclient.VisibilityEvent{
		EventType: "aegis.process.started",
		Payload:   []byte(`{"process_guid":"g1","pid":1,"name":"node.exe","command_line":"node @modelcontextprotocol/foo --mcp-server","collection_method":"t"}`),
	}
	b, err := BuildBatch([]ingestclient.VisibilityEvent{ev})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := RuleMatches(rule, b)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected match")
	}
}

func TestRuleMatches_DNS(t *testing.T) {
	raw := []byte(`{
		"rule_id": "test.dns",
		"match": {
			"dns": {
				"query_contains_any": ["ollama."]
			}
		}
	}`)
	var rule map[string]any
	if err := json.Unmarshal(raw, &rule); err != nil {
		t.Fatal(err)
	}
	ev := ingestclient.VisibilityEvent{
		EventType: "aegis.dns.observed",
		Payload:   []byte(`{"query":"ollama.internal.lab.local","answers":[],"correlation_method":"x","correlation_confidence":0.5}`),
	}
	b, err := BuildBatch([]ingestclient.VisibilityEvent{ev})
	if err != nil {
		t.Fatal(err)
	}
	ok, err := RuleMatches(rule, b)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected match")
	}
}
