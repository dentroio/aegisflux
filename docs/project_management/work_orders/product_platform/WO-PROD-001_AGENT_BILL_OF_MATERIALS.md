# WO-PROD-001: Agent Bill of Materials

**Status:** Done  
**Phase:** Product Differentiation  
**Primary owner:** Product / Backend / UI  

## Goal

Build the first product-grade Agent Bill of Materials so AegisFlux can answer: what AI-capable tools exist on this endpoint, what can they reach, and what evidence proves it?

## Why This Makes Aegis Stand Out

Most endpoint tools show alerts, processes, or software inventory. AegisFlux should show AI capability inventory: AI apps, coding agents, browser extensions, CLI tools, MCP servers, local model runtimes, model gateways, and the supporting process/network/DNS evidence.

This is a clearer wedge than another dashboard. It gives security, IT, and platform teams a thing they can immediately understand and want: a living inventory of AI-agent capability across endpoints.

## Scope

- Read-only Agent Bill of Materials model.
- Endpoint-level and fleet-level views.
- Evidence-backed source records.
- UI that presents meaningful categories, not raw telemetry buckets.

## Deliverables

- Defined ABOM item shape in `backend/ingest/internal/server/abom_query.go`:
  - `id` (deterministic per category + product)
  - `category` (taxonomy below)
  - `product`
  - `capability_tags`
  - `confidence` (`high`, `medium`, `low`, with multi-device promotion)
  - `device_ids`
  - `user_context` (forward-compatible field)
  - `evidence_refs` (`process:…`, `dns:…`, `event:…`, `finding:…`)
  - `first_seen_ms` / `last_seen_ms`
  - `recommended_review`
- Category taxonomy:
  - `ai_desktop_app`
  - `browser_ai_extension`
  - `coding_agent`
  - `cli_agent`
  - `mcp_endpoint`
  - `local_model_runtime`
  - `model_gateway`
  - `unknown_ai_automation`
- Backend aggregation endpoints:
  - `GET /v1/visibility/abom/fleet` — aggregates across fleet from process events, browser extensions, SASE/SSE components, DNS observations, and findings. Returns `category_count`, items, and `empty_help` when there is nothing to show.
  - `GET /v1/visibility/abom/device?device_id=…` — same logic filtered to the device.
- UI surfaces:
  - Reusable `AbomPanel` component (`ui/console/components/AbomPanel.tsx`).
  - Fleet route `/discover/abom` (and shell nav entry under Discover).
  - "AI Capability" tab on the agent detail page using the same panel scoped to the device.
  - Dashboard "Signal focus" card promotes the new view, plus a "Next best actions" link.
- Documented confidence rules: process matches start at `medium`, runtime/gateway/desktop/extension-with-broad-host-permissions promote to `high`. Cross-device repetition (>=3 devices) promotes to `high`. DNS-only and finding-only signals stay at `low` until corroborated.
- Backend tests in `backend/ingest/internal/server/abom_query_test.go` cover:
  - Local runtime + gateway aggregation (Ollama + `api.openai.com`).
  - Browser extension with broad host permissions + MCP server process.
  - Finding-only signals fall to `unknown_ai_automation` at `low` confidence.
  - Cross-device dedup and confidence promotion.
  - Empty-input safety.

## Acceptance Criteria

- [x] Operators can answer which AI-capable tools are present across the fleet (`/discover/abom`).
- [x] Operators can open an endpoint and see the AI capabilities that matter on that endpoint (Agent detail → `AI Capability` tab).
- [x] Each ABOM item has evidence links or source metadata (`evidence_refs`).
- [x] The UI avoids raw names like process/DNS as primary navigation labels (the panel uses category labels and capability tags).
- [x] Empty state explains what data is needed to populate ABOM (server returns `empty_help`, panel surfaces the message).
- [x] `npm run build` passes in `ui/console`.
- [x] Relevant backend tests added in `abom_query_test.go`. (Local Go test runner currently fails to link due to a pre-existing macOS Xcode CLT / Go SDK mismatch unrelated to this WO; CI runs the tests.)

## Dependencies

- WO-INV-001
- WO-API-001 recommended
- Existing visibility events for browser, process, DNS, findings, SASE/SSE, and detection packs

## Non-Goals

- No blocking or enforcement.
- No claim of complete AI inventory across all vendors.
- No external SaaS enrichment in the first slice.

## Suggested Verification

- Run the backend tests: `cd backend/ingest && go test ./internal/server -run TestBuildABOMItems`.
- Run `npm run build` in `ui/console`.
- Manually walk the lab data:
  - Fleet view via `Discover → Agent Bill of Materials`.
  - Endpoint view via Agents → choose endpoint → `AI Capability` tab.
- Confirm dashboard "Open Agent Bill of Materials" card and "Next best actions" link both deep-link into `/discover/abom`.
