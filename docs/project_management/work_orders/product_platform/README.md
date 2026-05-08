# AegisFlux Product Platform Work Orders

**Program:** AegisFlux product platform and Clarion-aligned management experience  
**Phase:** Phase 2 - Productization and AI-Agent Platform  
**Status:** Draft  
**Last updated:** May 8, 2026

## Objective

**Product promise:** Observe. Adapt. Enforce.  
**Positioning:** Adaptive security. Real-time protection.

Turn the current AegisFlux lab visibility foundation into a product-shaped platform that can be worked on safely by multiple agents.

The product direction is solid:

- AegisFlux is the AI-era endpoint evidence and local-control platform.
- AegisFlux endpoint agents stay lightweight, optimized, and observe-first.
- AegisFlux backend and UI turn endpoint evidence into investigation, inventory, detection packs, and draft controls.
- AegisFlux AI platform agents research, explain, validate, and draft, but do not bypass governance.
- AegisFlux adapts through signed detection packs, validated candidates, and continuous research-to-detection workflows.
- AegisFlux enforces only after evidence, simulation, approval, and rollback readiness are in place.
- Clarion remains the broader enterprise context and policy intelligence platform; AegisFlux eventually integrates as the host/workload evidence and endpoint-control subsystem.

## Architecture Baseline

These documents are the baseline for this program:

- [AegisFlux Product Roadmap](../../../AEGIS_PRODUCT_ROADMAP.md)
- [AegisFlux Platform Vision](../../../AEGIS_PLATFORM_VISION.md)
- [Dynamic AI Detection Strategy](../../../AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md)
- [AI Agent Platform Architecture](../../../AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md)
- [Sensor Fusion Architecture](../../../AEGIS_SENSOR_FUSION_ARCHITECTURE.md)
- [Agent Performance Architecture](../../../AEGIS_AGENT_PERFORMANCE_ARCHITECTURE.md)
- [UI Clarion Alignment](../../../AEGIS_UI_CLARION_ALIGNMENT.md)
- [Clarion UI Patterns to Adopt](../../../AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md)
- [AegisFlux and Clarion Integration Contract](../../plans/AEGIS_CLARION_INTEGRATION_CONTRACT.md)

## Product Workflow Target

The AegisFlux UI should follow the Clarion shell pattern:

- Top header: product identity, global search, notifications, documentation, AI assistant, health, user controls.
- Left navigation grouped by workflow:
  - Overview
  - Discover
  - Analyze
  - Control
  - Operate
  - Configure
- Dashboard stays clean and high-level.
- Device and evidence detail live behind drill-in pages.
- Management pages own configuration, AI providers, detection packs, privacy, audit, and settings.

## Work Order Sequence

