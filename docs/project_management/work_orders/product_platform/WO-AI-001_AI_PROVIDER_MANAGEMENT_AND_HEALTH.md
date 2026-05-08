# WO-AI-001: AI Provider Management and Health

**Status:** Completed (May 2026 — lab slice implemented)
**Phase:** Product Platform  
**Primary owner:** Backend / UI  

## Goal

Bring Clarion's governed AI provider pattern into AegisFlux: providers are configured, tested, selected, monitored, and surfaced as platform health.

## Scope

- Backend provider model.
- Management UI page.
- Health state used by future AI-assisted actions.

## Deliverables

- Provider model supporting:
  - local
  - OpenAI
  - Anthropic
  - Google
  - future enterprise gateway
- Default provider selection.
- Provider test endpoint.
- Health endpoint exposing AI status.
- UI page under Configure > Connectors > AI Providers.
- Header/dashboard AI health chip.

## Acceptance Criteria

- Operator can view providers and default provider.
- Operator can test provider connectivity.
- AegisFlux can report AI Available, Degraded, Unavailable, or Unknown.
- AI-assisted UI can disable actions when AI is unavailable.
- Provider secrets are never returned to the UI.

## Dependencies

- WO-PLAT-001
- Existing backend service decision on where provider config lives.

## Security Notes

- External provider support must be paired with WO-AI-002 before customer data is sent.
- Endpoint agents must not call external LLM providers.

