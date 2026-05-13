# Performance and load baseline (WO-OPS-007)

## Fixture sizes (lab)

| Size | Approx use |
|------|----------------|
| Small | `fixtures/visibility/lab-replay-sample.ndjson` (2 events) |
| Medium | Repeat POST of small fixture 50× in a shell loop |
| Stress | Generated NDJSON (out of scope here; use `scripts/` loop with larger batch files when added) |

## Repeatable commands

| Command | Measures |
|---------|----------|
| `./scripts/lab/replay-visibility-fixtures.sh` | Ingest accept latency (curl `time_total`) |
| `./scripts/lab/load-ingest-summaries.sh` | Summary route latency for dashboard/device/inventory |
| `cd ui/console && npm run build` | Console bundle build health |

## Threshold placeholders (lab — not CI gates yet)

Document machine OS, CPU model, and date next to any numbers you record.

| Route | Starting expectation |
|-------|----------------------|
| `summary/dashboard` | `< 2s` warm on small JSONL store |
| `summary/device` | `< 2s` |
| `POST /v1/visibility/events` (small batch) | `< 1s` localhost |

## Agent budget

- Linux agent emits `aegis.agent.performance` events (see agent code). Collect from ingest or logs during normal vs dynamic-pack runs and compare `process_cpu_percent` / `process_memory_rss_mb`.

## Bottlenecks to watch

1. Large JSONL ingest store without rotation.
2. Summary queries scanning full event history (lab volumes stay small).
3. detection-pipeline cold start when signing keys missing.

File follow-up work orders if optimization exceeds operational-readiness scope.
