# WO-AGENT-001: Agent Performance Budget Telemetry

**Status:** Complete
**Phase:** Product Platform  
**Primary owner:** Agent / Backend / UI  

## Goal

Prove that AegisFlux agents are low-resource and safe to run continuously. Every agent should report its own CPU, memory, collector runtime, skipped collectors, and health.

## Scope

- Windows and Linux lab agents.
- Backend ingest and UI display.
- Observe/report only.

## Deliverables

- Event schema: `aegis.agent.performance`.
- Fields:
  - device_id
  - agent_id
  - sensor_version
  - os
  - process_cpu_percent
  - process_memory_rss_mb
  - collector_runtime_ms
  - collector_name
  - collection_interval_ms
  - skipped_reason
  - event_queue_depth
  - spool_bytes
  - pack_eval_runtime_ms
- Backend storage/query support.
- UI widget: Agent Budget.
- Device detail collector performance panel.

## Acceptance Criteria

- Windows agent reports performance telemetry.
- Linux agent reports performance telemetry.
- Dashboard can show fleet-level max/avg CPU and memory.
- Device detail can show collector runtime.
- Agent continues operating if performance telemetry collection fails.

## Dependencies

- Active Windows/Linux lab agents.

## Product Requirement

Default target: AegisFlux should feel like it is not there. The agent must stay conservative on CPU, memory, disk, and network.

## Implementation Notes

- Added `aegis.agent.performance` visibility schema and example fixture.
- Windows and Linux agents now emit performance records for each collector plus dynamic-pack evaluation timing.
- Performance telemetry includes best-effort process CPU/RSS, collector runtime, queue depth, spool size, and pack evaluation runtime.
- Dashboard Agent Budget now computes fleet max/avg CPU and max RSS from performance events.
- Device drill-in Collector Health now includes budget summary metrics and a collector runtime table.

## Verification

- `cargo test` in `agents/linux-agent`
- `cargo test` in `agents/windows-agent`
- `npm run build` in `ui/console`
