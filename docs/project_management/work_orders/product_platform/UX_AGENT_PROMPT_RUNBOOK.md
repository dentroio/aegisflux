# AegisFlux UX Simplification Agent Prompt Runbook

**Purpose:** Copy-ready prompts for the UX simplification work orders. Each prompt includes commit and push instructions.

**Execution order:**

1. WO-UX-001: Clarion-Style Dashboard Simplification
2. WO-UX-002: Interaction and String Formatting System
3. WO-UX-003: Agents Workbench Simplification
4. WO-UX-004: Inventory Workbench Simplification
5. WO-UX-005: Detections, Control, and Operate UX Simplification

## Common Rules

- Start with `git status -sb`.
- Read `docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md`.
- Read the assigned work order.
- Review the Clarion references named in the work order.
- Do not create permanent right-side long-scroll columns when an item is clicked.
- Keep the design simple and operational.
- Format long strings intentionally.
- Run `npm run build` in `ui/console`.
- Update the work order status and implementation notes.
- Stage only files touched for the work order.
- Commit and push at the end.

## WO-UX-001 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-UX-001_CLARION_STYLE_DASHBOARD_SIMPLIFICATION.md

Goal: redesign the AegisFlux dashboard as a Clarion-style scan surface. Start from Clarion's Dashboard.tsx, DashboardWidget.tsx, useWidgetConfig.ts, WidgetEmptyState.tsx, and LazyWidgetMount.tsx patterns, but adapt them to AegisFlux.

Hard UX rules:
- Dashboard is a scan surface, not an investigation page.
- Clicking a metric, row, card, or widget must not create another permanent right-side column.
- Use deliberate navigation, a bounded modal, or a short inline expansion for detail.
- Long strings must not stretch cards, tables, or page width.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Read docs/AEGIS_CLARION_UI_PATTERNS_TO_ADOPT.md.
- Inspect ui/console/app/page.tsx and ui/console/components/shell/ConsoleShell.tsx.
- Inspect Clarion dashboard references listed in the work order.

Implementation scope:
- Rework / into bands: Readiness, Attention, Insights.
- Keep left nav visible.
- Keep dashboard content compact and bounded.
- Extract dashboard components if it reduces ui/console/app/page.tsx complexity.
- Preserve useful widget visibility behavior if possible.
- Route deeper investigation to Agents, Inventory, Detections, Controls, or Operate.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Visually check / at desktop and mobile widths if the dev server is available.
- Confirm clicks do not add a permanent right-side long-scroll column.

At the end:
- Run git status -sb.
- Stage only files changed for WO-UX-001.
- Commit with: git commit -m "Simplify dashboard UX"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-UX-002 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-UX-002_INTERACTION_AND_STRING_FORMATTING_SYSTEM.md

Goal: add shared interaction and formatting primitives so AegisFlux pages stay simple and long endpoint/security strings render cleanly.

Hard UX rules:
- Compact views truncate or summarize noisy data.
- Detail views reveal full values deliberately.
- No permanent right-side long-scroll columns by default.
- Tables must not stretch the page because of one long value.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Inspect ui/console/app/globals.css and existing console components.
- Inspect any dashboard components created by WO-UX-001 if present.

Implementation scope:
- Add reusable UI primitives for summary strips, bounded tables, formatted values, copy buttons, empty states, and temporary detail surfaces.
- Add helpers for hostnames, agent ids, IPs, MACs, hashes, paths, command lines, JSON/raw payloads, and dates.
- Adopt the primitives lightly where they naturally fit without redesigning every page.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Confirm long strings truncate/wrap appropriately in compact and detail contexts.

At the end:
- Run git status -sb.
- Stage only files changed for WO-UX-002.
- Commit with: git commit -m "Add UI formatting primitives"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-UX-003 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-UX-003_AGENTS_WORKBENCH_SIMPLIFICATION.md

Goal: simplify Agents into a focused workbench that answers which agents need attention without creating a long selected-agent side column.

Hard UX rules:
- Selecting an agent must not create a permanent right-side column with a long scroll.
- Open deliberate detail route /agents/[device_id], or use a bounded modal/temporary drawer with only critical summary.
- Keep list rows compact.
- Format long ids, pack ids, hostnames, and network values safely.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Read the work order.
- Inspect ui/console/components/AgentsManagementPanel.tsx and ui/console/app/agents/[device_id]/page.tsx.
- Inspect Clarion Devices.tsx and EndpointDetail.tsx references.

Implementation scope:
- Rework Agents around summary, filters, and bounded primary table/list.
- Move secondary detail to /agents/[device_id] or a bounded temporary detail surface.
- Compact detection-pack rollout.
- Use shared formatting primitives if available.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Check Linux and Windows lab agents render cleanly.
- Confirm selecting an agent does not create a permanent long-scroll right column.

At the end:
- Run git status -sb.
- Stage only files changed for WO-UX-003.
- Commit with: git commit -m "Simplify agents workbench UX"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-UX-004 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-UX-004_INVENTORY_WORKBENCH_SIMPLIFICATION.md

Goal: make Inventory a searchable, category-driven workbench instead of a long inventory report.

Hard UX rules:
- Clicking an inventory item must not create a permanent right-side long-scroll column.
- Use bounded modal, temporary drawer, inline expansion, or deliberate route for detail.
- Long extension ids, paths, command lines, vendors, and hostnames must render cleanly.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Read the work order.
- Inspect ui/console/components/InventoryPanel.tsx and ui/console/app/inventory/page.tsx.

Implementation scope:
- Add category summary, tabs/segmented controls, search, and bounded table/list.
- Group inventory by browser extensions, AI IDE/CLI tools, local model runtimes, SASE/SSE controls, and unknown signals.
- Route or reveal details deliberately.
- Use shared formatting primitives if available.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Confirm category filters and selected-item details do not create long page scrolls.

At the end:
- Run git status -sb.
- Stage only files changed for WO-UX-004.
- Commit with: git commit -m "Simplify inventory workbench UX"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```

## WO-UX-005 Prompt

```text
You are working in /Users/sgerhart/workspace/github/sgerhart/aegisflux.

Work order: docs/project_management/work_orders/product_platform/WO-UX-005_DETECTIONS_CONTROL_OPERATE_SIMPLIFICATION.md

Goal: simplify Detection Packs, Draft Controls, and Operational Event Feed so each has one clear primary task and bounded details.

Hard UX rules:
- Clicking candidates, packs, drafts, or events must not create a permanent long-scroll right column.
- Raw payloads should be hidden behind an explicit Raw view.
- Long ids, pack versions, hashes, event names, and payload values must format cleanly.

Before editing:
- Run git status -sb.
- Read docs/AEGIS_UI_SIMPLIFICATION_GUIDE.md.
- Read the work order.
- Inspect ui/console/app/detections/page.tsx, ui/console/app/control/controls/page.tsx, and ui/console/app/operate/events/page.tsx.

Implementation scope:
- Split detection candidates, signed packs, and rollout into tabs/segmented views.
- Make Draft Controls a queue plus bounded create/simulate detail.
- Make Operational Events a filterable timeline/table with bounded event detail.
- Use shared formatting primitives if available.
- Update the work order status and implementation notes.

Verification:
- Run npm run build in ui/console.
- Confirm selected items do not create permanent long-scroll columns.

At the end:
- Run git status -sb.
- Stage only files changed for WO-UX-005.
- Commit with: git commit -m "Simplify operational UX pages"
- Push with: git push
- Final response must include changed files, verification commands, commit hash, and push result.
```
