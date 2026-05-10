# AegisFlux Growth Agent Prompt Runbook

**Purpose:** Copy-ready prompts for the next AegisFlux growth wave after the first product-differentiation slice.

## North Star

Build toward the loop that makes AegisFlux worth using:

**Discover -> Explain -> Design -> Simulate -> Govern -> Adapt**

The product should not feel like another telemetry console. Every work order should help an operator understand AI-capable endpoint activity, trust the evidence, and design safer controls.

## Recommended Order

1. WO-GROWTH-001: ABOM Fleet Insights and Change Detection
2. WO-GROWTH-002: Evidence Graph UX and Explainability
3. WO-GROWTH-003: Control Simulation Depth and Decision History
4. WO-GROWTH-004: Agent Health and Readiness Scoring
5. WO-GROWTH-005: Adaptive Detection Workflow Maturity
6. WO-GROWTH-006: First-Value Demo Polish and Sample Scenarios
7. WO-GROWTH-007: Audit-Mode Enforcement Adapter Foundation

## Execution Rules for Every Agent

- Start with `git status -sb`.
- Read the assigned work order and relevant roadmap docs.
- Do not revert unrelated user or agent changes.
- Keep changes scoped to the assigned work order.
- Use operator-centered labels; avoid raw telemetry categories as primary UX.
- Preserve observe-only behavior unless the work order explicitly says audit-mode.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Stage only files changed for the work order.
- Commit and push at the end.

## WO-GROWTH-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-001_ABOM_FLEET_INSIGHTS_AND_CHANGE_DETECTION.md

Goal: turn ABOM from inventory into daily fleet insights: new, risky, widespread, low-confidence, and stale AI capabilities.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PRODUCT_ROADMAP.md and WO-PROD-001.
- Inspect ABOM backend query code and ABOM UI components/routes.

Implementation scope:
- Add ABOM insight categories and time-window/change detection.
- Add fleet UI sections for new, needs review, widespread, and endpoint hotspots.
- Keep raw evidence behind bounded detail.
- Update work order status and implementation notes.

Verification:
- Run backend tests for ABOM insight aggregation.
- Run npm run build in ui/console.
- Run npm run test:e2e if route coverage is affected.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-001.
- Commit with: git commit -m "Add ABOM fleet insights"
- Push with: git push.
```

## WO-GROWTH-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-002_EVIDENCE_GRAPH_UX_AND_EXPLAINABILITY.md

Goal: make evidence graph read like an explanation: what happened, why it matters, what is known, what is missing, and next step.

Before editing:
- Run git status -sb.
- Read WO-PROD-002 and docs/AEGIS_SENSOR_FUSION_ARCHITECTURE.md.
- Inspect evidence path backend and UI components/routes.

Implementation scope:
- Add plain-language evidence summaries and confidence reasons.
- Improve missing-evidence copy.
- Add links into ABOM and finding-to-control where available.
- Keep raw evidence bounded/collapsible.
- Update work order status and implementation notes.

Verification:
- Run existing evidence path backend tests.
- Run npm run build in ui/console.
- Manual walkthrough with complete and partial paths.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-002.
- Commit with: git commit -m "Improve evidence graph explainability"
- Push with: git push.
```

## WO-GROWTH-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-003_CONTROL_SIMULATION_DEPTH_AND_DECISION_HISTORY.md

Goal: deepen observe-only draft-control simulation and add decision history.

Before editing:
- Run git status -sb.
- Read WO-PROD-003 and WO-PLAT-006.
- Inspect draft-control backend state/API and control UI.

Implementation scope:
- Add richer simulation output and draft revision history.
- Add compare-before/after scope view.
- Add operator decision notes and operational events.
- Keep all behavior observe-only.
- Update work order status and implementation notes.

Verification:
- Run backend tests for draft simulation/history.
- Run npm run build in ui/console.
- Manual create/edit/simulate/history walkthrough.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-003.
- Commit with: git commit -m "Deepen control simulation history"
- Push with: git push.
```

## WO-GROWTH-004 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-004_AGENT_HEALTH_AND_READINESS_SCORING.md

Goal: add agent health/readiness scoring so operators know whether endpoint evidence can be trusted.

Before editing:
- Run git status -sb.
- Read WO-AGENT-001, WO-PLAT-005, and WO-GROWTH-004.
- Inspect agent summary APIs and Agents UI.

Implementation scope:
- Define readiness score dimensions and buckets.
- Add backend summary support.
- Add fleet and endpoint readiness UI.
- Explain what to fix first in plain language.
- Update work order status and implementation notes.

Verification:
- Run backend tests for score buckets.
- Run npm run build in ui/console.
- Manual check with online/stale lab agents.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-004.
- Commit with: git commit -m "Add agent readiness scoring"
- Push with: git push.
```

## WO-GROWTH-005 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-005_ADAPTIVE_DETECTION_WORKFLOW_MATURITY.md

Goal: mature research opportunity -> candidate -> simulation -> signed pack -> rollout -> retirement workflow.

Before editing:
- Run git status -sb.
- Read WO-PROD-004, WO-DET-002, WO-DET-003, and docs/AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md.
- Inspect research feed, detection candidates, signed pack, rollout, and operational event code.

Implementation scope:
- Add quality gates and simulation summaries.
- Link research rationale to candidates, packs, and rollout status.
- Add retirement/deprecation path.
- Emit operational events.
- Update work order status and implementation notes.

Verification:
- Run backend tests for lifecycle transitions.
- Run npm run build in ui/console.
- Manual walkthrough from research feed to rollout status.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-005.
- Commit with: git commit -m "Mature adaptive detection workflow"
- Push with: git push.
```

## WO-GROWTH-006 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-006_FIRST_VALUE_DEMO_POLISH_AND_SAMPLE_SCENARIOS.md

Goal: polish the first-value demo with sample scenarios, setup docs, and better empty states.

Before editing:
- Run git status -sb.
- Read WO-PROD-005 and docs/AEGIS_PRODUCT_ROADMAP.md.
- Inspect /demo, dashboard banner, ABOM, evidence, controls, and research routes.

Implementation scope:
- Add sample scenario docs and demo checklist.
- Improve empty states for demo-dependent routes.
- Add sample-data mode only if it does not pollute production paths.
- Update work order status and implementation notes.

Verification:
- Run npm run build in ui/console if UI changed.
- Manual walkthrough using lab or documented sample scenario.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-006.
- Commit with: git commit -m "Polish first value demo scenarios"
- Push with: git push.
```

## WO-GROWTH-007 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GROWTH-007_AUDIT_MODE_ENFORCEMENT_ADAPTER_FOUNDATION.md

Goal: create an audit-mode adapter foundation that proves safe policy delivery and match telemetry without blocking.

Before editing:
- Run git status -sb.
- Read WO-GROWTH-003, WO-GROWTH-004, WO-DET-003, and docs/AEGIS_PRODUCT_ROADMAP.md.
- Inspect signed pack rollout/status patterns and agent policy/control code.

Implementation scope:
- Define audit-mode policy bundle shape.
- Add backend staging/status path.
- Add one lab audit-mode adapter path where practical.
- Add UI status visibility.
- Document safety boundaries: audit-only, no blocking.
- Update work order status and implementation notes.

Verification:
- Run backend/agent tests for staging/status/receipt.
- Run lab smoke if adapter code lands.
- Run npm run build in ui/console if UI changed.

At the end:
- Run git status -sb.
- Stage only files changed for WO-GROWTH-007.
- Commit with: git commit -m "Add audit mode enforcement foundation"
- Push with: git push.
```

