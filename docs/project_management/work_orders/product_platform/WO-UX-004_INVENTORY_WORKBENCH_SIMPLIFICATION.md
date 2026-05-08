# WO-UX-004: Inventory Workbench Simplification

**Status:** Draft  
**Phase:** Product Platform UX  
**Primary owner:** UI  

## Goal

Make Inventory a simple, searchable workbench for AI tools, browser extensions, local model runtimes, and enterprise control components.

## Problem

Inventory data can easily become a long scroll of cards and raw fields. The operator needs quick answers first: what exists, where it is, what changed, and what needs attention.

## References

- `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`
- `docs/project_management/work_orders/product_platform/WO-INV-001_ENTERPRISE_AI_AND_CONTROL_INVENTORY.md`
- Aegis Inventory:
  - `ui/console/components/InventoryPanel.tsx`
  - `ui/console/app/inventory/page.tsx`

## Scope

- Inventory workbench page/panel.
- Existing inventory data only.
- Links from dashboard and agent detail into inventory filters.

## Deliverables

- Inventory page shaped as:
  - summary strip by inventory category
  - category tabs or segmented control
  - search/filter controls
  - bounded table/list
  - compact detail modal or deliberate detail route for selected inventory item
- Categories should include:
  - browser extensions
  - AI IDE/CLI tools
  - local model runtimes
  - SASE/SSE/control components
  - unknown/unclassified signals
- Clicking an inventory item must not create a permanent right-side long-scroll column.
- Long strings such as extension ids, paths, command lines, hostnames, and vendors must format cleanly.

## Acceptance Criteria

- Inventory can be scanned by category without scrolling through every raw record.
- Search and category filters reduce the visible set predictably.
- Selected item detail is bounded and focused.
- Dashboard and agent-detail links can deep-link or prefilter inventory if current routing supports it.
- `npm run build` passes in `ui/console`.

## Non-Goals

- No new inventory collectors.
- No enforcement action.
- No broad backend schema redesign.
