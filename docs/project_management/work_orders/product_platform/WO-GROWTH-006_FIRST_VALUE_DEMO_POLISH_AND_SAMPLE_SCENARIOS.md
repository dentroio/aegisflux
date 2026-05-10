# WO-GROWTH-006: First-Value Demo Polish and Sample Scenarios

**Status:** Done  
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

## Implementation Notes (Done)

### Documentation (`docs/demos/`)

- New `docs/demos/README.md` defines the operator narrative, the four product pillars and the
  routes that prove them, and the demo screenshot placeholders.
- New `docs/demos/SAMPLE_SCENARIOS.md` covers five scenarios end to end:
  - Browser AI extension
  - Coding agent (CLI)
  - Local model runtime
  - MCP endpoint exposure
  - Suspicious automation finding
  Each scenario lists steps, expected observations, and the routes that demonstrate them.
- New `docs/demos/CHECKLIST.md` captures required services, required agents, the expected route
  order, observations to confirm per scenario, reset/cleanup steps, and operator dos and don'ts.

### Console (`ui/console`)

- Added `app/demo/scenarios/page.tsx` rendering the same five scenarios in-product with deep
  links to the relevant routes plus external links to the docs. The page is gated by the existing
  lab auth flow so it stays consistent with the first-value tour.
- Updated `components/FirstValueTour.tsx` to add a `Sample scenarios` link in the tour header so
  operators can pivot from the tour to scenario references at any time.
- Extended `components/workbench/primitives.tsx` `EmptyState` with optional `hint` and `actions`
  slots without breaking existing call sites.
- Improved empty states on the demo-dependent panels to point operators at the sample scenarios:
  - `components/AbomPanel.tsx` — when ABOM has no items.
  - `components/EvidenceGraphPanel.tsx` — both no-path and no-evidence states.
  - `components/FindingToControlPanel.tsx` — when no evidence has been loaded.
  - `components/ResearchFeedPanel.tsx` — when the research feed view is empty.
- `npm run build` (Next.js 14) passes; the new `/demo/scenarios` route is built statically and
  no new lints are introduced.

### Acceptance Criteria

- A new operator can complete the demo flow without architecture knowledge: the first-value tour
  links into ABOM, evidence path, finding-to-control, and the research feed; sample scenarios are
  one click away from both the tour and any empty state.
- Empty states tell the operator how to get data: each demo-dependent panel now offers a
  `View sample scenarios` action and a hint that names the matching scenario.
- Demo docs include setup, route order, expected observations, and cleanup
  (`docs/demos/CHECKLIST.md` and `docs/demos/SAMPLE_SCENARIOS.md`).
- `npm run build` passes in `ui/console`.
- Demo path avoids claims of active enforcement: docs and the in-product page reinforce the
  observe-only framing on every scenario.

