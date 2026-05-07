# WO-PLAT-002: Agent List and Device Drill-In

**Status:** Complete  
**Phase:** Product Platform  
**Primary owner:** UI / Backend  

## Goal

Move endpoint detail out of the dashboard into a product workflow: agent list -> device detail -> evidence tabs.

## Scope

- Add an `/agents` page for fleet list and filtering.
- Add an `/agents/[device_id]` page for device detail.
- Keep dashboard high-level.

## Deliverables

- Agent list with OS/source, freshness, sensor version, event counts, finding count, extension count, and collector health.
- Device detail page with tabs:
  - Overview
  - AI Activity
  - Processes
  - Network
  - DNS
  - Browser
  - Inventory
  - Findings
  - Collector Health
- Deep links from dashboard agent panel to device detail.
- Empty states for no telemetry and stale telemetry.

## Acceptance Criteria

- User can click a device and land on a stable detail page.
- Device detail does not require selecting an agent from the dashboard.
- The page can load directly from URL.
- Windows and Linux lab devices render correctly.
- `npm run build` passes.

## Dependencies

- WO-PLAT-001
- Existing visibility APIs:
  - `/api/visibility/devices`
  - `/api/visibility/events`
  - `/api/visibility/processes`
  - `/api/visibility/flows`
  - `/api/visibility/dns`
  - `/api/visibility/findings`

## Non-Goals

- No enforcement actions.
- No AI-generated conclusions yet; that belongs to WO-AI-003.

## Implementation Update (May 7, 2026)

- Added direct device drill-in route: `/agents/[device_id]`.
- Device detail loads directly from URL and fetches device-scoped visibility evidence from ingest-backed console APIs.
- Added tabs for Overview, AI Activity, Processes, Network, DNS, Browser, Inventory, Findings, and Collector Health.
- Added stale telemetry treatment and empty states for missing evidence.
- Added links from the dashboard device panel and `/agents` list to the device detail page.
- Enriched `/agents` list rows with visibility event count, finding count, browser extension count, and collector status count.
- Verification: `npm run build` passes.
