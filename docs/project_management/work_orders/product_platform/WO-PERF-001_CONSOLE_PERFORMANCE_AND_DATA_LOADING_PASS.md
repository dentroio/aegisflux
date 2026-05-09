# WO-PERF-001: Console Performance and Data Loading Pass

**Status:** Implemented (measurement + targeted fixes)  
**Phase:** Product Platform Performance  
**Primary owner:** UI / Backend  

## Goal

Make the AegisFlux console feel fast and predictable by reducing unnecessary fetches, bounding rendered data, and documenting measurable performance baselines.

## Problem

The console currently mixes high-level workbench pages with large retained telemetry queries. Even after the first responsiveness pass, several risks remain:

- Agent detail still fetches many endpoint-specific datasets.
- Dashboard, inventory, detections, and events rely on client-side shaping of backend records.
- Large tables can become expensive if future changes bypass bounded rendering.
- Polling behavior may still be heavier than needed for a lab/operator console.

## Scope

- Console routes and components in `ui/console`.
- Existing backend visibility/action APIs only, unless a small read-only helper endpoint is clearly needed and documented.
- Performance measurement, not speculative rewrites.

## Deliverables

- Measure current load and interaction behavior for:
  - dashboard
  - agents workbench
  - agent detail
  - inventory
  - detections
  - operational events
- Identify top slow fetches and heavy render paths.
- Reduce unnecessary polling/fetch fan-out where safe.
- Ensure all major table/list surfaces are bounded and filterable.
- Add or update documentation describing:
  - fetch cadence by route
  - default API limits by route
  - known hotspots that should move to summary APIs
- If backend summary endpoints are needed, file or update WO-API-001 rather than burying broad backend work here.

## Acceptance Criteria

- `npm run build` passes in `ui/console`.
- Route smoke checks pass for `/`, `/agents`, `/agents/[device_id]`, `/inventory`, `/detections`, and `/operate/events`.
- No primary route makes large background visibility fetches while it is not the active panel.
- Table/list surfaces have bounded initial rendering or a documented reason they do not.
- Performance notes include before/after observations and remaining bottlenecks.
- The work order is updated with implementation notes and verification results.

## Dependencies

- WO-QA-001 is recommended first but not strictly required.
- WO-UX-001 through WO-UX-005.

## Non-Goals

- Do not introduce a new global state framework.
- Do not rewrite the backend storage layer.
- Do not hide important operator context just to improve numbers; improve summaries and drill-ins instead.

## Suggested Verification

- `npm run build` in `ui/console`.
- Route smoke checks against a local dev server.
- Browser devtools or script-based timing notes for the routes listed above.

## Implementation notes

### Before → after (network fan-out)

| Surface | Before (typical) | After |
|--------|-------------------|--------|
| Dashboard | 7 parallel visibility/action fetches | 1 `GET /api/visibility/summary/dashboard` |
| Agents workbench | `GET /agents` + `GET /visibility/devices` | 1 `GET /api/actions/console/summary/agents-workbench` (actions-api merges ingest devices) |
| Agent detail | 10 parallel visibility queries | 1 `GET /api/visibility/summary/device?device_id=…` (legacy fan-out retained as fallback if non-OK) |
| Inventory | 6 parallel visibility queries | 1 `GET /api/visibility/summary/inventory` (legacy fallback if non-OK) |

### Fetch cadence / background work

- **Dashboard:** 60s refresh; callback no-ops while `document.visibilityState === 'hidden'` so background tabs do not refresh.
- **Agents workbench:** 60s refresh with the same visibility guard.
- **Embedded panels:** Agents and inventory only mount when their home panel is active, so the dashboard route does not poll workbench APIs in the background.

### Remaining hotspots

- Agent detail still hydrates full tab datasets from the device summary payload (one round trip; body size unchanged vs the old parallel calls).
- Workbench summary depends on `INGEST_API_URL` from actions-api when merging visibility; misconfiguration yields agents without `visibility` (empty lab is still OK).

### Verification

- `npm run build` in `ui/console`; `npm run test:e2e` for route smoke and shell checks.

