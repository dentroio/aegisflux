# Aegis Sensor Fusion Architecture

## Goal

Aegis should detect AI-capable behavior across many applications and transports, even when one collection source is blind. The platform cannot depend on a single signal such as DNS cache, process names, or known domains. It needs sensor fusion: multiple weak signals correlated into evidence-backed findings.

The product goal is ambitious but realistic:

- Detect known AI services.
- Detect unknown AI-agent behavior.
- Detect browser, CLI, IDE, extension, local model, MCP, proxy, gateway, and agent-to-agent activity.
- Detect across DNS, DoH, direct IP, TLS, browser state, local services, process behavior, and file/config evidence.
- Produce explainable evidence and observe-only draft controls before enforcement.

## Why Sensor Fusion Is Required

The ChatGPT-on-Windows lab test exposed the problem clearly:

- `msedge.exe` produced outbound 443 flows.
- Windows DNS cache did not show `chatgpt.com` or `openai.com`.
- Flows had IPs but no hostname.
- The detector already knew about ChatGPT/OpenAI domains, but the evidence never reached it.

This is normal in modern endpoints. Browsers may use DoH, connection reuse, HTTP/3, CDN fronting, cached state, service workers, or proxy paths. Aegis must assume any single collector can miss the activity.

## Collection Layers

### Layer 1: Process and Lineage

Collect:

- Process image name.
- Executable path.
- Parent process.
- Command line when enabled.
- User/session.
- Signer/hash later.

Detect:

- AI tools launched directly.
- IDE plugins spawning agent runtimes.
- Browser child processes involved in AI usage.
- Shell automation near model calls.

### Layer 2: Network Flows

Collect:

- Local/remote IP and port.
- Protocol.
- PID/process attribution.
- Direction.
- Connection state where available.

Detect:

- Browser or CLI reaching suspicious destinations.
- Local model runtimes.
- Model gateway usage.
- Long-lived agent communication.

Current Windows implementation uses `netstat -ano`. Future Windows implementation should use ETW/WFP/IP Helper for richer attribution.

### Layer 3: DNS and Name Resolution

Collect:

- DNS cache.
- DNS Client ETW events.
- Resolver.
- Query type.
- Answers.
- Process attribution when available.

Detect:

- Known AI domains.
- New model gateway domains.
- MCP registry/package endpoints.
- Suspicious domain drift.

DNS cache alone is not enough because DoH and browser cache can bypass it.

### Layer 4: Browser Evidence

Collect:

- Recent browser history domains.
- Browser extension inventory.
- Browser profile metadata.
- Enterprise policy settings.
- DoH configuration.

Detect:

- Browser AI sessions.
- ChatGPT/Claude/Gemini/Copilot web usage.
- Browser AI extensions.
- Web agent tools that never appear in OS DNS cache.

The first Windows implementation should read recent Edge/Chrome/Brave history from copied SQLite snapshots and emit low-confidence domain observations.

### Layer 5: TLS, SNI, Proxy, and Gateway Evidence

Collect where allowed:

- TLS SNI from ETW/proxy/firewall.
- HTTP CONNECT destination from proxy logs.
- Enterprise gateway labels.
- Model gateway request metadata.

Detect:

- AI usage hidden behind CDNs.
- Direct IP connections enriched by gateway logs.
- Approved vs unapproved model providers.

This may come from endpoint, proxy, SASE, firewall, or Clarion integrations rather than only the endpoint agent.

### Layer 6: Local AI Runtime and Tool Bridges

Collect:

- Local listeners.
- OpenAI-compatible local endpoints.
- Ollama/LM Studio/local model ports.
- MCP client/server config.
- Tool manifests.
- IDE extension configs.
- CLI agent config files.

Detect:

- Local AI runtimes.
- Tool-capable agents.
- File-system, shell, browser, and repo tool bridges.
- Agent-to-agent listeners.

### Layer 7: Behavior and Baseline

Correlate over time:

- New model destination for a process.
- New local tool bridge.
- New browser AI session for a user.
- New repo/file access near model calls.
- New AI activity after package or extension install.

This is where Aegis becomes more than a signature engine.

## Detection Result Model

A finding should include:

- What was observed.
- Which sensors contributed.
- Which evidence is strong vs weak.
- Which process/user/device was involved.
- Which domains/IPs/tools were involved.
- Whether this is new behavior.
- What observe-only control is proposed.
- What data is missing before enforcement can be considered.

## Confidence Model

Example confidence levels:

- 0.30: Process name or generic domain only.
- 0.45: Browser history domain without process attribution.
- 0.60: DNS domain plus browser process present.
- 0.75: DNS/domain plus process-attributed flow.
- 0.85: Process, flow, DNS/SNI, and AI tool/config evidence.
- 0.95: Signed detection pack plus multiple independent sensors and simulation match.

## Immediate Build Path

1. Add Windows browser-history domain collection.
2. Expand known AI destination indicators.
3. Add detection-pack schema so indicators can update without rebuilding agents.
4. Add DNS-to-flow correlation by recent answer IP.
5. Add browser extension inventory.
6. Add local AI runtime and MCP config discovery.
7. Add ETW DNS Client collection.
8. Add proxy/SNI enrichment through Clarion or gateway integrations.

## Clarion Role

Clarion should eventually enrich Aegis endpoint evidence with network and enterprise context:

- Proxy logs.
- SASE events.
- Firewall flows.
- DNS infrastructure logs.
- Identity and device posture.
- Approved model gateway policy.

Aegis owns endpoint truth. Clarion owns enterprise context and orchestration. Together they can detect activity that neither side can reliably prove alone.

