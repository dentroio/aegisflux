# End-to-end pipeline smoke (WO-OPS-001)

Single entrypoint:

```bash
chmod +x scripts/lab/smoke-e2e-pipeline.sh
./scripts/lab/smoke-e2e-pipeline.sh
```

Environment variables match other lab scripts (`ACTIONS_URL`, `INGEST_URL`, `DETECTION_URL`, `LINUX_AGENT_UID`, `WINDOWS_AGENT_UID`). Replay device defaults to `lab-replay-device` (see `fixtures/visibility/lab-replay-sample.ndjson`).

## Optional flags

| Variable | Default | Effect |
|----------|---------|--------|
| `SKIP_HEALTH` | `0` | Set to `1` to skip `scripts/lab/health-sweep.sh`. |
| `SKIP_LAB_AGENT_ASSERT` | `0` | Set to `1` if lab agents are not registered (skips Linux/Windows UID presence on `GET /agents`). |
| `RUN_DETECTION_ROLLOUT` | `0` | Set to `1` to run `scripts/lab/smoke-detection-rollout.sh` after the core path (WO-DET-006). |

## Canonical scenarios (automated in the script)

| # | Scenario | Evidence |
|---|----------|----------|
| 1 | Service readiness | `GET {ACTIONS_URL}/readyz` JSON with `state` and `checks`. |
| 2 | Agent registry | `GET /agents` — optional assertion both lab UIDs appear. |
| 3 | Console merge path | `GET /console/summary/agents-workbench` includes `dependencies` with ingest `ok`. |
| 4 | Fleet readiness | `GET /console/summary/agent-readiness`. |
| 5 | Visibility ingest | Replay fixture → `POST /v1/visibility/events` accepted. |
| 6 | ABOM / summaries | `GET /v1/visibility/summary/dashboard`, `.../device`, `.../inventory` return `ok: true`. |
| 7 | Detection pack status | `GET {DETECTION_URL}/v1/agents/{uid}/detection-pack-status` (status may be `null`). |
| 8 | Operational events store | `GET /platform/operational-events`. |
| 9 | Detection opportunities | `GET /platform/detection-candidates`, `GET /platform/research-feed`. |
| 10 | Audit-mode lifecycle | Create → stage → status → match → revoke on `/platform/audit-bundles` (observe-only; **no blocking**). |
| 11 | (Optional) Full rollout | `RUN_DETECTION_ROLLOUT=1` runs `smoke-detection-rollout.sh`. |

## Manual console checklist

After the script passes, walk the console (or equivalent API-only review):

- Dashboard / home summary widgets if present.
- **Agents** workbench and agent detail (readiness strip, status `online`/`stale`/`offline`).
- **Inventory** / ABOM views fed by ingest summaries.
- **Detections** / research or candidate views if routed in your build.
- **Controls** — observe-only drafts only; no enforcement toggles.
- **Audit bundles** — list shows the smoke bundle lifecycle entries.

## Expected pass/fail interpretation

| Symptom | First place to look |
|---------|---------------------|
| `health-sweep` fails | `docker compose ps`, then per-service logs. |
| `/readyz` unhealthy | NATS not connected from actions-api container. |
| `/readyz` degraded | Ingest or detection URL wrong from actions-api; check compose `INGEST_API_URL` / `DETECTION_PIPELINE_URL`. |
| `GET /agents` missing lab UIDs | Tunnels, `check-agents.sh`, or set `SKIP_LAB_AGENT_ASSERT=1` for API-only CI. |
| `agents-workbench` ingest not `ok` | Ingest down or wrong `INGEST_API_URL` from actions-api. |
| Ingest summary `ok` false | Store not configured or query error — ingest logs. |
| `detection-pack-status` connection error | detection-pipeline container or port 8089. |
| Audit create 400 on mode | Only `audit` mode is allowed (enforcement modes rejected by design). |

## Timestamps and evidence

Capture for a release ticket:

- Shell output of `smoke-e2e-pipeline.sh` (or save to a log file: `./scripts/lab/smoke-e2e-pipeline.sh 2>&1 | tee wo-ops-001-evidence.log`).
- `date -u` and `git rev-parse HEAD` in the same note.

## Related

- `scripts/lab/check-agents.sh` — lab agent + tunnel path.
- `docs/ops/RELEASE_CANDIDATE_CHECKLIST.md` — folds this smoke into release flow.
