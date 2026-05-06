# Detection pack local evaluator contract (v1)

This document specifies how Windows and Linux endpoint agents must load, authenticate, bound, and evaluate **detection packs** that conform to `detection_pack.v1` (`schemas/detection/detection-pack.v1.schema.json`). It intentionally does **not** define distribution, rollout, or console APIs (see future work orders).

## Goals

- **Data-only**: Packs are declarative JSON. Evaluators must not interpret strings as code, load plug-ins, or execute expressions beyond the matcher types defined in the schema.
- **Signed and versioned**: Agents accept only packs whose cryptographic signature verifies against a configured trust anchor (see [Signing](#signing)).
- **Observe-only**: `mode` must be `observe`. Agents must not use v1 packs to enforce, block, or mutate system state.
- **OS-aware**: Rules may narrow evaluation with `target_os`. Packs declare `supported_os`; agents skip packs that do not include the host OS.
- **Bounded**: Evaluation respects numeric caps in `evaluator_limits` and must remain safe on low-resource hosts.

## Artifacts

| Artifact | Path |
|----------|------|
| JSON Schema | `schemas/detection/detection-pack.v1.schema.json` |
| Lab / parity fixture | `schemas/detection/examples/default-ai-markers.v1.pack.json` |
| Schema tests | `backend/detectionpack-schema/schema_test.go` |

## Load and validation pipeline

Agents should process an on-disk or in-memory pack through the following ordered steps. Any failure is a **hard reject** for that pack (do not partially evaluate).

1. **Parse** the document as UTF-8 JSON.
2. **Schema-validate** against `detection_pack.v1` (draft 2020-12). Reject on error.
3. **Version gate**: Compare `min_agent_version` to the running agent semver. Reject if the agent is older than required.
4. **OS gate**: Reject if the host OS is not listed in `supported_os`.
5. **Expiry**: If `expires_at` is set and the current time is past expiry, reject.
6. **Signature verify** over the canonical signed payload (see [Signing](#signing)). Reject if missing, unknown algorithm, unknown key, or verification fails.
7. **Observe gate**: Reject if `mode` is not `observe` (v1 allows only this value; future schema versions may add modes).
8. **Budget install**: Read `evaluator_limits` and configure per-batch evaluation budgets (see [Resource limits](#resource-limits)).

Until signing keys exist in a given environment, agents may run in a **lab-only** configuration that accepts a designated test key id. Production agents must not use lab-only trust.

## Signing

- The `signature` object contains `algorithm` (`ed25519` or `rs256`), `key_id`, and `value_b64` (base64 signature bytes).
- The signature must cover a **canonical serialization** of the pack with the `signature` field omitted or set to a fixed placeholder, so signers and verifiers agree on byte-exact content. Implementations should use a deterministic representation (for example JSON Canonicalization Scheme [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785)) for the signed object graph. Exact canonical rules will be pinned when the signing worker ships; until then, lab fixtures may use placeholder signatures that still satisfy the schema.
- **Reject** packs when `key_id` is not mapped to a trusted public key, when the algorithm is unsupported, or when verification fails.

## Resource limits

Agents must enforce **at least** the following bounds while evaluating a single visibility batch against one pack. They may apply stricter platform defaults.

| Field | Meaning |
|-------|---------|
| `max_wall_time_ms_per_batch` | Stop rule evaluation after this wall-clock budget elapses; emit telemetry that the pack evaluation truncated. |
| `max_heap_bytes` | Do not allocate unbounded matcher state; if parsing indexes would exceed this budget, reject the pack at load time or degrade safely. |
| `max_rules_evaluated_per_batch` | Upper bound on rules considered per batch (sorted by ascending `priority`, then `rule_id`). |
| `max_cpu_percent_soft` | Cooperative yield / backoff when sustained CPU exceeds this soft cap during evaluation. |
| `max_string_comparisons_per_rule` | Ceiling on substring / equality checks triggered by a single rule match attempt; stop and skip rule with reason `limit_exceeded`. |
| `max_clause_depth` | Maximum nesting depth for composite `op` nodes (`all_of` / `any_of`). |
| `max_clauses_per_rule` | Maximum total clause nodes visited while evaluating one rule’s `match` tree. |

If a limit triggers mid-batch, the agent finishes emitting any already-matched findings, marks the evaluation run as **partial**, and includes `pack_id`, `pack_version`, and the limit hit in diagnostic / status telemetry.

## Rule evaluation semantics

### Ordering

Consider rules in ascending `priority`, breaking ties by lexicographic `rule_id`. Stop early if `max_rules_evaluated_per_batch` is reached.

### OS filtering

If `target_os` is present, skip the rule unless the host OS is listed. Compare case-insensitively against normalized agent OS labels (`windows`, `linux`, `macos`).

### Composite clauses

- **`all_of`**: Every child clause must match.
- **`any_of`**: At least `min_match` children must match. If `min_match` is omitted, it defaults to `1`.
- Depth and total clause count must respect `max_clause_depth` and `max_clauses_per_rule`.

### Leaf matchers

All matching is **observational** over normalized fields from existing visibility events (process, DNS, flow, browser extension inventory, SASE component inventory). Matchers never spawn subprocesses, read arbitrary files, or call network APIs solely for detection.

| Leaf | Intent |
|------|--------|
| `process` | Executable identity, command line, parent process, and path substring checks. Honor `case_insensitive` (default `true`). |
| `dns` | DNS query hostname substring checks (aligned with `aegis.dns.observed`). |
| `flow` | Network flow attributes for the same process when `same_process_only` is true. `has_any_flow` matches if any attributed flow exists. |
| `browser_extension` | Extension inventory fields (aligned with `aegis.browser_extension.observed`). |
| `sase_component` | SSE/SASE inventory fields (aligned with `aegis.sase_component.observed`). |

Exact field mapping from events to matcher inputs is agent-specific but must be deterministic and documented per platform sensor.

### Outputs (observe-only)

When a rule matches, agents emit findings consistent with existing visibility contracts (for example `aegis.agent.detected` / `aegis.risk_finding.created`) using the rule’s `classification`, scores, `pattern_tags`, and `recommended_action`. v1 packs never imply enforcement.

### Telemetry

Agents should report **`pack_id` and `pack_version`** alongside detection output or collector status. For controller-driven lab rollout (WO-DET-003), agents may POST status to the detection-pipeline service and optionally emit **`aegis.detection_pack.status`** visibility events (see `schemas/visibility/detection-pack-status.schema.json`).

## Explicit non-goals (this work order)

- Downloading, staging, or rolling out packs from cloud services.
- User interface or API for pack management.
- Changing correlator or ingest behavior.

## Related documents

- `docs/project_management/work_orders/product_platform/WO-DET-001_DETECTION_PACK_SCHEMA_AND_LOCAL_EVALUATOR_CONTRACT.md`
- `schemas/visibility/README.md`
- `docs/AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md`
