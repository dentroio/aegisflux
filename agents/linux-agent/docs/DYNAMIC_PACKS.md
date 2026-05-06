# Linux dynamic detection packs (WO-DET-004)

Observe-only evaluation of signed `detection_pack.v1` documents from the WO-DET-003 detection-pipeline controller. The agent never executes pack code, never enforces policy from packs, and only emits existing visibility events (`aegis.agent.detected`, `aegis.risk_finding.created`) plus optional controller status posts.

## Controller contract (WO-DET-003)

Plain HTTP to `AEGIS_CONTROLLER_URL` (lab; no TLS in the reference transport):

| Method | Path | Purpose |
|--------|------|--------|
| GET | `/detection-packs/latest?os=linux&agent_version={semver}` | Discover latest compatible artifact metadata |
| GET | `/detection-packs/{pack_id}/artifact?os=linux&agent_version={semver}` | Download signed pack JSON |
| POST | `/agents/{agent_uid}/detection-pack-status` | Report rollout / validation state |

Artifact responses should include `X-Content-SHA256` (hex). The agent verifies that header against the downloaded body when present.

## Trust and verification

1. **Hash:** body SHA-256 matches `X-Content-SHA256` and the `sha256` field from the `latest` response.
2. **Signature:** detached Ed25519 over `aegis.detection_pack.v1\0` || SHA256(unsigned JSON), where unsigned JSON is the pack object without `signature`, serialized with stable object key order (`serde_json` / Go `json.Marshal` on maps).
3. **Schema / policy:** `schema_version`, `mode: observe`, `supported_os` includes `linux`, semver `sensor_version >= min_agent_version`, optional `expires_at`, `signature.algorithm == ed25519`, rule shape checks.
4. **Compatibility:** controller may return `406` on artifact fetch; the agent treats that as incompatible and keeps evaluating the last verified cache entry when possible.

Configure trust with **`AEGIS_DETECTION_PACK_PUBLIC_KEY`**: standard Base64 encoding of the raw 32-byte Ed25519 public key (same material the pipeline exposes to operators).

## Cache layout

Under `AEGIS_DETECTION_PACK_CACHE` or, by default, `{parent of AEGIS_EVENT_SPOOL}/detection-pack/`:

| File | Meaning |
|------|--------|
| `active_verified.pack.json` | Last fully verified artifact bytes |
| `active_verified.meta.json` | `artifact_id`, `pack_id`, `pack_version`, `sha256` |
| `previous_verified.pack.json` | Prior active pack (promoted on successful upgrade) |
| `previous_verified.meta.json` | Metadata for previous |

A failed download or verification does **not** delete the previous verified files; the agent continues with the last good active (or rolls back to previous when active fails local re-validation).

## Findings and evidence

`aegis.agent.detected` has a fixed payload schema. Pack identity is carried in **evidence** entries:

- `detection_pack_id`
- `detection_pack_version`
- `rule_id`

`required_evidence` from the rule is mapped to simple evidence rows where possible (`process`, `network_flow`, etc.).

## Status reporting

`POST .../detection-pack-status` includes WO-DET-003 fields such as `rollout_state`, `reason_codes`, `reason_detail`, `signature_status`, `hash_status`, `schema_status`, `compatibility_status`, and active/previous pack ids/versions when known. `emit_visibility` is sent as `false` from the agent; the pipeline may still record status internally.

## Rust toolchain

`rust-toolchain.toml` requests **1.82.0**. A direct dependency pin on `base64ct = 1.6.0` avoids transitive crates that require unstable Cargo `edition2024` parsing on older toolchains.
