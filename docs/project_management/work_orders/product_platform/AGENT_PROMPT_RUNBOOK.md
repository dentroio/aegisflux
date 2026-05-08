# AegisFlux Product Platform Agent Prompt Runbook

**Purpose:** Keep the remaining work-order queue and copy-ready agent prompts in one place. Each prompt tells the agent to verify, commit, and push at the end of the work order.

**Queue status as of May 8, 2026:**

- Remaining existing work orders before creating new ones: **7**
- New UI work orders created from recent Clarion design discussions: **2**
- Total open work orders after adding the new UI work orders: **9**

## Execution Rules for Every Agent

Every agent should follow these rules:

- Start with `git status -sb`.
- Read the referenced work order and relevant docs before editing.
- Do not revert or overwrite unrelated user/agent changes.
- Keep changes scoped to the work order.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Commit and push at the end.
- If the worktree contains unrelated dirty files, stage only the files owned by the work order.

## Recommended Order

1. WO-PLAT-001: close/reconcile shell status.
2. WO-PLAT-005: Clarion-style Agent Workbench and Detail.
3. WO-PLAT-003: Custom Dashboard Widget Framework.
4. WO-AI-001: AI Provider Management and Health.
5. WO-AI-002: AI Privacy and Audit Foundation.
6. WO-AI-003: Endpoint Evidence Analyst.
7. WO-CTRL-001: Observe-Only Draft Controls and Simulation.
8. WO-PLAT-006: Operational Event Feed.
9. WO-INT-001: Clarion Integration API Slice.

## WO-PLAT-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PLAT-001_CLARION_ALIGNED_CONSOLE_SHELL.md

Goal: verify and close the Clarion-aligned console shell work. The AegisFlux shell should follow Clarion's pattern: persistent top header, persistent left workflow navigation, breadcrumb row, and a main/right content area that scrolls independently.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_CLARION_ALIGNMENT.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect ui/console/app/page.tsx, ui/console/app/layout.tsx, and any shell-related components.

Implementation scope:
- Reconcile the current shell against the work-order acceptance criteria.
- Ensure the left menu remains visible while dashboard, agents, and inventory panels render in the main/right panel.
- Ensure /agents and /inventory deep links preserve shell navigation or redirect into the shell panel.
- If shell pieces are still too tangled, extract only small reusable shell helpers/components that reduce risk and match the existing code style.
- Update WO-PLAT-001 status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Verify http://127.0.0.1:3030, /agents, and /inventory behavior if the local dev server is available.
- Do not modify unrelated work orders or backend files.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PLAT-001.
- Commit with: git commit -m "Complete Clarion-aligned console shell"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PLAT-005 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PLAT-005_CLARION_STYLE_AGENT_WORKBENCH_AND_DETAIL.md

Goal: bring Clarion's endpoint workbench and endpoint detail patterns into AegisFlux Agents.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Read docs/AEGIS_UI_CLARION_ALIGNMENT.md.
- Read the Clarion references:
  - /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Devices.tsx
  - /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/endpoints/EndpointDetail.tsx
- Inspect Aegis files:
  - ui/console/components/AgentsManagementPanel.tsx
  - ui/console/app/agents/[device_id]/page.tsx
  - ui/console/components/InventoryPanel.tsx
  - ui/console/app/page.tsx

Implementation scope:
- Convert Agents into a Clarion-style workbench inside the persistent shell.
- Add KPI/context cards, quick-filter tabs with counts, search/scoped filters, table/card view toggle, and persisted view/column preferences.
- Rework agent detail into a Clarion-style investigation surface with identity header, freshness row, status chips, Evidence Confidence, Network Context, Detection / Policy Context, and Next Best Action cards.
- Add tabs: Overview, Evidence, Inventory, Detection Packs, Performance, Policy.
- Use deterministic Next Best Action logic; do not add AI calls.
- Keep text compact and prevent overflow.
- Update WO-PLAT-005 status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Verify the Agents shell panel and at least one Linux and one Windows agent detail path if local data is available.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PLAT-005.
- Commit with: git commit -m "Add Clarion-style agent workbench"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PLAT-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PLAT-003_CUSTOM_DASHBOARD_WIDGET_FRAMEWORK.md

