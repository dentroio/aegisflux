# Aegis AI Agent Platform Architecture

## Purpose

Aegis should use AI agents the same way Clarion does: as governed product capabilities, not as a decorative chat feature. Clarion already has useful patterns for provider management, agent routing, health checks, privacy controls, audit records, and context-aware operator assistance. Aegis should carry those patterns forward and specialize them for AI endpoint visibility, dynamic detection, and safe control design.

The important distinction is that Aegis has two kinds of agents:

- Endpoint agents: lightweight Windows, Linux, and future macOS collectors that observe local evidence and apply approved detection packs.
- AI platform agents: backend/cloud-side research and reasoning workers that help operators understand evidence, generate detection candidates, and draft observe-only controls.

AI platform agents must never directly bypass governance to instruct endpoint agents. They create candidates. Aegis validates, signs, stages, audits, and then distributes approved packs or control drafts through normal platform workflows.

## Clarion Patterns To Reuse

### Provider Management

Clarion has an AI connector that supports multiple provider types, a default provider, provider health, test chat, and provider error visibility. Aegis should reuse the same product model:

- Local provider support for lab/private deployments.
- External provider support for OpenAI, Anthropic, Google, and future enterprise gateways.
- One default provider for standard agent work.
- Provider test actions and connection health.
- Visible provider errors and last-connected timestamps.

In Aegis, this belongs under Management > Connectors > AI Providers.

### Registered AI Agents

Clarion exposes registered agents as system-defined capabilities. Aegis should do the same. Operators should not think in raw prompts. They should see named agents with clear jobs, inputs, outputs, and governance state.

Initial Aegis registered agents:

- Ecosystem Research Agent: tracks new AI tools, domains, protocols, package names, browser extensions, IDE extensions, MCP servers, local model runtimes, and agent-to-agent frameworks.
- Detection Authoring Agent: converts research into candidate detection-pack changes with required evidence, confidence, risk, and false-positive notes.
- Pack Validation Agent: tests candidate packs against lab telemetry and known fixtures before approval.
- Endpoint Evidence Analyst: explains what happened on a selected endpoint using process, DNS, flow, browser, extension, SASE, and finding evidence.
- Control Drafting Agent: proposes observe-only controls with scope, blast radius, rollback, and evidence references.
- Privacy Review Agent: checks whether prompts, evidence payloads, and detection packs expose secrets, identifiers, or unnecessary customer data.

### AI Health Status

Clarion's AI service health pattern should become an Aegis status surface. Every AI-assisted panel should show whether AI is available, degraded, unavailable, or unknown. If unavailable, Aegis should keep the workflow usable with deterministic summaries and stored evidence.

Aegis UI behavior:

- Dashboard shows AI service status as a small management signal, not as the main dashboard story.
- Endpoint drill-in shows AI assistance status near AI actions.
- Detection-pack pages show whether research, validation, and signing workers are healthy.
- Controls remain disabled or marked draft-only if required AI validation is unavailable.

### Context-Aware Actions

Clarion's device modal asks specific questions with structured context rather than opening a generic chat window. Aegis should follow that model.

Examples:

- Explain AI activity on this endpoint.
- Summarize evidence for this finding.
- Draft an observe-only control.
- Identify likely AI tool capability.
- Compare this endpoint to its baseline.
- Explain why this detection matched.
- Suggest what the endpoint agent should observe next.

The UI should send a bounded context object, not a loose data dump. For endpoint detail, context should include device id, sensor version, process evidence, DNS evidence, flow evidence, browser extension evidence, SASE component evidence, collector health, current detection pack versions, and selected finding ids.

### Privacy And Audit Controls

Clarion's external LLM privacy controls are directly relevant to Aegis because endpoint evidence can contain sensitive hostnames, usernames, tokens, command lines, file paths, URLs, IPs, and business context.

Aegis should include:

- Sanitize external LLM requests.
- Audit external LLM requests.
- Store redacted preview only.
- Redact IP addresses and CIDRs.
- Redact MAC addresses.
- Redact usernames, emails, and hostnames.
- Redact command-line secrets and environment variables.
- Redact file paths when requested.
- Block raw secrets.
- Record provider, model, route, agent name, operation, redaction count, status, and error.

This must be a first-class management section, not an afterthought.

## Aegis-Specific AI Agent Loop

The differentiating loop should be:

1. Research agents discover new AI ecosystem signals.
2. Detection authoring creates candidate rules and required evidence chains.
3. Validation tests candidates against lab telemetry and historical customer telemetry where allowed.
4. Privacy review checks payload and rule metadata.
5. Human or policy approval promotes the candidate to a signed detection pack.
6. Endpoint agents fetch the signed pack and evaluate locally within CPU and memory budgets.
7. Findings flow back to Aegis with evidence and pack version.
8. Endpoint Evidence Analyst explains findings and Control Drafting Agent proposes observe-only controls.

This creates a continuous update process without making endpoint agents heavy or uncontrolled.

## Backend Model

Aegis should add backend concepts that mirror Clarion but are tuned for detection operations:

- `ai_providers`: provider configuration, status, default provider, error state.
- `ai_registered_agents`: system-defined agent metadata, route, status, allowed operations.
- `ai_agent_runs`: every agent invocation with request id, operator id, context type, provider, model, status, duration, and error.
- `ai_privacy_settings`: global tenant controls for external providers.
- `ai_privacy_audit`: sanitized prompt preview, redaction counts, provider, model, route, operation, status.
- `detection_research_items`: research findings waiting for authoring.
- `detection_pack_candidates`: generated but unapproved pack changes.
- `detection_pack_validations`: simulation results, fixtures, false-positive notes.
- `signed_detection_packs`: approved packs available for endpoint agents.

## UI Model

The Aegis management UI should eventually include these left-menu sections:

- Dashboard: clean platform overview and customizable widgets.
- Agents: endpoint list and device drill-in.
- AI Activity: findings, AI destinations, AI-capable tools, and Agent Bill of Materials.
- Inventory: browser extensions, IDE extensions, CLI tools, local model runtimes, SASE/SSE components.
- Detection Packs: versions, candidate changes, validation status, endpoint rollout.
- Controls: draft controls, simulations, approvals, rollback plans.
- Connectors: AI providers, SASE/SSE, identity, browser management, network sources.
- Settings: privacy, audit, roles, retention, performance budgets.

Dashboard stays light. Detailed AI-agent workflows belong behind management pages.

## Endpoint Agent Constraints

AI platform agents do not run on endpoints. Endpoint agents must remain low-resource collectors and local evaluators:

- No external LLM calls from endpoint agents.
- No heavy local inference in the normal agent.
- Detection packs are data, not arbitrary code.
- Packs are signed, versioned, scoped, and rollbackable.
- Endpoint agents report CPU, memory, runtime, skipped collectors, pack version, and collector health.
- New detection intelligence should update without rebuilding the endpoint agent.

## First Implementation Slice

The first useful Aegis implementation should be small and concrete:

1. Add an AI Providers management page modeled after Clarion's provider connector.
2. Add AI service health to the dashboard and endpoint drill-in.
3. Add Endpoint Evidence Analyst action on the agent detail view.
4. Store every AI request as an auditable run record.
5. Add privacy settings before enabling external providers by default.
6. Add a detection-pack candidate schema for dynamic AI detection.
7. Use lab telemetry from Windows and Linux to validate one generated candidate pack.

## Product Principle

Aegis should not say, "Ask AI anything about your endpoint."

Aegis should say, "Aegis agents continuously research the AI ecosystem, turn new behavior into validated detection intelligence, and explain endpoint evidence in a way operators can trust, approve, and control."
