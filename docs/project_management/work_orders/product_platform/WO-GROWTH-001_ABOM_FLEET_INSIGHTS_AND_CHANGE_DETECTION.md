# WO-GROWTH-001: ABOM Fleet Insights and Change Detection

**Status:** Draft  
**Phase:** Product Growth / Differentiation  
**Primary owner:** Product / Backend / UI  

## Goal

Turn the Agent Bill of Materials from an inventory list into a daily-use fleet insight surface that highlights what is new, risky, widespread, or worth review.

## Why This Matters

ABOM is AegisFlux's clearest customer-facing wedge. The next version should not merely answer "what exists?" It should answer:

- What changed?
- What is new on sensitive endpoints?
- Which AI capabilities are spreading?
- Which tools have enough evidence to trust?
- Which findings should become controls?

This is how Aegis becomes a product people open every morning.

## Scope

- Fleet-level ABOM insight model.
- Change detection against prior observations.
- UI callouts for new/risky/widespread AI capability.
- Operator-centered language and empty states.

## Deliverables

- Add ABOM insight categories:
  - newly observed
  - newly observed on sensitive endpoint
  - high-confidence AI capability
  - low-confidence needs review
  - widespread capability
  - stale/disappeared capability
- Add backend support for first-seen / last-seen / previous-window comparisons.
- Add UI sections:
  - "New since last review"
  - "Needs review"
  - "Widespread capabilities"
  - "Endpoint hotspots"
- Add filters by category, confidence, device, capability tag, and time window.
- Add "create review note" or "send to finding/control workflow" affordance where existing APIs support it.
- Update product docs with ABOM positioning and demo narrative.

## Acceptance Criteria

- Operator can identify new AI capabilities without scanning the whole inventory.
- Operator can filter ABOM by confidence and time window.
- Endpoint-level ABOM shows why a capability matters and when it first appeared.
- Empty states explain what data is needed to populate insights.
- `npm run build` passes in `ui/console`.
- Backend tests cover at least one new insight category and empty data behavior.

## Dependencies

- WO-PROD-001
- WO-API-001
- WO-QA-001 recommended

## Non-Goals

- No external enrichment feed.
- No enforcement.
- No broad ABOM taxonomy rewrite unless required by the insight model.

## Suggested Verification

- Backend tests for ABOM insight aggregation.
- `npm run build` in `ui/console`.
- `npm run test:e2e` if the harness covers the new route or sidebar.

