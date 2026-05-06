package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaCompiler loads detection_pack.v1 from repo or SCHEMA_PATH.
type SchemaCompiler struct {
	url string
}

func NewSchemaCompiler() (*SchemaCompiler, error) {
	return &SchemaCompiler{
		url: "https://aegisflux.local/schemas/detection/detection-pack.v1.schema.json",
	}, nil
}

func (c *SchemaCompiler) schemaBytes() ([]byte, error) {
	if p := os.Getenv("DETECTION_PACK_SCHEMA_PATH"); p != "" {
		return os.ReadFile(p)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("schema path resolution failed")
	}
	// internal/pack -> backend/detection-pipeline -> repo root
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	p := filepath.Join(repoRoot, "schemas", "detection", "detection-pack.v1.schema.json")
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", p, err)
	}
	return b, nil
}

func (c *SchemaCompiler) Compile() (*jsonschema.Schema, error) {
	b, err := c.schemaBytes()
	if err != nil {
		return nil, err
	}
	comp := jsonschema.NewCompiler()
	comp.Draft = jsonschema.Draft2020
	if err := comp.AddResource(c.url, strings.NewReader(string(b))); err != nil {
		return nil, fmt.Errorf("add resource: %w", err)
	}
	sch, err := comp.Compile(c.url)
	if err != nil {
		return nil, fmt.Errorf("compile: %w", err)
	}
	return sch, nil
}

func (c *SchemaCompiler) Validate(doc any) error {
	sch, err := c.Compile()
	if err != nil {
		return err
	}
	return sch.Validate(doc)
}
