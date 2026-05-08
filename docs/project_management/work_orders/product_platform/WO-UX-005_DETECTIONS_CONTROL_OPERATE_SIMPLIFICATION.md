# WO-UX-005: Detections, Control, and Operate UX Simplification

**Status:** Draft  
**Phase:** Product Platform UX  
**Primary owner:** UI  

## Goal

Simplify the Detection Packs, Draft Controls, and Operational Event Feed pages so each workflow has a clear primary task and does not become a long report.

## Problem

These pages contain important operational data, but candidates, rollout status, signed packs, draft controls, simulations, and audit/event streams can overwhelm the operator when stacked vertically.

## References

- `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`
- Aegis pages:
  - `ui/console/app/detections/page.tsx`
  - `ui/console/app/control/controls/page.tsx`
  - `ui/console/app/operate/events/page.tsx`

## Scope

- Detection Packs page.
- Draft Controls page.
- Operational Event Feed page.
- Existing backend APIs only unless a tiny read-only summary endpoint is clearly required.

## Deliverables

- Detection Packs:
  - split candidates, signed packs, and rollout into tabs or segmented views.
  - show the current work queue first.
  - keep selected candidate details bounded in modal/route/inline expansion, not a permanent long right column.
- Draft Controls:
  - focus on draft queue and simulation status.
  - put create/edit detail in a modal or compact form state.
  - keep observe-only posture visible but not repetitive.
- Operational Events:
  - make it a filterable timeline/table.
  - provide event details in a bounded modal.
  - hide raw payloads behind an explicit raw view.
- Apply shared string formatting for ids, pack versions, hashes, event names, and payloads.

## Acceptance Criteria

- Each page has one clear primary task.
- Clicking an item does not create a new permanent long-scroll column.
- Raw data is available but not shown by default.
- Long strings do not stretch layout.
- `npm run build` passes in `ui/console`.

## Non-Goals

- Do not alter detection-pack lifecycle semantics.
- Do not add enforcement.
- Do not redesign the entire console shell.
