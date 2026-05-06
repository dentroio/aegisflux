# WO-DET-005: Windows Dynamic Detection Pack Evaluator

**Status:** Draft  
**Phase:** Product Platform  
**Primary owner:** Windows Agent  

## Goal

Add the first Windows agent implementation for controller-managed dynamic detection packs. The Windows agent should discover the latest compatible observe-only pack, verify it, cache it, evaluate it against locally collected visibility events, emit findings, and report pack status back to the controller.

This is observe-only evaluation. It must not enforce, block, redirect, quarantine, change WFP/Firewall policy, or execute pack-provided code.

## Scope

- Windows agent pack discovery from the `WO-DET-003` latest-pack API.
- Artifact retrieval and local cache.
- Signature, hash, schema, OS, version, mode, and expiration checks.
- Data-only rule evaluation against Windows visibility events.
- Dynamic finding emission using the existing visibility event contract.
- Detection-pack status reporting to the controller.
- Rollback to previous verified cached pack when the latest pack is rejected.

## Deliverables

- Windows agent configuration:
  - `AEGIS_CONTROLLER_URL`
  - `AEGIS_DETECTION_PACKS_ENABLED`
  - `AEGIS_DETECTION_PACK_CACHE`
  - `AEGIS_DETECTION_PACK_PUBLIC_KEY` or controller trust material
- Pack client:
  - `GET /detection-packs/latest?os=windows&agent_version={version}`
  - `GET /detection-packs/{pack_id}/artifact`
  - `POST /agents/{agent_uid}/detection-pack-status`
- Local cache layout for active and previous verified packs under `C:\ProgramData\Aegis\Agent\`.
- Local evaluator for the `WO-DET-001` rule model.
- Status reporting fields from `WO-DET-003`.
- Tests for:
  - valid pack applied.
  - unsigned pack rejected.
  - hash mismatch rejected.
  - expired pack rejected.
  - unsupported OS rejected.
  - unsupported agent version rejected.
  - non-observe-only pack rejected.
  - previous verified pack retained after rejection.

## Acceptance Criteria

- Windows agent can fetch and apply a compatible lab observe-only detection pack.
- Windows agent emits dynamic findings that identify the pack id, pack version, and rule id.
- Windows agent reports applied/rejected/stale/expired/incompatible status to the controller.
- Windows agent preserves the previous verified pack when a new pack is rejected.
- Evaluation is data-only and bounded; no arbitrary code execution is possible.
- Existing static Windows detections continue to run.
- Agent still works when the controller is unavailable by using the last verified cached pack.

## Dependencies

- `WO-DET-001`
- `WO-DET-003`
- Windows lab heartbeat/registration.

## Non-Goals

- No Linux implementation.
- No WFP, Firewall, endpoint enforcement, or blocking.
- No production rollout.
- No AI calls from the endpoint.
- No arbitrary scripts, regex engines with unsafe features, or unbounded evaluation.

## Security Notes

- Reject by default when trust, schema, compatibility, or mode cannot be proven.
- Never delete the previous verified pack until a new pack is fully verified and cached.
- Include rejection reasons in status reports.

