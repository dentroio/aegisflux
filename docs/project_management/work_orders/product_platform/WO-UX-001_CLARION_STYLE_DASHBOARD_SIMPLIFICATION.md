# WO-UX-001: Clarion-Style Dashboard Simplification

**Status:** Implemented  
**Phase:** Product Platform UX  
**Primary owner:** UI  

## Goal

Rebuild the AegisFlux dashboard as a simple, Clarion-aligned scan surface. It should show the right information quickly without becoming a long report page or spawning extra columns when the operator clicks something.

## Problem

The current dashboard mixes platform health, widgets, agents, selected agent details, and operational data into one large surface. It contains useful information, but the hierarchy is too weak and the page invites long scrolling.

## References

- `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`
- `docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md`
- Clarion dashboard:
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Dashboard.tsx`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/DashboardWidget.tsx`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/hooks/useWidgetConfig.ts`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/WidgetEmptyState.tsx`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/LazyWidgetMount.tsx`
- Aegis dashboard:
  - `ui/console/app/page.tsx`
  - `ui/console/components/shell/ConsoleShell.tsx`

## Scope

- Dashboard route `/`.
- Dashboard-only widgets and summary components.
- Local widget visibility persistence if already present or easy to preserve.
- Navigation from dashboard cards/widgets to deeper pages.

## Deliverables

- A dashboard layout with these bands:
  - Readiness: platform status, telemetry health, AI/provider health, and next best action.
  - Attention: only actionable warnings, hidden when nothing needs attention.
  - Insights: a short KPI strip and compact widgets.
- Dashboard widgets that are bounded in height and do not create long vertical data dumps.
- Dashboard clicks that navigate to the appropriate workbench or open a small modal; they must not add a permanent right-side detail column.
- Raw data removed from the dashboard default view.
- Clear loading, empty, error, and no-data states.
- Long strings formatted safely with truncation/tooltips/copy where needed.
- `ui/console/app/page.tsx` reduced by extracting dashboard-only components where useful.

## Acceptance Criteria

- The dashboard is usable without long scrolling at common desktop widths.
- Clicking a dashboard metric, row, or widget does not create a new permanent right-side column.
- Dashboard cards show only summary information and route users to deeper workflows for details.
- Long strings do not stretch cards, tables, or page width.
- The left navigation remains visible.
- `npm run build` passes in `ui/console`.
- A local visual check confirms `/` is professional at desktop and mobile widths.

## Non-Goals

- Do not redesign Agents, Inventory, Detection Packs, Controls, or Events in this work order.
- Do not add new backend APIs unless a tiny read-only summary endpoint is clearly required.
- Do not add drag-and-drop layout.
- Do not expose raw evidence on the dashboard by default.

## Implementation Notes

Prefer small components under `ui/console/components/dashboard/` or `ui/console/components/workbench/` over growing `ui/console/app/page.tsx`.

Dashboard widget content should stay compact. If a widget needs more than a small table, short list, or chart, link to a workbench page.

### 2026-05-07 Implementation Update

- Reworked dashboard `/` into clearer scan bands:
  - Readiness: platform health hero and compact readiness stats.
  - Attention: conditional warning strip shown only when actionable signals exist.
  - Insights: bounded widget grid with local visibility controls retained.
- Removed dashboard's permanent right-side selected-agent detail column.
- Replaced deep inline detail behavior with deliberate navigation links to workbench routes (Agents, Inventory, Detections, Controls, Operate).
- Kept left navigation and shell behavior intact.
- Preserved compact list behavior and constrained row rendering to avoid long-string layout stretch.
- Verification completed with `npm run build` in `ui/console` (pass).
