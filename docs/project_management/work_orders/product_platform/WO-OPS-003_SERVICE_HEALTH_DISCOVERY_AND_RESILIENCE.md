# WO-OPS-003: Service Health, Discovery, and Resilience

**Status:** Complete  
**Phase:** Operational Readiness  
**Primary owner:** Backend / Platform  

## Goal

Give every AegisFlux service a consistent health/readiness contract and make service-to-service calls fail clearly instead of creating ambiguous no-data states.

## Problem

Several services expose health endpoints, but behavior is not yet uniform. The console and backend summaries depend on multiple local services, and misconfiguration can look like empty lab data. Operators need clear service health, dependency health, timeouts, and retry behavior.

## Scope

- Backend services in `backend/*`.
- Console proxy/API dependency handling where applicable.
- Docker Compose service definitions and local runbooks.
- Health, readiness, dependency status, timeouts, and error reporting.

## Deliverables

- Inventory all service ports, health endpoints, and readiness dependencies.
- Standardize response shape for health/readiness where practical:
  - service name,
  - version/build when available,
  - uptime,
  - dependency statuses,
  - degraded vs unhealthy state.
- Add missing health/readiness endpoints to high-impact services.
- Set explicit client timeouts for service-to-service calls that feed summaries or agent status.
- Add clear error messages for dependency-down vs empty-data responses.
- Update Docker Compose health checks where supported.
- Document the local service map and expected startup order.

## Acceptance Criteria

- Each active backend service has a documented health endpoint.
- Readiness distinguishes "process alive" from "dependencies available" for services that need dependencies.
- Summary endpoints fail or degrade with explicit dependency context instead of returning misleading empty data.
- Local runbook includes a small health sweep command.
- Tests cover at least one degraded dependency path for a summary or service client.

## Dependencies

- WO-API-001.
- WO-OPS-002 recommended.

## Non-Goals

- No full service mesh.
- No Kubernetes-only implementation.
- No broad rewrite of backend service boundaries.

## Implementation notes (May 12, 2026)

- **Inventory:** `docs/ops/SERVICE_HEALTH_MAP.md` lists ports, paths, and startup order.
- **actions-api:** `GET /readyz` returns JSON with `state` (`ready` / `degraded` / `unhealthy`), per-dependency checks for NATS, ingest, and detection-pipeline; HTTP 503 when NATS is unavailable. `GET /healthz` remains plain liveness for backward compatibility.
- **Console merge:** `GET /console/summary/agents-workbench` includes `dependencies` with ingest probe status so ingest outages are not mistaken for an empty fleet without context. Tests: `TestAgentsWorkbenchSummary_*`.
- **Sweep:** `scripts/lab/health-sweep.sh` curls core services from the host.
- **Compose:** `actions-api` and `etl-enrich` Docker healthchecks; `INGEST_API_URL` / `DETECTION_PIPELINE_URL` wired for in-network readiness checks on actions-api.

## Suggested Verification

- Run the health sweep with services up and with one dependency intentionally stopped.
- Run relevant backend tests for touched health/client code.
- Confirm console shows usable degraded state where applicable.
