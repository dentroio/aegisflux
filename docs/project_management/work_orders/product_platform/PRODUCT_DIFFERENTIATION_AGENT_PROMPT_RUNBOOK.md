# AegisFlux Product Differentiation Agent Prompt Runbook

**Purpose:** Copy-ready prompts for work orders that make AegisFlux stand out as an evidence-backed AI-era endpoint control-design platform.

## Recommended Order

1. WO-PROD-001: Agent Bill of Materials
2. WO-PROD-002: Evidence Graph Investigation Path
3. WO-PROD-003: Finding-to-Control Design Workflow
4. WO-PROD-004: AI Research Feed and Detection Opportunities
5. WO-PROD-005: First-Value Demo and Operator Onboarding

## Product Bar

Each work order should make at least one of these true:

- Aegis detects AI-agent behavior that ordinary tools miss.
- Aegis explains endpoint behavior in a way an operator can trust.
- Aegis turns evidence into a safer control proposal.
- Aegis reduces blast radius before enforcement.
- Aegis gives Clarion endpoint context it cannot get from network telemetry alone.
- Aegis makes AI usage governable without blocking useful work.

## Execution Rules for Every Agent

- Start with `git status -sb`.
- Read the assigned work order and relevant product docs.
- Do not revert unrelated user or agent changes.
- Keep changes scoped to the assigned work order.
- Use operator-centered labels and avoid raw telemetry buckets as primary UX.
- Preserve observe-only behavior unless the work order explicitly says otherwise.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Stage only files changed for the work order.
- Commit and push at the end.

## WO-PROD-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PROD-001_AGENT_BILL_OF_MATERIALS.md

Goal: build the first Agent Bill of Materials slice so AegisFlux can show AI-capable tools, capabilities, reachability, and evidence across endpoints.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PRODUCT_ROADMAP.md and docs/AEGIS_PLATFORM_VISION.md.
- Read the work order.
- Inspect existing inventory and visibility code before designing anything new.

Implementation scope:
- Define the ABOM item shape and category taxonomy.
- Add backend aggregation or summary support if needed.
- Add fleet and endpoint UI surfaces if in scope.
- Keep raw telemetry behind bounded detail.
- Update the work order status and implementation notes.

Verification:
- Run relevant backend tests for touched modules.
- Run npm run build in ui/console if UI changes are made.
- Verify with available lab agent data or documented empty-state behavior.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PROD-001.
- Commit with: git commit -m "Add Agent Bill of Materials slice"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PROD-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PROD-002_EVIDENCE_GRAPH_INVESTIGATION_PATH.md

Goal: create an evidence path that connects findings to process, network, DNS, endpoint, detection, and draft-control context.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PRODUCT_ROADMAP.md and docs/AEGIS_SENSOR_FUSION_ARCHITECTURE.md.
- Read the work order.
- Inspect existing visibility storage/API and agent detail UI.

Implementation scope:
- Define node and edge types.
- Build the smallest useful path API or path builder.
- Add compact UI path presentation if in scope.
- Show missing evidence clearly.
- Keep raw records available behind bounded detail.
- Update the work order status and implementation notes.

Verification:
- Run backend tests for path building.
- Run npm run build in ui/console if UI changes are made.
- Verify a complete and partial-evidence scenario.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PROD-002.
- Commit with: git commit -m "Add evidence graph investigation path"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PROD-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PROD-003_FINDING_TO_CONTROL_DESIGN_WORKFLOW.md

Goal: turn findings into observe-only draft controls with evidence, scope, blast-radius preview, and rollback notes.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PRODUCT_ROADMAP.md.
- Read WO-CTRL-001 and WO-PROD-003.
- Inspect control and finding APIs/UI.

Implementation scope:
- Add or refine draft-control proposal shape.
- Generate first proposals from AI-related findings.
- Add UI flow from finding to proposed observe-only control if in scope.
- Add operational events for draft creation/update if practical.
- Keep enforcement inactive.
- Update the work order status and implementation notes.

Verification:
- Run backend tests for draft-control creation/simulation inputs.
- Run npm run build in ui/console if UI changes are made.
- Verify the UI clearly says observe-only, not enforcing.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PROD-003.
- Commit with: git commit -m "Add finding to control design workflow"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PROD-004 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PROD-004_AI_RESEARCH_FEED_AND_DETECTION_OPPORTUNITIES.md

Goal: create a governed research opportunity queue that turns AI ecosystem intelligence into detection candidates.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md.
- Read WO-DET-002 and WO-PROD-004.
- Inspect detection candidate and operational event code.

Implementation scope:
- Define research opportunity lifecycle.
- Add storage/API/UI for opportunities if in scope.
- Seed examples for MCP, coding agents, browser automation, and local model runtimes.
- Allow promote-to-candidate or reject.
- Emit operational events where practical.
- Update the work order status and implementation notes.

Verification:
- Run backend tests for opportunity lifecycle.
- Run npm run build in ui/console if UI changes are made.
- Verify promotion/rejection behavior.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PROD-004.
- Commit with: git commit -m "Add AI research opportunity queue"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PROD-005 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PROD-005_FIRST_VALUE_DEMO_AND_OPERATOR_ONBOARDING.md

Goal: create a first-value workflow that shows why AegisFlux matters in under five minutes.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PRODUCT_ROADMAP.md and docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Read the work order.
- Inspect dashboard, agents, inventory, controls, and docs.

Implementation scope:
- Define and implement the guided first-value path where practical.
- Use operator-centered labels.
- Avoid marketing-only pages; build product workflow entry points.
- Add missing-data empty states and demo preparation docs.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console if UI changes are made.
- Run route checks for the guided path.
- Manually walk through with available lab data or documented empty states.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PROD-005.
- Commit with: git commit -m "Add first value demo workflow"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

