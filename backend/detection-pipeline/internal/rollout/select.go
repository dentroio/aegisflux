package rollout

import (
	"fmt"
	"strings"

	"aegisflux/backend/detection-pipeline/internal/model"
)

// SelectLatestCompatible picks the newest signed artifact compatible with host OS and agent version.
// If packIDFilter is non-empty, only artifacts for that pack_id are considered.
func SelectLatestCompatible(artifacts []*model.SignedPackArtifact, hostOS, agentSemver, packIDFilter string, pol Policy) (*model.SignedPackArtifact, *PackMeta, error) {
	var bestArt *model.SignedPackArtifact
	var bestMeta *PackMeta
	for _, art := range artifacts {
		if art == nil {
			continue
		}
		meta, err := ParsePackMeta(art)
		if err != nil {
			continue
		}
		if !pol.AllowPack(meta.PackID) {
			continue
		}
		if packIDFilter != "" && meta.PackID != packIDFilter {
			continue
		}
		if !meta.Compatible(hostOS, agentSemver) {
			continue
		}
		if bestArt == nil {
			bestArt, bestMeta = art, meta
			continue
		}
		// Prefer higher semver for same pack_id; if tie, newer artifact wins.
		if meta.PackID == bestMeta.PackID {
			if SemverGT(meta.PackVersion, bestMeta.PackVersion) {
				bestArt, bestMeta = art, meta
			} else if meta.PackVersion == bestMeta.PackVersion && art.CreatedAtMS > bestArt.CreatedAtMS {
				bestArt, bestMeta = art, meta
			}
			continue
		}
		// Different pack_ids: prefer newer signing time (lab default).
		if art.CreatedAtMS > bestArt.CreatedAtMS {
			bestArt, bestMeta = art, meta
		}
	}
	if bestArt == nil {
		return nil, nil, fmt.Errorf("no compatible signed pack for os=%s agent=%s", hostOS, agentSemver)
	}
	return bestArt, bestMeta, nil
}

// FindArtifactByPackID returns the artifact for pack_id and optional exact version (semver string).
// If version is empty, returns latest compatible for default OS/linux and agent 0.0.0 - not ideal.
// Use FindArtifactByPackIDVersion with explicit version or use SelectLatestCompatible with filter.
func FindArtifactByPackAndVersion(artifacts []*model.SignedPackArtifact, packID, version string) (*model.SignedPackArtifact, *PackMeta, error) {
	var best *model.SignedPackArtifact
	var bestMeta *PackMeta
	for _, art := range artifacts {
		meta, err := ParsePackMeta(art)
		if err != nil {
			continue
		}
		if !strings.EqualFold(meta.PackID, packID) {
			continue
		}
		if version != "" && meta.PackVersion != version {
			continue
		}
		if version != "" {
			return art, meta, nil
		}
		if best == nil {
			best, bestMeta = art, meta
			continue
		}
		if SemverGT(meta.PackVersion, bestMeta.PackVersion) ||
			(meta.PackVersion == bestMeta.PackVersion && art.CreatedAtMS > best.CreatedAtMS) {
			best, bestMeta = art, meta
		}
	}
	if best == nil {
		return nil, nil, fmt.Errorf("pack %q version %q not found", packID, version)
	}
	return best, bestMeta, nil
}
