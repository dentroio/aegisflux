# Demo / lab checklist

Use this checklist to bring a clean lab environment into a demo-ready state and to reset it
afterwards.

The demo is observe-only. Nothing in this checklist enables enforcement on agents.

## 1. Required services

The compose stack should expose at minimum the following backends:

- `actions-api` (default `http://localhost:8083`)
- `ingest` (default `http://localhost:9091`)
- `bpf-registry` (default `http://localhost:8090`)
- `orchestrator` (default `http://localhost:8084`)
- `decision-api` (default `http://localhost:8087`)
- `detection-pipeline` (default `http://localhost:8089`)
- `console` (Next.js dev server, default `http://localhost:3000`)

Verify each is healthy before starting the demo. The console health bar at the top of every page
will render `Lab` if running against the bundled lab settings.

## 2. Required agents

At least two endpoint agents should be reporting:

- One Linux/macOS host that produces process, DNS, and flow telemetry.
- One Windows or browser-equipped host that reports browser extensions and SASE/DNS events.

Agents should have heartbeat within the last few minutes; a stale agent will degrade readiness.

## 3. Expected routes

The full route order for a credible walkthrough:

1. `/` — landing dashboard with first-value tour banner.
2. `/demo` — five-minute first-value tour.
3. `/discover/abom` — Agent Bill of Materials with fleet insights.
4. `/agents` — fleet readiness strip and per-agent status.
5. `/analyze/evidence` — evidence path with operator narrative.
6. `/analyze/findings` — finding list to feed the design step.
7. `/control/controls` — finding-to-control draft with simulation and history.
8. `/analyze/research` — AI research feed and detection workflow board.
9. `/demo/scenarios` — sample scenarios reference.

## 4. Expected observations

For each scenario in [`SAMPLE_SCENARIOS.md`](SAMPLE_SCENARIOS.md), confirm:

- ABOM populates within ~1 minute of agent telemetry.
- Evidence path shows the operator narrative block (not just raw nodes).
- A draft control simulation card renders with non-zero matches in the lab.
- Research feed shows at least one item with a linked candidate id.
- Detection workflow board shows the candidate at the expected stage.

## 5. Reset / cleanup steps

To reset between demos without restarting the stack:

- Reset the first-value tour from `/demo` (`Reset progress` button) to clear localStorage state.
- Use the actions-api `DELETE /platform/draft-controls/{id}` for any draft control created during
  the walkthrough.
- Use `POST /platform/research-feed/{id}` with `{ "status": "new" }` to revert any promoted
  research item back to `new` (or rely on the lab seed reset).
- Use `POST /platform/detection-candidates/{id}/retire` with a note like `demo reset` for any
  candidate that should not appear in the next walkthrough.
- Restart the agents only if a host accumulates stale telemetry that overwhelms the panels.

## 6. Operator dos and don'ts

- Always lead with the narrative: discover → investigate → design → adapt.
- Always show the observe-only disclaimer in finding-to-control and detection candidate detail.
- Avoid claims of automatic enforcement; the demo intentionally does not modify endpoint state.
- If a panel renders an empty state, click `View sample scenarios` to redirect into the right
  scenario instead of explaining around the gap.
