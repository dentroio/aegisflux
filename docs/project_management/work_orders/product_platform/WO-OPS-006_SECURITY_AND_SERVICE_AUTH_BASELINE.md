# WO-OPS-006: Security and Service Authentication Baseline

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Backend / Security / Agent  

## Goal

Establish the minimum security baseline for local/lab operation before any broader deployment work: authentication boundaries, secret handling, transport assumptions, and safe defaults.

## Problem

The platform now has agents, APIs, detection packs, audit bundles, and AI/provider settings. Before expanding beyond the lab, the project needs an explicit baseline for which calls are trusted, which require authentication, how secrets are handled, and where TLS/mTLS or signed artifacts are required next.

## Scope

- Agent registration/authentication path.
- Backend service-to-service calls.
- Console lab auth and management APIs.
- Detection pack and audit-mode bundle signing/verification assumptions.
- Local secrets and `.env` handling.

## Deliverables

- Document current trust boundaries for lab mode.
- Inventory unauthenticated or weakly authenticated endpoints and classify them:
  - acceptable lab-only,
  - needs token/API key,
  - future mTLS/TLS candidate,
  - should be local-only.
- Add or tighten authentication middleware for high-risk write endpoints where practical.
- Confirm signed detection-pack and audit-bundle contracts reject invalid modes/signatures where implemented.
- Document secret storage and environment variable expectations.
- Add security regression tests for the highest-risk auth checks.

## Acceptance Criteria

- The repo has a clear lab security baseline and a deployment-hardening gap list.
- High-risk write endpoints are either authenticated, local-only with documentation, or explicitly marked lab-only.
- Secrets are not required in committed files.
- Tests cover at least one denied unauthenticated write path and one signed-artifact rejection path where available.
- No endpoint behavior enables blocking/enforcement by default.

## Dependencies

- WO-AI-002.
- WO-DET-001 through WO-DET-003.
- WO-GROWTH-007.

## Non-Goals

- No production compliance certification.
- No full PKI rollout unless separately scoped.
- No production enforcement enablement.

## Implementation notes (May 12, 2026)

- **Baseline doc:** `docs/ops/LAB_SECURITY_BASELINE.md` — trust boundaries, endpoint classification table, secret handling, hardening gaps.
- **Regression tests:** ingest malformed-body test; actions-api workbench ingest-dependency surfacing test (documents lab “no silent empty merge” expectation).

## Suggested Verification

- Run backend auth/security tests for touched services.
- Attempt representative unauthenticated write calls and confirm expected denial or documented lab-only behavior.
- Verify no secrets are added to git status.
