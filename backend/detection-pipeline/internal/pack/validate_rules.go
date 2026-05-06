package pack

import (
	"encoding/json"
	"fmt"
	"time"

	"aegisflux/backend/detection-pipeline/internal/model"
)

// ValidateProposedRules validates proposed_rules as the rules array of a syntactically valid pack.
func ValidateProposedRules(c *model.Candidate) error {
	var rules any
	if err := json.Unmarshal(c.ProposedRules, &rules); err != nil {
		return fmt.Errorf("proposed_rules json: %w", err)
	}
	var limits any
	if err := json.Unmarshal(c.EvaluatorLimits, &limits); err != nil {
		return fmt.Errorf("evaluator_limits json: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	doc := map[string]any{
		"schema_version":        "detection_pack.v1",
		"pack_id":               c.PackID,
		"pack_version":          c.PackVersion,
		"created_at":            now,
		"min_agent_version":     c.MinAgentVersion,
		"supported_os":          toAnySlice(c.SupportedOS),
		"mode":                  "observe",
		"expires_at":            nil,
		"rollback_pack_version": nil,
		"evaluator_limits":      limits,
		"rules":                 rules,
		"signature": map[string]any{
			"algorithm": "ed25519",
			"key_id":    "schema-check",
			"value_b64": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		},
	}
	if c.Author != "" {
		doc["author"] = c.Author
	} else if c.Source != "" {
		doc["source"] = c.Source
	} else {
		doc["author"] = "validation"
	}
	comp, err := NewSchemaCompiler()
	if err != nil {
		return err
	}
	return comp.Validate(doc)
}
