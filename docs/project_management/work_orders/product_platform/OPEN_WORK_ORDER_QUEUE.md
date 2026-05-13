# AegisFlux Open Work Order Queue

**Last updated:** May 12, 2026  
**Scope:** Currently open/planned product-platform work orders  
**Queue owner:** Operational readiness lead / coordinating agent  

## Current Open Work Orders

All earlier visibility, platform, UX, product-differentiation, and growth work orders are marked complete, done, or implemented. The active queue is the operational-readiness wave:

| Queue | Work Order | Status | Why It Is Here |
|-------|------------|--------|----------------|
| 1 | WO-OPS-002: Lab Agent Connectivity and Heartbeat Reliability | Complete | Agent freshness and tunnel reliability must be understandable before e2e failures mean anything. |
| 2 | WO-OPS-003: Service Health, Discovery, and Resilience | Complete | Services need consistent health/readiness and dependency failure behavior before deeper validation. |
| 3 | WO-OPS-004: Ingest and ETL Hardening | Complete | Event validation, dedupe, replay, and storage recovery need to be stable before relying on summaries. |
| 4 | WO-OPS-005: Observability, Metrics, and Logging Baseline | Complete | Health and ingest failure modes need useful logs, metrics, and operational events. |
| 5 | WO-OPS-001: End-to-End Pipeline Validation | Complete | Once the lab and services are diagnosable, prove endpoint signal through console evidence. |
| 6 | WO-OPS-006: Security and Service Authentication Baseline | Complete | After the active endpoints and flows are clear, document/tighten trust boundaries. |
| 7 | WO-OPS-007: Performance, Load, and Resource Baseline | Complete | Use the stabilized replay/smoke paths to measure realistic latency, throughput, and agent budget. |
| 8 | WO-OPS-008: Deployment, Release, and Operator Readiness | Complete | Package the validated state into a release-candidate checklist last. |

## Processing Rules

- Process the queue in numeric order unless the coordinating agent explicitly splits parallel work.
- Start each work order by reading its full file and the dependencies listed in it.
- Update the assigned work order with implementation notes and verification results before closing it.
- Preserve observe-only and audit-only safety boundaries.
- Do not expand an operational-readiness work order into a new product feature; document feature gaps as follow-up work.
- Commit each work order separately so the queue can be reviewed and reverted independently if needed.

## Dependency Gates

| Gate | Required Before | Gate Condition |
|------|-----------------|----------------|
| Lab Reliability Gate | WO-OPS-001, WO-OPS-007, WO-OPS-008 | Linux and Windows lab agent status can be classified as online, stale, or offline with a clear reason. |
| Health/Readiness Gate | WO-OPS-001, WO-OPS-005, WO-OPS-008 | Core services expose documented health/readiness and dependency failures are explicit. |
| Data Pipeline Gate | WO-OPS-001, WO-OPS-007, WO-OPS-008 | Ingest handles malformed, duplicate, partial, and replayed data predictably. |
| Diagnosis Gate | WO-OPS-001, WO-OPS-007, WO-OPS-008 | Common lab failures are explainable from health, logs, metrics, and operational events. |
| E2E Validation Gate | WO-OPS-006, WO-OPS-007, WO-OPS-008 | Canonical lab smoke proves endpoint signal to console evidence and audit-mode status. |
| Release Gate | WO-OPS-008 | Security baseline, performance baseline, and known limitations are documented. |

## Safe Parallelization

Default processing should be sequential. If multiple agents are available, these splits are low conflict:

| Track | Work Orders | Notes |
|-------|-------------|-------|
| Lab track | WO-OPS-002 | Keep ownership of tunnel scripts, heartbeat checks, and agent status docs isolated. |
| Platform health track | WO-OPS-003 | Can run beside WO-OPS-004 if service-health changes do not touch ingest internals. |
| Data pipeline track | WO-OPS-004 | Can run beside WO-OPS-003 if ingest/ETL changes stay scoped. |
| Security track | WO-OPS-006 | Can begin after WO-OPS-003 inventories service endpoints; avoid changing endpoints that WO-OPS-001 is actively validating. |

WO-OPS-005 should wait until at least part of WO-OPS-003 or WO-OPS-004 has landed. WO-OPS-001 should wait until WO-OPS-002, WO-OPS-003, and WO-OPS-004 are substantially complete. WO-OPS-007 should wait for WO-OPS-001 and WO-OPS-004 replay/smoke paths. WO-OPS-008 should remain last.

## Agent Pickup Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Use docs/project_management/work_orders/product_platform/OPEN_WORK_ORDER_QUEUE.md as the source of truth for work-order order.

Pick the first work order whose status is Planned and whose dependency gates are satisfied. Read that work order fully, then execute it according to docs/project_management/work_orders/product_platform/OPERATIONAL_READINESS_AGENT_PROMPT_RUNBOOK.md.

Before editing:
- Run git status -sb.
- Confirm no earlier Planned work order is still blocking this one.
- Do not revert unrelated changes.

During execution:
- Keep scope limited to the assigned work order.
- Preserve observe-only and audit-only safety boundaries.
- Update the assigned work order with implementation notes and verification results.

At the end:
- Run the verification listed in the work order.
- Run git status -sb.
- Stage only files changed for this work order.
- Commit and push this work order separately.
```
