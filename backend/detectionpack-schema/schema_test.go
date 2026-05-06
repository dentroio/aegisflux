package detectionpackschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func mustCompilePackSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "schemas", "detection", "detection-pack.v1.schema.json")
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	schemaURL := "https://aegisflux.local/schemas/detection/detection-pack.v1.schema.json"
	if err := c.AddResource(schemaURL, strings.NewReader(string(b))); err != nil {
		t.Fatalf("add resource: %v", err)
	}
	sch, err := c.Compile(schemaURL)
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return sch
}

func loadFixturePack(t *testing.T) map[string]any {
	t.Helper()
	root := repoRoot(t)
	p := filepath.Join(root, "schemas", "detection", "examples", "default-ai-markers.v1.pack.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	return doc
}

func TestDefaultFixturePackValidates(t *testing.T) {
	sch := mustCompilePackSchema(t)
	doc := loadFixturePack(t)
	if err := sch.Validate(doc); err != nil {
		t.Fatalf("fixture should validate: %v", err)
	}
}

func TestPackSchema_RejectsInvalidDocuments(t *testing.T) {
	sch := mustCompilePackSchema(t)

	tests := []struct {
		name  string
		mutate func(map[string]any)
	}{
		{
			name: "unsupported_mode",
			mutate: func(m map[string]any) {
				m["mode"] = "enforce"
			},
		},
		{
			name: "wrong_schema_version",
			mutate: func(m map[string]any) {
				m["schema_version"] = "detection_pack.v0"
			},
		},
		{
			name: "missing_signature",
			mutate: func(m map[string]any) {
				delete(m, "signature")
			},
		},
		{
			name: "missing_author_and_source",
			mutate: func(m map[string]any) {
				delete(m, "author")
				delete(m, "source")
			},
		},
		{
			name: "extra_root_property",
			mutate: func(m map[string]any) {
				m["executable_payload"] = "no"
			},
		},
		{
			name: "rules_over_max",
			mutate: func(m map[string]any) {
				rules := m["rules"].([]any)
				for len(rules) <= 500 {
					rules = append(rules, rules[0])
				}
				m["rules"] = rules
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := loadFixturePack(t)
			tt.mutate(doc)
			if err := sch.Validate(doc); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPackSchema_FlatDocumentStillInvalid(t *testing.T) {
	sch := mustCompilePackSchema(t)
	doc := map[string]any{"hello": "world"}
	if err := sch.Validate(doc); err == nil {
		t.Fatal("expected validation error")
	}
}