Goal: make the dashboard customizable without making it noisy.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_CLARION_ALIGNMENT.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect ui/console/app/page.tsx and any dashboard/widget components.

Implementation scope:
- Create or formalize a dashboard widget registry with id, title, description, data source, default size, and enabled state.
- Persist widget preferences locally unless a backend config already exists.
- Add show/hide and ordering controls without complex drag-and-drop unless already easy.
- Include default widgets: Platform Status, Endpoint Freshness, AI Activity, Detection Pack Coverage, Agent Performance Budget, Enterprise Control Inventory.
- Make empty/loading states polished.
- Update WO-PLAT-003 status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Verify widget preferences survive refresh in the browser/local storage if a dev server is available.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PLAT-003.
- Commit with: git commit -m "Add customizable dashboard widgets"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-AI-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-AI-001_AI_PROVIDER_MANAGEMENT_AND_HEALTH.md

Goal: bring Clarion's governed AI provider pattern into AegisFlux.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect Clarion AI provider UI/API patterns where useful:
  - /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/connectors/AI.tsx
  - /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/components/AIProviderModal.tsx
  - /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/hooks/useAIServiceHealth.ts
- Inspect Aegis backend services and UI routing before choosing where provider config should live.

Implementation scope:
- Add provider model for local, OpenAI, Anthropic, Google, and future enterprise gateway.
- Add default provider selection.
- Add provider test endpoint and health endpoint.
- Add UI under Configure / Connectors / AI Providers in the persistent shell.
- Add AI health chip to header/dashboard if it can be done cleanly.
- Never return provider secrets to the UI.
- Update WO-AI-001 status and implementation notes.

Verification:
- Run backend tests for touched backend packages.
- Run npm run build in ui/console.
- Verify provider list and test/health behavior with safe local/mock config.

At the end:
- Run git status -sb.
- Stage only files changed for WO-AI-001.
- Commit with: git commit -m "Add AI provider management and health"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-AI-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-AI-002_AI_PRIVACY_AND_AUDIT_FOUNDATION.md

Goal: make all AI-agent usage auditable and safe for endpoint evidence.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md.
- Read WO-AI-001 implementation notes and code.
- Inspect Clarion privacy/audit patterns if available.

Implementation scope:
- Add privacy settings for external AI requests.
- Add redaction pipeline for IPs/CIDRs, MACs, usernames, emails, hostnames, command-line secrets, file paths, and raw secrets.
- Add AI run records and privacy audit records.
- Add UI under Configure / Settings or Configure / AI Providers.
- Ensure raw provider secrets and raw sensitive payloads are not exposed.
- Update WO-AI-002 status and implementation notes.

Verification:
- Add/run tests for redaction examples: command lines, tokens, IPs, MACs, usernames/emails, hostnames, and file paths.
- Run relevant backend tests.
- Run npm run build in ui/console.

At the end:
- Run git status -sb.
- Stage only files changed for WO-AI-002.
- Commit with: git commit -m "Add AI privacy and audit foundation"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-AI-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-AI-003_ENDPOINT_EVIDENCE_ANALYST.md

Goal: add the first AegisFlux AI platform agent action: explain AI activity on a selected endpoint using bounded, auditable context.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md.
- Read WO-AI-001 and WO-AI-002 implementation notes and code.
- Inspect agent/device detail UI from WO-PLAT-005 if complete.

Implementation scope:
- Register AI agent endpoint_evidence_analyst.
- Add backend route that builds bounded endpoint context from findings, DNS, process, flow, browser extension, SASE/SSE, collector health, and detection-pack status.
- Add UI action on device/agent detail: Explain AI activity.
- Response sections: Assessment, Evidence, Confidence, Recommended next action.
- If AI is unavailable, show deterministic fallback summary.
- Link AI run/audit record to device id.
- Update WO-AI-003 status and implementation notes.

