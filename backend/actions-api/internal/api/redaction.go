package api

import (
	"regexp"
	"strings"
)

var (
	reIPv4 = regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(?:/\d{1,2})?\b`)
	reMAC  = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{2}[:-]){5}[0-9a-f]{2}\b`)
	reMail = regexp.MustCompile(`\b[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}\b`)
	rePath = regexp.MustCompile(`\B/(?:usr|home|var|etc|tmp|opt|bin|lib|Users|Program Files)(?:/[a-zA-Z0-9._-]+)+\b`)
	reTok  = regexp.MustCompile(`(?i)\bBearer\s+[a-zA-Z0-9._-]{8,}\b`)
	reKey  = regexp.MustCompile(`(?i)\b(?:api[_-]?key|token|secret)\s*[=:]\s*[a-zA-Z0-9._-]{8,}\b`)
)

// RedactLine applies lab privacy transforms for outbound strings (WO-AI-002).
func RedactLine(s string, p PrivacySettings) string {
	out := s
	if p.RedactSecrets {
		out = reTok.ReplaceAllString(out, "Bearer [REDACTED]")
		out = reKey.ReplaceAllString(out, "secret=[REDACTED]")
	}
	if p.RedactIPs {
		out = reIPv4.ReplaceAllString(out, "[REDACTED_IP]")
	}
	if p.RedactMACs {
		out = reMAC.ReplaceAllString(out, "[REDACTED_MAC]")
	}
	if p.RedactEmails {
		out = reMail.ReplaceAllString(out, "[REDACTED_EMAIL]")
	}
	if p.RedactUsers {
		out = regexp.MustCompile(`(?i)\buser(?:name)?\s*[:=]\s*\S+`).ReplaceAllString(out, "user: [REDACTED]")
	}
	if p.RedactHosts {
		out = regexp.MustCompile(`\b[a-z0-9.-]+\.(?:local|lan|corp|internal)\b`).ReplaceAllStringFunc(out, func(h string) string {
			if strings.Contains(h, " ") {
				return h
			}
			return "[REDACTED_HOST]"
		})
	}
	if p.RedactPaths {
		out = rePath.ReplaceAllString(out, "[REDACTED_PATH]")
	}
	if p.RedactCmd {
		out = regexp.MustCompile(`(?i)(--password|--token|--secret)\s+\S+`).ReplaceAllString(out, `$1 [REDACTED]`)
	}
	return out
}
