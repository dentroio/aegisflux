# AegisFlux Development and Clarion Integration Plan

**Status:** Draft  
**Owner:** AegisFlux / Clarion technical planning  
**Last updated:** April 24, 2026

## 1. Purpose

AegisFlux is the host and workload visibility and enforcement layer for Clarion. Clarion is the broader context-aware policy intelligence platform that owns enterprise context, network identity, policy intent, risk decisions, and integrations with systems such as ISE, TrustSec, firewalls, proxies, gateways, SIEM/SOAR, MDM, and EDR.

This plan defines how to mature AegisFlux as a standalone host/workload platform while creating a clean path for Clarion integration.

## 2. Product Boundary

### Clarion Owns

- Enterprise policy intent and policy authoring
- Context graph across users, devices, sessions, SGTs, destinations, applications, and risk
- Network/session identity from ISE, pxGrid, NetFlow/IPFIX, Zeek, firewall logs, DNS, DHCP, AD, and endpoint inventories
- Risk scoring and policy decisions across network, endpoint, gateway, and application layers
- Orchestration into ISE, TrustSec/SGT, SGACL, firewalls, proxies, model gateways, API gateways, SIEM/SOAR, MDM, and EDR
- User-facing policy, investigation, and operations workflows

### AegisFlux Owns

- Host and workload agent lifecycle
- Endpoint identity and agent identity
- Process inventory and process lineage
- Process-to-flow attribution
- DNS and destination observations from the host perspective
- Container, cgroup, namespace, and workload context
- AI-agent and automation behavior signals
- Local telemetry collection and signed reporting
- Local policy cache
- Host/workload enforcement through eBPF, cgroup hooks, tc, nftables, WFP, Windows Firewall, local proxy, and future macOS controls
- Enforcement audit events and rollback

### Shared Contract

AegisFlux and Clarion should integrate through explicit events and APIs, not shared implicit database writes. AegisFlux emits host/workload evidence. Clarion evaluates policy and risk. AegisFlux applies local enforcement when Clarion chooses endpoint/workload enforcement.

## 3. Architecture Tracks

### Track A: AegisFlux Core Agent

Goal: make the AegisFlux agent reliable, identifiable, secure, and deployable.

Deliverables:

- Stable agent identity and UID persistence
- Secure registration and authentication
- Signed telemetry
- Policy receipt and local policy cache
- Agent heartbeat, health, version, and capability reporting
- Agent upgrade and rollback model
- Platform capability discovery for Linux, Windows, macOS future

Exit criteria:

- Agent can register, survive restart, reconnect, receive policy, report health, and prove identity.
- Backend can distinguish host identity, agent identity, runtime capabilities, and policy compatibility.

### Track B: Host and Workload Visibility

Goal: provide Clarion-grade context from the endpoint and workload layer.

Deliverables:

- Process start/stop telemetry
- Parent/child process lineage
- Process executable path, hash, signer, command line, user/session
- Network connection telemetry with PID/process attribution
- DNS observation and flow-to-domain correlation
- Container/cgroup/network namespace metadata
- Local application classification: browser, shell, developer tool, database client, automation tool, AI assistant, local model runtime
- Evidence bundles for investigations

Exit criteria:

- AegisFlux can answer: which user, process, application, workload, or suspected agent generated this flow?

### Track C: AI Agent and Automation Detection

Goal: detect AI-agent-like and automation behavior without assuming every detection is binary or malicious.

Signal families:

- Known AI application or agent parent process: Cursor, VS Code extensions, browser automation, local agent runners
- Known frameworks and runtimes: LangChain, AutoGen, CrewAI, Semantic Kernel, MCP clients/servers, Playwright, Selenium
- LLM/API/model destinations: hosted model APIs, private model gateway, local LLM servers
- Tool behavior: repo read, file system traversal, shell execution, browser automation, API calls, git operations
- Process chains: app -> node/python -> tool/runtime -> network destination
- Behavioral loops: repeated read/plan/tool-call/network patterns
- Local model runtime behavior: Ollama, LM Studio, llama.cpp, vector database access
- Risk modifiers: sensitive repo access, credential access, production destination, unknown cloud upload, encoded shell command, admin privileges

Outputs:

- `agent_likelihood`
- `detected_patterns`
- `confidence`
- `risk_score`
- `recommended_enforcement`
- supporting evidence

Exit criteria:

- AegisFlux can classify likely AI-agent activity with confidence and evidence, while avoiding a brittle allow/block-only model.

### Track D: Local Enforcement

Goal: apply precise host/workload controls with low blast radius.

Deliverables:

- Linux eBPF/cgroup/tc/nftables policy enforcement
- Windows WFP/Windows Firewall policy enforcement
- Per-process destination block
- Per-process redirect to proxy/model gateway
- Per-workload/container policy
- Enforcement audit log
- Policy TTL and rollback
- Fail-safe and fail-open/fail-closed modes by policy class

