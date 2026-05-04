# WO-PLAT-001: Clarion-Aligned Console Shell

**Status:** Started  
**Phase:** Product Platform  
**Primary owner:** UI  

## Goal

Make the AegisFlux console follow Clarion's core app-shell workflow: full-width top header, left workflow navigation, breadcrumb row, and clean dashboard content.

## Scope

- Console shell and navigation only.
- Dashboard remains the landing page.
- Menu items can be placeholders until their pages are implemented.

## Deliverables

- Top header with product identity, search placeholder, notification icon, documentation icon, AI assistant icon, health state, and user control area.
- Left navigation grouped by:
  - Overview
  - Discover
  - Analyze
  - Control
  - Operate
  - Configure
- Breadcrumb row.
- Dashboard content area that scrolls independently from the header/sidebar.
- Route/page placeholder plan for every menu item.

## Acceptance Criteria

- Console visually resembles Clarion's app shell layout.
- Dashboard no longer owns every workflow.
- Header is stable across the console.
- Left menu remains visible on dashboard.
- `npm run build` passes.
- Dev console returns HTTP 200.

## Dependencies

- Current AegisFlux console.
- Clarion frontend layout reference at `/Users/stevengerhart/workspace/github/sgerhart/clarion/frontend/src/components/Layout.tsx`.

## Implementation Notes

- Existing commit `e73c04d` is the first shell alignment pass.
- Next UI pass should convert shell pieces into reusable components before adding more pages.

