# WO-AI-002: AI Privacy and Audit Foundation

**Status:** Draft  
**Phase:** Product Platform  
**Primary owner:** Backend / UI  

## Goal

Make all AI-agent usage auditable and safe for endpoint evidence. AegisFlux must redact sensitive data before external provider calls and record every AI run.

## Scope

- Privacy settings.
- Redaction pipeline.
- AI run and audit records.
- UI management page.

## Deliverables

- Privacy settings:
  - sanitize external requests
  - audit external requests
  - store redacted preview only
  - redact IP addresses and CIDRs
  - redact MAC addresses
  - redact usernames, emails, and hostnames
  - redact command-line secrets
  - redact file paths
  - block raw secrets
- AI run record with provider, model, agent name, route, operation, status, duration, error.
- AI privacy audit record with redaction count and sanitized preview.
- UI under Configure > Settings > AI Privacy or Configure > Connectors > AI Providers.

## Acceptance Criteria

- Every AI request creates an auditable run record.
- External requests use redaction when enabled.
- Raw provider secrets and raw sensitive payloads are not exposed in UI.
- Operators can see recent AI audit records.
- Tests cover redaction examples for command lines, tokens, IPs, MACs, and usernames.

## Dependencies

- WO-AI-001

## Non-Goals

- No autonomous approval.
- No endpoint-side LLM calls.

