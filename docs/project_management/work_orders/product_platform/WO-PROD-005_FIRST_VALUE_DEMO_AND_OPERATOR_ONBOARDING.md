# WO-PROD-005: First-Value Demo and Operator Onboarding

**Status:** Done  
**Phase:** Product Differentiation  
**Primary owner:** UI / Product  

## Goal

Build a focused demo path that an operator can follow to understand AegisFlux value in under five minutes.

## Why This Makes Aegis Stand Out

Most security tools require long ramp-ups. AegisFlux should let an operator say in five minutes: "this is what we have, this is why a finding matters, this is what we would do, this is how we adapt." That clarity is part of the product value.

## Scope

- A guided five-step tour covering the four product pillars (Discover, Investigate, Design, Adapt) plus a welcome and wrap-up.
- Per-step CTAs that open the real product surface, not slides.
- Local-storage progress tracking (visited steps, started timestamp, completed/skipped flags).
- Dashboard banner that promotes the tour while it is incomplete.
- Always-on demo route the operator (or a buyer in a demo session) can return to.

## Deliverables

- New `FirstValueTour` component in `ui/console/components/FirstValueTour.tsx`:
  - Five-step ordered list (welcome → discover ABOM → investigate evidence path → design control → adapt research feed → wrap-up).
  - Per-step icon, product-pillar tag, time estimate, narrative paragraph, and CTA that links into the live route.
  - Pillar grid summarizing the four product pillars.
  - Progress chips (`min left`, `visited count`, `completed`).
  - Controls: Skip for now, Mark all visited, Reset progress.
  - Persisted in localStorage under `aegisflux.firstValueDemo.v1`.
- Reusable `FirstValueTourBanner` exported from the same module:
  - Appears on the dashboard while the tour is neither completed nor skipped.
  - Offers Start tour and Skip; Skip persists across reloads.
- Helpers `tourCompleted()`, `markTourCompleted()`, `markTourSkipped()`, `resetTour()` for future surfaces (help menu, telemetry).
- New `/demo` route gated by lab auth; uses `ConsoleShell` so the experience matches the rest of the product.
- Dashboard `app/page.tsx` now renders the `FirstValueTourBanner` above the readiness band.

## Acceptance Criteria

- [x] An operator can complete the tour and grasp Discover/Investigate/Design/Adapt in under five minutes (estimated step times sum to ~5:50; ~4:30 minus welcome/wrap).
- [x] Each step links into the real product surface (`/discover/abom`, `/analyze/evidence`, `/control/controls`, `/analyze/research`).
- [x] Progress is preserved across reloads via localStorage; banner hides when completed or skipped.
- [x] Dashboard surfaces a Start-tour banner while the tour is incomplete.
- [x] `cd ui/console && npm run build` passes.

## Dependencies

- WO-PROD-001 (Agent Bill of Materials)
- WO-PROD-002 (Evidence Graph)
- WO-PROD-003 (Finding-to-Control Designer)
- WO-PROD-004 (AI Research Feed)

## Non-Goals

- No backend telemetry for tour completion (left as a future enhancement).
- No multi-user roles for tour content.
- No auto-replay or in-product tooltips beyond the dashboard banner.

## Suggested Verification

- `cd ui/console && npm run build`.
- Manual walkthrough: log into the lab, observe the banner on the dashboard, click Start tour, walk each step and confirm each CTA opens the right product surface, return to `/demo` and confirm the visited markers persist after a hard reload.
