# Aegis UI Clarion Alignment

## Goal

Aegis should evolve as a Clarion-compatible management experience. The UI can have its own AI-security focus, but navigation, density, controls, and operational language should feel familiar to Clarion users.

Reference frontend:

```text
/Users/stevengerhart/workspace/github/sgerhart/clarion/frontend
```

## Clarion Patterns To Reuse

- Workflow sidebar grouped by phases: dashboard, discover, analyze, secure, operate, configure.
- Compact KPI cards with clear operational labels and short health hints.
- Dense tables and drill-ins for repeated investigation work.
- Connector and settings sections for external systems and platform management.
- Health/readiness status surfaces that explain missing telemetry.
- Restrained Tailwind utility styling with light/dark support.
- Lucide icons for navigation, actions, status, and compact controls.

## Aegis Navigation Direction

Initial Aegis sections should map cleanly to Clarion concepts:

| Aegis Section | Purpose |
|---------------|---------|
| Dashboard | AI/security posture, endpoint health, recent findings, collector coverage |
| Devices | Windows/Linux/macOS agent inventory, resource health, last seen, collector status |
| AI Activity | browser AI, CLI AI, IDE agents, local models, MCP/tool bridges |
| Evidence | processes, flows, DNS, browser history, extensions, SASE/SSE components |
| Findings | explainable detections and observe-only control recommendations |
| Controls | draft policies, staged controls, rollback plans, enforcement integrations |
| Connectors | Clarion, SASE/SSE, proxy, firewall, identity, model gateway integrations |
| Management | agents, detection packs, resource budgets, schedules, certificates, tenants |

## Management UI Requirements

The UI must grow beyond a visibility console. It needs first-class management for:

- Agent enrollment, version, health, and resource usage.
- Collector enablement, cadence, and performance budgets.
- Detection-pack updates and approval state.
- SSE/SASE, proxy, identity, firewall, and Clarion connectors.
- Enterprise browser and extension inventory.
- Observe-only policies before enforcement.
- Evidence retention and privacy controls.

## Design Guardrails

- Keep the first screen operational, not marketing-oriented.
- Prefer compact panels, tables, tabs, and drill-ins over large decorative sections.
- Use cards for repeated records and dashboard widgets, not nested page structure.
- Keep wording concrete: what was observed, why it matters, what is missing, and what can be done next.
- Make resource impact visible wherever agent health appears.

The UI should eventually look like Aegis belongs inside Clarion, while still making AI detection, endpoint evidence, and agent governance the center of the experience.
