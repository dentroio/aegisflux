# Lab security baseline (WO-OPS-006)

This document describes **lab / local compose** posture only. It is not a production compliance statement.

## Trust boundaries

- **Agents → ingest / actions-api:** Plain HTTP in lab; no mTLS. Agents identify with `agent_uid` and heartbeat JSON. Treat the lab network as a trusted segment.
- **Service-to-service:** Same Docker network; no inter-service auth in default compose.
- **Console → backend:** Typically Next.js server or browser to `actions-api` / `ingest` as configured for dev.

## Endpoint classification (summary)

| Area | Auth today | Classification |
|------|------------|------------------|
| `GET /healthz`, `GET /readyz`, `GET /ops/metrics` | None | Lab-only observability |
| `GET /agents`, `GET /console/summary/*` | None | Lab-only read |
| `POST /agents/heartbeat` | None | Lab-only write (registration side-effect) |
| `POST /agents/broadcast`, agent send routes | NATS-backed | Lab-only; misuse can enqueue gateway traffic |
| Detection signed pack APIs | Signature verification on artifacts | Lab trust on keys configured in pipeline |
| Audit bundles | Observe-only paths per product rules | Review WO-GROWTH-007 |

## Secret handling

- Use `.env` or shell exports for tokens; never commit real secrets.
- Vault dev token in compose is **development only**.

## Regression tests

- **Ingest:** malformed JSONL rejected with `400` (`TestHandleVisibilityEventsRejectsMalformedJSONL`).
- **Workbench:** ingest outage surfaces in `dependencies` array (`TestAgentsWorkbenchSummary_IngestFailureSurfacesDependency`) so empty agent merge is not mistaken for “no agents in fleet” without context.

## Hardening gap list (non-lab)

- mTLS or signed requests for agent writes.
- Authenticated operator APIs for broadcast and configuration.
- Centralized audit export for authz decisions.