| ID | Title | Primary Output | Depends On |
|----|-------|----------------|------------|
| WO-PLAT-001 | Clarion-Aligned Console Shell | Shared shell, header, nav groups, route placeholders | None |
| WO-PLAT-002 | Agent List and Device Drill-In | `/agents` list and `/agents/[device_id]` detail workflow | WO-PLAT-001 |
| WO-PLAT-003 | Custom Dashboard Widget Framework | Persisted dashboard widget layout/config | WO-PLAT-001 |
| WO-AI-001 | AI Provider Management and Health | AI providers page, default provider, health endpoint/UI | WO-PLAT-001 |
| WO-AI-002 | AI Privacy and Audit Foundation | Redaction settings and AI run/audit records | WO-AI-001 |
| WO-AI-003 | Endpoint Evidence Analyst | First context-aware AI action on device detail | WO-PLAT-002, WO-AI-002 |
| WO-DET-001 | Detection Pack Schema and Local Evaluator Contract | Versioned signed-pack schema for endpoint agents | Existing visibility schemas |
| WO-DET-002 | Dynamic AI Detection Candidate Pipeline | Research item -> candidate -> validation -> approved pack workflow | WO-DET-001, WO-AI-002 |
| WO-DET-003 | Detection Pack Controller Rollout and Agent Status | Latest approved pack API, artifact retrieval, and per-agent pack status | WO-DET-001, WO-DET-002 |
| WO-DET-004 | Linux Dynamic Detection Pack Evaluator | Linux fetch/verify/cache/evaluate/status for observe-only packs | WO-DET-001, WO-DET-003 |
| WO-DET-005 | Windows Dynamic Detection Pack Evaluator | Windows fetch/verify/cache/evaluate/status for observe-only packs | WO-DET-001, WO-DET-003 |
| WO-LAB-001 | Agent Tunnel and Ingest Reliability | Reliable lab agent connectivity and ingest reachability | WO-DET-004, WO-DET-005 |
| WO-DET-006 | Repeatable Detection Rollout Smoke | One-command lab validation for signed-pack rollout | WO-DET-002, WO-DET-003, WO-DET-004, WO-DET-005, WO-LAB-001 |
| WO-AGENT-001 | Agent Performance Budget Telemetry | CPU/memory/runtime collector budget events and UI | Windows/Linux agents reporting |
| WO-INV-001 | Enterprise AI and Control Inventory | Browser, IDE, CLI, local model, SASE/SSE inventory pages | Visibility events and agents |
| WO-PLAT-004 | Detection Pack Status Visibility | Pack rollout health in agent list, device detail, and rollout views | WO-PLAT-002, WO-DET-003, WO-DET-006 |
| WO-PLAT-005 | Clarion-Style Agent Workbench and Detail Experience | Agent workbench, Clarion-style detail tabs, evidence confidence, next best action | WO-PLAT-001, WO-PLAT-002, WO-PLAT-004, WO-AGENT-001, WO-INV-001 |
| WO-CTRL-001 | Observe-Only Draft Controls and Simulation | Finding -> draft control -> historical match simulation | WO-PLAT-002, WO-DET-001 |
| WO-PLAT-006 | Operational Event Feed | Auditable platform event feed for rollout, approval, AI, controls, and integration activity | WO-PLAT-001, WO-DET-002, WO-DET-003 |
| WO-INT-001 | Clarion Integration API Slice | AegisFlux evidence export and Clarion import contract implementation | WO-PLAT-002, WO-INV-001 |

## Parallelization Guidance

These can run in parallel with low conflict:

- WO-PLAT-001 and WO-DET-001 after agreeing on route and schema names.
- WO-AI-001 and WO-AGENT-001 after backend ownership is clear.
- WO-INV-001 can start from existing browser/SASE events while WO-AI work proceeds.
- WO-DET-003 should start after WO-DET-002 defines signed approved-pack artifacts; Windows/Linux evaluator work can branch from its status contract.
- WO-DET-004 and WO-DET-005 should run in parallel after WO-DET-003 stabilizes its endpoint contract; keep Linux and Windows write scopes separate.
- WO-LAB-001 should run immediately after evaluator basics exist, because lab instability will obscure real rollout bugs.
- WO-DET-006 should follow WO-LAB-001 and become the regression check for the dynamic detection path.
- WO-PLAT-004 can start once WO-PLAT-002 exists and the WO-DET-003 status contract is stable.
- WO-CTRL-001 should wait until drill-in and detection-pack schema stabilize.
- WO-PLAT-005 should happen before AI-heavy endpoint work so the detail surface has the right Clarion-style structure.
- WO-PLAT-006 can start once at least detection candidate/rollout events are available; it can add producers incrementally as AI, controls, and Clarion export work land.
- WO-INT-001 should wait until the AegisFlux evidence model is not churning daily.

## Phase Exit Criteria

- AegisFlux UI has a Clarion-like shell and clear workflow navigation.
- Dashboard is high-level and customizable without becoming an investigation page.
- Operators can drill from agent list into device detail.
- AI providers, AI health, privacy, and audit are first-class management capabilities.
- Endpoint evidence can be explained by an audited AI action.
- Detection packs are data, signed/versioned, and safe for low-resource endpoint evaluation.
- Detection-pack rollout status is visible per device and per pack.
- The lab has a repeatable smoke test that proves dynamic detection rollout end to end.
- Agent resource usage is visible and bounded.
- Inventory surfaces show AI tools, browser extensions, local model runtimes, and enterprise control components.
- Findings can produce observe-only draft controls and simulations.
- Clarion integration has a concrete first API/event slice.

## Non-Goals

- No production enforcement by default.
- No external LLM calls from endpoint agents.
- No arbitrary code in detection packs.
- No broad UI rewrite outside the active AegisFlux console unless explicitly scoped.
- No Clarion database coupling.
