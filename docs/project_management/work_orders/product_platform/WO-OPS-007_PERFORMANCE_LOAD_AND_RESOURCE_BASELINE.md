# WO-OPS-007: Performance, Load, and Resource Baseline

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** QA / Backend / Agent / UI  

## Goal

Measure and document practical performance limits for the lab console, backend APIs, ingest path, and endpoint agent resource budgets.

## Problem

Console performance and agent budget telemetry have first-pass coverage, but the platform still needs realistic load baselines: event throughput, summary latency, UI route responsiveness, WebSocket/heartbeat scale assumptions, and agent CPU/memory behavior during collection and pack evaluation.

## Scope

- Console route timing and bundle/build checks.
- Ingest event replay throughput.
- Summary API latency under fixture data.
- Agent heartbeat/status update cadence.
- Agent resource telemetry and detection-pack evaluator runtime.

## Deliverables

- Define small, medium, and stress fixture sizes for lab testing.
- Add load/replay scripts or documented commands for ingest and summary APIs.
- Measure baseline latency for dashboard, agents workbench, agent detail, inventory, detections, events, and audit bundles.
- Measure agent CPU/memory/runtime budget events during normal and detection-pack evaluation paths.
- Document thresholds that should fail CI/lab validation later.
- Identify top bottlenecks and file follow-up work if fixes exceed this order.

## Acceptance Criteria

- Baseline numbers are documented with machine/context details.
- Ingest and summary APIs have repeatable load/replay commands.
- Console route checks include at least HTTP status and coarse response/build timing.
- Agent budget telemetry is visible or its absence is called out as a blocker.
- Follow-up bottlenecks are ranked by operator impact.

## Dependencies

- WO-AGENT-001.
- WO-PERF-001.
- WO-API-001.
- WO-OPS-004 recommended.

## Non-Goals

- No enterprise-scale benchmark claim.
- No premature optimization without measurement.
- No new infrastructure requirement beyond local/lab tooling.

## Implementation notes (May 12, 2026)

- **Doc:** `docs/ops/PERFORMANCE_BASELINE.md` — fixture sizing, threshold placeholders, agent budget pointers.
- **Scripts:** `scripts/lab/replay-visibility-fixtures.sh`, `scripts/lab/load-ingest-summaries.sh` for repeatable ingest + summary timing from the host.

## Suggested Verification

- Run replay/load scripts against a local lab stack.
- Run console build and route smoke checks.
- Compare measured values to documented thresholds.
