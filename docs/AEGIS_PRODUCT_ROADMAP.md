# AegisFlux Product Roadmap

## Product Thesis

**Observe. Adapt. Enforce.**  
**Adaptive security. Real-time protection.**

AegisFlux should become the evidence-to-control platform for the AI-era endpoint. The market already has tools that detect, alert, block, and aggregate. AegisFlux can be different by making every control recommendation traceable to live evidence: process, command line, network flow, DNS, identity, policy, blast radius, and rollback plan.

The practical wedge is simple: observe first, adapt continuously, explain clearly, draft safely, and enforce only when the operator trusts the evidence.

The larger platform vision is captured in [AEGIS_PLATFORM_VISION.md](AEGIS_PLATFORM_VISION.md): AegisFlux should mature as a first-of-kind standalone innovation platform, then integrate into Clarion through explicit events and APIs as Clarion's host/workload evidence and local-control subsystem.

Clarion's AI-agent patterns should also carry into AegisFlux. AegisFlux should support governed AI providers, registered AI platform agents, privacy controls, audit records, endpoint-specific AI assistance, and research-to-detection workflows. See [AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md](AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md).

## Product Loop

The product should be shaped around one memorable operator loop:

**Discover -> Explain -> Design -> Simulate -> Govern -> Adapt**

- **Discover:** Agent Bill of Materials shows AI-capable tools, local runtimes, model gateways, browser/IDE/CLI agents, and capability changes across the fleet.
- **Explain:** Evidence graph turns endpoint telemetry into a plain-language investigation path with confidence and missing-evidence callouts.
- **Design:** Findings become observe-only draft controls with scope, evidence, and rollback notes.
- **Simulate:** Operators see historical matches, affected endpoints/users/destinations, and breakage risk before staging anything.
- **Govern:** Agent health/readiness, decision history, approvals, operational events, and audit records make trust visible.
- **Adapt:** Research opportunities become detection candidates, simulated packs, signed observe-only rollout, and retired detections when stale.

## Current Lab State

The lab now has a working visibility foundation:

- Windows endpoint collection with process, network, DNS, and finding events.
- Linux endpoint collection with the same ingest contract.
- Mac-hosted ingest, storage, and console access through lab tunnels.
- Device discovery across reporting endpoints.
- Investigation drill-in from a process or finding to linked processes, flows, DNS records, and detections.
- Observe-only UI that can show evidence without pretending to enforce policy yet.
- Agent Bill of Materials, evidence path, finding-to-control designer, AI research feed, first-value demo, console summary APIs, and UI regression harness have initial lab slices.

This is enough to demonstrate the core loop: endpoint evidence arrives, AegisFlux connects it into a path, then AegisFlux starts drafting the control that would reduce risk.

## Differentiation

### Versus EDR

EDR is optimized for malware detection and response. AegisFlux should focus on understandable control design: what should this endpoint be allowed to do, why, and what happens if we change it?

### Versus SIEM

SIEM centralizes logs and correlation. AegisFlux should centralize evidence that can become action. The output is not only an alert; it is a candidate policy with supporting proof and rollback.

### Versus ZTNA, SASE, and NAC

These platforms often enforce network access but lack deep endpoint process context. AegisFlux can bridge endpoint behavior to enforcement surfaces: host firewall, eBPF, WFP, SASE policy, NAC identity groups, and segmentation tags.

### Versus CNAPP and CSPM

Cloud posture tools see cloud configuration well. AegisFlux should see the human and machine activity that reaches those services from real endpoints, especially AI tooling, automation, scripts, browsers, and local agents.

## Core Concepts

### Agent Bill of Materials

AegisFlux should maintain an Agent Bill of Materials for every endpoint:

- AI desktop apps and browser AI sessions.
- CLI agents, shell automation, scheduled tasks, and script runners.
- Model gateways, API endpoints, and local model runtimes.
- The processes, users, DNS names, and remote services each tool touches.
- The policies currently observing or controlling that activity.

This gives customers an inventory they do not reliably get today: which AI-capable agents exist, what they can reach, and what evidence supports that conclusion.

### Dynamic AI Detection

AI tooling changes too quickly for static endpoint signatures to carry the product. AegisFlux should support a research-to-detection loop where cloud-side research agents monitor new AI tools, protocols, model gateways, MCP servers, browser agents, coding agents, and agent-to-agent ecosystems, then generate signed observe-only detection packs for endpoint agents.

See [AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md](AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md) for the detailed architecture.

The AI platform itself should be managed like a first-class enterprise connector, following the Clarion model: provider configuration, registered agents, health status, privacy redaction, and auditable runs.

### Finding to Draft Control

Every finding should be able to produce a draft control:

- Proposed action: observe, restrict, deny, quarantine, segment, require approval, or monitor.
- Scope: process path, signer, command marker, user, host, remote IP, DNS name, port, protocol, or identity group.
- Evidence: the exact process, flow, DNS, and finding records used.
- Blast radius: what else would be affected if the control became active.
- Rollback: how AegisFlux would revert the change and what success telemetry should look like.

