# Service health map (WO-OPS-003)

Host ports follow `docker-compose.yml` defaults on the developer machine. In-container listeners may differ (e.g. ingest listens on `9090` inside the container, mapped to host `9091`).

| Service | Host port | Liveness | Readiness / detail |
|---------|-----------|----------|---------------------|
| actions-api | 8083 | `GET /healthz` (plain `ok`) | `GET /readyz` JSON: NATS, ingest, detection-pipeline |
| actions-api | 8083 | — | `GET /ops/metrics` Prometheus text (`actions_api_heartbeat_accepted_total`) |
| ingest | 9091 | `GET /healthz` JSON | `GET /readyz` JSON |
| ingest | 9091 | `GET /metrics` | Prometheus |
| detection-pipeline | 8089 | `GET /healthz` | — |
| etl-enrich | 8088 | `GET /healthz` JSON + `dependencies` | `GET /readyz` JSON snapshot |
| etl-enrich | 8088 | `GET /metrics` | Prometheus text |
| config-api | 8085 | `GET /healthz` JSON | `GET /readyz` |
| correlator | 8082 | `GET /healthz`, `GET /readyz` | `GET /metrics` |
| decision | 8087 | `GET /healthz`, `GET /readyz` | — |
| websocket-gateway | 8080 | `GET /health` JSON | — |
| NATS | 4222 | TCP | monitoring `8222` |

## Health sweep

From repo root with compose up:

```bash
chmod +x scripts/lab/health-sweep.sh
./scripts/lab/health-sweep.sh
```

Optional verbose bodies: `VERBOSE=1 ./scripts/lab/health-sweep.sh`

## Startup order (lab)

1. Data dependencies: Timescale, Neo4j, Vault, NATS (compose starts these).
2. Ingest and config-api (consumers of NATS / DB).
3. detection-pipeline (depends on ingest URL).
4. actions-api, correlator, decision, etl-enrich, websocket-gateway.

Distroless images (ingest, detection-pipeline) do not ship shell tools; rely on host-side `curl` in `health-sweep.sh` rather than in-container `HEALTHCHECK` for those services.

## Compose health checks

`actions-api` and `etl-enrich` include Docker `healthcheck` definitions where a probe command is available (Alpine `wget` for actions-api; Python `urllib` for etl-enrich).
