# WO-GROWTH-005: Adaptive Detection Workflow Maturity

**Status:** Draft  
**Phase:** Product Growth / Adapt  
**Primary owner:** Detection / AI Platform / UI  

## Goal

Mature the research feed into an end-to-end adaptive detection workflow: opportunity, candidate, simulation, governance, signed pack, rollout visibility, and retirement.

## Why This Matters

The "Adapt" pillar is only compelling if the workflow feels governed and repeatable. Aegis should show how new AI-agent behavior becomes safe observe-only detection without rebuilding endpoint agents.

## Scope

- Research opportunity lifecycle.
- Candidate quality gates.
- Simulation result display.
- Pack release/retirement governance.
- Rollout visibility cross-links.

## Deliverables

- Add quality gates:
  - required evidence fields
  - expected false-positive notes
  - simulation match count
  - expiration date
  - reviewer/approval notes
  - rollback/deprecation notes
- Link research opportunity -> candidate -> signed pack -> rollout status.
- Add detection workflow status board.
- Add candidate simulation summary in the UI.
- Add pack retirement/deprecation path for stale detections.
- Emit operational events for promote, approve, sign, deploy, retire.

## Acceptance Criteria

- Operator can follow a detection from research opportunity to rollout status.
- Candidate cannot appear release-ready without evidence requirements and simulation summary.
- Signed pack status links back to research rationale.
- Retired/deprecated packs are visible and auditable.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console`.

## Dependencies

- WO-PROD-004
- WO-DET-002
- WO-DET-003
- WO-PLAT-004
- WO-PLAT-006

## Non-Goals

- No automatic external internet research unless separately approved.
- No automatic endpoint enforcement.
- No replacing signed-pack trust model.

## Suggested Verification

- Backend tests for lifecycle transitions.
- `npm run build` in `ui/console`.
- Manual walkthrough from research feed to detection pack rollout.

