# WO-VIS-010: macOS Agent Scaffold

**Status:** Complete
**Phase:** Visibility and Observability
**Primary owner:** Agent
**Target environment:** macOS Apple Silicon first, Intel compatibility later

## Goal

Create a secure Rust scaffold for the macOS Aegis agent so macOS can become a supported visibility platform after the Windows Phase 1 path is proven.

## Scope

Scaffold only. No Endpoint Security entitlement request, system extension, Network Extension, PF rule, blocking, quarantine, or MDM integration.

## Deliverables

- `agents/macos-agent` Rust crate
- `unsafe_code = "forbid"`
- No third-party dependencies in the initial skeleton
- Agent heartbeat event
- Process snapshot events for lab visibility without privileged install
- Collector status placeholders for network and DNS visibility
- Local JSONL event spool
- Security baseline documentation
- Lab-mode run command
- Optional localhost-only `--post` to Aegis ingest

## Acceptance Criteria

- Crate builds on Apple Silicon macOS.
- Crate runs in `--once --stdout` lab mode without root.
- Event envelope includes OS and architecture.
- Lab mode emits `aegis.process.started` snapshot events with command-line collection disabled by default.
- `--post` can send the `--once` batch to local Aegis ingest without installing an agent service.
- README documents Endpoint Security and entitlement constraints.
- Security doc explicitly excludes enforcement from Phase 1.

## Dependencies

- None

## Risks

- Production process visibility requires Endpoint Security entitlements.
- Network visibility may require Network Extension or approved APIs.
- Code signing, notarization, hardened runtime, and entitlement review are required before production use.

## Completion Evidence

Completed April 26, 2026.

- `cargo fmt --check`
- `cargo check`
- `cargo test`
- `git diff --check`
- `AEGIS_EVENT_SPOOL=/tmp/aegis-macos-agent-smoke/events.jsonl AEGIS_PROCESS_SNAPSHOT_LIMIT=5 cargo run -- --once --stdout`
- `AEGIS_BACKEND_URL=http://127.0.0.1:19091 AEGIS_EVENT_SPOOL=/tmp/aegis-macos-agent-smoke/events.jsonl AEGIS_PROCESS_SNAPSHOT_LIMIT=5 cargo run -- --once --post`

The smoke run emitted heartbeat, process collector status, five `aegis.process.started` snapshot events, and pending network/DNS collector status events without requiring root or an installed agent.
