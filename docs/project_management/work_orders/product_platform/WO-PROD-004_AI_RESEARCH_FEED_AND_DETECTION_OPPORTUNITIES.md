# WO-PROD-004: AI Research Feed and Detection Opportunities

**Status:** Done  
**Phase:** Product Differentiation  
**Primary owner:** Backend / UI  

## Goal

Build a research feed view that turns new AI ecosystem intelligence into governed detection opportunities (with scope, indicators, evidence requirements, and a draft pack id).

## Why This Makes Aegis Stand Out

This is the "Adapt" part of the AegisFlux story. AI is changing fast — new desktop apps, browser extensions, and agent protocols appear weekly. The product should make staying current feel governed, not chaotic. Each piece of research has a clear lifecycle, a draft pack, and an audit trail.

## Scope

- Backend research items with lifecycle (`new`, `scoped`, `ready_for_pack`, `promoted`, `declined`).
- Indicators, evidence requirements, suggested detection (logic, scope, confidence, expected noise, guard rails).
- Operator notes, status transitions, and promote-to-pack flow.
- Lab seeds covering local model runtime, coding agent, and browser AI extension categories.
- UI feed with category filters, status filter, search, detail modal, and operator actions.

## Deliverables

- New `ResearchItem`, `ResearchIndicator`, `ResearchSuggestedRule` types in `backend/actions-api/internal/api/platform_state.go`.
- Lab seeds in `seedResearchItems()` covering Ollama exposure, Claude Code MCP usage, and sideloaded browser AI extensions.
- New routes registered in `platform_routes.go`:
  - `GET /platform/research-feed` — list with optional `category` and `status` filters; returns total, status_counts, and category_counts.
  - `POST /platform/research-feed` — ingest a new item (manual / external feeder).
  - `GET /platform/research-feed/{id}` — single item.
  - `PATCH /platform/research-feed/{id}` — update status, operator notes, indicators, suggested detection, proposed pack id, risk score.
  - `POST /platform/research-feed/{id}/promote` — flip status to `promoted`, allocate a `proposed_pack_id`, and record the governance note (observe-only by default).
- Audit: `research.ingested`, `research.updated`, `research.promoted` operational events for every transition.
- New `ResearchFeedPanel` component:
  - KPI strip (total, new, scoped, ready_for_pack),
  - category filter buttons, status select, free-text search,
  - bounded research table with title, summary, indicator chips, status badge, source link, risk score, and per-row actions: Mark scoped, Ready for pack, Promote (with confirm), Notes, Decline,
  - read-only detail modal with indicators, evidence required, suggested detection (logic, scope, confidence, expected noise, guard rails), and a governance callout,
  - operator notes / status modal,
  - governance footer reinforcing observe-only and pointing to the finding-to-control designer.
- New `/analyze/research` route gated by lab auth, and the shell nav now lists "AI Research Feed" under Analyze.
- Dashboard "Next best actions" gains a deep link to the research feed.

## Acceptance Criteria

- [x] Research items move through a clear lifecycle (`new` → `scoped` → `ready_for_pack` → `promoted`/`declined`).
- [x] Each item lists indicators, evidence requirements, suggested detection logic and scope, and operator-editable notes.
- [x] Promotion creates a proposed pack id and records an `research.promoted` operational event with `observe_only: true`.
- [x] Three lab seeds make the feed feel real out of the box.
- [x] `cd ui/console && npm run build` passes.
- [x] `cd backend/actions-api && go vet ./...` passes.

## Dependencies

- WO-PROD-001 (provides ABOM context) helpful but not required.
- WO-PROD-003 (finding-to-control designer) for the cross-link.

## Non-Goals

- No automatic enabling of detections.
- No external feed ingestion pipeline (manual POST is sufficient for the lab story).
- No multi-tenant research scoping.

## Suggested Verification

- `cd backend/actions-api && go vet ./...`.
- `cd ui/console && npm run build`.
- Manual walkthrough: open `/analyze/research`, filter by category, mark an item scoped, mark another ready for pack, promote it, confirm it moves into the Promoted state with a draft pack id and that the operational events feed records the transitions.
