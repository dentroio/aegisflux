# AegisFlux Operational Readiness Agent Prompt Runbook

**Purpose:** Copy-ready prompts for the operational-readiness work-order batch after the product-platform, UX, differentiation, and growth waves reached lab-slice completion.

## Recommended Order

Use [Open Work Order Queue](OPEN_WORK_ORDER_QUEUE.md) as the source of truth. Current order:

1. WO-OPS-002: Lab Agent Connectivity and Heartbeat Reliability
2. WO-OPS-003: Service Health, Discovery, and Resilience
3. WO-OPS-004: Ingest and ETL Hardening
4. WO-OPS-005: Observability, Metrics, and Logging Baseline
5. WO-OPS-001: End-to-End Pipeline Validation
6. WO-OPS-006: Security and Service Authentication Baseline
7. WO-OPS-007: Performance, Load, and Resource Baseline
8. WO-OPS-008: Deployment, Release, and Operator Readiness

## Queue Gates

- Start with WO-OPS-002 unless a coordinating agent has already completed it.
- Do not start WO-OPS-001 until lab status, service health, and ingest/ETL behavior are diagnosable.
- Do not start WO-OPS-007 until replay/smoke paths exist from WO-OPS-001 and WO-OPS-004.
- Keep WO-OPS-008 last; it packages the validated state rather than creating it.

## Execution Rules for Every Agent

- Start with `git status -sb`.
- Check [Open Work Order Queue](OPEN_WORK_ORDER_QUEUE.md) and pick the first uncompleted work order whose gates are satisfied.
- Read the assigned work order before editing.
- Read relevant docs listed in the work order.
- Do not revert unrelated user or agent changes.
- Keep changes scoped to the assigned work order.
- Preserve observe-only and audit-only safety boundaries.
- Run the verification listed in the work order.
- Update the work order status and implementation notes.
- Stage only files changed for the work order.
- Commit and push at the end.

## Shared Context

The previous waves are considered lab-slice complete. These work orders should focus on reliability, diagnosability, validation, and release readiness rather than new product features. If a missing product feature is discovered, document it as a follow-up instead of expanding the operational-readiness scope.

## WO-OPS-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-002_LAB_AGENT_CONNECTIVITY_AND_HEARTBEAT_RELIABILITY.md

Goal: make Linux and Windows lab agent online/stale/offline state reliable and diagnosable.

Before editing:
- Run git status -sb.
- Read WO-OPS-002, WO-LAB-001, WO-GROWTH-004, and the lab tunnel scripts/docs.
- Inspect Actions API agent status handling and console agent status display.

Implementation scope:
- Define heartbeat freshness thresholds.
- Add/update health checks and lab runbooks for tunnels, local agent health, backend reachability, and last heartbeat.
- Improve stale/offline reason display if a small backend/UI change is needed.
- Update WO-OPS-002 with implementation notes and verification results.

Verification:
- Run the documented health checks for Linux and Windows paths.
- Confirm GET /agents reflects heartbeat freshness.
- Confirm console agent status agrees with API state or document any known difference.
```

## WO-OPS-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-003_SERVICE_HEALTH_DISCOVERY_AND_RESILIENCE.md

Goal: standardize service health/readiness and make dependency failures explicit.

Before editing:
- Run git status -sb.
- Read WO-OPS-003 and inspect backend service health handlers plus docker-compose.yml.

Implementation scope:
- Inventory service health/readiness endpoints.
- Add missing high-impact health/readiness responses.
- Add explicit client timeouts and degraded dependency messages for summary/service calls.
- Update compose health checks and local docs where practical.
- Update WO-OPS-003 with implementation notes.

Verification:
- Run health sweep with services available.
- Simulate one dependency-down case and confirm clear degraded/unhealthy output.
- Run relevant backend tests.
```

## WO-OPS-004 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-004_INGEST_AND_ETL_HARDENING.md

Goal: harden ingest and ETL/enrich validation, dedupe, errors, replay, and recovery behavior.

Before editing:
- Run git status -sb.
- Read WO-OPS-004, WO-VIS-004, WO-VIS-005, and WO-API-001.
- Inspect backend/ingest and backend/etl-enrich.

Implementation scope:
- Add tests for malformed, duplicate, partial, and replayed visibility events.
- Improve structured validation and processing errors.
- Add or update replay fixtures.
- Harden ETL/enrich health/readiness/metrics and startup failure behavior.
- Document storage modes and recovery steps.
- Update WO-OPS-004 with implementation notes.

