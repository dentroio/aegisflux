# Visibility and Observability Work Orders

**Program:** AegisFlux host/workload visibility for Clarion  
**Phase:** Phase 1 - AegisFlux Stabilization and Visibility  
**Status:** Phase 1 complete (repository deliverables; operator VM snapshot remains environment-specific)  
**Last updated:** May 10, 2026

## Objective

Build the first useful AegisFlux visibility loop:

1. A Windows test machine runs normal applications, developer tools, browser AI, local scripts, and downloaded software.
2. The AegisFlux Windows sensor observes process, user, network, DNS, and application context.
3. The AegisFlux backend receives and stores visibility events.
4. The AegisFlux UI or API can show device -> user -> process -> parent process -> destination -> classification -> evidence.
5. Early AI-agent and automation detection produces explainable, non-blocking findings.

This phase is observability only. Do not build blocking or WFP enforcement yet.

## Work Order Sequence

| ID | Title | Primary Output | Depends On |
|----|-------|----------------|------------|
| WO-VIS-001 | Windows Test Machine Baseline | `windows-dev-agent-01` ready for repeatable tests | None |
| WO-VIS-002 | Windows Sensor Process Inventory | Process start/stop and lineage events | WO-VIS-001 |
| WO-VIS-003 | Windows Network and DNS Attribution | PID/process-to-flow and DNS context | WO-VIS-002 |
| WO-VIS-004 | Visibility Event Schemas | Versioned event schemas for process, flow, DNS, detection | None |
| WO-VIS-005 | Backend Ingest and Storage | Backend accepts and stores visibility events | WO-VIS-004 |
| WO-VIS-006 | Visibility API and UI Surface | Query/display device, process, flow, and evidence | WO-VIS-005 |
| WO-VIS-007 | AI-Agent Detection Pack 1 | Non-blocking IDE/browser/local-script detections | WO-VIS-002, WO-VIS-003, WO-VIS-005 |
| WO-VIS-008 | Test Harness and Evidence Capture | Repeatable scenario runner and expected outputs | WO-VIS-001, WO-VIS-007 |
| WO-VIS-009 | Clarion Mapping Draft | AegisFlux visibility events mapped to Clarion context objects | WO-VIS-004 |
| WO-VIS-010 | macOS Agent Scaffold | Secure Rust macOS visibility-agent baseline | None |

## Phase Exit Criteria

- AegisFlux can reliably identify process lineage on Windows.
- AegisFlux can attribute outbound network flows to process and user context.
- AegisFlux can correlate DNS/domain context to observed flows.
- AegisFlux can classify at least three early AI/automation scenarios with evidence:
  - Browser AI usage
  - IDE assistant helper process
  - Local Python or Node agent script
- Backend can store and query visibility events.
- UI or API can show an investigation path for a single flow.
- Clarion integration mapping is documented, even if not implemented.

macOS is tracked as a parallel scaffold/future visibility lane. Windows remains the first full Phase 1 implementation target.

## Non-Goals

- No enforcement or blocking.
- No endpoint quarantine.
- No SGT changes.
- No production deployment.
- No claim that AI-agent detection is complete or high-confidence across all tools.

## Related Planning

- [AegisFlux Development and Clarion Integration Plan](../../plans/AEGIS_DEVELOPMENT_AND_CLARION_INTEGRATION_PLAN.md)
