# Aegis Dynamic AI Detection Strategy

## Answer

Yes, Aegis should be able to detect new AI agents and new ways of doing AI on devices, but not by relying on a static list of product names. The detection model must be dynamic, behavior-based, continuously updated, and evidence-backed.

The core idea: Aegis should run a research-to-detection loop. Aegis cloud-side research agents watch the AI ecosystem, convert discoveries into detection intelligence, simulate the detections against observed lab and customer telemetry, then publish signed detection packs to endpoint agents. Endpoint agents stay lightweight and safe: they collect evidence, evaluate approved rules, and report findings without enforcing unless a separate policy workflow approves it.

## Why Static Detection Will Fail

AI tooling changes too quickly for hardcoded detections to be enough. New tools appear as:

- Desktop apps.
- IDE plugins and coding agents.
- Browser extensions and web AI sessions.
- CLI tools and shell wrappers.
- Local model runtimes.
- MCP servers and clients.
- Agent-to-agent protocols.
- Automation frameworks that do not call themselves AI.
- Enterprise model gateways and proxy services.

The detection surface is not just "is ChatGPT installed?" It is "is this endpoint running a tool-using autonomous process that can read, write, browse, call tools, exfiltrate context, or delegate work?"

## Market Signals To Track

These are current examples that show why Aegis needs adaptive detection:

- Anthropic's Model Context Protocol standardizes how AI applications connect to tools and data sources. That creates useful enterprise patterns, but also a new endpoint detection surface: MCP clients, MCP servers, tool manifests, local transports, and tool execution. Source: https://docs.anthropic.com/en/docs/mcp
- The broader AI ecosystem is moving toward agent-to-agent communication. Google introduced Agent2Agent in April 2025, and the Linux Foundation announced the Agent2Agent project in June 2025. That means future endpoint activity may be initiated by another agent, not directly by a human. Sources: https://www.linuxfoundation.org/press/linux-foundation-launches-the-agent2agent-protocol-project-to-enable-secure-intelligent-communication-between-ai-agents and https://github.com/a2aproject/A2A
- OpenAI's Agents SDK describes agents using hosted tools, function tools, agents-as-tools, local MCP servers, web search, file search, code interpreter, and computer use. That means detection must look for tool invocation patterns and local bridges, not only model API domains. Source: https://openai.github.io/openai-agents-js/guides/tools

## Detection Philosophy

Aegis should treat AI as a behavior class, not a vendor list.

### Detect Capabilities

The first question is what capability appeared on the endpoint:

- Model access.
- Tool calling.
- Browser or GUI control.
- File search or repository search.
- Code execution.
- Local model inference.
- Remote model gateway access.
- Agent-to-agent delegation.
- MCP tool serving.
- Secret, token, or credential access.

### Detect Evidence Chains

Aegis should never emit a high-confidence finding from one weak signal. It should build evidence chains:

- Process path and signer.
- Command line and parent process.
- Open files, config files, and environment variables.
- DNS query and remote endpoint.
- TLS SNI or proxy metadata where available.
- Local ports and loopback services.
- Browser extension or IDE plugin identity.
- MCP server/client manifest.
- Tool names and permission scope.
- User identity and session context.

### Detect Drift

An endpoint may become risky because behavior changes:

- A known process starts calling a new model domain.
- A developer tool starts hosting a local MCP server.
- A browser extension begins using automation APIs.
- A script runner starts touching secrets and model gateways in the same session.
- A model client starts communicating with peer agents.

This is where Aegis can be stronger than static security tools: baseline what AI behavior is normal, then detect meaningful changes.

## Aegis Research Agents

Aegis should have cloud-side research agents with specific jobs.

### Ecosystem Research Agent

Continuously monitors public sources for new AI tools, protocols, packages, domains, and behaviors.

Inputs:

- Official vendor docs.
- GitHub releases and package registries.
- MCP server registries.
- A2A protocol/project updates.
- Browser extension stores.
- CLI package ecosystems.
- Security advisories and CVEs.
- Customer-submitted unknown findings.

Outputs:

- Candidate product names.
- Process names.
- Package names.
- File paths.
- Config markers.
- Network destinations.
- Protocol markers.
- Suspicious behavior hypotheses.

### Detection Authoring Agent

Converts research into detection candidates.

Outputs:

- YAML or JSON detection pack.
- Required evidence fields.
- Confidence model.
- Risk scoring guidance.
- False-positive notes.
- Test fixtures.
- Rollback or deprecation metadata.

### Simulation Agent

Tests candidate detections against historical Aegis telemetry before release.

Questions:

- How many endpoints would match?
- Which evidence fields are missing?
- Is the detection too broad?
- Does it match normal developer activity?
- Does it produce useful draft controls?

