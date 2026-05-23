# AegisFlux AI-Native Leap Agent Prompt Runbook

**Purpose:** Copy-ready prompts for the AI-native capability wave after operational readiness.

## Recommended Order

Use [AI-Native Leap Work Order Queue](AI_NATIVE_LEAP_WORK_ORDER_QUEUE.md) as the source of truth:

1. WO-AGENTS-001: AI Agent Harness and Tool Runtime
2. WO-AGENTS-002: Evidence-Bound Reasoning Contract
3. WO-GOV-001: Agent Governance, Memory, and Decision Ledger
4. WO-AGENTS-003: Endpoint Analyst Deep Agent
5. WO-ADAPT-001: Detection Opportunity Research Agents
6. WO-ADAPT-002: Detection Candidate Simulation Harness
7. WO-CONTROL-002: Control Design Copilot
8. WO-FLEET-001: AI Capability Drift Radar
9. WO-ENFORCE-001: Enforcement Readiness Scorecard
10. WO-DEMO-002: Autonomous Demo Scenario Generator

## Execution Rules for Every Agent

- Start with `git status -sb`.
- Confirm the operational-readiness queue is complete or explicitly deferred.
- Read [AegisFlux AI-Native Leap Architecture](../../../AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md).
- Read the assigned work order before editing.
- Do not revert unrelated user or agent changes.
- Keep changes scoped to the assigned work order.
- Preserve observe-only and audit-only safety boundaries.
- Do not add endpoint LLM calls.
- Do not add autonomous enforcement, blocking, quarantine, or direct policy publish.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Stage only files changed for the work order.
- Commit and push at the end.

## WO-AGENTS-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-AGENTS-001_AI_AGENT_HARNESS_AND_TOOL_RUNTIME.md

Goal: create the governed AI platform agent harness with job lifecycle, typed tools, and auditable runs.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md and WO-AGENTS-001.
- Inspect existing AI provider/privacy/audit code and Endpoint Evidence Analyst code.

Implementation scope:
- Add agent job/run/tool contracts.
- Register initial system agents.
- Implement at least three read-only tools.
- Capture tool calls, provider/model, status, duration, and errors.
- Add minimal run list/detail API or UI.
- Update WO-AGENTS-001 with implementation notes.
```

## WO-AGENTS-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-AGENTS-002_EVIDENCE_BOUND_REASONING_CONTRACT.md

Goal: require evidence refs, confidence, assumptions, missing evidence, and safety boundaries for governed agent conclusions.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md and WO-AGENTS-002.
- Inspect evidence path, finding, ABOM, and operational event models.

Implementation scope:
- Define EvidenceBoundConclusion schema.
- Add evidence reference types.
- Add validation for product-impacting agent outputs.
- Add UI rendering pattern for cited evidence and missing evidence.
- Update WO-AGENTS-002 with implementation notes.
```

## WO-GOV-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-GOV-001_AGENT_GOVERNANCE_MEMORY_AND_DECISION_LEDGER.md

Goal: add scoped product memory and decision ledger for AI recommendations and operator decisions.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md and WO-GOV-001.
- Inspect AI audit, operational events, draft controls, detection candidates, and audit bundles.

Implementation scope:
- Define memory and decision ledger entries.
- Add append/query APIs.
- Link recommendations to run, evidence, and decision outcome.
- Respect privacy/redaction settings.
- Update WO-GOV-001 with implementation notes.
```

## Later Work Order Prompt Pattern

For `WO-AGENTS-003`, `WO-ADAPT-001`, `WO-ADAPT-002`, `WO-CONTROL-002`, `WO-FLEET-001`, `WO-ENFORCE-001`, and `WO-DEMO-002`, use this pattern:

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: <assigned work order path>

Before editing:
- Run git status -sb.
- Read docs/AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md.
- Read the assigned work order and its dependencies.
- Confirm earlier AI-native queue items that this depends on are complete.

Implementation scope:
- Keep changes scoped to the assigned work order.
- Use the agent harness, evidence-bound reasoning contract, governance ledger, and privacy controls where applicable.
- Preserve observe-only and audit-only safety boundaries.
- Update the assigned work order with implementation notes and verification results.

Verification:
- Run the verification listed in the assigned work order.
- Run relevant UI/backend tests for touched modules.

At the end:
- Run git status -sb.
- Stage only files changed for the assigned work order.
- Commit and push separately.
```
