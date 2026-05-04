# Aegis Platform Vision

## North Star

Aegis should become a first-of-kind, feature-rich AI-era endpoint and workload intelligence platform. The near-term lab is important, but the larger goal is bigger: build the system that understands AI-capable activity on devices, proves it with evidence, drafts safe controls, and eventually feeds that intelligence into Clarion.

Aegis should be innovative enough to stand alone, and disciplined enough to integrate cleanly.

## What Aegis Should Become

Aegis should not be positioned as only an endpoint agent, EDR clone, or telemetry collector. It should become the evidence and control-design layer for devices and workloads.

The platform should answer questions that current tools struggle with:

- What AI-capable tools exist on this endpoint?
- Which processes can call models, tools, browsers, shells, files, repositories, or secrets?
- Which local agents, MCP servers, browser extensions, IDE plugins, and automation frameworks are active?
- What network destinations, model gateways, and local model runtimes are being used?
- Which behavior is normal for this device and which behavior is new?
- What control would reduce risk without breaking normal work?
- What evidence proves that the control is justified?
- What is the blast radius, and how do we roll it back?

## Product Pillars

### 1. Dynamic AI Discovery

Aegis should continuously discover AI-agent capabilities on devices. Detection must be dynamic, with signed detection packs produced from research, simulation, and approval workflows.

This includes:

- AI apps and browser AI usage.
- Coding agents and IDE plugins.
- CLI agents and shell automation.
- MCP clients and servers.
- Agent-to-agent protocols.
- Local model runtimes.
- Enterprise model gateways.
- Tool calling and file/repository access.

### 2. Agent Bill of Materials

Aegis should maintain an Agent Bill of Materials for every endpoint and workload. This becomes a differentiated asset for Clarion later because it gives enterprise context systems a device-level view of AI capability, not just asset inventory.

### 3. Evidence Graph

Aegis should connect process, parent process, command line, flow, DNS, user, destination, detection, finding, draft control, and rollback evidence into one explainable path.

The UI and API should make it easy to move from:

finding to process to flow to DNS to draft control to simulation to approval.

The collection strategy should follow [AEGIS_SENSOR_FUSION_ARCHITECTURE.md](AEGIS_SENSOR_FUSION_ARCHITECTURE.md): Aegis must combine process, network, DNS, browser, TLS/proxy, local runtime, config, and baseline signals so modern transports like DoH, HTTP/3, gateways, and agent-to-agent protocols do not create blind spots.

### 4. Observe-First Control Design

Aegis should never rush to blocking. The platform should draft observe-only controls first, simulate historical impact, stage policy with approval, then enforce only when the evidence and rollback path are strong.

### 5. Management Plane

As the UI evolves, Aegis needs a full management area:

- Agent fleet health.
- Detection pack management.
- Research intelligence feed.
- Policy simulation.
- Draft control approval.
- Update channels and rollout cohorts.
- Integrations.
- Users, roles, and audit.
- Clarion export and handoff status.

### 6. Clarion Integration

Aegis should eventually integrate into Clarion as the host/workload evidence and local-control subsystem. Clarion should get richer context from Aegis, and Aegis should receive approved decisions from Clarion when endpoint or workload-local action is needed.

The integration should remain event-driven and API-driven. Aegis should not depend on Clarion internals, and Clarion should not depend on Aegis local storage internals.

## Standalone First, Integrated Later

The correct sequence is:

1. Make Aegis excellent as a standalone product experience.
2. Prove Windows and Linux visibility with live lab devices.
3. Add dynamic AI detection packs and Agent Bill of Materials.
4. Add draft controls, simulation, approval, and rollback.
5. Add management views and operational health.
6. Export Aegis evidence into Clarion context objects.
7. Let Clarion use Aegis evidence in broader enterprise decisions.
8. Let Clarion send approved local-control decisions back to Aegis.

This avoids reducing Aegis to a narrow Clarion feature too early. Aegis should mature as the innovation engine, then Clarion can consume it as a powerful subsystem.

## Clarion Value

Clarion becomes stronger when it can reason over Aegis evidence:

- Device has an AI coding agent.
- Agent launched shell commands.
- Agent accessed a repo and contacted a model gateway.
- Agent used an MCP server with file-system tools.
- Endpoint process touched a sensitive destination.
- Aegis has an observe-only draft control with blast-radius analysis.

Clarion can then decide whether the right action is:

- No action.
- Increased monitoring.
- User or manager review.
- Model gateway policy.
- SASE or proxy policy.
- ISE/SGT posture change.
- Local endpoint control through Aegis.
- Cross-system escalation.

## Innovation Bar

For Aegis to be first-of-kind, each feature should pass at least one of these tests:

- It detects AI-agent behavior that normal endpoint tools miss.
- It explains endpoint behavior in a way an operator can trust.
- It turns evidence into a safer control proposal.
- It reduces blast radius before enforcement.
- It gives Clarion context that Clarion cannot get from network telemetry alone.
- It makes AI usage governable without blocking innovation.

## Near-Term Product Commitments

- Keep endpoint collection reliable and lightweight.
- Keep dynamic detections observe-only until simulation exists.
- Make detection packs signed, versioned, expiring, and auditable.
- Build management UI early enough that operations do not become invisible.
- Keep Clarion integration explicit, versioned, and contract-based.
- Preserve Aegis as an innovation platform while designing for eventual Clarion integration.
