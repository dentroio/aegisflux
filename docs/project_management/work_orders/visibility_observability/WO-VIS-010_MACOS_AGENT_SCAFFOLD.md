# WO-VIS-010: macOS Agent Scaffold

**Status:** Draft  
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
- Collector status placeholders for process, network, and DNS visibility
- Local JSONL event spool
- Security baseline documentation
- Lab-mode run command

## Acceptance Criteria

- Crate builds on Apple Silicon macOS.
- Crate runs in `--once --stdout` lab mode without root.
- Event envelope includes OS and architecture.
- README documents Endpoint Security and entitlement constraints.
- Security doc explicitly excludes enforcement from Phase 1.

## Dependencies

- None

## Risks

- Production process visibility requires Endpoint Security entitlements.
- Network visibility may require Network Extension or approved APIs.
- Code signing, notarization, hardened runtime, and entitlement review are required before production use.

## Completion Evidence

- `cargo fmt --check`
- `cargo check`
- `cargo test`
- `cargo run -- --once --stdout`
