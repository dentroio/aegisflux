# Operations docs (WO-OPS-001 … WO-OPS-008)

| Document | Purpose |
|----------|---------|
| [E2E_PIPELINE_SMOKE.md](E2E_PIPELINE_SMOKE.md) | WO-OPS-001 canonical e2e smoke, manual console checklist, failure modes |
| [SERVICE_HEALTH_MAP.md](SERVICE_HEALTH_MAP.md) | Ports, health/readiness URLs, compose notes |
| [OBSERVABILITY_BASELINE.md](OBSERVABILITY_BASELINE.md) | Log fields, metrics, first five lab checks |
| [LAB_SECURITY_BASELINE.md](LAB_SECURITY_BASELINE.md) | Lab trust boundaries and endpoint classification |
| [INGEST_ETL_LAB.md](INGEST_ETL_LAB.md) | Ingest storage, dedupe, replay, ETL failure modes |
| [PERFORMANCE_BASELINE.md](PERFORMANCE_BASELINE.md) | Fixture sizes, load scripts, threshold placeholders |
| [RELEASE_CANDIDATE_CHECKLIST.md](RELEASE_CANDIDATE_CHECKLIST.md) | Clone-to-validated-lab operator path |

Related scripts: `scripts/lab/smoke-e2e-pipeline.sh`, `scripts/lab/health-sweep.sh`, `scripts/lab/replay-visibility-fixtures.sh`, `scripts/lab/load-ingest-summaries.sh`, `scripts/lab/check-agents.sh`.
