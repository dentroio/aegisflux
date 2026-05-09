# WO-PROD-003: Finding-to-Control Design Workflow

**Status:** Draft  
**Phase:** Product Differentiation  
**Primary owner:** Product / Backend / UI  

## Goal

Turn findings into useful observe-only draft controls with evidence, proposed scope, expected blast radius, and rollback notes.

## Why This Makes Aegis Stand Out

AegisFlux should not stop at "something happened." The product should help operators decide what safe control would reduce risk and why. That moves Aegis from telemetry into control design.

This is the core product loop: evidence becomes a safe, explainable control proposal.

## Scope

- Finding detail action to create or review draft control.
- Draft-control model refinement.
- Simulation inputs and blast-radius preview.
- Operator decision notes.

## Deliverables

- Add draft-control proposal shape:
  - proposed action
  - scope fields
  - evidence references
  - confidence
  - expected matches
  - expected breakage risk
  - rollback note
  - status
  - operator notes
- Generate first proposals from:
  - AI-related findings
  - suspicious automation
  - known model gateway access
  - local model runtime exposure
- Add UI flow:
  - finding summary
  - proposed control
  - evidence path
  - simulation/blast-radius preview
  - save as observe-only draft
- Add audit/operational event when a draft is created or updated.

## Acceptance Criteria

- Operator can create an observe-only draft control from a finding.
- Draft shows why the scope was chosen.
- Draft shows what historical activity would have matched.
- Draft makes clear that enforcement is not active.
- Draft has a rollback note even if rollback execution is future work.
- Relevant backend tests pass.
- `npm run build` passes in `ui/console` if UI changes are made.

## Dependencies

- WO-CTRL-001
- WO-PROD-002 strongly recommended
- WO-PLAT-006 for operational event feed

## Non-Goals

- No active blocking.
- No policy deployment to endpoint agents.
- No approval workflow beyond draft/notes in the first slice.

## Suggested Verification

- Backend tests for draft-control creation and simulation inputs.
- UI route checks for controls and finding-driven draft flow.

