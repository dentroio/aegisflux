# WO-QA-001: UI Rendering and Navigation Regression Harness

**Status:** Implemented  
**Phase:** Product Platform Quality  
**Primary owner:** UI / QA  

## Goal

Create repeatable UI regression checks that catch broken rendering, auth redirects, left-nav persistence, row navigation, and oversized-scroll regressions before they reach manual testing.

## Problem

Recent UI work exposed several fragile areas:

- Routes could render but still feel broken because the shell, left menu, or content panel was wrong.
- Agent detail could stall on session checks.
- Agent rows needed clearer click-through behavior.
- Data-heavy pages could become long, slow scroll surfaces.

The project needs a small, reliable harness that verifies the actual operator paths we keep touching.

## Scope

- AegisFlux console UI routes under `ui/console`.
- Local dev/prod build route checks.
- Browser-level assertions where practical.
- Documentation for running the checks locally and from future CI.

## Deliverables

- Add a UI regression harness that verifies:
  - `/` dashboard renders after lab auth.
  - `/agents` redirects or opens the agents shell panel.
  - agent workbench rows navigate to `/agents/[device_id]`.
  - `/agents/[device_id]` handles authenticated and unauthenticated sessions.
  - `/inventory` redirects or opens the inventory shell panel.
  - `/detections`, `/control/controls`, and `/operate/events` render inside the persistent shell.
  - left navigation remains visible on primary console routes.
  - no page displays only an intermediate state such as `Checking session...` after navigation settles.
- Add route smoke checks for HTTP status and basic content.
- Add a short runbook for:
  - local execution
  - expected dependencies
  - known lab auth setup
  - future CI integration

## Acceptance Criteria

- `npm run build` passes in `ui/console`.
- The new harness can run locally with a single documented command.
- The harness checks at least dashboard, agents, agent detail, inventory, detections, controls, and events.
- The harness fails when a route renders only a loading/session placeholder after a reasonable wait.
- The harness verifies row-click navigation from Agents to agent detail where fixture data is available.
- The work order is updated with implementation notes and verification results.

## Dependencies

- WO-PLAT-001
- WO-UX-001 through WO-UX-005
- Current lab auth behavior (`admin` / `admin`)

## Non-Goals

- Do not build a full visual snapshot platform in this work order.
- Do not require production-like auth.
- Do not redesign UI while adding tests.
- Do not require live endpoint telemetry for every assertion; use graceful fixture/fallback behavior where needed.

## Suggested Verification

- `npm run build` in `ui/console`.
- New UI regression command documented by this work order.
- Manual route check if browser automation is unavailable.

## Implementation notes

- Added Playwright harness under `ui/console/e2e/console-regression.spec.ts` with `npm run test:e2e` (see `ui/console/playwright.config.ts`).
- The config builds the console and serves `next start` on **127.0.0.1:3041** by default (`PW_PORT` overrides) so checks do not collide with a dev server on 3030.
- Lab auth is seeded via `localStorage` (`aegisflux.labAuth=admin`) in the authenticated suite; the unauthenticated case clears it before hitting agent detail.
- `ConsoleShell` exposes `data-testid="console-sidebar-nav"` for stable shell assertions.
- Row navigation test skips when the workbench has no agent rows (no fixture agents).
- **Verification:** `npm run build` and `npm run test:e2e` in `ui/console` (7 passed, 1 skipped in a no-agent lab).

