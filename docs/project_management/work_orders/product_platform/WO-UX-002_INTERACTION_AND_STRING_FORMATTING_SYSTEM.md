# WO-UX-002: Interaction and String Formatting System

**Status:** Draft  
**Phase:** Product Platform UX  
**Primary owner:** UI  

## Goal

Create shared UI primitives and rules that keep AegisFlux pages simple, prevent accidental long-scroll layouts, and format noisy endpoint/security strings cleanly.

## Problem

Multiple pages display agent ids, paths, command lines, hashes, pack ids, event ids, IPs, MACs, and raw payloads differently. Detail selection also tends to create stacked cards or right-side panels that become another long scroll.

## References

- `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`
- Clarion dashboard and endpoint patterns:
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Dashboard.tsx`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Devices.tsx`
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/endpoints/EndpointDetail.tsx`
- Aegis shell:
  - `ui/console/components/shell/ConsoleShell.tsx`
  - `ui/console/app/globals.css`

## Scope

- Shared UI components and formatting helpers only.
- Small targeted adoption on the dashboard if WO-UX-001 has already landed; otherwise include examples but avoid broad page rewrites.

## Deliverables

- Shared components for:
  - `SummaryStrip`
  - `KpiTile`
  - `WorkbenchHeader`
  - `FilterBar`
  - `BoundedTable`
  - `DetailModal` or `DetailDrawer` for temporary detail
  - `FormattedValue`
  - `CopyValueButton`
  - `EmptyState`
- Shared formatting helpers for:
  - hostnames
  - agent ids
  - IPs/CIDRs
  - MAC addresses
  - hashes
  - paths
  - command lines
  - JSON/raw payloads
  - dates/relative age
- CSS/Tailwind conventions that constrain table cells and prevent text overflow.

## Acceptance Criteria

- Long values are truncated in compact views and readable in deliberate detail views.
- Tables cannot stretch the page horizontally because of one long field.
- Detail surfaces have bounded height and their own internal scroll only when necessary.
- Components are accessible enough for keyboard and screen-reader basics.
- `npm run build` passes in `ui/console`.

## Non-Goals

- Do not redesign every page in this work order.
- Do not introduce a large UI framework.
- Do not make permanent right-side detail columns the default pattern.

## Implementation Notes

The key behavior is restraint: compact views show what matters, detail views reveal more only when asked, and raw values never break the layout.
