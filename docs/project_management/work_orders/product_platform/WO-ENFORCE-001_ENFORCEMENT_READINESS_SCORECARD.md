# WO-ENFORCE-001: Enforcement Readiness Scorecard

**Status:** Planned  
**Phase:** AI-Native Leap / Safe Enforcement Readiness  
**Primary owner:** Control / Agent / Governance  

## Goal

Add an enforcement readiness scorecard that explains what evidence, simulation, agent health, approval, rollback, and audit-mode results are still missing before any control could be considered for blocking enforcement.

## Problem

AegisFlux will eventually enforce, but trust depends on showing why a control is or is not ready. Operators need a readiness model that prevents premature enforcement and turns gaps into concrete next steps.

## Scope

- Draft controls.
- Audit-mode bundles and match results.
- Agent readiness/health.
- Simulation outcomes.
- Approval and rollback metadata.

## Deliverables

- Define readiness dimensions:
  - evidence completeness,
  - evidence confidence,
  - agent readiness,
  - detection-pack/status freshness,
  - simulation quality,
  - audit-mode observation,
  - rollback plan,
  - approval state,
  - blast-radius acceptability.
- Add backend readiness calculation for draft controls and audit bundles.
- Add UI scorecard with blockers, warnings, and next actions.
- Add explicit "not enforcement" language and safety boundaries.

## Acceptance Criteria

- Scorecard can explain why a control is not ready.
- Readiness does not enable blocking behavior.
- Missing prerequisites are actionable and evidence-linked.
- Audit-mode results improve readiness only when matches and rollback data are sufficient.

## Dependencies

- WO-GROWTH-003.
- WO-GROWTH-004.
- WO-GROWTH-007.
- WO-CONTROL-002 recommended.
- WO-GOV-001 recommended.

## Non-Goals

- No production enforcement.
- No automatic promotion from audit to blocking.
- No hidden policy publish.

## Suggested Verification

- Tests for readiness scoring buckets and blockers.
- Manual scorecard check for a draft-only control and an audit-mode bundle.