Verification:
- Run ingest tests.
- Replay fixture events into local ingest.
- Query dashboard/device/inventory summaries after replay.
```

## WO-OPS-005 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-005_OBSERVABILITY_METRICS_AND_LOGGING_BASELINE.md

Goal: establish practical logs, metrics, and operational-event checks for lab diagnosis.

Before editing:
- Run git status -sb.
- Read WO-OPS-005, WO-PLAT-006, and the backend services that emit operational events.

Implementation scope:
- Define baseline structured log fields.
- Add high-value counters/gauges for ingest, heartbeat, detection-pack status, audit bundles, and summary degraded responses.
- Add lightweight request/event correlation where practical.
- Document the first five checks for lab failures.
- Update WO-OPS-005 with implementation notes.

Verification:
- Trigger one happy path and one failure path.
- Confirm logs, metrics, and operational events explain both.
- Run relevant backend tests.
```

## WO-OPS-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-001_END_TO_END_PIPELINE_VALIDATION.md

Goal: prove the complete lab path from endpoint signal to console evidence and audit-mode status.

Before editing:
- Run git status -sb.
- Read WO-OPS-001 plus WO-DET-006 and WO-GROWTH-007.
- Inspect existing lab smoke scripts and docs/demos.

Implementation scope:
- Define canonical smoke scenarios for heartbeat, ingest, detection-pack status, inventory/ABOM summary, detection opportunity, and audit-mode lifecycle.
- Add one command or clear runbook for running the checks.
- Add automated assertions where practical.
- Document expected API/UI evidence and common failure modes.
- Update WO-OPS-001 with implementation notes.

Verification:
- Run the new e2e smoke path from a fresh shell.
- Check /agents, summary APIs, operational events, and console routes.
```

## WO-OPS-006 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-006_SECURITY_AND_SERVICE_AUTH_BASELINE.md

Goal: document and tighten the local/lab security baseline before broader deployment work.

Before editing:
- Run git status -sb.
- Read WO-OPS-006, WO-AI-002, WO-DET-001, WO-DET-003, and WO-GROWTH-007.

Implementation scope:
- Inventory trust boundaries and unauthenticated endpoints.
- Classify endpoints as lab-only, needs token/API key, future TLS/mTLS, or local-only.
- Tighten high-risk write endpoints where practical.
- Add regression tests for denied unauthenticated writes and signed-artifact rejection where available.
- Document secret/environment handling.
- Update WO-OPS-006 with implementation notes.

Verification:
- Run touched backend auth/security tests.
- Attempt representative unauthenticated write calls.
- Confirm no secrets are added to git.
```

## WO-OPS-007 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-007_PERFORMANCE_LOAD_AND_RESOURCE_BASELINE.md

Goal: measure practical load, latency, and resource baselines for the lab.

Before editing:
- Run git status -sb.
- Read WO-OPS-007, WO-AGENT-001, WO-PERF-001, and WO-API-001.

Implementation scope:
- Define small/medium/stress fixture sizes.
- Add load/replay scripts or documented commands for ingest and summaries.
- Measure console route timing and summary API latency.
- Measure or verify agent resource telemetry during normal and pack-evaluation paths.
- Document thresholds and ranked bottlenecks.
- Update WO-OPS-007 with implementation notes.

Verification:
- Run replay/load commands.
- Run console build and route smoke checks.
- Compare measured values to documented thresholds.
```

## WO-OPS-008 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-OPS-008_DEPLOYMENT_RELEASE_AND_OPERATOR_READINESS.md

Goal: package the lab into a repeatable release-candidate checklist.

Before editing:
- Run git status -sb.
- Read WO-OPS-008 and the completed WO-OPS work orders.
- Inspect docs/demos, docs/README.md, README.md, and local setup docs.

Implementation scope:
- Create a clone-to-validated-lab checklist.
- Link required startup, health sweep, agent heartbeat, e2e smoke, console walkthrough, and reset steps.
- Add known limitations for lab-only, observe-only, and audit-only behavior.
- Document release evidence capture and CI/CD gaps.
- Update WO-OPS-008 with implementation notes.

Verification:
- Run the checklist from a fresh shell as far as the local environment allows.
- Capture pass/fail evidence and known blockers.
```
