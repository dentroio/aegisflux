# WO-GROWTH-006: First-Value Demo Polish and Sample Scenarios

**Status:** Draft  
**Phase:** Product Growth / Onboarding  
**Primary owner:** Product / UI / Docs  

## Goal

Polish the first-value demo into a credible operator and buyer walkthrough with sample scenarios, better empty states, and a repeatable lab setup.

## Why This Matters

AegisFlux needs a short path to "I get it." The demo should show the full product loop:

Discover AI capability -> Explain evidence -> Design observe-only control -> Simulate blast radius -> Adapt detection.

## Scope

- Demo/onboarding route.
- Sample scenario setup.
- Operator copy and docs.
- Demo data checklist.

## Deliverables

- Add sample scenario docs:
  - browser AI extension
  - coding agent / CLI agent
  - local model runtime
  - model gateway access
  - suspicious automation finding
- Add demo checklist:
  - required services
  - required agents
  - expected routes
  - reset/cleanup steps
- Improve empty states on demo-dependent routes.
- Add "use sample data" mode if feasible without polluting production paths.
- Add screenshots or screenshot placeholders for demo docs.
- Add a short operator narrative that sales/customer teams can follow.

## Acceptance Criteria

- A new operator can complete the demo flow without architecture knowledge.
- Empty states tell the operator how to get data.
- Demo docs include setup, route order, expected observations, and cleanup.
- `npm run build` passes in `ui/console` if UI changes are made.
- Demo path avoids claims of active enforcement.

## Dependencies

- WO-PROD-005
- WO-GROWTH-001 through WO-GROWTH-005 recommended

## Non-Goals

- No marketing landing page.
- No fake enforcement.
- No production installer workflow.

## Suggested Verification

- `npm run build` in `ui/console` if changed.
- Manual walkthrough using lab or sample scenario docs.

