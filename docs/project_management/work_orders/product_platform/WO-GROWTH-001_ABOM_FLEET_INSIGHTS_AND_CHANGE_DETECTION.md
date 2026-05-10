# WO-GROWTH-001: ABOM Fleet Insights and Change Detection

**Status:** Done  
**Phase:** Product Growth / Differentiation  
**Primary owner:** Product / Backend / UI  

## Goal

Turn the Agent Bill of Materials from an inventory list into a daily-use fleet insight surface that highlights what is new, risky, widespread, or worth review.

## Why This Matters

ABOM is AegisFlux's clearest customer-facing wedge. The next version should not merely answer "what exists?" It should answer:

- What changed?
- What is new on sensitive endpoints?
- Which AI capabilities are spreading?
- Which tools have enough evidence to trust?
- Which findings should become controls?

This is how Aegis becomes a product people open every morning.

## Scope

- Fleet-level ABOM insight model.
- Change detection against prior observations.
- UI callouts for new/risky/widespread AI capability.
- Operator-centered language and empty states.

## Deliverables

- Backend insight aggregator in `backend/ingest/internal/server/abom_insights.go`:
  - Insight categories `newly_observed`, `newly_observed_high_attention`, `high_confidence`, `low_confidence_needs_review`, `widespread`, `stale`.
  - Time-window/change detection using `first_seen_ms` against a configurable `since_ms` cutoff (defaults to 24 hours) and stale detection against a configurable `stale_after_ms` cutoff (defaults to 7 days).
  - Endpoint hotspots ranked by high-attention status, total ABOM rows, and low-confidence pressure.
  - High-attention device set derived from devices with open findings.
  - Per-row `reason` strings written in operator language.
- New endpoint `GET /v1/visibility/abom/insights` returning sections, hotspots, fleet size, and the windows used to compute the response. Empty fleet returns helpful onboarding copy.
- Backend tests in `abom_insights_test.go` cover:
  - mixed inputs producing newly observed, widespread, high/low confidence, and stale rows;
  - high-attention promotion when an item lives on a device with open findings;
  - hotspot ranking (high-attention devices first);
  - empty inputs returning section scaffolding without errors.
- Console `AbomPanel` extended:
  - new "Fleet insights" block above the table with cards for each insight section, threshold strings, top items, and "Filter table" jumps for confidence sections.
  - endpoint hotspots strip with deep links into agent detail.
  - new confidence filter chip group (`all` / `high` / `medium` / `low`) and time-window selector (`24h`, `3d`, `7d`, `30d`) tied to the insights API.
  - per-row "Design control" deep link into the finding-to-control designer with `device_id` / `finding_id` query params.

## Acceptance Criteria

- [x] Operator can identify new AI capabilities without scanning the whole inventory (newly observed cards).
- [x] Operator can filter ABOM by confidence and time window (chip group + window select).
- [x] Endpoint-level ABOM still shows why a capability matters and when it first appeared (existing `recommended_review` + `formatRelative`).
- [x] Empty states explain what data is needed to populate insights (per-section "no rows in this slice yet" copy and overall onboarding hint when the fleet is empty).
- [x] `npm run build` passes in `ui/console`.
- [x] Backend tests cover at least one new insight category and empty data behavior (`TestBuildABOMInsights_Categories`, `TestBuildABOMInsights_EmptyInputs`).

## Dependencies

- WO-PROD-001
- WO-API-001
- WO-QA-001 recommended

## Non-Goals

- No external enrichment feed.
- No enforcement.
- No broad ABOM taxonomy rewrite unless required by the insight model.

## Suggested Verification

- `cd backend/ingest && go vet ./internal/server/...` (Go test runner blocked locally by an unrelated SecTrust linker bug; CI runs `go test`).
- `cd ui/console && npm run build`.
- `cd ui/console && npm run test:e2e` if the harness covers the new route or sidebar.
