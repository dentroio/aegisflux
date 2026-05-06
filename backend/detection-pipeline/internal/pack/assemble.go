package pack

import (
	"encoding/json"
	"fmt"
	"time"

	"aegisflux/backend/detection-pipeline/internal/model"
)

// AssemblePack builds a full detection_pack.v1 map from an approved candidate (unsigned).
func AssemblePack(c *model.Candidate) (map[string]any, error) {
	var rules any
	if err := json.Unmarshal(c.ProposedRules, &rules); err != nil {
		return nil, fmt.Errorf("proposed_rules: %w", err)
	}
	var limits any
	if len(c.EvaluatorLimits) > 0 {
		if err := json.Unmarshal(c.EvaluatorLimits, &limits); err != nil {
			return nil, fmt.Errorf("evaluator_limits: %w", err)
		}
	} else {
		return nil, fmt.Errorf("evaluator_limits required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	pack := map[string]any{
		"schema_version":      "detection_pack.v1",
		"pack_id":             c.PackID,
		"pack_version":        c.PackVersion,
		"created_at":          now,
		"min_agent_version":   c.MinAgentVersion,
		"supported_os":        toAnySlice(c.SupportedOS),
		"mode":                "observe",
		"expires_at":          nil,
		"rollback_pack_version": nil,
		"evaluator_limits":    limits,
		"rules":               rules,
	}
	if c.Author != "" {
		pack["author"] = c.Author
	}
	if c.Source != "" {
		pack["source"] = c.Source
	}
	if len(c.References) > 0 {
		var refs any
		if err := json.Unmarshal(c.References, &refs); err == nil {
			pack["references"] = refs
		}
	}
	// Placeholder; signing step replaces this.
	pack["signature"] = map[string]any{
		"algorithm":  "ed25519",
		"key_id":     "unsigned",
		"value_b64":  "AA==",
	}
	return pack, nil
}

func toAnySlice(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}
