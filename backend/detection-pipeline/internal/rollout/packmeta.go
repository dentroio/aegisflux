package rollout

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aegisflux/backend/detection-pipeline/internal/model"
)

// PackMeta is parsed detection_pack.v1 metadata for controller decisions.
type PackMeta struct {
	PackID          string
	PackVersion     string
	MinAgentVersion string
	SupportedOS     []string
	Mode            string
	ExpiresAt       *time.Time
	SHA256Hex       string
}

// ParsePackMeta reads a signed artifact JSON body into rollout metadata.
func ParsePackMeta(art *model.SignedPackArtifact) (*PackMeta, error) {
	if art == nil || len(art.PackJSON) == 0 {
		return nil, fmt.Errorf("artifact missing pack json")
	}
	var doc struct {
		PackID          string   `json:"pack_id"`
		PackVersion     string   `json:"pack_version"`
		MinAgentVersion string   `json:"min_agent_version"`
		SupportedOS     []string `json:"supported_os"`
		Mode            string   `json:"mode"`
	}
	if err := json.Unmarshal(art.PackJSON, &doc); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(art.PackJSON)
	hexSum := hex.EncodeToString(sum[:])
	meta := &PackMeta{
		PackID:          doc.PackID,
		PackVersion:     doc.PackVersion,
		MinAgentVersion: doc.MinAgentVersion,
		SupportedOS:     doc.SupportedOS,
		Mode:            doc.Mode,
		SHA256Hex:       hexSum,
	}
	if art.SHA256Hex != "" {
		meta.SHA256Hex = art.SHA256Hex
	}
	var root map[string]any
	if err := json.Unmarshal(art.PackJSON, &root); err == nil {
		if v, ok := root["expires_at"]; ok && v != nil {
			if s, ok := v.(string); ok && s != "" {
				t, err := time.Parse(time.RFC3339Nano, s)
				if err != nil {
					t, err = time.Parse(time.RFC3339, s)
				}
				if err == nil {
					utc := t.UTC()
					meta.ExpiresAt = &utc
				}
			}
		}
	}
	return meta, nil
}

// Compatible reports whether the pack may run on hostOS with agentSemver.
func (m *PackMeta) Compatible(hostOS, agentSemver string) bool {
	hostOS = strings.ToLower(strings.TrimSpace(hostOS))
	if hostOS == "" {
		return false
	}
	ok := false
	for _, s := range m.SupportedOS {
		if strings.EqualFold(strings.TrimSpace(s), hostOS) {
			ok = true
			break
		}
	}
	if !ok {
		return false
	}
	if strings.TrimSpace(m.Mode) != "" && strings.TrimSpace(m.Mode) != "observe" {
		return false
	}
	if m.ExpiresAt != nil && !time.Now().UTC().Before(*m.ExpiresAt) {
		return false
	}
	return SemverGE(strings.TrimSpace(agentSemver), strings.TrimSpace(m.MinAgentVersion))
}
