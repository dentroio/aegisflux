# WO-OPS-001: End-to-End Pipeline Validation

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Backend / Agent / QA  

## Goal

Prove the complete AegisFlux lab path works repeatably from endpoint signal to operator-visible evidence and safe control/audit-mode status.

## Problem

The major product and visibility slices exist, but the remaining risk is integration drift. Individual services and UI surfaces can pass while the full chain silently breaks between agents, ingest, summaries, detections, operational events, and audit-mode bundle status.

## Scope

- Windows and Linux lab agents.
- Actions API, ingest, detection pipeline, operational events, console summaries, and audit-mode bundle status.
- Repeatable smoke scripts and documented expected results.
- Observe-only and audit-only behavior only.

## Deliverables

- Define one canonical lab scenario for each path:
  - agent heartbeat and registration,
  - visibility event ingestion,
  - detection-pack status reporting,
  - finding or detection opportunity surfacing,
  - ABOM/inventory summary update,
  - audit-mode bundle stage/status/match/revoke.
- Add a single command or runbook that executes the canonical smoke checks.
- Capture expected API responses and UI surfaces for each scenario.
- Add automated assertions where practical and manual checkpoints where endpoint/VM state is required.
- Document the known failure modes and the first place to look for each.

## Acceptance Criteria

- A clean lab run can demonstrate endpoint signal through console visibility without code edits.
- The smoke checks clearly distinguish service-down, tunnel-down, stale-agent, and no-data states.
- Linux and Windows agent paths are both represented, even if one has a reduced manual checkpoint.
- Audit-mode validation confirms no blocking/enforcement behavior occurs.
- Results are documented with timestamps, commands, and pass/fail interpretation.

## Dependencies

- WO-VIS-001 through WO-VIS-010.
- WO-DET-006.
- WO-GROWTH-007.
- WO-OPS-002 is recommended first if lab agents are unstable.

## Non-Goals

- No production enforcement.
- No new product UI unless required to expose an existing missing status.
- No replacement of unit tests or route regression tests.

## Implementation notes (May 12, 2026)

- **Single command:** `scripts/lab/smoke-e2e-pipeline.sh` — runs health sweep (optional), actions `/readyz`, `GET /agents`, agents workbench + readiness summaries, ingest replay, ingest summary APIs, detection-pack status, operational events, research feed + detection candidates list, and full audit-bundle lifecycle (draft → staged → status → match → revoke). Optional `RUN_DETECTION_ROLLOUT=1` chains `smoke-detection-rollout.sh` (WO-DET-006).
- **Runbook:** `docs/ops/E2E_PIPELINE_SMOKE.md` — scenario table, manual console checklist, pass/fail matrix (service-down vs tunnel vs stale agent vs no-data), evidence capture guidance.
- **Automated assertions:** JSON shape checks via embedded `python3` in the shell script; non-blocking lab UIDs with `SKIP_LAB_AGENT_ASSERT=1` for API-only environments.

## Suggested Verification

- Run the new end-to-end smoke command or runbook from a fresh shell.
- Check `GET /agents`, console summary APIs, detection-pack status, and operational events.
- Perform a manual console walkthrough for dashboard, agents, inventory, detections, controls, and audit bundles.
