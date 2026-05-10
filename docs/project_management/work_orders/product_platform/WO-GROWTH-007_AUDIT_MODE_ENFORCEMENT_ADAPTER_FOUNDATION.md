# WO-GROWTH-007: Audit-Mode Enforcement Adapter Foundation

**Status:** Draft  
**Phase:** Product Growth / Safe Enforcement  
**Primary owner:** Backend / Agent / Policy  

## Goal

Create the foundation for future enforcement by adding an audit-mode adapter path that can evaluate policy effects without blocking traffic or changing endpoint behavior.

## Why This Matters

AegisFlux's promise ends with Enforce, but enforcement must be earned. Audit-mode adapters let the product prove:

- a control can be represented safely,
- an endpoint can receive it,
- matches can be observed,
- rollback can be described,
- no blocking occurs yet.

This is the bridge from control design to safe change management.

## Scope

- Audit-mode policy bundle model.
- Agent-side receipt/status contract.
- Backend staging and status visibility.
- No blocking or deny behavior.

## Deliverables

- Define signed audit-mode policy bundle shape:
  - bundle id/version
  - mode: audit-only
  - scope
  - expected match telemetry
  - expiration
  - rollback metadata
  - approval references
- Add backend staging route for audit-mode bundles.
- Add agent status contract for accepted/rejected/stale/incompatible audit bundle.
- Add one lab adapter target:
  - Linux audit-only path, or
  - Windows audit-only path
- Add UI visibility:
  - staged bundles
  - endpoint compatibility
  - audit match events
  - rollback/readiness notes
- Add safety docs explaining why this is not enforcement.

## Acceptance Criteria

- Audit-mode bundle can be staged without blocking behavior.
- Endpoint status reports accepted/rejected/incompatible/stale.
- UI clearly distinguishes audit-mode from enforcement.
- Operational events record staging and status changes.
- Relevant backend/agent tests pass.
- Repeatable lab smoke documented.

## Dependencies

- WO-GROWTH-003
- WO-GROWTH-004
- WO-DET-003 pattern for signed rollout/status
- WO-PLAT-006

## Non-Goals

- No deny/block/quarantine.
- No production enforcement.
- No broad adapter matrix; one lab path is enough.

## Suggested Verification

- Backend tests for bundle staging/status.
- Agent tests for audit-mode receipt where available.
- Lab smoke showing audit-mode match telemetry and no blocking.

