# WO-UX-003: Agents Workbench Simplification

**Status:** Implemented  
**Phase:** Product Platform UX  
**Primary owner:** UI  

## Goal

Simplify Agents into a focused workbench that helps an operator find agents needing attention without long stacked detail columns.

## Problem

The current Agents surface includes filters, rollout telemetry, an agent list, selected agent details, labels, notes, system info, network info, eBPF capabilities, and detection-pack status in one long layout. Selecting an agent effectively creates another scroll-heavy detail area.

## References

- `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`
- `docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md`
- Clarion endpoint list:
  - `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Devices.tsx`
- Aegis Agents:
  - `ui/console/components/AgentsManagementPanel.tsx`
  - `ui/console/app/agents/page.tsx`
  - `ui/console/app/agents/[device_id]/page.tsx`

## Scope

- Agents workbench panel/page.
- Links into agent detail.
- Summary/rollout information already available from existing APIs.

## Deliverables

- Agents page shaped as:
  - concise summary strip
  - filter/search controls
  - primary bounded table/list
  - no permanent stacked right-side detail column
- Clicking an agent should either:
  - navigate to `/agents/[device_id]`, or
  - open a bounded modal/temporary drawer with only critical summary and actions.
- Detection-pack rollout should be a compact view or tab, not a full stacked section above the agent list.
- Agent list rows should show only:
  - identity
  - freshness/status
  - OS/platform
  - pack health
  - recent event/finding signal
  - action to open detail
- Labels, notes, capabilities, network, and raw pack metadata should move to the deliberate detail route or a compact detail modal.
- Long strings must use the shared formatting rules from WO-UX-002 if available.

## Acceptance Criteria

- Agents can be scanned without long vertical scrolling.
- Selecting an agent does not create a permanent right-side column with its own long scroll.
- The workbench clearly answers: online/stale, pack health, budget pressure, findings, and last seen.
- Linux and Windows lab agents both render cleanly.
- Long agent ids, hostnames, pack ids, and network fields do not overflow.
- `npm run build` passes in `ui/console`.

## Non-Goals

- Do not remove the agent detail route.
- Do not add enforcement controls.
- Do not change backend agent schemas unless a display-critical summary is impossible without a tiny read-only addition.

## Implementation Notes

### 2026-05-08 Implementation Update

- Reworked `ui/console/components/AgentsManagementPanel.tsx` into a focused workbench layout:
  - concise summary strip (total, stale, pack issues, budget pressure)
  - compact filter/search controls
  - single primary bounded table surface
- Removed permanent selected-agent right-side detail column and long stacked detail cards from the workbench.
- Added deliberate surface switch between:
  - `Agents` primary scan table
  - `Rollout` compact detection-pack table
- Kept detail behavior deliberate by routing to `/agents/[device_id]` via `Open detail` actions.
- Retained trust metadata access through bounded `DetailModal` (temporary surface only).
- Applied shared formatting primitives from WO-UX-002 for long ids and hostnames.
- Verification completed with `npm run build` in `ui/console` (pass).
