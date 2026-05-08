# WO-INV-001: Enterprise AI and Control Inventory

**Status:** Done  
**Phase:** Product Platform  
**Primary owner:** Backend / UI / Agent  

**Notes (2026-05-07):** Shipped console route `/inventory` (Discover → Inventory) with fleet KPIs and aggregated tables for browser extensions, enterprise browsers (from extension telemetry), SASE/SSE components, and AI-related DNS destinations, plus observe-only placeholders for IDE/CLI/local MCP/EDR categories. Uses existing `/api/visibility/*` proxies only. Device drill-in links to `/agents/[device_id]`; device header links to filtered fleet inventory (`?device=`).

## Goal

Build the first inventory view for AI-capable tools and enterprise control components observed on endpoints.

## Scope

- Inventory pages and backend normalization.
- Uses existing browser extension and SASE/SSE component events.
- Adds placeholders for IDE, CLI, and local model runtime inventory.

## Deliverables

- Inventory page under Discover > Inventory.
- Inventory categories:
  - Browser extensions
  - Enterprise browsers
  - IDE extensions
  - CLI AI tools
  - Local model runtimes
  - MCP clients/servers
  - SASE/SSE components
  - EDR/MDM/security agents
- Device-to-inventory relationships.
- Risk/context columns:
  - source
  - version
  - permissions/evidence
  - first seen
  - last seen
  - device count
  - confidence

## Acceptance Criteria

- Browser extension inventory renders from current Windows telemetry.
- SASE/SSE inventory renders Palo Alto, Zscaler, Cisco, or other observed components when available.
- Empty inventory categories have useful empty states.
- Device detail links back to inventory records.

## Dependencies

- Current browser extension and SASE/SSE events.
- WO-PLAT-002 for device drill-in links.

## Non-Goals

- No blocking/remediation.
- No claim of complete enterprise browser support until specific browser integrations are implemented.

