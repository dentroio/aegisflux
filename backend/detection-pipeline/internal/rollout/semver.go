package rollout

import (
	"fmt"
	"strconv"
	"strings"
)

// SemverGE returns true if agentVersion >= minVersion (both "major.minor.patch" style).
func SemverGE(agentVersion, minVersion string) bool {
	if minVersion == "" {
		return true
	}
	a, errA := parseSemver(agentVersion)
	b, errB := parseSemver(minVersion)
	if errA != nil || errB != nil {
		return agentVersion >= minVersion
	}
	if a[0] != b[0] {
		return a[0] > b[0]
	}
	if a[1] != b[1] {
		return a[1] > b[1]
	}
	return a[2] >= b[2]
}

// SemverGT returns true if a > b.
func SemverGT(a, b string) bool {
	aa, errA := parseSemver(a)
	bb, errB := parseSemver(b)
	if errA != nil || errB != nil {
		return a > b
	}
	if aa[0] != bb[0] {
		return aa[0] > bb[0]
	}
	if aa[1] != bb[1] {
		return aa[1] > bb[1]
	}
	return aa[2] > bb[2]
}

func parseSemver(v string) ([3]int, error) {
	var out [3]int
	v = strings.TrimSpace(v)
	if v == "" {
		return out, fmt.Errorf("empty")
	}
	// Strip pre-release suffix for ordering (lab).
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			break
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return out, err
		}
		out[i] = n
	}
	return out, nil
}
