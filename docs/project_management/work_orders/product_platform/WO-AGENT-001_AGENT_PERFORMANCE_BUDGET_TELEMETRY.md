# WO-AGENT-001: Agent Performance Budget Telemetry

**Status:** Draft  
**Phase:** Product Platform  
**Primary owner:** Agent / Backend / UI  

## Goal

Prove that Aegis agents are low-resource and safe to run continuously. Every agent should report its own CPU, memory, collector runtime, skipped collectors, and health.

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

Default target: Aegis should feel like it is not there. The agent must stay conservative on CPU, memory, disk, and network.