The first implementation should stay observe-only. The UI can show the draft, but backend policy staging and enforcement adapters should come later.

## Roadmap

### Now: Visibility Foundation

- Keep Windows and Linux reporting reliably into ingest.
- Normalize endpoint events into stable schemas.
- Improve device health: event freshness, collector version, last successful run, and tunnel status.
- Add better sample detections for AI agents, suspicious automation, model gateways, and unusual outbound flows.
- Keep the console focused on evidence and investigation rather than broad dashboards.
- Keep UI regression and performance checks part of every product slice.

### Near: Product Pull and Trust

- Polish ABOM into fleet insights: new, risky, widespread, low-confidence, and stale AI capabilities.
- Refine evidence graph UX so operators see what happened, why it matters, what is missing, and what to do next.
- Deepen draft-control simulation with affected endpoints/users/destinations, breakage risk, and decision history.
- Add agent health/readiness scoring so evidence trust is visible.
- Improve the first-value demo with sample scenarios and polished empty states.
- Add signed policy bundle models without enabling enforcement by default.

### Mid: Governed Adaptation and Audit-Mode Controls

- Add policy staging, approval gates, and rollback plans.
- Add device health scoring and deployment readiness.
- Add signed agent registration and mTLS transport.
- Add policy diff views: what changed, why, who approved it, and which evidence justified it.
- Mature adaptive detection workflow from research opportunity to candidate, simulation, signed pack, rollout, and retirement.
- Add lab enforcement adapters for Windows Firewall/WFP and Linux nftables/eBPF in audit mode first.
- Show audit-mode policy delivery/status and match telemetry without blocking behavior.

### Later: Enterprise Control Plane

- Add production enforcement adapters for host firewall, WFP, eBPF, SASE, NAC, SGT, and cloud access controls.
- Add identity enrichment from directory, EDR, MDM, NAC, and network source-of-truth systems.
- Add autonomous change windows where AegisFlux can stage, watch, verify, and roll back under human-approved constraints.
- Add packaged installers, upgrade channels, fleet management, and tenant isolation.
- Add compliance reporting that proves why a control exists and when it last matched real behavior.

### Next Leap: AI-Native Security Operating Loop

After operational readiness, AegisFlux should add a governed deep-agent layer rather than isolated AI features. See [AegisFlux AI-Native Leap Architecture](AEGIS_AI_NATIVE_LEAP_ARCHITECTURE.md).

- Add an AI agent harness with typed tools, run audit, privacy controls, and approval gates.
- Require evidence-bound reasoning for agent conclusions: evidence refs, confidence, assumptions, missing evidence, and safety boundary.
- Add Endpoint Analyst as the first deep agent for device/finding investigation.
- Add research and detection-authoring agents that turn ecosystem/fleet signals into scored detection opportunities.
- Add candidate simulation before detection-pack approval.
- Add Control Design Copilot for observe-only proposals, blast radius, rollback, and approval packets.
- Add fleet AI capability drift radar for new, risky, widespread, stale, and low-confidence capability changes.
- Add governance memory and decision ledger so AI recommendations and operator decisions are durable.
- Add enforcement readiness scorecards that explain missing prerequisites without enabling blocking behavior.

## Demo Narrative

The strongest demo path is:

1. A Windows or Linux endpoint reports process, flow, DNS, and finding evidence.
2. The operator selects a finding.
3. AegisFlux shows the investigation path: process, command, remote service, DNS, and risk.
4. AegisFlux drafts an observe-only control from the evidence.
5. The operator can see the proposed scope, expected blast radius, and rollback plan.
6. AegisFlux makes clear that enforcement is not active until policy staging and approval exist.

This tells the market that AegisFlux is not another alert feed. It is a control design system grounded in endpoint truth.

## Risks

- Weak evidence linkage will make draft controls feel speculative.
- Enforcement too early could damage trust; observe-only and simulation need to come first.
- Agent reliability matters more than breadth. A small, reliable collector beats a large, fragile one.
- Product language must stay concrete. "AI security" is crowded; "AI agent bill of materials" and "evidence-backed controls" are clearer.
- Integrations can sprawl. Start with the local endpoint loop, then attach network and identity context.

## 30/60/90 Day Plan

### 30 Days

- Polish ABOM fleet insights and change detection.
- Refine evidence graph explainability.
- Add richer draft-control simulation and decision history.
- Add agent health/readiness scoring.
- Keep QA/performance harnesses green for all console product paths.

### 60 Days

- Mature adaptive detection workflow from research feed to signed pack rollout and retirement.
- Polish first-value demo scenarios and lab setup docs.
- Add signed policy bundle models.
- Add operator approval and decision history.
- Start one host-level audit-mode adapter.

### 90 Days

- Add rollback execution planning for the first audit-mode adapter.
- Add identity and network enrichment from a source-of-truth integration.
- Add packaged endpoint deployment.
- Add reporting for Agent Bill of Materials, risky AI activity, and proposed controls.
- Prepare a focused customer-style demo around evidence-backed AI agent governance.
