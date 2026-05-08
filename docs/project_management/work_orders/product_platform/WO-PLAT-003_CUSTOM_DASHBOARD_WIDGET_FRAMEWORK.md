# WO-PLAT-003: Custom Dashboard Widget Framework

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** UI / Backend  

## Goal

Make the dashboard customizable without making it noisy. Users should be able to choose a small number of high-level widgets that fit their role.

## Scope

- Dashboard widget catalog.
- Persisted layout/config.
- Role-friendly default widget sets.

## Deliverables

- Widget registry with id, title, description, data source, default size, and enabled state.
- Persisted widget preferences, initially local storage or backend config depending on available storage.
- Customize panel for show/hide and ordering.
- Default widgets:
  - Platform Status
  - Endpoint Freshness
  - AI Activity
  - Detection Pack Coverage
  - Agent Performance Budget
  - Enterprise Control Inventory
- Empty and loading states.

## Acceptance Criteria

- User widget choices persist across refresh.
- Dashboard remains readable with zero, two, or many agents.
- Long text does not overflow widgets.
- `npm run build` passes.

## Dependencies

- WO-PLAT-001

## Non-Goals

- No complex drag-and-drop in first slice unless already easy.
- No per-user auth model required for first lab implementation.

