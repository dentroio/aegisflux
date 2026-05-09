# WO-PROD-003: Finding-to-Control Design Workflow

**Status:** Done  
**Phase:** Product Differentiation  
**Primary owner:** Backend / UI  

## Goal

Turn findings into useful observe-only draft controls with evidence, proposed scope, expected blast radius, and rollback notes. The draft is a piece of design output, not a runtime change.

## Why This Makes Aegis Stand Out

This is where AegisFlux moves from telemetry to control design. The product captures *why a finding exists* and *how a thoughtful operator would respond*, with rollback. That is the value above SIEM and EDR, even before any enforcement plane lands.

## Scope

- Backend draft control state with evidence reference, proposed action, scope, blast radius, rollback notes, observe-only flag, operator notes.
- Backend simulation of expected match counts using lab telemetry only.
- UI flow that goes from finding → evidence → proposed control → simulation → save as observe-only draft.
- Audit trail of created and updated drafts.

## Deliverables

- Extended `DraftControl` model in `backend/actions-api/internal/api/platform_state.go` with:
  - `source_finding_title`, `source_device_id`,
  - `confidence`, `expected_matches`, `expected_breakage_risk`,
  - `blast_radius_notes`, `rollback_steps`,
  - `operator_notes`, `updated_at_ms`, `simulation_device_id`, `simulation_at_ms`.
- `POST /platform/draft-controls` (existing route) now seeds default scope, blast radius, blast-radius notes, and rollback steps when a payload is sparse, emits a `draft_control.created` operational event tagged with the source device, and returns the full draft body.
- New `GET /platform/draft-controls/{id}` returns a single draft for inspection.
- New `PATCH /platform/draft-controls/{id}` updates operator notes, status, scope, blast radius, breakage risk, and rollback plan, emitting a `draft_control.updated` operational event.
- `POST /platform/draft-controls/{id}/simulate` now records the simulation device and timestamp on the draft and emits an enriched `draft_control.simulated` event.
- New `FindingToControlPanel` component in the console:
  - inputs for finding id, source device, simulation device,
  - reads the evidence path through `/api/visibility/evidence-path` and seeds the proposal from the synthesized draft control,
  - editable proposal form with proposed action, scope selectors, blast radius summary + notes, rollback plan + steps, expected breakage risk, operator notes, confidence,
  - "Save observe-only draft" → POST to actions-api,
  - "Simulate blast radius" → POST `/{id}/simulate` and shows projected match count on the saved draft,
  - amber observe-only banner reinforcing that drafts do not enforce policy.
- `/control/controls` page reorganized:
  - "Finding-to-control designer" mode (default) renders the new panel,
  - "Draft queue" mode keeps the legacy refresh / simulate / detail flow and gains an operator-notes modal with status update (`draft_observe_only`, `draft_in_review`, `draft_archived`),
  - draft rows display source device id, finding title, and confidence,
  - `Suspense` wrapper added so the route can read `?finding_id=` / `?device_id=` query params from deep links.
- Endpoint detail page Policy tab now deep-links into the designer with `?finding_id=…&device_id=…` for the lead finding.

## Acceptance Criteria

- [x] An operator can go from finding → draft control without leaving the console (designer panel auto-loads evidence, seeds the proposal, accepts edits, saves, simulates).
- [x] Each draft includes evidence references, scope, blast radius (summary + notes), and rollback (plan + steps); a saved draft also carries source device, confidence, breakage risk, operator notes, simulation device, and timestamps.
- [x] Simulation runs against lab telemetry only (`simulateMatches` is deterministic against `device_id+draft_id`) and the result is logged on the draft and as an operational event.
- [x] All drafts default to observe-only (`draft_observe_only` status, `expected_effect: observe_only`, breakage risk reads "low (observe-only; no enforcement)") and the UI reinforces this through banners.
- [x] `cd ui/console && npm run build` passes.

## Dependencies

- WO-PROD-002 (evidence path supplies the seed data for proposals)
- Existing actions-api state and simulation primitives

## Non-Goals

- No enforcement.
- No policy engine integration.
- No multi-tenant draft sharing.

## Suggested Verification

- `cd backend/actions-api && go vet ./...` (backend build/test parity in CI).
- `cd ui/console && npm run build`.
- Manual walkthrough: open `/control/controls`, default into the designer, paste a lab finding id (or pre-seed via `?finding_id=…`), click Load evidence → Save observe-only draft → Simulate. Confirm the queue mode shows the new draft with source device, confidence, simulation count, and supports operator notes via the Notes button.
