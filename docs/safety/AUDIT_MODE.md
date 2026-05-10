# Audit-mode bundles: why this is not enforcement

AegisFlux's roadmap ends with an Enforce pillar, but enforcement must be earned. The
**audit-mode bundle** is the first step of that journey. It looks like a policy bundle and is
delivered through the same staging contract a future enforcement bundle would use, but it
**never blocks, denies, or quarantines**.

## What an audit bundle does

- Declares scope (selectors) that an endpoint can evaluate.
- Declares expected match telemetry — what the endpoint should emit when scope matches.
- Declares expiration and rollback metadata so a reviewer can audit the entire path.
- Records approval references so the bundle can be tied back to operator decisions.
- Is signed/staged with an explicit `mode: audit` flag that the agent must honor.

## What an audit bundle does not do

- **No deny / block / quarantine.** The contract literally rejects any other mode.
- **No silent enforcement.** Match telemetry is the only side effect.
- **No production rollout matrix.** One lab adapter target is enough for the foundation.
- **No replacing the signed-pack trust model.** Audit bundles complement detection packs.

## Endpoint contract

An endpoint that receives an audit bundle must:

1. Validate that `mode == "audit"`. Reject any other mode with `incompatible`.
2. Persist the bundle and acknowledge with a status report (`accepted`, `rejected`,
   `incompatible`, or `stale` — pending is the default while waiting).
3. Evaluate the bundle's scope against the endpoint's local telemetry path.
4. When scope matches, send a match record (observe-only) back to the platform via
   `POST /platform/audit-bundles/{id}/match`.
5. On revoke or expiration, stop evaluating the bundle.

The wire-level shape of these calls is defined in
[`AUDIT_MODE_BUNDLE_CONTRACT.md`](AUDIT_MODE_BUNDLE_CONTRACT.md).

## Why this is safe

- The platform-side handler refuses any mode other than `audit`.
- The agent-side contract is observe-only. There is no allowed `block` or `deny` action.
- Bundles are time-bounded via `expires_at_ms`; stale bundles are reported by the agent.
- Revocation is explicit and recorded in history with operational events.
- The UI is labelled **Audit-only** in the breadcrumbs and detail modals so operators do not
  mistake the bundle for enforcement.

## How this maps to the roadmap

This is foundation work. A future enforcement bundle would re-use:

- the same staging route shape,
- the same endpoint receipt/status contract,
- the same revocation and history event model,

but with an additional, separately-approved `mode: enforce` and a controlled rollout matrix.
That is a separate work order.