### Release Governance Agent

Prepares signed detection-pack releases but does not publish without policy gates.

Checks:

- Schema validity.
- Unit tests.
- Simulation result.
- Required evidence.
- Expiration date.
- Version and changelog.
- Approval record.

## Endpoint Agent Update Model

Endpoint agents should not need a rebuild for every new AI tool.

They should support signed detection packs:

```yaml
pack_id: aegis.ai.dynamic.2026-05
version: 1
mode: observe-only
expires_at: 2026-08-01T00:00:00Z
rules:
  - id: mcp.local-server.observed
    classification: ai_tool_bridge
    risk_score: 35
    required_evidence:
      - process
      - local_listener
      - config_marker
    match:
      command_line_any:
        - "mcp"
        - "model context protocol"
      file_path_any:
        - ".mcp.json"
        - "mcpServers"
      local_port_behavior: true
    finding:
      title: "Local MCP tool bridge observed"
      recommended_action: "review"
```

The endpoint agent evaluates the pack locally and emits normal Aegis events:

- `aegis.agent.detected`
- `aegis.risk_finding.created`
- Future: `aegis.ai_tool_inventory.observed`
- Future: `aegis.detection_pack.status`

## Dynamic Detection Layers

### Layer 1: Known Indicators

These are fast and useful, but never enough alone:

- Process names.
- Package names.
- App bundle IDs.
- Browser extension IDs.
- Common file paths.
- Known model domains.
- Known MCP server names.

### Layer 2: Behavioral Indicators

These generalize better:

- Tool execution after model interaction.
- Code execution plus model gateway access.
- Local service exposing tool manifests.
- Browser automation plus external AI service.
- IDE plugin spawning shell commands.
- Script reading files then calling a model API.
- Repeated prompt-sized outbound payloads.

### Layer 3: Protocol Indicators

Aegis should understand emerging AI protocols:

- MCP client/server configuration.
- Tool schema manifests.
- Agent-to-agent messages.
- Local OpenAI-compatible endpoints.
- Model gateway proxy traffic.
- Local vector stores and retrieval indexes.

### Layer 4: Graph and Baseline Indicators

Correlate over time:

- New AI capability appeared on this device.
- New process-to-model path.
- New tool bridge connected to sensitive directories.
- New agent communication outside expected network zones.
- Same AI tool appears across multiple devices after a package update.

## Product Differentiator

Aegis can be different by producing an Agent Bill of Materials, continuously:

- Which AI-capable tools exist on this endpoint?
- Which tools can call other tools?
- Which tools can access files, browsers, shells, secrets, or repositories?
- Which model providers or gateways do they reach?
- Which user or automation context launched them?
- Which controls are only observing, staged, or enforced?

Most products will say, "We saw an app." Aegis should say, "This endpoint has an AI agent capability with these tools, this network path, this identity context, this evidence, and this safe control proposal."

## Safety Rules

Dynamic does not mean reckless.

- Research agents do not directly instruct endpoint agents.
- All detection packs are schema-validated.
- All packs are signed.
- Packs default to observe-only.
- High-risk detections require multiple evidence categories.
- Expiring rules are required for experimental intelligence.
- Customers can pin, disable, or stage packs by cohort.
- Endpoint agents report pack version and health.

## Next Build Slice

1. Define a detection-pack schema for Windows and Linux agents.
2. Move current hardcoded AI markers into a default local pack.
3. Add `aegis.detection_pack.status` events so the console knows which pack version each device is running.
4. Add a backend endpoint to publish available detection packs.
5. Add console visibility for Agent Bill of Materials.
6. Add a lab-only research pack for MCP/local model/tool-bridge behavior.

## Detection Pack Fields

Minimum fields:

- `pack_id`
- `version`
- `mode`
- `expires_at`
- `rules`
- `required_evidence`
- `match`
- `classification`
- `risk_score`
- `finding`
- `false_positive_notes`
- `references`

Future fields:

- `simulation_result`
- `approval`
- `rollout_cohort`
- `rollback`
- `customer_overrides`
- `deprecation_reason`

## Near-Term Research Topics

- MCP server and client discovery.
- Local OpenAI-compatible model endpoints.
- Browser AI usage and extensions.
- IDE coding agents.
- CLI agent frameworks.
- Agent-to-agent communication.
- Model gateway and proxy products.
- Secret access near model calls.
- Tool execution after AI interaction.

## Positioning

Aegis should not market this as another static AI app inventory. The stronger message is:

> Aegis continuously discovers AI agent capabilities on endpoints, proves them with evidence, and turns them into safe observe-only controls before enforcement is considered.

