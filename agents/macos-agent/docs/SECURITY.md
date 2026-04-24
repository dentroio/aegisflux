# macOS Agent Security Baseline

## Design Rules

- The agent must run with the least privilege required for active collectors.
- The agent must not expose an inbound network listener by default.
- Telemetry must use an outbound-only control/telemetry channel.
- Event schemas must be validated before backend ingestion.
- Secrets must not be written to logs or event payloads.
- The collector layer must be separate from event normalization and transport.
- Enforcement work is excluded from Phase 1.

## macOS-Specific Rules

- Prefer Endpoint Security Framework for process/file visibility when production entitlements are available.
- Do not use deprecated kernel extensions.
- Do not add Network Extension enforcement until visibility is reliable and policy semantics are defined.
- Do not assume x86_64; Apple Silicon is the primary macOS deployment target, with Intel macOS as a compatibility target where needed.
- Code signing, notarization, hardened runtime, and entitlement review are required before any production release.

## Rust Safety Rules

- `unsafe_code` is forbidden in the initial crate.
- New dependencies require review for:
  - maintenance health
  - license
  - transitive dependency count
  - macOS API surface
  - known advisories
- Panics are not acceptable as normal error handling.
- Parsing and event construction should be fuzzable once schemas stabilize.

## Phase 1 Runtime Posture

- Runs in visibility-only mode.
- Emits heartbeat and future process, flow, DNS, and detection evidence.
- Does not block traffic.
- Does not modify PF, Network Extension policy, or MDM posture.
- Does not quarantine endpoints.

## Future Hardening

- Code signing and notarization
- Hardened runtime
- Endpoint Security entitlement review
- mTLS or signed telemetry envelope
- Tamper detection for service stop, binary modification, policy disablement, and log clearing
- Signature verification for local policy cache
- SBOM and dependency audit in CI
