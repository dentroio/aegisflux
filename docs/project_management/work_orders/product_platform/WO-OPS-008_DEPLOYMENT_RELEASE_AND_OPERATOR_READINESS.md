# WO-OPS-008: Deployment, Release, and Operator Readiness

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Platform / Docs / QA  

## Goal

Package the completed lab slices into a repeatable release candidate workflow that a new operator or developer can bring up, validate, and troubleshoot.

## Problem

The repo contains many implemented capabilities and docs, but a release candidate needs one coherent path: prerequisites, startup, configuration, health sweep, smoke validation, demo data, known limitations, and rollback/reset steps.

## Scope

- Local Docker/Compose or equivalent lab startup.
- Console startup and build validation.
- Linux and Windows lab agent connectivity assumptions.
- Demo and smoke runbooks.
- Release notes and known limitations.

## Deliverables

- Create a release-candidate checklist covering:
  - prerequisites,
  - environment variables,
  - service startup,
  - health sweep,
  - agent heartbeat validation,
  - e2e smoke,
  - console walkthrough,
  - reset/cleanup.
- Consolidate or link the key runbooks needed for a new operator.
- Add a known-limitations section that clearly distinguishes lab-only, observe-only, and audit-only behavior.
- Document how to capture evidence for a release candidate pass/fail.
- Identify CI/CD gaps and deployment automation follow-ups.

## Acceptance Criteria

- A new developer/operator can follow one checklist from clone to validated lab state.
- The checklist references the operational readiness work orders and their verification commands.
- Known limitations are explicit and do not imply production enforcement readiness.
- Release evidence includes service health, agent status, route smoke, and e2e smoke results.
- Reset/cleanup steps are documented.

## Dependencies

- WO-OPS-001 through WO-OPS-007.
- Existing demo docs in `docs/demos`.

## Non-Goals

- No production deployment guarantee.
- No Kubernetes-only migration.
- No packaging/signing beyond the current lab artifact model unless separately scoped.

## Implementation notes (May 12, 2026)

- **Checklist:** `docs/ops/RELEASE_CANDIDATE_CHECKLIST.md` links health sweep, agents check, replay, optional detection smoke, console build, known limitations, reset steps, and CI/CD gap notes.
- **Index:** `docs/ops/README.md` lists all ops docs and lab scripts.

## Suggested Verification

- Run the release-candidate checklist from a fresh shell.
- Confirm health sweep, agent status, e2e smoke, and console walkthrough all pass or produce clear known-limitations notes.
