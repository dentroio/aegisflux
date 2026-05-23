# WO-AGENTS-001: AI Agent Harness and Tool Runtime

**Status:** Implemented (lab)  
**Phase:** AI-Native Leap  
**Primary owner:** Backend / AI Platform / UI  

## Goal

Create the governed runtime that lets AegisFlux run specialized AI platform agents with bounded tools, auditable jobs, privacy controls, and approval gates.

## Problem

AegisFlux has AI-assisted slices, but the next product leap needs a durable agent harness instead of one-off AI actions. Agents must be able to investigate, draft, simulate, and validate while preserving evidence, privacy, and governance.

## Scope

- Backend agent job model and lifecycle.
- Registered agent metadata.
- Tool registry with typed inputs/outputs.
- Agent run audit records.
- Minimal UI/API surface to view runs and status.
- No endpoint-side AI execution.

## Deliverables

- Define `ai_agent_jobs`, `ai_agent_tools`, `ai_agent_tool_calls`, and `ai_agent_runs` contracts.
- Register initial system agents: Endpoint Analyst, Detection Researcher, Pack Author, Simulation Agent, Control Designer, Governance Reviewer.
- Add job lifecycle: queued, running, needs human input, completed, failed, cancelled, superseded.
- Add typed tool registry with at least three read-only tools:
  - device evidence summary,
  - findings/evidence path lookup,
  - detection candidate lookup or draft stub.
- Capture provider/model, prompts or redacted prompt preview, tool calls, duration, status, and error.
- Add UI/API view for recent agent runs and run detail.

## Acceptance Criteria

- Agents run through a single harness rather than bespoke action handlers.
- Tool calls are typed, timed, and audited.
- Agent run records are queryable by agent, device/finding context, status, and time.
- External provider calls respect existing privacy settings.
- No agent can call enforcement or endpoint mutation tools.

## Dependencies

- WO-OPS-008.
- WO-AI-001 through WO-AI-003.
- WO-AI-002 privacy/audit foundation.

## Non-Goals

- No autonomous enforcement.
- No broad chat UI.
- No arbitrary tool execution.
- No endpoint LLM calls.

## Suggested Verification

- Backend tests for job lifecycle and tool-call audit.
- UI build if run surfaces are added.
- Manual run of a stub/read-only agent job with captured tool calls.

## Implementation notes (May 12, 2026)

- **Harness:** `backend/actions-api/internal/agentsharness` — system agent registry, read-only tool registry with JSON Schema metadata, `Harness.Run` with job lifecycle constants, per-tool timing and typed input/output capture, redacted prompt preview via existing privacy redaction.
- **HTTP (Actions API):** `GET /platform/ai/agent-harness/agents`, `…/tools`, `…/runs` (query: `agent_id`, `device_id`, `finding_id`, `status`), `GET …/runs/{run_id}`, `POST …/agent-harness/jobs`. `POST /platform/ai/endpoint-evidence-analyst` delegates to harness agent `endpoint_analyst` and still appends legacy `AIRunRecord` for `/platform/ai/runs`.
- **Persistence:** In-memory on `PlatformData` (`AgentHarnessJobs`, `AgentHarnessRuns`). Relational contract and seed tools: `backend/internal/db/migrate/003_ai_agent_harness.sql`.
- **UI:** Connectors → AI Providers page lists recent harness runs and run detail (tool calls, redacted preview).
- **Follow-ups:** Bind harness to Postgres using migration 003; implement async transitions for `needs_human_input`, `cancelled`, `superseded`; route additional platform AI entry points through `POST …/agent-harness/jobs`.

## Verification results (lab)

| Check | Result |
|-------|--------|
| `go test ./...` in `backend/actions-api` | Pass |
| `npx tsc --noEmit` in `ui/console` | Pass |
| Manual | `POST /platform/ai/agent-harness/jobs` with `agent_id` + `device_id`; confirm `runs` list and detail show tool calls |
