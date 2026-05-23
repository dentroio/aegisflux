# AegisFlux AI-Native Leap Architecture

## Purpose

The next AegisFlux leap is to move from an evidence and control-design platform with AI-assisted features into an AI-native security operating system for endpoint evidence, adaptive detections, and governed local controls.

The leap is not "add chat." It is a governed agent harness where specialized AI platform agents can investigate, research, draft, simulate, and validate inside strict evidence, privacy, audit, and approval boundaries.

Endpoint agents remain lightweight. AI platform agents run in the backend/platform layer and produce auditable work products. They do not directly enforce policy, bypass approval, or execute arbitrary endpoint code.

## North Star

**AegisFlux deep agents continuously discover AI-capable endpoint behavior, explain evidence, design safe controls, simulate impact, govern approvals, and adapt detection intelligence.**

The product loop remains:

**Discover -> Explain -> Design -> Simulate -> Govern -> Adapt**

The AI-native leap makes that loop agent-driven:

- **Discover:** Fleet Scout and Research agents identify new AI tools, protocols, capability drift, MCP/A2A/local-runtime signals, and risky reachability.
- **Explain:** Endpoint Analyst agents turn process, DNS, flow, ABOM, finding, and collector-health evidence into cited investigation narratives.
- **Design:** Control Designer agents draft observe-only controls, scope, rollback, expected telemetry, and approval packets.
- **Simulate:** Simulation agents replay detection candidates and draft controls against historical/lab evidence.
- **Govern:** Governance agents enforce privacy, evidence citation, human approval, prompt/tool audit, and decision-ledger requirements.
- **Adapt:** Detection Researcher and Pack Author agents mature research into signed observe-only detection packs.

## Architectural Layers

### 1. Agent Harness

The harness is the runtime for AI platform agents.

Core responsibilities:

- Register system-defined agents and allowed operations.
- Create agent jobs and track lifecycle.
- Provide a bounded tool registry.
- Capture prompts, model/provider metadata, tool calls, evidence snapshots, outputs, approvals, and errors.
- Enforce privacy/redaction policy before external provider calls.
- Require evidence references and confidence metadata for agent conclusions.
- Make runs replayable from captured inputs and tool results.

Initial agent job states:

- `queued`
- `running`
- `needs_human_input`
- `completed`
- `failed`
- `cancelled`
- `superseded`

### 2. Evidence-Bound Reasoning Contract

Every agent output that makes a claim must include:

- Evidence references.
- Confidence level and reason.
- Missing evidence.
- Assumptions.
- Recommended next step.
- Safety boundary.
- Whether human approval is required.

This prevents AI output from becoming unaudited speculation. AegisFlux should be comfortable saying "unknown" when evidence is missing.

### 3. Tool Registry

Agents should use explicit tools instead of free-form backend access.

Initial tool categories:

- Evidence query tools: device summary, process lineage, DNS/flow lookup, findings, ABOM, operational events.
- Detection tools: research item lookup, candidate draft, schema validation, simulation replay, pack status.
- Control tools: draft-control lookup, blast-radius simulation, audit-bundle status.
- Governance tools: privacy preview, approval request, decision-ledger append.
- Release tools: smoke check, health sweep, performance baseline lookup.

Tools must have typed inputs/outputs, authorization rules, timeouts, and audit records.

### 4. Memory and Decision Ledger

Agent memory must be product memory, not unbounded chat memory.

Memory types:

- Device memory: baseline capabilities, recurring findings, collector health, known gaps.
- Fleet memory: widespread AI tools, new/risky capabilities, stale detections, rollout health.
- Detection memory: research lineage, candidate versions, validation results, approval decisions, retirement reason.
- Control memory: draft revisions, simulations, operator decisions, rollback plans.
- Run memory: prompts, tools, outputs, redactions, provider/model, status, duration.

The decision ledger records who or what recommended a change, what evidence supported it, what was approved/rejected, and why.

### 5. Adaptive Detection Factory

The factory turns ecosystem signals and endpoint evidence into governed detection intelligence.

Flow:

1. Research agents collect new AI ecosystem signals.
2. Opportunities are scored for relevance, novelty, risk, confidence, and expected false positives.
3. Pack Author agents draft candidate detection-pack changes.
4. Validation agents run schema, fixture, and historical replay checks.
5. Privacy/Governance agents verify safe metadata and evidence handling.
6. Humans or policy gates approve promotion.
7. Packs are signed and rolled out through existing detection-pack workflows.
8. Rollout and finding quality feed back into lifecycle decisions.

### 6. Control Design Copilot

The copilot turns findings and evidence paths into observe-only control proposals.

It should produce:

- Proposed scope.
- Evidence references.
- Expected match telemetry.
- Blast radius.
- Breakage risks.
- Rollback notes.
- Expiration/review date.
- Approval packet.

The copilot can recommend audit-mode staging, but cannot enable blocking behavior.

## Safety Boundaries

- No endpoint LLM calls.
- No arbitrary code in detection packs.
- No agent can directly enforce policy.
- No blocking/quarantine/deny behavior by default.
- External model calls must pass privacy policy.
- Agent outputs that affect packs, controls, audit bundles, or approvals must be auditable.
- Human approval or explicit policy gates are required for promotion.

## First Capability Wave

The first AI-native wave should build the substrate before adding spectacle:

1. AI agent harness and tool runtime.
2. Evidence-bound reasoning contract.
3. Endpoint Analyst deep agent.
4. Detection opportunity research agents.
5. Detection candidate simulation harness.
6. Control Design Copilot.
7. Fleet AI capability drift radar.
8. Governance, memory, and decision ledger.
9. Autonomous demo/scenario generator.
10. Enforcement readiness scorecard.

## Relationship to Operational Readiness

Operational readiness must land first. The AI-native wave depends on reliable agent status, service health, ingest/ETL behavior, observability, e2e validation, security baseline, performance baselines, and release readiness.

The AI-native wave should not try to repair foundational lab instability. If it finds reliability gaps, it should file or reopen operational-readiness work rather than burying fixes inside agent features.
