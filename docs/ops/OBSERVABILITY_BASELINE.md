# Observability baseline (WO-OPS-005)

## Baseline structured log fields (target)

When adding or changing logs in backend services, prefer these keys (where applicable):

| Field | Meaning |
|-------|---------|
| `time` / timestamp | RFC3339 UTC |
| `service` | Short service name (`actions-api`, `ingest`, …) |
| `level` | info / warn / error |
| `msg` | Human-readable message |
| `operation` | e.g. `heartbeat`, `ingest_visibility`, `summary` |
| `request_id` | When available from headers or generated |
| `agent_uid` / `device_id` | When handling agent or host scoped work |
| `error` | Error string without stack secrets |

Do not log raw tokens, API keys, or full visibility payloads in production-style builds.

## Metrics (lab)

| Location | What to scrape |
|----------|----------------|
| `ingest` `GET /metrics` | `events_total`, `events_invalid_total`, `events_deduped_total`, NATS errors |
| `actions-api` `GET /ops/metrics` | `actions_api_heartbeat_accepted_total` |
| `etl-enrich` `GET /metrics` | `etl_enrich_*` gauges and counters |

## First five checks when the lab looks wrong

1. **`./scripts/lab/health-sweep.sh`** — which dependency is down.
2. **`./scripts/lab/check-agents.sh`** — tunnels, compose, agent `online` / `stale` / `offline`.
3. **`curl -s "$ACTIONS_URL/readyz" | jq .`** — NATS + ingest + detection from actions-api.
4. **`curl -s "$INGEST_URL/metrics" | rg deduped|invalid|events_total`** — ingest path health.
5. **`docker compose ps`** — exited or unhealthy containers.

## Happy / sad path smoke

- **Happy:** successful `POST /agents/heartbeat` increments `actions_api_heartbeat_accepted_total` (see `/ops/metrics`).
- **Sad:** stop ingest container; `/readyz` should show `degraded` with ingest check failing while `/healthz` on actions-api stays `ok`.
