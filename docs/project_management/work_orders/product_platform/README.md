# AegisFlux Product Platform Work Orders

**Program:** AegisFlux product platform and Clarion-aligned management experience  
**Phase:** Phase 3 complete; operational readiness and AI-native leap planned  
**Status:** Living roadmap (completed waves plus planned readiness/AI-native waves)  
**Last updated:** May 12, 2026

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

## Current Product Loop

The next work-order wave drives AegisFlux toward the loop that should make people want to use it:

**Discover -> Explain -> Design -> Simulate -> Govern -> Adapt**

- **Discover:** Agent Bill of Materials and fleet AI-capability insights.
- **Explain:** Evidence graph and plain-language investigation paths.
- **Design:** Findings become observe-only control proposals.
- **Simulate:** Operators see blast radius before any enforcement work.
- **Govern:** Agent health, decision history, operational audit, and approval readiness.
- **Adapt:** Research opportunities mature into governed signed detection packs.

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
- [AegisFlux UI Simplification Guide](../../../AEGIS_UI_SIMPLIFICATION_GUIDE.md)
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

The completed product-platform sequence is retained below for history. The current open operational-readiness queue is tracked in [Open Work Order Queue](OPEN_WORK_ORDER_QUEUE.md). The next AI-native capability wave is tracked in [AI-Native Leap Work Order Queue](AI_NATIVE_LEAP_WORK_ORDER_QUEUE.md).

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
| WO-UX-001 | Clarion-Style Dashboard Simplification | Dashboard scan surface with bounded widgets and deliberate click-throughs | WO-PLAT-001, WO-PLAT-003 |
| WO-UX-002 | Interaction and String Formatting System | Shared bounded-detail and long-string formatting primitives | WO-UX-001 |
| WO-UX-003 | Agents Workbench Simplification | Focused agents table/list without permanent long-scroll detail column | WO-UX-002, WO-PLAT-005 |
| WO-UX-004 | Inventory Workbench Simplification | Searchable category-driven inventory workbench | WO-UX-002, WO-INV-001 |
| WO-UX-005 | Detections, Control, and Operate UX Simplification | Bounded candidate/draft/event workflows | WO-UX-002, WO-DET-002, WO-CTRL-001, WO-PLAT-006 |
| WO-QA-001 | UI Rendering and Navigation Regression Harness | Repeatable checks for shell, auth, route rendering, row navigation, and bounded pages | WO-UX-001 through WO-UX-005 |
| WO-PERF-001 | Console Performance and Data Loading Pass | Measured route performance, bounded data loading, and fetch cadence cleanup | WO-QA-001 recommended |
| WO-API-001 | Console Summary Endpoints | Read-only backend summaries for dashboard, agents, agent detail, and inventory | WO-PERF-001 findings |
| WO-PROD-001 | Agent Bill of Materials | Evidence-backed inventory of AI-capable tools, capabilities, and reachability | WO-INV-001, WO-API-001 recommended |
| WO-PROD-002 | Evidence Graph Investigation Path | Finding-to-process-to-network-to-control evidence path | WO-API-001, WO-PROD-001 recommended |
| WO-PROD-003 | Finding-to-Control Design Workflow | Findings become observe-only draft controls with evidence, blast radius, and rollback notes | WO-CTRL-001, WO-PROD-002 |
| WO-PROD-004 | AI Research Feed and Detection Opportunities | Research opportunities become governed detection candidates | WO-DET-002, WO-AI-002, WO-PLAT-006 |
| WO-PROD-005 | First-Value Demo and Operator Onboarding | Five-minute guided workflow that shows AegisFlux value | WO-PROD-001 through WO-PROD-003 recommended |
| WO-GROWTH-001 | ABOM Fleet Insights and Change Detection | New/risky/widespread AI capability insights for daily review | WO-PROD-001, WO-API-001 |
| WO-GROWTH-002 | Evidence Graph UX and Explainability | Plain-language evidence explanations with confidence reasons and missing evidence | WO-PROD-002, WO-PROD-003 |
| WO-GROWTH-003 | Control Simulation Depth and Decision History | Richer blast-radius simulation, draft revisions, and operator decision history | WO-PROD-003, WO-PLAT-006 |
| WO-GROWTH-004 | Agent Health and Readiness Scoring | Trust/readiness score for endpoint evidence and future control decisions | WO-AGENT-001, WO-API-001 |
| WO-GROWTH-005 | Adaptive Detection Workflow Maturity | Research -> candidate -> simulation -> signed pack -> rollout -> retirement workflow | WO-PROD-004, WO-DET-003 |
| WO-GROWTH-006 | First-Value Demo Polish and Sample Scenarios | Credible buyer/operator demo path with sample scenarios and setup docs | WO-PROD-005 |
| WO-GROWTH-007 | Audit-Mode Enforcement Adapter Foundation | Audit-only policy bundle delivery/status path without blocking behavior | WO-GROWTH-003, WO-GROWTH-004 |
| WO-OPS-001 | End-to-End Pipeline Validation | Repeatable lab smoke from endpoint signal to console evidence and audit-mode status | WO-OPS-002 recommended, WO-DET-006, WO-GROWTH-007 |
| WO-OPS-002 | Lab Agent Connectivity and Heartbeat Reliability | Reliable online/stale/offline diagnosis for Linux and Windows lab agents | WO-LAB-001, WO-GROWTH-004 |
| WO-OPS-003 | Service Health, Discovery, and Resilience | Consistent health/readiness contracts and dependency failure behavior | WO-API-001, WO-OPS-002 recommended |
| WO-OPS-004 | Ingest and ETL Hardening | Validation, dedupe, replay, storage recovery, and ETL/enrich error handling | WO-VIS-004, WO-VIS-005, WO-API-001 |
| WO-OPS-005 | Observability, Metrics, and Logging Baseline | Practical logs, metrics, and operational-event checks for lab diagnosis | WO-PLAT-006, WO-OPS-003 |
| WO-OPS-006 | Security and Service Authentication Baseline | Lab trust-boundary inventory, endpoint auth tightening, and secret-handling baseline | WO-AI-002, WO-DET-003, WO-GROWTH-007 |
| WO-OPS-007 | Performance, Load, and Resource Baseline | Measured route/API/ingest/agent resource baselines and thresholds | WO-AGENT-001, WO-PERF-001, WO-API-001 |
| WO-OPS-008 | Deployment, Release, and Operator Readiness | Clone-to-validated-lab checklist, known limitations, release evidence, and reset steps | WO-OPS-001 through WO-OPS-007 |
| WO-AGENTS-001 | AI Agent Harness and Tool Runtime | Governed AI platform agent jobs, tools, run audit, and status | WO-OPS-008, WO-AI-001 through WO-AI-003 |
| WO-AGENTS-002 | Evidence-Bound Reasoning Contract | Required evidence refs, confidence, missing evidence, assumptions, and safety boundary for agent outputs | WO-AGENTS-001 |
| WO-GOV-001 | Agent Governance, Memory, and Decision Ledger | Scoped product memory and durable approval/rejection/recommendation ledger | WO-AGENTS-001, WO-AGENTS-002, WO-AI-002 |
| WO-AGENTS-003 | Endpoint Analyst Deep Agent | Evidence-bound device/finding investigation agent | WO-AGENTS-001, WO-AGENTS-002, WO-GOV-001 recommended |
| WO-ADAPT-001 | Detection Opportunity Research Agents | Research/fleet signals become scored detection opportunities | WO-AGENTS-001, WO-AGENTS-002, WO-PROD-004 |
| WO-ADAPT-002 | Detection Candidate Simulation Harness | Candidate packs replayed against fixtures/historical evidence before approval | WO-ADAPT-001, WO-DET-001 through WO-DET-003 |
| WO-CONTROL-002 | Control Design Copilot | Evidence-bound finding-to-control proposals with blast radius and rollback | WO-AGENTS-002, WO-GROWTH-003, WO-GROWTH-007 |
| WO-FLEET-001 | AI Capability Drift Radar | Daily fleet radar for new/risky/widespread/stale AI capability changes | WO-GROWTH-001, WO-GROWTH-004, WO-ADAPT-001 recommended |
| WO-ENFORCE-001 | Enforcement Readiness Scorecard | Explains missing prerequisites before future enforcement consideration | WO-GROWTH-003, WO-GROWTH-004, WO-GROWTH-007 |
| WO-DEMO-002 | Autonomous Demo Scenario Generator | Lab-safe scenario fixtures and validation for AI-native demo paths | WO-OPS-001, WO-ADAPT-002 recommended |

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
- WO-UX-001 should run before adding new UI capability because the dashboard is currently carrying too much detail.
- WO-UX-002 should follow WO-UX-001 so the later workbench pages share detail and string-formatting behavior.
- WO-UX-003, WO-UX-004, and WO-UX-005 can run after WO-UX-002; keep their page ownership separate to avoid conflicts.
- WO-QA-001 should start the next phase so the UI regressions discovered during refinement become repeatable checks.
- WO-PERF-001 should follow or run alongside WO-QA-001; keep it focused on measurement and low-risk fetch/render improvements.
- WO-API-001 should use WO-PERF-001 findings to choose which summary endpoints remove the most UI fan-out.
- WO-PROD-001 is the first differentiation work order because Agent Bill of Materials is the clearest customer-facing wedge.
- WO-PROD-002 should follow once the inventory and summary model can support evidence-backed drill-ins.
- WO-PROD-003 is the core evidence-to-control workflow; keep it observe-only until simulation and approval mature.
- WO-PROD-004 can run in parallel with WO-PROD-001/002 because it extends the dynamic detection loop rather than the UI data model.
- WO-PROD-005 should package the first-value story after the first three product-differentiation slices exist.
- WO-GROWTH-001 should lead the growth wave because ABOM is the clearest "open daily" feature.
- WO-GROWTH-002 and WO-GROWTH-003 should follow together: evidence must explain controls, and controls must be simulatable.
- WO-GROWTH-004 can run in parallel because trust/readiness scoring supports every later control decision.
- WO-GROWTH-005 matures the Adapt workflow and can run alongside growth UX work if detection/API ownership is separate.
- WO-GROWTH-006 should package the demo after ABOM, evidence, and controls are polished enough to tell the story cleanly.
- WO-GROWTH-007 comes last in this wave; audit-mode is a bridge to enforcement, not a shortcut around trust.
- WO-OPS-002 should lead operational readiness because stale/offline lab agents will confuse every later validation task.
- WO-OPS-003 and WO-OPS-004 can run in parallel if ownership is split between platform health and ingest/ETL.
- WO-OPS-005 should follow enough health/ingest work to instrument real failure modes.
- WO-OPS-001 should run once agent reliability and health checks are stable enough to make e2e failures meaningful.
- WO-OPS-006 can run in parallel with validation after the current endpoint/API inventory is clear.
- WO-OPS-007 should use the stabilized smoke/replay paths from WO-OPS-001 and WO-OPS-004.
- WO-OPS-008 comes last and packages the validated state into a release-candidate checklist.
- WO-AGENTS-001 and WO-AGENTS-002 must lead the AI-native wave; all deep agents depend on the harness and evidence-bound output contract.
- WO-GOV-001 should land before agents make product-impacting recommendations at scale.
- WO-AGENTS-003 is the first useful deep-agent experience after the harness exists.
- WO-ADAPT-001 and WO-ADAPT-002 form the adaptive detection factory and should precede automated pack lifecycle expansion.
- WO-CONTROL-002 should wait for evidence-bound reasoning and governance so control proposals are auditable.
- WO-FLEET-001 can run after ABOM/growth work and should link into Endpoint Analyst and Adapt workflows.
- WO-ENFORCE-001 is a readiness scorecard only; it does not enable enforcement.
- WO-DEMO-002 should package the implemented AI-native flows late in the wave.

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
- Core UI navigation and rendering paths have repeatable regression coverage.
- Console performance bottlenecks are measured, documented, and reduced where practical.
- High-impact workbench pages can use backend summaries instead of broad client-side telemetry fan-out.
- AegisFlux can show an Agent Bill of Materials that is more meaningful than raw software inventory.
- A finding can be explained through an evidence path and turned into an observe-only draft control proposal.
- A new operator can understand AegisFlux's value through a short first-value workflow.
- ABOM highlights new, risky, widespread, and low-confidence AI capabilities.
- Evidence explanations are readable without raw telemetry knowledge.
- Draft controls show blast radius, revision history, and operator decisions.
- Agent readiness scores clarify when evidence can be trusted.
- Adaptive detection lifecycle is auditable from research to rollout and retirement.
- Audit-mode policy delivery can be tested without blocking endpoint behavior.
- Linux and Windows lab agents have reliable online/stale/offline diagnosis.
- Service health/readiness makes dependency failures explicit.
- Ingest and ETL paths are tested against malformed, duplicate, partial, and replayed data.
- Operators can diagnose common lab failures from health, metrics, logs, and operational events.
- The full lab pipeline can be validated from endpoint signal to console evidence with one documented path.
- The lab security baseline and deployment-hardening gaps are documented.
- Performance, load, and agent resource baselines are measured and repeatable.
- A release-candidate checklist can bring a fresh operator from clone to validated lab state.
- AI platform agents run through a governed harness with typed tools and auditable runs.
- Agent conclusions are evidence-bound and show confidence, assumptions, and missing evidence.
- Endpoint Analyst can investigate device/finding context as a deep, evidence-citing agent.
- Research agents can turn ecosystem and fleet signals into scored detection opportunities.
- Candidate detection packs can be simulated before approval.
- Control Design Copilot can draft observe-only proposals without enabling enforcement.
- Fleet drift radar highlights daily AI capability changes.
- Governance memory and decision ledger make AI recommendations and operator decisions durable.
- Enforcement readiness is scored without enabling blocking behavior.

## Non-Goals

- No production enforcement by default.
- No external LLM calls from endpoint agents.
- No arbitrary code in detection packs.
- No broad UI rewrite outside the active AegisFlux console unless explicitly scoped.
- No Clarion database coupling.
