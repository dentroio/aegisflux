# WO-VIS-006: Visibility API and UI Surface

**Status:** Initial investigation and findings query APIs complete  
**Phase:** Visibility and Observability  
**Primary owner:** Backend / UI  

## Goal

Expose the first useful investigation workflow for Aegis visibility data.

## Scope

API and UI/API surface only. This work order assumes events are already ingested and stored.

## Investigation Views

Minimum API or UI should answer:

- What processes ran on this device?
- What process created this outbound connection?
- What parent process launched this process?
- What user/session owned this process?
- What DNS/domain context is associated with this flow?
- Was this process classified as browser, IDE helper, script, local agent, automation, or unknown?
- What evidence supports the classification?

## Deliverables

- Device visibility endpoint - initial raw event query available via `GET /v1/visibility/events`
- Process detail endpoint - initial process event query available via `GET /v1/visibility/processes`
- Flow detail endpoint - initial flow event query available via `GET /v1/visibility/flows`
- DNS observation endpoint - initial DNS event query available via `GET /v1/visibility/dns`
- Detection/risk finding endpoint - initial findings query available via `GET /v1/visibility/findings`
- Timeline query by device and time range
- Minimal API output for investigation path - initial combined path available via `GET /v1/visibility/investigation`
- Example investigation for `cursor.exe -> python.exe -> model/API destination`

## Acceptance Criteria

- User can query a Windows device and see recent process start/end events. **Initial API complete.**
- User can select a process and see parent, command line, user, and outbound flows.
- User can query flow events and see process attribution plus remote destination fields. **Initial API complete.**
- User can query DNS observations and see domain, answers, resolver, and process correlation evidence. **Initial API complete.**
- User can see early AI/automation detection evidence when present. **Initial API complete.**
- The surface is read-only and cannot apply enforcement.

## Dependencies

- WO-VIS-005
- WO-VIS-007 for detection evidence display

## Risks

- Existing UI may assume current Aegis policy/agent objects only. Keep first surface simple.
- Avoid building a polished investigation UI before the data model stabilizes.

## Completion Evidence

- API examples documented in `backend/ingest/README.md`
- Recorded investigation path using lab scenario data
