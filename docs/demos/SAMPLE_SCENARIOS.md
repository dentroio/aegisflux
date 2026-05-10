# Sample demo scenarios

Five short scenarios that exercise the AegisFlux first-value workflow end to end. Each scenario
runs against lab data only and remains observe-only.

The same scenarios are rendered in-product at `/demo/scenarios` so an operator can click directly
into the relevant routes.

---

## 1. Browser AI extension

A managed Chromium endpoint installs a popular AI assistant extension that exfiltrates clipboard
content to a third-party model.

**Steps**

1. Confirm the agent has reported recent extension inventory.
2. Open `Discover → Agent Bill of Materials` and filter by category `browser_ai_extension`.
3. Pick the highlighted extension and review its findings, last seen, and devices list.
4. Click `Design control` on a finding to draft an observe-only block of the extension origin.

**Expected observations**

- ABOM lists at least one `browser_ai_extension` item with multiple devices.
- A finding shows clipboard read paired with outbound DNS to an AI-model domain.
- A draft control records the finding id, scope, and observe-only flag.

**Routes**

- `/discover/abom`
- `/analyze/evidence`
- `/control/controls`

---

## 2. Coding agent (CLI)

A developer machine runs a coding agent that reads source code and uploads a project tree to an
external completion API.

**Steps**

1. Open `Discover → Activity` to see recent process executions on the developer host.
2. Pivot into `Analyze → Evidence path` for the coding-agent process.
3. Review the parent process, file reads, DNS lookups, and the model gateway destination.
4. Promote the suggested observe-only detection from the research feed if not already linked.

**Expected observations**

- Evidence path shows a clear chain from shell → coding agent → outbound API.
- Operator label and confidence reason explain why the chain is suspicious.
- Research feed item is linked to a detection candidate in the workflow board.

**Routes**

- `/discover/activity`
- `/analyze/evidence`
- `/analyze/research`

---

## 3. Local model runtime

A local model runtime (e.g., Ollama-style) listens on localhost and is reachable from any process
on the host.

**Steps**

1. Open `Discover → ABOM` and filter by `local_model_runtime`.
2. Inspect the listening sockets, devices, and confidence score.
3. Open `Finding-to-control` and draft an observe-only egress control around the runtime port.
4. Run the lab simulation to see projected matches before any rollout.

**Expected observations**

- ABOM item lists the runtime with at least one listening port and bind address.
- Draft control captures scope (port, process, host) and a rollback note.
- Latest simulation shows match count, top processes, and destinations.

**Routes**

- `/discover/abom`
- `/control/controls`

---

## 4. MCP endpoint exposure

A workstation exposes an MCP endpoint that allows an external agent to invoke arbitrary tools on
the host.

**Steps**

1. Open ABOM and filter by `mcp_endpoint`.
2. Open the device detail and review readiness and connectivity.
3. Open `Evidence path` to view recent invocations of the MCP endpoint.
4. Promote the related research item and review the candidate quality gate.

**Expected observations**

- ABOM lists the MCP endpoint with confidence and devices.
- Agent readiness explains heartbeat, event ingestion, and connectivity health.
- Detection candidate appears on the workflow board in the `simulated` stage.

**Routes**

- `/discover/abom`
- `/agents`
- `/analyze/research`

---

## 5. Suspicious automation finding

An RPA-style automation chains a browser AI extension and a CLI agent to read clipboard, upload,
and write files.

**Steps**

1. Open `Analyze → Findings` to locate the high-risk automation finding.
2. Pivot into the evidence path and review the missing-evidence callouts.
3. Open the AI research feed for related intelligence; promote if scoped.
4. Watch the candidate move from `new` → `simulated` → `reviewed` and check the gate.

**Expected observations**

- Finding ties to multiple ABOM items and a single evidence narrative.
- Research feed item shows a linked candidate id.
- Workflow board shows the candidate progressing with quality-gate status.

**Routes**

- `/analyze/findings`
- `/analyze/evidence`
- `/analyze/research`
