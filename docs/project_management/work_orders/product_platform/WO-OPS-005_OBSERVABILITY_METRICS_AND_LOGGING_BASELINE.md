# WO-OPS-005: Observability, Metrics, and Logging Baseline

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Backend / Platform / QA  

## Goal

Create a practical observability baseline so failures in the lab can be diagnosed from metrics, logs, and operational events without reading every service manually.

## Problem

AegisFlux has operational events and several service health endpoints, but there is not yet a consistent baseline for metrics names, log fields, request correlation, or lab troubleshooting. That makes e2e failures slower to triage and hides regressions until manual walkthroughs.

## Scope

- Backend services and console API proxy logs where practical.
- Operational event stream usage.
- Metrics endpoints already present or easy to add.
- Local lab troubleshooting docs.

## Deliverables

- Define baseline log fields:
  - timestamp,
  - service,
  - request id or event id,
  - agent/device id when applicable,
  - operation,
  - status/error reason.
- Inventory existing metrics and add missing high-value counters/gauges for:
  - ingest accepted/rejected/deduped events,
  - agent heartbeat updates,
  - detection-pack status reports,
  - audit-mode bundle transitions,
  - summary endpoint errors/degraded responses.
- Add lightweight request correlation where practical.
- Document "first five checks" for lab failures.
- Ensure operational events are emitted for important product state changes, not routine noisy telemetry.

## Acceptance Criteria

- Common lab failures can be triaged with a documented sequence of health, metrics, and event checks.
- At least ingest, actions-api, and audit-mode transitions expose useful counters or structured logs.
- Logs avoid leaking secrets or full sensitive payloads.
- Operational events remain bounded and meaningful.
- The observability docs include example commands and expected outputs.

## Dependencies

- WO-PLAT-006.
- WO-OPS-003.
- WO-OPS-004 recommended.

## Non-Goals

- No full distributed tracing platform requirement.
- No hosted observability vendor integration.
- No verbose raw telemetry logging by default.

## Implementation notes (May 12, 2026)

- **Doc baseline:** `docs/ops/OBSERVABILITY_BASELINE.md` — log field targets, first five checks, happy/sad path notes.
- **actions-api metrics:** `GET /ops/metrics` exposes `actions_api_heartbeat_accepted_total` (incremented on successful heartbeat handling).
- **Ingest:** existing `/metrics` plus new `events_deduped_total` counter (WO-OPS-004) supports pipeline triage.

## Suggested Verification

- Trigger one happy path and one failure path.
- Confirm logs, metrics, and operational events show enough context to explain both.
- Run relevant backend tests for any new metrics/event code.
