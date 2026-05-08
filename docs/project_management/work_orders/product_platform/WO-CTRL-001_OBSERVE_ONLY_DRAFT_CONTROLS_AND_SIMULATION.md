# WO-CTRL-001: Observe-Only Draft Controls and Simulation

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** Backend / UI  

## Goal

Turn findings into draft controls that are explainable and simulated before any enforcement work begins.

## Scope

- Observe-only draft controls.
- Historical match simulation.
- UI review workflow.

## Deliverables

- Draft control model:
  - id
  - source finding/investigation
  - proposed action
  - scope selectors
  - evidence references
  - expected effect
  - blast radius
  - rollback plan
  - status
- Simulation endpoint that runs draft scope against historical events.
- UI page under Control > Controls.
- Finding detail action: "Draft observe-only control".

## Acceptance Criteria

- A finding can generate a draft control.
- User can see exactly which historical events would match.
- UI clearly says enforcement is not active.
- Draft can be accepted, rejected, or left pending.
- All actions are auditable.

## Dependencies

- WO-PLAT-002
- WO-DET-001

## Non-Goals

- No WFP/eBPF/nftables enforcement.
- No automatic approval.