Exit criteria:

- AegisFlux can block or redirect the risky process without quarantining the whole endpoint unless policy requires escalation.

### Track E: Backend and Policy Pipeline

Goal: make AegisFlux backend services production-shaped and Clarion-compatible.

Deliverables:

- Ingest API for host telemetry
- WebSocket or message bus protocol for agent communication
- Policy decision ingestion from Clarion
- Policy compiler for endpoint/workload enforcement targets
- BPF artifact registry and signing
- Orchestrator for deployment, rollback, and status
- Event schema versioning
- Audit trail for policy decisions and enforcement actions

Exit criteria:

- Clarion can send a policy decision to AegisFlux, AegisFlux can compile/apply it, and Clarion can observe enforcement status.

### Track F: Clarion Integration

Goal: connect AegisFlux host/workload context into Clarion without collapsing project boundaries.

Deliverables:

- AegisFlux -> Clarion telemetry events
- Clarion -> AegisFlux policy decision API
- Shared identity model: device, user, session, process, application, agent, flow, destination, policy decision, enforcement action
- Clarion context graph ingestion of AegisFlux evidence
- Clarion UI workflow for endpoint/process/agent evidence
- Clarion policy workflow that can select endpoint enforcement through AegisFlux
- ISE/SGT coordination based on AegisFlux risk posture

Exit criteria:

- Clarion can reason over AegisFlux endpoint context and choose local enforcement, gateway enforcement, network action, or escalation.

## 4. Test Machine and Lab Strategy

Testing must include controlled machines that intentionally run normal applications, AI agents, automation tools, developer tools, downloaded software, and customer-like workloads. The lab should validate both detection and enforcement.

### Lab Tiers

| Tier | Purpose | Example Environment |
|------|---------|---------------------|
| Tier 0 | Unit and schema tests | Local CI, mocked events |
| Tier 1 | Single-host functional tests | One Linux VM and one Windows VM |
| Tier 2 | Multi-host workflow tests | Developer workstation, server, database, model gateway |
| Tier 3 | Network integration tests | ISE/TrustSec simulator or real lab, firewall/proxy hooks |
| Tier 4 | Customer-pattern validation | Repeatable lab packs for finance, engineering, admin, browser-heavy users |

### Required Test Machines

| Machine | OS | Purpose |
|---------|----|---------|
| `linux-dev-agent-01` | Ubuntu | Linux agent, eBPF enforcement, developer tooling, containers |
| `linux-server-01` | Ubuntu/RHEL | Server/workload telemetry and east-west flows |
| `windows-dev-agent-01` | Windows 11 | Browser, Cursor/VS Code, PowerShell, WFP/ETW testing |
| `windows-business-01` | Windows 11 | Browser/SaaS/downloaded software behavior |
| `macos-agent-01` | macOS | Future sensor research and parity testing |
| `prod-db-sim-01` | Linux/Postgres | Sensitive destination for policy tests |
| `model-gateway-01` | Linux | Approved LLM/model gateway target |
| `unknown-cloud-sim-01` | Linux | Simulated unapproved upload/API destination |
| `ise-lab-01` | ISE or simulator | SGT/CoA/session context integration |
| `firewall-proxy-lab-01` | Firewall/proxy simulator | Gateway and policy integration testing |

### AI Agent Detection Test Packs

Each test pack should define setup, expected telemetry, expected classification, expected policy decision, and expected enforcement result.

| Pack | Scenario | Expected Result |
|------|----------|-----------------|
| Browser AI | Browser-based AI assistant accesses SaaS/model API | Classified as browser/app AI use, not autonomous host agent |
| IDE AI Assistant | Cursor or VS Code launches node/python helper and calls model API | Process chain maps app -> helper -> model destination |
| Local Agent Script | Python AutoGen/LangChain/CrewAI script uses tools and network | High `agent_likelihood` with framework/tool evidence |
| MCP Server | Node MCP filesystem server reads repo and serves an agent | MCP process and file access context detected |
| Browser Automation | Playwright/Selenium controls browser and logs into SaaS | Automation behavior detected with browser child process context |
| Local LLM | Ollama/LM Studio/lama.cpp invoked with file/repo access | Local model runtime classified, external network not required |
| Downloaded Tool | Unknown downloaded binary makes unusual network connections | Unknown software risk, signer/hash evidence captured |
| Customer App | Line-of-business app reaches expected services | Classified as normal application baseline |
| Database Client | DB client from developer machine reaches dev/prod DB | Dev allowed, prod blocked or escalated by policy |
| Shell Automation | PowerShell/bash encoded or scripted network behavior | Suspicious automation detected and optionally blocked |

### Enforcement Validation Cases

