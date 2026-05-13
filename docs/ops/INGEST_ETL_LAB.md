# Ingest and ETL — lab operations (WO-OPS-004)

## Validation and errors

- Malformed JSONL lines return **400** with a plain-text error body derived from `readVisibilityEvents`.
- Validator failures return **400** with `event validation failed` prefix.
- Duplicate `event_id` values already persisted are **ignored** (no duplicate rows); `events_deduped_total` increments on that path.

## Storage (JSONL lab mode)

- Default path inside ingest container: `AEGIS_VISIBILITY_STORE_PATH` (see compose), typically `/data/visibility-events.jsonl`.
- **Recovery:** stop ingest, backup the JSONL file, truncate or replace corrupted lines, restart ingest. For SQLite-backed tests only, delete the test DB file.

## Replay fixtures

Sample file: `fixtures/visibility/lab-replay-sample.ndjson`

```bash
chmod +x scripts/lab/replay-visibility-fixtures.sh
./scripts/lab/replay-visibility-fixtures.sh
```

Then query summaries (see `scripts/lab/load-ingest-summaries.sh`).

## ETL / enrich

- `GET /healthz` returns `healthy` when all dependencies connected, otherwise `degraded`, with `dependencies` booleans.
- `GET /readyz` returns full snapshot; HTTP **503** when not ready.
- Processing counters live under `GET /metrics`.

## Bounded requests

- Visibility POST bodies are capped (`maxVisibilityRequestBytes` in ingest HTTP server).
