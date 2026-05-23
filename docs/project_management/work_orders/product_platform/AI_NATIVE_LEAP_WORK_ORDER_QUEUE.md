# AegisFlux AI-Native Leap Work Order Queue

**Last updated:** May 12, 2026  
**Scope:** Next capability wave after operational readiness  
**Queue owner:** AI platform lead / coordinating agent  

## Prerequisite

Complete or explicitly defer the operational-readiness queue before starting this wave:

- [Open Work Order Queue](OPEN_WORK_ORDER_QUEUE.md)

The AI-native wave assumes the lab can be validated, service health is clear, ingest/ETL behavior is stable, security boundaries are documented, and performance baselines exist.

## Ordered Work Orders

| Queue | Work Order | Status | Why It Is Here |
|-------|------------|--------|----------------|
| 1 | WO-AGENTS-001: AI Agent Harness and Tool Runtime | **Implemented (lab)** | Runtime, registry, audited jobs/runs/tools; lab in-memory store with SQL migration for future Postgres. |
| 2 | WO-AGENTS-002: Evidence-Bound Reasoning Contract | **Implemented (lab)** | Evidence-bound schema, harness validation, API + console rendering. |
| 3 | WO-GOV-001: Agent Governance, Memory, and Decision Ledger | **Next** — Planned | Adds durable decision and memory primitives before agents make product-impacting recommendations. |
| 4 | WO-AGENTS-003: Endpoint Analyst Deep Agent | Planned | First deep agent using the harness and evidence contract on device/finding investigations. |
| 5 | WO-ADAPT-001: Detection Opportunity Research Agents | Planned | Starts the adaptive research-to-detection loop with scored opportunities. |
| 6 | WO-ADAPT-002: Detection Candidate Simulation Harness | Planned | Adds replay/simulation proof before detection candidates can be approved. |
| 7 | WO-CONTROL-002: Control Design Copilot | Planned | Turns evidence-bound findings into observe-only control proposals and approval packets. |
| 8 | WO-FLEET-001: AI Capability Drift Radar | Planned | Promotes ABOM into a daily fleet change/risk radar linked to analyst and adapt workflows. |
| 9 | WO-ENFORCE-001: Enforcement Readiness Scorecard | Planned | Explains what is missing before any future enforcement consideration, without enabling enforcement. |
| 10 | WO-DEMO-002: Autonomous Demo Scenario Generator | Planned | Packages the new capabilities into repeatable, evidence-backed demo scenarios. |

## Processing Rules

- Do not start this queue until the operational-readiness queue is complete or consciously deferred with documented risk.
- Process in numeric order by default.
- Keep endpoint agents lightweight; no endpoint LLM calls.
- All agent work must use the harness, tool registry, privacy controls, and evidence-bound contract once those exist. **Lab state (May 12, 2026):** harness, privacy, and evidence-bound validation on product-impacting harness completions are in place on Actions API.
- No agent may directly enforce policy, publish blocking controls, or bypass approval gates.
- Commit each work order separately.

## Current wave status (May 12, 2026)

- **WO-AGENTS-001** and **WO-AGENTS-002** are closed for the **lab slice** (harness + evidence-bound contract, validation, UI). **Follow-ups:** Postgres persistence for harness tables; async job states; deeper visibility joins for evidence refs (process/DNS/flow row IDs).
- **Next default pickup:** [WO-GOV-001](WO-GOV-001_AGENT_GOVERNANCE_MEMORY_AND_DECISION_LEDGER.md).

## Safe Parallelization

Default is sequential until WO-GOV-001 lands (harness + evidence-bound contract are in place). After that:

- WO-GOV-001 and WO-AGENTS-003 can overlap if write scopes are separated.
- WO-ADAPT-001 can begin while Endpoint Analyst UI work continues.
- WO-FLEET-001 can begin after drift data contracts are agreed.
- WO-DEMO-002 should stay late so scenarios reflect real implemented flows.

## Agent Pickup Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Use docs/project_management/work_orders/product_platform/AI_NATIVE_LEAP_WORK_ORDER_QUEUE.md as the source of truth for the AI-native wave.

Before starting:
- Confirm docs/project_management/work_orders/product_platform/OPEN_WORK_ORDER_QUEUE.md is complete or explicitly deferred (operational readiness queue is complete as of May 12, 2026).
- Pick the first Planned AI-native work order whose dependencies are satisfied (default next: WO-GOV-001).
- Read docs/AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md and the assigned work order.

During execution:
- Keep scope limited to the assigned work order.
- Use governed tools and evidence-bound outputs once the harness exists.
- Preserve observe-only and audit-only boundaries.
- Update the assigned work order with implementation notes and verification results.

At the end:
- Run the verification listed in the work order.
- Stage only files changed for this work order.
- Commit and push this work order separately.
```
