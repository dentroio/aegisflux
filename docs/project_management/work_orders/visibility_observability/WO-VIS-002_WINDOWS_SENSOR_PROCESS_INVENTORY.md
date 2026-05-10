# WO-VIS-002: Windows Sensor Process Inventory

**Status:** Complete (`aegis.process.started` snapshot + `aegis.process.ended` via `--lifecycle` / `--interval-secs` diff; ETW optional for production hardening)  
**Phase:** Visibility and Observability  
**Primary owner:** Agent  
**Target environment:** `windows-dev-agent-01`

## Goal

Build the first Windows visibility sensor capability: process start/stop telemetry, process metadata, user/session context, and parent/child lineage.

## Scope

Visibility only. The sensor observes and reports process context but does not block or alter behavior.

## Required Telemetry

For each process event:

- `event_id`
- `event_type`: `process.started` or `process.ended`
- `timestamp`
- `device_id`
- `agent_id`
- `pid`
- `ppid`
- `process_guid` or stable local process instance ID
- `parent_process_guid` if available
- `name`
- `path`
- `command_line`
- `user`
- `logon_session_id`
- `integrity_level` if available
- `sha256` if available without unacceptable overhead
- `publisher` or signer if available
- `sensor_version`

## Implementation Notes

Preferred collection options to evaluate:

- Windows ETW process provider
- Windows Event Log / Security events where available
- WMI process start trace as fallback
- Sysmon integration as optional lab source, not a required customer dependency

The implementation should separate collection from event normalization. We need to be able to replace the collector without changing the backend event contract.

## Deliverables

- Windows sensor process collector
- Normalized `aegis.process.started` and `aegis.process.ended` events
- Local debug log for collector health
- Basic buffering/retry for backend outages
- Unit tests for event normalization
- Functional test that launches `notepad.exe`, `powershell.exe`, `python.exe`, and `node.exe`

## Acceptance Criteria

- Starting and stopping common processes produces normalized events.
- Parent/child lineage is captured for:
  - `powershell.exe` launching `python.exe`
  - `cursor.exe` or `Code.exe` launching helper processes when available
  - `cmd.exe` launching a script
- Command line is captured when permissions allow.
- Sensor does not require enforcement privileges.
- Sensor overhead is acceptable for lab use and measured.

## Dependencies

- WO-VIS-001
- WO-VIS-004 for final event schema alignment

## Risks

- Some command-line or signer fields may require elevated permissions.
- ETW collection reliability and event volume need validation.
- Parent process may exit before enrichment completes. Cache process metadata.

## Completion Evidence

- Process inventory in `aegisflux/agents/windows-agent` (`windows.process` collector).
- **Lifecycle mode:** `--lifecycle` or `--interval-secs` emits `aegis.process.started` / `aegis.process.ended` via snapshot diff and `AEGIS_PROCESS_STATE_PATH` (see agent README).
- `cargo test` passes for the agent crate.
- Command-line collection is opt-in and disabled by default.
- Optional hardening: kernel ETW for exit codes and richer session fields (production follow-up).
