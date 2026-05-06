package rollout

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Policy is lab-only rollout configuration (WO-DET-003).
type Policy struct {
	LabOnlyEnabled   bool
	AllowedPackIDs   map[string]struct{}
	StaleAfterMS     int64
}

func LoadPolicyFromEnv() Policy {
	lab := strings.TrimSpace(os.Getenv("DETECTION_ROLLOUT_LAB_ONLY")) != "false"
	p := Policy{
		LabOnlyEnabled: lab,
		StaleAfterMS:   24 * time.Hour.Milliseconds(),
	}
	if raw := strings.TrimSpace(os.Getenv("DETECTION_ROLLOUT_ALLOWED_PACK_IDS")); raw != "" {
		p.AllowedPackIDs = make(map[string]struct{})
		for _, id := range strings.Split(raw, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				p.AllowedPackIDs[id] = struct{}{}
			}
		}
	}
	if v := strings.TrimSpace(os.Getenv("DETECTION_ROLLOUT_STALE_AFTER_MS")); v != "" {
		if ms, err := strconv.ParseInt(v, 10, 64); err == nil && ms > 0 {
			p.StaleAfterMS = ms
		}
	}
	return p
}

// AllowPack returns false if an allowlist is configured and packID is not listed.
func (p Policy) AllowPack(packID string) bool {
	if len(p.AllowedPackIDs) == 0 {
		return true
	}
	_, ok := p.AllowedPackIDs[packID]
	return ok
}
