# WO-ADAPT-001: Detection Opportunity Research Agents

**Status:** Planned  
**Phase:** AI-Native Leap / Adapt  
**Primary owner:** AI Platform / Detection / Product  

## Goal

Add governed research agents that turn AI ecosystem signals and observed endpoint behavior into scored detection opportunities.

## Problem

AI tooling changes too quickly for static lists. AegisFlux needs a repeatable way to collect new AI tool/protocol/runtime signals, relate them to fleet evidence, and decide what deserves a detection candidate.

## Scope

- Research item model and scoring.
- Agent harness jobs for research ingestion and opportunity scoring.
- Inputs from existing research feed, ABOM/fleet evidence, and manually entered signals.
- No automatic pack promotion.

## Deliverables

- Define detection opportunity fields: source, signal type, evidence refs, novelty, fleet relevance, risk, confidence, expected false positives, recommended next action.
- Add Research Agent and Detection Researcher agent definitions.
- Add tools for listing ABOM/fleet signals and existing research items.
- Add opportunity scoring with clear rationale.
- Add UI/API surface to review opportunities and create candidate-draft jobs.

## Acceptance Criteria

- Research opportunities are evidence-linked and scored.
- Operators can distinguish public ecosystem signals from locally observed fleet signals.
- Opportunities can be promoted only to candidate draft, not directly to signed packs.
- All research-agent runs are audited.

## Dependencies

- WO-AGENTS-001.
- WO-AGENTS-002.
- WO-PROD-004.
- WO-GROWTH-005.

## Non-Goals

- No unsupervised external crawling without source controls.
- No automatic deployment of detections.
- No vendor-name-only detections without behavior/evidence context.

## Suggested Verification

- Tests for opportunity scoring and empty/no-evidence behavior.
- Manual creation of one opportunity from ABOM evidence and one from research feed input.
