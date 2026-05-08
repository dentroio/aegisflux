# WO-PLAT-005: Clarion-Style Agent Workbench and Detail Experience

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** UI  

## Goal

Bring Clarion's endpoint workbench and endpoint detail patterns into AegisFlux so Agents becomes a dense, operational workbench instead of a basic management page.

## Scope

- Agents workbench page inside the persistent AegisFlux shell.
- Agent/device detail experience aligned to Clarion endpoint detail.
- UI-only first slice, using existing Actions API and Visibility API data.

## Deliverables

- Agent workbench with:
  - KPI/context cards.
  - quick-filter tabs with counts.
  - search and scoped filters.
  - table/card view toggle.
  - persisted column visibility and view preference.
  - clear empty/loading/error states.
- Agent detail with:
  - strong identity header.
  - freshness row.
  - status chips.
  - Evidence Confidence, Network Context, Detection / Policy Context, and Next Best Action cards.
  - tabs: Overview, Evidence, Inventory, Detection Packs, Performance, Policy.
- Inventory and detection-pack links from the detail page that preserve shell navigation.
- Deterministic Next Best Action logic based on stale status, findings, detection-pack status, collector health, and agent budget telemetry.

## Acceptance Criteria

- The left navigation remains visible while using Agents and detail workflows.
- Agent workbench supports quick filtering for online, stale, pack applied, pack rejected, needs attention, and budget pressure.
- Agent detail is usable for both Linux and Windows lab agents.
- Text does not overflow compact cards, buttons, tabs, or table cells.
- `npm run build` passes in `ui/console`.
- Local dev console returns the Agents shell panel successfully.

## Dependencies

- WO-PLAT-001
- WO-PLAT-002
- WO-PLAT-004
- WO-AGENT-001
- WO-INV-001

## Non-Goals

- No backend schema changes unless a small gap blocks the UI.
- No enforcement actions.
- No AI-generated explanations; deterministic guidance only.

## Reference

- `docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md`
- Clarion endpoint list: `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Devices.tsx`
- Clarion endpoint detail: `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/endpoints/EndpointDetail.tsx`

