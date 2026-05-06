package api

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"

	"aegisflux/backend/detection-pipeline/internal/model"
)

func (s *Server) maybeEmitDetectionPackStatus(ctx context.Context, st *model.AgentPackStatus, controllerBase string) error {
	if st.DeviceID == "" || st.AgentUID == "" {
		return nil
	}
	sv := strings.TrimSpace(st.ReportedAgentVersion)
	if sv == "" {
		sv = "0.0.0"
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(st.AgentUID))
	seq := int64(h.Sum32()%999_999_000) + 1
	ev := map[string]any{
		"schema_version": "visibility.v1",
		"event_id":       fmt.Sprintf("dps-%s-%d", st.AgentUID, st.UpdatedAtMS),
		"event_type":     "aegis.detection_pack.status",
		"timestamp_ms":   st.UpdatedAtMS,
		"source":         "aegis-detection-pipeline",
		"device_id":      st.DeviceID,
		"agent_id":       st.AgentUID,
		"sensor_version": sv,
		"sequence":       seq,
		"payload": map[string]any{
			"controller_endpoint":  strings.TrimSpace(controllerBase),
			"rollout_state":        string(st.RolloutState),
			"reason_codes":         st.ReasonCodes,
			"signature_status":     nonEmptyOr(st.SignatureStatus, "unknown"),
			"hash_status":          nonEmptyOr(st.HashStatus, "unknown"),
			"schema_status":        nonEmptyOr(st.SchemaStatus, "unknown"),
			"compatibility_status": nonEmptyOr(st.CompatibilityStatus, "unknown"),
		},
	}
	pl := ev["payload"].(map[string]any)
	if strings.TrimSpace(st.ReasonDetail) != "" {
		pl["reason_detail"] = st.ReasonDetail
	}
	if strings.TrimSpace(st.ActivePackID) != "" {
		pl["active_pack_id"] = st.ActivePackID
	}
	if strings.TrimSpace(st.ActivePackVersion) != "" {
		pl["active_pack_version"] = st.ActivePackVersion
	}
	if strings.TrimSpace(st.PreviousPackID) != "" {
		pl["previous_pack_id"] = st.PreviousPackID
	}
	if strings.TrimSpace(st.PreviousPackVersion) != "" {
		pl["previous_pack_version"] = st.PreviousPackVersion
	}
	body, err := json.Marshal([]any{ev})
	if err != nil {
		return err
	}
	return s.ingest.PostVisibilityEvents(ctx, body)
}

func nonEmptyOr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
