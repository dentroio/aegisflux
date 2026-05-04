# Aegis Product Roadmap

## Product Thesis

Aegis should become the evidence-to-control platform for the AI-era endpoint. The market already has tools that detect, alert, block, and aggregate. Aegis can be different by making every control recommendation traceable to live evidence: process, command line, network flow, DNS, identity, policy, blast radius, and rollback plan.

The practical wedge is simple: observe first, explain clearly, draft safely, enforce only when the operator trusts the evidence.

The larger platform vision is captured in [AEGIS_PLATFORM_VISION.md](AEGIS_PLATFORM_VISION.md): Aegis should mature as a first-of-kind standalone innovation platform, then integrate into Clarion through explicit events and APIs as Clarion's host/workload evidence and local-control subsystem.

Clarion's AI-agent patterns should also carry into Aegis. Aegis should support governed AI providers, registered AI platform agents, privacy controls, audit records, endpoint-specific AI assistance, and research-to-detection workflows. See [AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md](AEGIS_AI_AGENT_PLATFORM_ARCHITECTURE.md).

## Current Lab State

The lab now has a working visibility foundation:

- Windows endpoint collection with process, network, DNS, and finding events.
- Linux endpoint collection with the same ingest contract.
- Mac-hosted ingest, storage, and console access through lab tunnels.
- Device discovery across reporting endpoints.
- Investigation drill-in from a process or finding to linked processes, flows, DNS records, and detections.
- Observe-only UI that can show evidence without pretending to enforce policy yet.

This is enough to demonstrate the core loop: endpoint evidence arrives, Aegis connects it into a path, then Aegis starts drafting the control that would reduce risk.

## Differentiation

### Versus EDR

EDR is optimized for malware detection and response. Aegis should focus on understandable control design: what should this endpoint be allowed to do, why, and what happens if we change it?

### Versus SIEM

SIEM centralizes logs and correlation. Aegis should centralize evidence that can become action. The output is not only an alert; it is a candidate policy with supporting proof and rollback.

### Versus ZTNA, SASE, and NAC

These platforms often enforce network access but lack deep endpoint process context. Aegis can bridge endpoint behavior to enforcement surfaces: host firewall, eBPF, WFP, SASE policy, NAC identity groups, and segmentation tags.

### Versus CNAPP and CSPM

Cloud posture tools see cloud configuration well. Aegis should see the human and machine activity that reaches those services from real endpoints, especially AI tooling, automation, scripts, browsers, and local agents.

## Core Concepts

### Agent Bill of Materials

Aegis should maintain an Agent Bill of Materials for every endpoint:

- AI desktop apps and browser AI sessions.
- CLI agents, shell automation, scheduled tasks, and script runners.
- Model gateways, API endpoints, and local model runtimes.
- The processes, users, DNS names, and remote services each tool touches.
- The policies currently observing or controlling that activity.

This gives customers an inventory they do not reliably get today: which AI-capable agents exist, what they can reach, and what evidence supports that conclusion.

### Dynamic AI Detection

AI tooling changes too quickly for static endpoint signatures to carry the product. Aegis should support a research-to-detection loop where cloud-side research agents monitor new AI tools, protocols, model gateways, MCP servers, browser agents, coding agents, and agent-to-agent ecosystems, then generate signed observe-only detection packs for endpoint agents.

See [AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md](AEGIS_DYNAMIC_AI_DETECTION_STRATEGY.md) for the detailed architecture.

The AI platform itself should be managed like a first-class enterprise connector, following the Clarion model: provider configuration, registered agents, health status, privacy redaction, and auditable runs.

### Finding to Draft Control

Every finding should be able to produce a draft control:

- Proposed action: observe, restrict, deny, quarantine, segment, require approval, or monitor.
- Scope: process path, signer, command marker, user, host, remote IP, DNS name, port, protocol, or identity group.
- Evidence: the exact process, flow, DNS, and finding records used.
- Blast radius: what else would be affected if the control became active.
- Rollback: how Aegis would revert the change and what success telemetry should look like.

The first implementation should stay observe-only. The UI can show the draft, but backend policy staging and enforcement adapters should come later.

## Roadmap

### Now: Visibility Foundation

- Keep Windows and Linux reporting reliably into ingest.
- Normalize endpoint events into stable schemas.
- Improve device health: event freshness, collector version, last successful run, and tunnel status.
- Add better sample detections for AI agents, suspicious automation, model gateways, and unusual outbound flows.
- Keep the console focused on evidence and investigation rather than broad dashboards.

### Near: Evidence Graph and Draft Controls

- Persist process-to-flow-to-DNS-to-finding relationships as first-class investigation records.
- Add draft controls generated from findings and investigations.
- Add control simulation: show matching historical events before any policy is staged.
- Add operator notes and decision history.
- Add signed policy bundle models without enabling enforcement by default.

### Mid: Safe Change Management

- Add policy staging, approval gates, and rollback plans.
- Add device health scoring and deployment readiness.
- Add signed agent registration and mTLS transport.
- Add policy diff views: what changed, why, who approved it, and which evidence justified it.
- Add lab enforcement adapters for Windows Firewall/WFP and Linux nftables/eBPF in observe or audit mode first.

### Later: Enterprise Control Plane

- Add production enforcement adapters for host firewall, WFP, eBPF, SASE, NAC, SGT, and cloud access controls.
- Add identity enrichment from directory, EDR, MDM, NAC, and network source-of-truth systems.
- Add autonomous change windows where Aegis can stage, watch, verify, and roll back under human-approved constraints.
- Add packaged installers, upgrade channels, fleet management, and tenant isolation.
- Add compliance reporting that proves why a control exists and when it last matched real behavior.

## Demo Narrative

The strongest demo path is:

1. A Windows or Linux endpoint reports process, flow, DNS, and finding evidence.
2. The operator selects a finding.
3. Aegis shows the investigation path: process, command, remote service, DNS, and risk.
4. Aegis drafts an observe-only control from the evidence.
5. The operator can see the proposed scope, expected blast radius, and rollback plan.
6. Aegis makes clear that enforcement is not active until policy staging and approval exist.

This tells the market that Aegis is not another alert feed. It is a control design system grounded in endpoint truth.

## Risks

- Weak evidence linkage will make draft controls feel speculative.
- Enforcement too early could damage trust; observe-only and simulation need to come first.
- Agent reliability matters more than breadth. A small, reliable collector beats a large, fragile one.
- Product language must stay concrete. "AI security" is crowded; "AI agent bill of materials" and "evidence-backed controls" are clearer.
- Integrations can sprawl. Start with the local endpoint loop, then attach network and identity context.

## 30/60/90 Day Plan

### 30 Days

- Harden lab deployment for Windows, Linux, and Mac-hosted ingest.
- Add device health and collector freshness to the console.
- Expand AI-agent detection patterns.
- Add observe-only draft control panels in the UI.
- Document lab demos and repeatable setup.

### 60 Days

- Store investigation relationships and draft controls in the backend.
- Add policy simulation against historical events.
- Add signed policy bundle models.
- Add operator approval and decision history.
- Start one host-level enforcement adapter in audit mode.

### 90 Days

- Add rollback execution for the first enforcement adapter.
- Add identity and network enrichment from a source-of-truth integration.
- Add packaged endpoint deployment.
- Add reporting for Agent Bill of Materials, risky AI activity, and proposed controls.
- Prepare a focused customer-style demo around evidence-backed AI agent governance.
