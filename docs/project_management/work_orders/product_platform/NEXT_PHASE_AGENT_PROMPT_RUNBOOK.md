# AegisFlux Next Phase Agent Prompt Runbook

**Purpose:** Copy-ready prompts for the next work-order batch after the product-platform and UX work orders reached lab-slice completion.

## Recommended Order

1. WO-QA-001: UI Rendering and Navigation Regression Harness
2. WO-PERF-001: Console Performance and Data Loading Pass
3. WO-API-001: Console Summary Endpoints

## Execution Rules for Every Agent

- Start with `git status -sb`.
- Read the assigned work order before editing.
- Read relevant docs listed in the work order.
- Do not revert unrelated user or agent changes.
- Keep changes scoped to the assigned work order.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Stage only files changed for the work order.
- Commit and push at the end.

## WO-QA-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-QA-001_UI_RENDERING_AND_NAVIGATION_REGRESSION_HARNESS.md

Goal: add repeatable UI regression checks for the AegisFlux console so shell rendering, auth redirects, row navigation, and bounded-detail behavior do not regress.

Before editing:
- Run git status -sb.
- Read the work order.
- Inspect ui/console/package.json, ui/console/app/page.tsx, ui/console/components/shell/ConsoleShell.tsx, ui/console/components/AgentsManagementPanel.tsx, and ui/console/app/agents/[device_id]/page.tsx.

Implementation scope:
- Add the smallest practical local regression harness for console routes.
- Verify dashboard, agents, agent detail, inventory, detections, controls, and operational events.
- Include a lab-auth setup path for browser-level tests.
- Check that primary routes render inside the persistent shell with left navigation visible.
- Check that agent rows navigate to agent detail when data is available.
- Check that agent detail does not remain stuck on Checking session after navigation settles.
- Document how to run the harness locally and how to wire it into CI later.
- Update WO-QA-001 status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Run the new regression command.
- If browser automation is unavailable, document the blocker and run route smoke checks.

At the end:
- Run git status -sb.
- Stage only files changed for WO-QA-001.
- Commit with: git commit -m "Add console UI regression harness"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PERF-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PERF-001_CONSOLE_PERFORMANCE_AND_DATA_LOADING_PASS.md

Goal: measure and improve console responsiveness without hiding meaningful operator context.

Before editing:
- Run git status -sb.
- Read WO-PERF-001.
- Inspect ui/console/app/page.tsx, ui/console/components/AgentsManagementPanel.tsx, ui/console/app/agents/[device_id]/page.tsx, ui/console/components/InventoryPanel.tsx, ui/console/app/detections/page.tsx, and ui/console/app/operate/events/page.tsx.

Implementation scope:
- Record current fetch cadence and route data limits.
- Identify the top slow/high-fan-out routes.
- Reduce unnecessary polling and fetch fan-out where safe.
- Ensure large tables/lists remain bounded and searchable.
- Do not introduce a broad state-management rewrite.
- If summary APIs are needed, update WO-API-001 notes rather than expanding scope.
- Update WO-PERF-001 status and implementation notes with before/after observations.

Verification:
- Run npm run build in ui/console.
- Run route smoke checks for /, /agents, /agents/[device_id], /inventory, /detections, and /operate/events.
- Run the WO-QA-001 harness if it exists.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PERF-001.
- Commit with: git commit -m "Improve console performance baselines"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-API-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-API-001_CONSOLE_SUMMARY_ENDPOINTS.md

Goal: add read-only backend summary endpoints so the console can load operator summaries without client-side fan-out over raw telemetry.

Before editing:
- Run git status -sb.
- Read WO-API-001 and WO-PERF-001 implementation notes if present.
- Inspect backend visibility/action API services and the UI routes that currently aggregate raw telemetry.

Implementation scope:
- Define summary payloads for the highest-impact route first.
- Prefer dashboard, agents workbench, or agent detail based on WO-PERF-001 findings.
- Implement read-only endpoints with empty-data behavior.
- Add tests for summary shape and no-data behavior.
- If scope permits, update the UI to consume one summary endpoint.
- Keep raw telemetry APIs intact.
- Update WO-API-001 status and implementation notes.

Verification:
- Run relevant backend tests for touched modules.
- Run npm run build in ui/console if UI changes are made.
- Run route smoke checks for any migrated UI surface.

At the end:
- Run git status -sb.
- Stage only files changed for WO-API-001.
- Commit with: git commit -m "Add console summary endpoints"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

