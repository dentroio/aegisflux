# WO-VIS-003: Windows Network and DNS Attribution

**Status:** Draft  
**Phase:** Visibility and Observability  
**Primary owner:** Agent  
**Target environment:** `windows-dev-agent-01`

## Goal

Attribute outbound network connections and DNS observations to the Windows process and user context collected by the Aegis sensor.

## Scope

Visibility only. No WFP block rules, firewall changes, or proxy redirection.

## Required Flow Telemetry

For each network flow event:

- `event_id`
- `event_type`: `flow.started` or `flow.ended`
- `timestamp`
- `device_id`
- `agent_id`
- `pid`
- `process_guid` if available
- `process_name`
- `process_path`
- `user`
- `protocol`
- `local_ip`
- `local_port`
- `remote_ip`
- `remote_port`
- `remote_hostname` if correlated
- `direction`
- `bytes_sent` and `bytes_received` if available
- `connection_state`

## Required DNS Telemetry

For each DNS observation:

- `event_id`
- `event_type`: `dns.observed`
- `timestamp`
- `device_id`
- `agent_id`
- `query`
- `answers`
- `resolver`
- `pid` or process attribution when available
- `correlation_method`

## Implementation Notes

Collection options to evaluate:

- ETW TCP/IP provider for connection events
- Windows DNS Client ETW provider for DNS observations
- Periodic `GetExtendedTcpTable` or equivalent API for snapshot correlation
- WFP event observation only if it does not imply enforcement work

Correlation should tolerate gaps. When DNS cannot be attributed directly to a PID, correlate by timestamp, remote IP, and recent query cache with a confidence score.

## Deliverables

- Network connection collector
- DNS observation collector
- Process-flow correlation cache
- DNS-flow correlation cache
- Normalized `aegis.flow.started`, `aegis.flow.ended`, and `aegis.dns.observed` events
- Functional tests for browser, PowerShell, Python, Node, and Git outbound flows

## Acceptance Criteria

- A browser connection to a known domain is attributed to the browser process.
- A Python script connection is attributed to `python.exe` and its parent process.
- A Node script connection is attributed to `node.exe`.
- A Git operation is attributed to `git.exe`.
- DNS/domain context is visible for at least browser and script scenarios.
- Events include a correlation confidence or method when attribution is inferred.

## Dependencies

- WO-VIS-001
- WO-VIS-002
- WO-VIS-004 for final event schema alignment

## Risks

- DNS over HTTPS may hide local DNS observations. Browser configuration must be controlled in the lab.
- Short-lived connections can be missed by polling-only approaches. Prefer event-driven collection where feasible.
- PID reuse can create incorrect attribution. Use process instance IDs and timestamps.

## Completion Evidence

- Sample flow and DNS event JSON
- Scenario logs for browser, Python, Node, Git, PowerShell
- Documented attribution confidence behavior
