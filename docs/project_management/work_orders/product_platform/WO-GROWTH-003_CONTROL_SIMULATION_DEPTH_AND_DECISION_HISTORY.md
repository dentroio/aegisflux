# WO-GROWTH-003: Control Simulation Depth and Decision History

**Status:** Done  
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

- Backend `DraftControl` extended with:
  - `history` slice of `DraftDecisionEntry` records (action, actor, status, changed keys, note, before/after snapshots, optional simulation id).
  - `simulations` slice of `DraftSimulationResult` records (match count, matched device ids, matched users, top process paths, top destinations, window, confidence, expected breakage, scope snapshot, summary).
- New helpers in `platform_misc.go`:
  - `snapshotDraft`, `diffSnapshotKeys`, `decideHistoryAction` for compact diff/lifecycle classification.
  - `appendDraftHistory` (bounded at 50 entries, auto-stamped).
  - `buildSimulationResult` produces a deterministic observe-only projection from the draft + device id so repeat simulations are stable in the lab.
- Updated draft handlers:
  - `POST /platform/draft-controls` seeds the initial `created` history entry with an after-snapshot, then emits `draft_control.created`.
  - `PATCH /platform/draft-controls/{id}` records a typed history entry (`scope_edited`, `note_added`, `status_changed`, `updated`) with full before/after snapshots, accepts new `blast_radius_notes`, `rollback_steps`, and `decision_note` fields, and emits the matching operational event.
  - `POST /platform/draft-controls/{id}/simulate` builds a `DraftSimulationResult`, appends both the simulation record and a `simulated` history entry, returns the projection in the response, and emits a richer `draft_control.simulated` event.
- Backend tests in `draft_history_test.go`:
  - `TestDiffSnapshotKeys_DetectsScopeAndNotes`
  - `TestDiffSnapshotKeys_DetectsStatus`
  - `TestDecideHistoryAction`
  - `TestBuildSimulationResult_ProducesObservableProjection` (asserts observe-only mode, positive count, top lists, deterministic re-run)
  - `TestAppendDraftHistory_BoundedAndStamped`
- Console `FindingToControlPanel`:
  - "Decision note for the simulation run" input next to Save/Simulate buttons.
  - New `SimulationCard` displays the latest projection: matched events, matched devices, matched users, top process paths, top destinations, window, confidence, expected breakage.
  - New `DecisionHistorySection` lists the most recent 8 history entries with action chip, status, changed keys, optional operator note, and a collapsible before/after JSON view for compare-before-approve.
  - Saved-draft state widened to carry `history`, `simulations`, and the latest simulation.

## Acceptance Criteria

- [x] Operator can understand the blast radius of a draft without raw telemetry — SimulationCard surfaces matched events, devices, users, top process paths, top destinations, and window.
- [x] Draft changes are recorded in decision history — `DraftDecisionEntry` written on create/edit/simulate with before/after snapshots.
- [x] Simulation can be rerun after scope edits — patch records `scope_edited`; rerun appends a new simulation record and history entry.
- [x] UI clearly says observe-only and not enforcing — existing observe-only callout plus mode badge in SimulationCard.
- [x] Relevant backend tests pass — see `draft_history_test.go`.
- [x] `npm run build` passes in `ui/console`.

## Dependencies

- WO-PROD-003
- WO-PROD-002
- WO-PLAT-006

## Non-Goals

- No active enforcement.
- No policy bundle rollout.
- No multi-tenant approval system.

## Suggested Verification

- `cd backend/actions-api && go test ./internal/api -run "TestDiffSnapshotKeys|TestDecideHistoryAction|TestBuildSimulationResult|TestAppendDraftHistory"`.
- `cd ui/console && npm run build`.
- Manual walkthrough: create draft → simulate → edit scope → resimulate → inspect history.
