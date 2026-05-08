# AegisFlux Clarion UI Patterns to Adopt

**Purpose:** Capture design and architectural decisions from the AegisFlux / Clarion UI alignment discussions so they do not remain only in chat.

**Reference UI:** `/Users/sgerhart/workspace/github/sgerhart/clarion/frontend`

## Product Direction

AegisFlux should feel like a Clarion-compatible operational console while staying focused on AI-era endpoint evidence, agent governance, detection packs, inventory, and observe-only control design.

The desired shape is not a marketing dashboard. It is a workbench for repeated operator workflows:

- Find endpoints and agents that need attention.
- Understand evidence quality and telemetry freshness.
- Inspect AI-capable tools, local models, MCP/tool bridges, browser extensions, and SASE/SSE components.
- Review detection-pack rollout and agent resource budgets.
- Draft controls only after evidence, simulation, approval, and rollback readiness.

## Persistent Shell Pattern

Borrow Clarion's shell model:

- Full-width top header.
- Persistent left navigation grouped by workflow.
- Breadcrumb row above page content.
- Only the main/right content panel scrolls.
- Detail pages and management panels render inside the shell rather than replacing it.

AegisFlux panel navigation should keep the left menu visible. Agents, Inventory, Detection Packs, Controls, AI Providers, and Settings should all use the same shell.

## Agent and Endpoint Workbench Pattern

Clarion's endpoint list should be the model for AegisFlux Agents:

- KPI cards at the top for the current workbench context.
- Quick-filter tabs with counts.
- Search and scoped filters.
- Table/card view toggle.
- Column visibility controls.
- Persisted list preferences.
- Fast inspection from the list, with optional full detail.

Candidate Aegis quick filters:

- All
- Online
- Stale
- Pack Applied
- Pack Rejected
- Needs Attention
- High Risk
- Budget Pressure

## Detail Page Pattern

Clarion's endpoint detail page has the right structure for AegisFlux device/agent detail:

- Back link.
- Strong identity header with icon, name, agent UID, host/device ID, OS, version, IP, and labels.
- Status chips for freshness, platform, detection-pack state, findings, and inventory flags.
- Edit/manage actions in a compact action group.
- Freshness row showing last seen and first seen.
- Four context cards:
  - Evidence Confidence
  - Network Context
  - Detection / Policy Context
  - Next Best Action
- Tabs for deeper surfaces.

Recommended Aegis detail tabs:

- Overview
- Evidence
- Inventory
- Detection Packs
- Performance
- Policy

## Detail Layering

Clarion uses both quick detail modals and full detail pages. AegisFlux should follow that pattern:

- Modal or side panel for fast inspection from dense lists.
- Full detail page for investigations, tabs, history, actions, and audit.

This should apply to:

- Agents/devices.
- Findings.
- Inventory items.
- Detection-pack candidates.
- Draft controls.

## Evidence Confidence Pattern

Clarion's identity source confidence should become AegisFlux evidence confidence.

Evidence sources should be visible as contributing signals:

- Agent heartbeat.
- Process telemetry.
- DNS telemetry.
- Flow telemetry.
- Browser extension telemetry.
- SASE/SSE telemetry.
- Detection-pack findings.
- Collector status.
- Agent resource budget events.

The UI should distinguish "not observed" from "not available" and explain what collector or integration would improve confidence.

## Next Best Action Pattern

Every major AegisFlux workbench should include deterministic operator guidance before AI is involved.

Examples:

- Agent stale: check tunnel, heartbeat, or agent service.
- Pack rejected: inspect reason, signature/hash/schema/compatibility status.
- Budget pressure: review collector runtime and queue/spool metrics.
- Inventory gap: enable collector or verify integration source.
- Finding present: inspect evidence and draft observe-only control.
- AI unavailable: show deterministic evidence summary and disable AI-only actions.

AI should enhance this guidance later, not replace deterministic safety logic.

## Staleness and Trust Pattern

Borrow Clarion's staleness banners wherever telemetry trust matters:

- Agent list and detail.
- Inventory pages.
- Detection-pack rollout.
- Findings.
- Draft control simulation.
- AI provider health and audit.

Stale data should be visible, not silently mixed with fresh state.

## AI Side Panel Pattern

Clarion's Calyx side panel is the preferred model for general AI assistance:

- AI assistant opens as a right-side panel.
- Context-aware page actions remain in the page.
- AI actions send bounded context objects, not arbitrary data dumps.
- If AI is unavailable, page workflows remain useful through deterministic summaries.

AegisFlux should eventually have an Aegis AI side panel plus named page actions such as:

- Explain AI activity on this endpoint.
- Summarize detection-pack rollout failures.
- Explain inventory risk for this tool.
- Draft observe-only control from this finding.

## Operational Event Feed Pattern

AegisFlux needs a Clarion-style operational/audit feed for platform actions:

- Agent registration, heartbeat, stale/offline transitions.
- Detection candidate validation, approval, rejection, signing.
- Detection-pack rollout checks, applies, rejects, rollbacks.
- Inventory refresh and collector status changes.
- AI provider tests and AI run/audit events.
- Draft control creation, simulation, approval, rejection.
- Clarion export attempts and results.

The feed should be filterable by device, agent, pack, candidate, control, operation type, status, and time.

## Design Guardrails

- Keep AegisFlux dense, operational, and scan-friendly.
- Prefer tabs, tables, compact cards, filters, and drill-ins.
- Use cards for records/widgets, not nested page framing.
- Keep labels concrete: what was observed, what is missing, what changed, and what action is safe.
- Never imply enforcement is active unless it is explicitly approved and implemented.
- Keep endpoint agents lightweight and observe-first.

