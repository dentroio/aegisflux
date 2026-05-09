# WO-API-001: Console Summary Endpoints

**Status:** Implemented  
**Phase:** Product Platform API  
**Primary owner:** Backend / UI  

## Goal

Add read-only summary endpoints that let the console load meaningful operator views without composing large telemetry datasets in the browser.

## Problem

AegisFlux has useful visibility data, but the UI often has to fetch several raw datasets and aggregate them client-side. That makes pages slower, duplicates logic, and encourages screens to expose raw telemetry names instead of operator concepts.

## Scope

- Backend read APIs that summarize existing stored visibility/action data.
- UI migration for one or more high-impact routes.
- Contracts and tests for summary payload shape.

## Deliverables

- Define and implement initial summary endpoint candidates:
  - dashboard summary
  - agents workbench summary
  - agent detail summary
  - inventory category summary
- Each summary should answer operator questions directly:
  - Which agents need attention?
  - Which endpoint is stale or unhealthy?
  - What AI/control activity matters?
  - What findings require triage?
  - What is the next best action?
- Keep raw telemetry available through existing drill-in APIs.
- Add backend tests for summary shape and empty-data behavior.
- Update the UI to consume at least one summary endpoint if scope permits.
- Document endpoint contracts and any migration left for later work orders.

## Acceptance Criteria

- Summary APIs are read-only.
- Empty lab data returns useful zero/empty summaries rather than errors.
- Backend tests pass for added summary code.
- `npm run build` passes in `ui/console` if UI changes are made.
- The UI no longer needs to fan out to raw datasets for at least one high-impact summary surface, or the work order documents why migration is deferred.
- The work order is updated with implementation notes and verification results.

## Dependencies

- WO-PERF-001 findings should guide endpoint priority.
- Existing visibility/action APIs and stores.

## Non-Goals

- Do not remove raw telemetry endpoints.
- Do not add write/enforcement behavior.
- Do not couple AegisFlux to Clarion storage.
- Do not implement long-term analytics warehousing in this slice.

## Suggested Verification

- Relevant backend unit/integration tests.
- `go test ./...` for touched Go modules where practical.
- `npm run build` in `ui/console` if UI code changes.

## Implementation notes

### Ingest (read-only) summaries — `backend/ingest`

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/v1/visibility/summary/dashboard` | Devices list + pre-aggregated dashboard model (replaces multi-fetch client aggregation for the home dashboard). |
| GET | `/v1/visibility/summary/device?device_id=` | Single bundle: devices list, events, processes, flows, dns, findings, typed extension/sase/collector/performance event streams. |
| GET | `/v1/visibility/summary/inventory` | Single bundle for inventory workbench (devices, extension/sase events, dns, processes, findings). |

Proxied from the console as `/api/visibility/summary/...` via existing `next.config.js` rewrites.

### Actions API — workbench merge

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/console/summary/agents-workbench` | Same JSON shape as `GET /agents`, with optional `visibility` on each agent populated from ingest (`INGEST_API_URL`, default `http://localhost:9091`). |

Proxied as `GET /api/actions/console/summary/agents-workbench`.

### Tests

- `backend/ingest/internal/server/summary_handlers_test.go` — empty dashboard summary + device `device_id` validation.
- Run with `CGO_ENABLED=0 go test ./internal/server/...` if the platform linker rejects cgo.

### Verification

- `npm run build` in `ui/console`; console routes consume the new endpoints as described in WO-PERF-001 notes.