- Block only `python agent_runner.py` from `prod-db-sim-01`; keep browser traffic working.
- Redirect model API calls to `model-gateway-01`.
- Deny unknown cloud upload from browser when user/data sensitivity requires it.
- Move endpoint to `AI_Active_Device` or `AI_Restricted_Device` only when policy/risk threshold is met.
- Trigger full quarantine only after tamper, confirmed compromise, or repeated high-severity behavior.

## 5. Event and API Contract Plan

### AegisFlux -> Clarion Events

- `aegis.agent.registered`
- `aegis.agent.heartbeat`
- `aegis.process.started`
- `aegis.process.ended`
- `aegis.flow.started`
- `aegis.flow.ended`
- `aegis.dns.observed`
- `aegis.agent.detected`
- `aegis.enforcement.applied`
- `aegis.enforcement.failed`
- `aegis.endpoint.risk.changed`
- `aegis.sensor.tamper_detected`

### Clarion -> AegisFlux Commands

- `clarion.policy.publish`
- `clarion.enforcement.apply`
- `clarion.enforcement.remove`
- `clarion.agent.command`
- `clarion.sensor.config.update`
- `clarion.evidence.request`

### Minimum Shared Object Model

- `device`
- `user`
- `network_session`
- `process`
- `application`
- `agent`
- `flow`
- `destination`
- `risk_finding`
- `policy_decision`
- `enforcement_action`

## 6. Development Phases

### Phase 1: AegisFlux Stabilization and Visibility

Work orders:

- [Visibility and Observability Work Orders](../work_orders/visibility_observability/README.md)

Build:

- Agent identity and registration hardening
- Signed telemetry envelope
- Process inventory and process lineage
- Process-to-flow mapping on Linux and Windows
- Basic app classification
- Lab Tier 1 machines
- Initial event schemas

Exit:

- AegisFlux can reliably report host/process/flow context from Linux and Windows into its backend.

### Phase 2: AI-Agent Detection and Evidence

Build:

- AI-agent signal extraction
- Detection confidence model
- Evidence bundles
- Browser, IDE, local agent, MCP, downloaded software, and shell automation test packs
- Detection dashboard in AegisFlux UI

Exit:

- AegisFlux can detect representative AI-agent and automation scenarios with explainable evidence.

### Phase 3: Local Enforcement

Build:

- Linux local enforcement
- Windows local enforcement
- Policy compiler for local controls
- Local policy cache, TTL, rollback
- Enforcement audit events
- Lab Tier 2 validation

Exit:

- AegisFlux can block/redirect specific process/workload flows with rollback and auditability.

### Phase 4: Clarion Context Integration

Build:

- AegisFlux event ingestion into Clarion
- Clarion context graph mapping
- Shared object IDs and correlation rules
- Clarion investigation view for host/process/agent evidence
- Clarion policy decision export to AegisFlux

Exit:

- Clarion can use AegisFlux context in policy decisions and display AegisFlux evidence in investigations.

### Phase 5: Clarion Orchestration Integration

Build:

- Clarion decisions selecting endpoint, gateway, firewall, or SGT actions
- ISE/SGT posture updates based on AegisFlux risk
- Gateway/model-gateway policy coordination
- Multi-control enforcement audit trail
- Lab Tier 3 integration

Exit:

- Clarion can coordinate AegisFlux local enforcement with network and gateway controls.

### Phase 6: Customer Pattern Validation

Build:

- Repeatable customer lab packs
- Baseline behavior models
- False-positive and false-negative review loop
- Performance, scale, and operational tests
- Deployment playbooks

Exit:

- AegisFlux and Clarion can be validated against customer-like endpoint, workload, and application behavior.

## 7. Immediate Next Steps

1. Convert the Clarion endpoint architecture PDF into a maintained Markdown source file.
2. Create an AegisFlux-to-Clarion architecture mapping document.
3. Define v1 event schemas for process, flow, DNS, agent detection, and enforcement status.
4. Stand up Tier 1 lab machines: one Linux VM and one Windows 11 VM.
5. Implement the first three detection packs: IDE AI assistant, local agent script, and browser AI.
6. Add a test evidence format so every detection result includes why it fired.
7. Decide whether AegisFlux remains a named subsystem or becomes branded as Clarion Host Enforcement.

## 8. Open Questions

- Should AegisFlux keep its own UI long-term, or become an operator/developer console while Clarion owns the primary customer UI?
- Which platform gets first-class enforcement priority: Linux eBPF or Windows WFP?
- What is the minimum event model Clarion needs before context graph integration starts?
- Should AI-agent detection run primarily on the endpoint, in the backend, or as a split model?
- How much local file-system evidence can be collected without creating privacy or compliance issues?
- What customer environments should define the first validation packs?
- What is the supported mode when endpoint enforcement fails: monitor, gateway-only enforcement, or network escalation?
