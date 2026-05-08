# WO-INT-001: Clarion Integration API Slice

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** Backend / Integration  

## Goal

Implement the first concrete AegisFlux -> Clarion integration slice without coupling databases or collapsing product boundaries.

## Scope

- AegisFlux evidence export.
- Clarion-compatible shape.
- Documentation and sample payloads.

## Deliverables

- API/export for device evidence summary:
  - device id
  - agent id
  - OS/source
  - freshness
  - AI activity summary
  - inventory summary
  - findings
  - evidence links
- Event contract for:
  - `aegis.device.observed`
  - `aegis.ai_activity.summarized`
  - `aegis.inventory.item_observed`
  - `aegis.finding.created`
- Sample payloads for Windows and Linux lab devices.
- Clarion mapping notes for endpoint, device, user/session, flow, destination, and finding context.

## Acceptance Criteria

- AegisFlux can export a device evidence summary without Clarion running.
- Payloads validate against documented schema.
- Contract avoids direct DB writes.
- Clarion can consume or mock the payload independently.

## Dependencies

- WO-PLAT-002
- WO-INV-001
- Existing integration contract document.

## Non-Goals

- No production Clarion UI changes in this work order.
- No bidirectional enforcement API yet.

