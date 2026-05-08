# AegisFlux UI Simplification Guide

**Status:** Design baseline  
**Last updated:** May 8, 2026

## Purpose

AegisFlux has enough data now that the UI needs stronger hierarchy. The next UI pass should make the console calmer, more intentional, and easier to scan without hiding important operational evidence.

## Clarion Patterns To Carry Forward

Use Clarion's dashboard as the nearest reference:

- `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Dashboard.tsx`
- `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/DashboardWidget.tsx`
- `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/hooks/useWidgetConfig.ts`
- `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/WidgetEmptyState.tsx`
- `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/dashboard/LazyWidgetMount.tsx`

The target is not visual cloning. The target is Clarion's information architecture:

- readiness band first
- next best action near the top
- a short KPI strip
- compact insight widgets
- clear empty states
- explicit navigation for deep work
- no accidental page expansion when something is selected

## Hard UX Rules

- Clicking a row, card, metric, or widget must not create another permanent right-side column that starts a long scroll.
- Details must appear in one of these patterns:
  - replace the main work area with a focused detail view
  - open a modal or temporary drawer for a small amount of detail
  - navigate to a deliberate detail route
  - expand a single row inline for short evidence only
- Each page should have one primary scroll area.
- A page should not stack many unrelated cards vertically just because data exists.
- Raw evidence belongs behind a tab, accordion, modal, or explicit "Raw" view.
- Dashboard is a scan surface, not an investigation page.
- Agents, Inventory, Detections, Controls, and Events are workbenches, not reports.
- Long strings must be intentionally formatted with truncation, wrapping, monospace, copy affordances, and tooltips where appropriate.

## String Formatting Rules

Use consistent formatting for noisy values:

- Hostnames, agent ids, pack ids, hashes, MAC addresses, IPs, paths, command lines, and event ids should use compact monospace styling.
- Long identifiers should truncate in tables and cards, with full value available through title/tooltip or copy button.
- File paths and command lines should wrap only in detail views, never in compact table rows.
- JSON and raw payloads should be hidden behind a formatted code block in a detail tab or modal.
- Tables should use fixed or constrained columns so a long value cannot stretch the page.
- Badge text must stay short; use detail views for explanation.

## Page Shape

Default workbench page shape:

1. Header: title, short subtitle, primary actions.
2. Summary strip: 4-6 high-signal metrics only.
3. Controls: search, saved filters, compact filter chips.
4. Main surface: table, timeline, or widget grid.
5. Detail: deliberate route, modal, temporary drawer, or inline expansion. Avoid permanent nested right columns.

## Dashboard Target

The dashboard should answer:

- Is the platform healthy?
- What needs attention?
- What changed recently?
- Where should the operator go next?

The dashboard should not answer every detailed evidence question. Those belong in Agents, Inventory, Detections, Controls, and Operate.
