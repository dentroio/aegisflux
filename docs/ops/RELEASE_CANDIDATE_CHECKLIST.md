# Release candidate checklist — lab (WO-OPS-008)

Use this after WO-OPS-001 … WO-OPS-007 work is reflected on `main`. Evidence columns are suggestions for a release ticket.

## Prerequisites

- [ ] Docker and Compose installed; ports 8083, 9091, 8089, 8088, … available.
- [ ] Repo cloned; no uncommitted secrets (`git status` clean of `.env`).

## Environment

- [ ] Copy or export required lab URLs (`ACTIONS_URL`, `INGEST_URL`, …) per `docs/ops/SERVICE_HEALTH_MAP.md`.
- [ ] Optional: `AEGIS_WINDOWS_HOST`, `AEGIS_LINUX_HOST` for remote tunnel checks.

## Startup

- [ ] `docker compose up -d`
- [ ] Wait until `docker compose ps` shows **Up** (and healthy where defined).

## Health and readiness

- [ ] `./scripts/lab/health-sweep.sh` — pass
- [ ] `curl -s http://127.0.0.1:8083/readyz` — `state` documented (`ready` vs `degraded`)

## Agents

- [ ] `./scripts/lab/check-agents.sh` — expected agents online or documented `stale` reason

## Data path

- [ ] `./scripts/lab/replay-visibility-fixtures.sh` — `202 Accepted`
- [ ] `./scripts/lab/smoke-e2e-pipeline.sh` — full WO-OPS-001 path (or `SKIP_LAB_AGENT_ASSERT=1` in CI without agents)
- [ ] `./scripts/lab/load-ingest-summaries.sh` — HTTP 200 and reasonable `time_total`

## Detection / smoke (optional but recommended)

- [ ] `./scripts/lab/smoke-detection-rollout.sh` when tunnels and agents are configured (WO-DET-006).

## Console

- [ ] `cd ui/console && npm run build` — success

## Known limitations (must be explicit)

- **Lab-only:** no production SLA; plain HTTP; dev Vault token.
- **Observe-only:** controls and simulations do not enforce blocking.
- **Audit-only:** audit bundles do not imply enforcement readiness (see product docs).

## Reset / cleanup

- [ ] `docker compose down` (optional `-v` to drop volumes — **destructive**)

## CI/CD gaps (follow-up)

- Compose health not attached to all distroless services.
- No automated release artifact signing in this checklist path.

## References

- Work orders: `docs/project_management/work_orders/product_platform/WO-OPS-*.md`
- Ops docs index: `docs/ops/README.md`
