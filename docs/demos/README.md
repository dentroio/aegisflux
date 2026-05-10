# AegisFlux first-value demo

This folder contains everything needed to run a credible AegisFlux walkthrough for an operator or
buyer in roughly five minutes.

The demo never enables enforcement. Detections, controls, and packs surfaced through the demo
remain observe-only.

## Operator narrative

> AegisFlux helps operators see AI-capable tools on the endpoint, trust why a finding matters,
> design observe-only controls with rollback, and adapt detection as new AI tools appear — without
> rebuilding endpoint agents.

The four product pillars and the routes that prove them:

| Pillar      | Route                                  | What the operator sees                                                                                |
| ----------- | -------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| Discover    | `/discover/abom`                       | Agent Bill of Materials with categories, capabilities, devices, confidence, and fleet insights.       |
| Investigate | `/analyze/evidence`                    | Evidence path narrative connecting finding → process → flow → DNS → endpoint → control.               |
| Design      | `/control/controls`                    | Finding-to-control draft with scope, blast radius, simulation, and decision history.                  |
| Adapt       | `/analyze/research`                    | AI research feed with detection candidate workflow board and quality gates.                            |

The operator can also run the always-on `First-value tour` from `/demo` to step through these
in order.

## Files

- [`SAMPLE_SCENARIOS.md`](SAMPLE_SCENARIOS.md) — five concrete scenarios with steps and expected
  observations. The same content is rendered in-product at `/demo/scenarios`.
- [`CHECKLIST.md`](CHECKLIST.md) — pre-demo checklist (services, agents, expected routes, reset
  steps) for a repeatable lab walkthrough.

## Screenshot placeholders

The following screenshot placeholders are referenced from the docs above. Capture them once a lab
environment is stable and drop the PNGs into `docs/demos/screenshots/` with the suggested names.

| Placeholder file                  | Suggested capture                                                       |
| --------------------------------- | ----------------------------------------------------------------------- |
| `screenshots/abom-fleet.png`      | ABOM with fleet insights strip and a populated table.                   |
| `screenshots/evidence-narrative.png` | Evidence path with the operator narrative block.                     |
| `screenshots/draft-control.png`   | Finding-to-control with simulation card and decision history.           |
| `screenshots/agent-readiness.png` | Agent detail Health tab with readiness explanation.                     |
| `screenshots/detection-board.png` | Research feed with detection workflow board.                            |

The PNGs are intentionally not committed yet; this README is the canonical placeholder set.