Verification:
- Verify action works for Windows lab device.
- Verify action works for Linux lab device even without browser evidence.
- Run relevant backend tests.
- Run npm run build in ui/console.

At the end:
- Run git status -sb.
- Stage only files changed for WO-AI-003.
- Commit with: git commit -m "Add endpoint evidence analyst"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-CTRL-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-CTRL-001_OBSERVE_ONLY_DRAFT_CONTROLS_AND_SIMULATION.md

Goal: turn findings into observe-only draft controls that are explainable and simulated before any enforcement work begins.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_PLATFORM_VISION.md.
- Read docs/AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect visibility findings APIs and detection-pack schema.

Implementation scope:
- Add draft control model with source finding, proposed action, scope selectors, evidence references, expected effect, blast radius, rollback plan, and status.
- Add simulation endpoint that runs draft scope against historical events.
- Add UI under Control / Controls in the persistent shell.
- Add finding/detail action: Draft observe-only control.
- Make UI unmistakably observe-only; no enforcement.
- Update WO-CTRL-001 status and implementation notes.

Verification:
- Add/run backend tests for draft model and simulation.
- Run npm run build in ui/console.
- Verify a finding can generate a draft and show historical matches.

At the end:
- Run git status -sb.
- Stage only files changed for WO-CTRL-001.
- Commit with: git commit -m "Add observe-only draft controls"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-PLAT-006 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-PLAT-006_OPERATIONAL_EVENT_FEED.md

Goal: create a first operational event feed for platform actions, status transitions, approvals, rollout events, AI activity, and integration exports.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect Clarion audit feed: /Users/sgerhart/workspace/github/sgerhart/clarion/frontend/src/pages/Audit.tsx
- Inspect detection candidate, rollout, and agent status APIs.

Implementation scope:
- Add operational event model and append/list API.
- Add UI under Operate / Operational Event Feed in the persistent shell.
- Add filters for type, status, subject, device/agent, and time.
- Add producers where practical for detection candidate lifecycle and detection-pack rollout events.
- Leave hooks for AI events, draft control events, and Clarion export events if those work orders are not complete yet.
- Update WO-PLAT-006 status and implementation notes.

Verification:
- Add/run backend tests for event append/list.
- Run npm run build in ui/console.
- Verify feed handles no events and recent events.

At the end:
- Run git status -sb.
- Stage only files changed for WO-PLAT-006.
- Commit with: git commit -m "Add operational event feed"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-INT-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-INT-001_CLARION_INTEGRATION_API_SLICE.md

Goal: implement the first concrete AegisFlux to Clarion integration slice without coupling databases or collapsing product boundaries.

Before editing:
- Run git status -sb.
- Read docs/project_management/plans/AEGIS_CLARION_INTEGRATION_CONTRACT.md.
- Read docs/AEGIS_PLATFORM_VISION.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect current ingest/visibility APIs and inventory outputs.

Implementation scope:
- Add API/export for device evidence summary without requiring Clarion to run.
- Include device id, agent id, OS/source, freshness, AI activity summary, inventory summary, findings, and evidence links.
- Document event contracts for aegis.device.observed, aegis.ai_activity.summarized, aegis.inventory.item_observed, and aegis.finding.created.
- Add sample payloads for Windows and Linux lab devices.
- Add Clarion mapping notes for endpoint, device, user/session, flow, destination, and finding context.
- Avoid direct DB writes to Clarion and avoid Clarion repo changes.
- Update WO-INT-001 status and implementation notes.

Verification:
- Add/run tests for schema/payload validation.
- Verify export works without Clarion running.
- Run relevant backend tests.

At the end:
- Run git status -sb.
- Stage only files changed for WO-INT-001.
- Commit with: git commit -m "Add Clarion integration API slice"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

