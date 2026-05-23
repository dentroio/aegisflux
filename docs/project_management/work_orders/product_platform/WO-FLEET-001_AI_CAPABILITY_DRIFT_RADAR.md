# WO-FLEET-001: AI Capability Drift Radar

**Status:** Planned  
**Phase:** AI-Native Leap / Discover  
**Primary owner:** Backend / UI / Detection  

## Goal

Create a fleet radar that highlights new, risky, widespread, stale, or low-confidence AI capabilities and behavior changes.

## Problem

ABOM inventory is valuable, but the daily operator question is "what changed and what matters?" AegisFlux needs a drift radar that turns fleet evidence into prioritized review items.

## Scope

- ABOM/fleet insight summaries.
- Capability change detection.
- Detection opportunity links.
- Endpoint readiness and evidence confidence.

## Deliverables

- Define capability drift categories:
  - new AI-capable tool,
  - new model gateway destination,
  - new local runtime,
  - new MCP/tool bridge signal,
  - risky reachability change,
  - widespread adoption,
  - stale/low-confidence evidence.
- Add backend summary or job that produces ranked drift items.
- Add UI radar surface with filters by endpoint, user, tool class, risk, and confidence.
- Link drift items to Endpoint Analyst and Detection Opportunity workflows.

## Acceptance Criteria

- Operators can see what changed since a selected time window.
- Drift items include evidence refs, confidence, first seen, last seen, affected endpoints, and recommended next action.
- Low-confidence items are marked as such rather than overstated.
- Radar does not replace raw ABOM/inventory drill-in.

## Dependencies

- WO-GROWTH-001.
- WO-GROWTH-004.
- WO-ADAPT-001 recommended.

## Non-Goals

- No automatic policy changes.
- No vendor-only risk scoring.
- No high-volume raw telemetry table as the primary UI.

## Suggested Verification

- Tests for drift classification from fixture evidence.
- UI build and manual radar walkthrough with sample scenarios.
