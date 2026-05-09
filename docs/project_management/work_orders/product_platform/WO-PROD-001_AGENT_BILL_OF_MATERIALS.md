# WO-PROD-001: Agent Bill of Materials

**Status:** Draft  
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

- Define ABOM item shape:
  - id
  - category
  - product/tool name
  - capability tags
  - confidence
  - device ids
  - user/session context if available
  - evidence references
  - first seen / last seen
  - recommended review action
- Create category taxonomy:
  - AI desktop app
  - browser AI extension/session
  - coding agent / IDE plugin
  - CLI agent / shell automation
  - MCP client/server
  - local model runtime
  - enterprise model gateway
  - unknown AI-like automation
- Add backend summary or aggregation path for ABOM items.
- Add UI surfaces:
  - fleet ABOM inventory
  - endpoint ABOM panel
  - detail modal with supporting evidence
- Document confidence rules and false-positive handling.

## Acceptance Criteria

- Operators can answer which AI-capable tools are present across the fleet.
- Operators can open an endpoint and see the AI capabilities that matter on that endpoint.
- Each ABOM item has evidence links or source metadata.
- The UI avoids raw names like process/DNS as primary navigation labels.
- Empty state explains what data is needed to populate ABOM.
- `npm run build` passes in `ui/console` if UI changes are made.
- Relevant backend tests pass if backend code changes.

## Dependencies

- WO-INV-001
- WO-API-001 recommended
- Existing visibility events for browser, process, DNS, findings, SASE/SSE, and detection packs

## Non-Goals

- No blocking or enforcement.
- No claim of complete AI inventory across all vendors.
- No external SaaS enrichment in the first slice.

## Suggested Verification

- Backend tests for ABOM aggregation.
- UI build and route checks.
- Manual check with Windows and Linux lab agent data.

