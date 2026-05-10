# WO-GROWTH-003: Control Simulation Depth and Decision History

**Status:** Draft  
**Phase:** Product Growth / Control Design  
**Primary owner:** Backend / UI  

## Goal

Deepen finding-to-control simulation so operators can trust draft controls before any future enforcement work begins.

## Why This Matters

AegisFlux should stand out by reducing blast radius before enforcement. A draft control is only useful if the operator can see:

- what would match,
- who or what would be affected,
- why the scope was selected,
- what changed after edits,
- who decided what and when.

## Scope

- Draft-control simulation depth.
- Decision history.
- Compare-before-approve UX.
- Observe-only posture remains explicit.

## Deliverables

- Add richer simulation outputs:
  - matched event count
  - matched endpoints
  - matched users/sessions where available
  - top process paths
  - top destinations/domains
  - time-window summary
  - confidence and expected breakage risk
- Add draft revision history:
  - created
  - edited scope
  - simulated
  - reviewed
  - archived
- Add compare view between current and previous draft scope.
- Add operator decision notes and reviewer/status transitions.
- Add operational events for meaningful state transitions.
- Update control UI to keep simulation results compact and explainable.

## Acceptance Criteria

- Operator can understand the blast radius of a draft without raw telemetry.
- Draft changes are recorded in decision history.
- Simulation can be rerun after scope edits.
- UI clearly says observe-only and not enforcing.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console`.

## Dependencies

- WO-PROD-003
- WO-PROD-002
- WO-PLAT-006

## Non-Goals

- No active enforcement.
- No policy bundle rollout.
- No multi-tenant approval system.

## Suggested Verification

- Backend tests for simulation and draft history.
- `npm run build` in `ui/console`.
- Manual walkthrough: create draft, simulate, edit scope, resimulate, inspect history.

