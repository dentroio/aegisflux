# WO-GROWTH-007: Audit-Mode Enforcement Adapter Foundation

**Status:** Done  
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

## Implementation Notes (Done)

### Backend (`backend/actions-api/internal/api`)

- Extended `PlatformData` to include `AuditBundles []AuditBundle` and added a bounded
  `appendAuditBundle` helper.
- New types in `platform_state.go`: `AuditBundle`, `AuditBundleStatus`, `AuditBundleMatch`, and
  `AuditBundleEvent`. Bundle fields cover scope, expected match telemetry, approval refs,
  rollback notes, source candidate/draft id, expiration, status, endpoint statuses, matches, and
  history.
- New `audit_bundles.go` enforces an observe-only contract:
  - `mode` must be `audit`; any other mode returns `400`.
  - Lifecycle: `draft -> staged -> revoked` with `expired` reserved for time-based transitions.
  - Endpoints report `pending | accepted | rejected | incompatible | stale`; any other value is
    rejected with `400`.
  - Matches are accepted only while the bundle is `staged`; after revoke, match calls return
    `409`.
  - Each transition appends an entry to the bundle's `history` and emits a typed
    `OperationalEvent` (`audit_bundle.created/updated/staged/endpoint_status/match/revoked`).
- Routes registered in `platform_routes.go`:
  - `GET/POST /platform/audit-bundles`
  - `GET/PATCH /platform/audit-bundles/{id}`
  - `POST /platform/audit-bundles/{id}/{stage|status|match|revoke}`
- Tests in `audit_bundles_test.go` cover mode enforcement, title required, full lifecycle
  (create → stage → endpoint accept → record match → revoke → match rejected), unsupported
  endpoint statuses, and the `summarizeEndpointStatuses` aggregator.

### Frontend (`ui/console`)

- New `app/control/audit-bundles/page.tsx` route gated by lab auth, with a prominent
  `Audit-only` banner that links to the safety doc.
- New `components/AuditBundlesPanel.tsx` renders:
  - A summary strip (totals, staged, accepted endpoints, endpoints with matches).
  - A table per bundle with status, endpoint stats, match counts, and inline actions
    (`Stage`, `Report status`, `Record match`, `Revoke`).
  - A detail modal showing scope, expected telemetry, approval refs, rollback notes, full
    endpoint table, recent matches, history, and an `Audit-only` disclaimer.
  - A create-bundle modal that defaults `mode` to `audit` and starts the bundle in `draft`.
- Added a header link from `/control/controls` to `/control/audit-bundles` so operators can pivot
  from draft controls into audit-mode staging.
- `npm run build` passes; no new lints introduced. The new route is built statically.

### Documentation

- New `docs/safety/AUDIT_MODE.md` explains why this is not enforcement: contract refuses any
  other mode, no allowed deny/block/quarantine, time-bounded via expiration, explicit
  revocation, and Audit-only labelling throughout the UI.
- New `docs/safety/AUDIT_MODE_BUNDLE_CONTRACT.md` defines the wire-level shape of bundles,
  endpoint receipt/status reporting, match reporting, and operational events.
- New `docs/demos/AUDIT_MODE_LAB_SMOKE.md` is a repeatable lab smoke that creates a bundle,
  stages it, reports an endpoint accept, records a match, verifies operational events, then
  revokes and confirms the post-revoke match is rejected with `409`.

### Acceptance Criteria

- Audit-mode bundle can be staged without blocking behavior: `POST /stage` flips the status to
  `staged`; the contract refuses any non-`audit` mode at create time.
- Endpoint status reports `accepted | rejected | incompatible | stale` (pending is the
  default while waiting); enforced by an explicit allow-list and unit-tested.
- UI clearly distinguishes audit-mode from enforcement: `Audit-only` health badge, banner,
  detail-modal disclaimer, and inline link to the safety doc.
- Operational events record staging and status changes: `audit_bundle.*` events emitted on
  every transition and surfaced in the bundle history and the operational-events stream.
- Relevant backend tests pass (`go vet ./...` and audit-bundle test suite). Repeatable lab
  smoke documented in `docs/demos/AUDIT_MODE_LAB_SMOKE.md`.
- Endpoint receipt/match contract documented in
  `docs/safety/AUDIT_MODE_BUNDLE_CONTRACT.md` so a future Linux/Windows agent change can
  implement the receipt path without ambiguity.

