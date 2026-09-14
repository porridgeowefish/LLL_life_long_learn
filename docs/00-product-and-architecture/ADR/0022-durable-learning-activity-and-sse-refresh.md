# ADR-0022: Durable Learning Activity And SSE Refresh

Status: accepted
Owner: project maintainer
Date: 2026-09-14
Last reviewed: 2026-09-14
Source of truth: learning-activity write and refresh boundary.
Extends: ADR-0012, ADR-0015

## Context

The home and usage heatmaps aggregate `progress/events.jsonl`, but the active
teacher conversation and the assistant publication lifecycle did not write
there. A heatmap could therefore remain unchanged after the main learning
workflow completed. The existing app-wide SSE connection also had no activity
notification, so an already open dashboard could not refresh promptly.

## Decision

`modules/learning.RecordActivity` is the one persistence entry point for a
learning activity. The event ID is its idempotency key. The HTTP server bridges
every newly added event to `learning-activity-updated` on the existing global
SSE connection, with only `{projectSlug}` as the payload. It emits nothing for
an already-recorded event.

The teacher service reports a response only after it has been durably recorded
with successful provider completion. Bootstrap records it as
`teacher-response:<responseId>` with one activity point. The assistant
dispatcher reports only after task state is durably `succeeded` and its result
contains an updated core asset or published deliverable. Bootstrap records it
as `assistant-publication:<taskId>` with one activity point. Failures,
interruptions, no-output successes, and recovery replays do not add another
point.

`AppShell` remains the sole `EventSource` owner. A small activity-refresh hook
subscribes through that existing fan-out registry and invalidates TanStack
Query's `['activity']` entries. Home and usage heatmaps then refetch their
normal REST summaries. SSE is notification only; `GET /api/activity` remains
the recovery source after refresh or a missed notification.

## Consequences

- New completed teacher exchanges and published assistant work appear in both
  heatmaps without a browser refresh.
- Existing reading, practice, Ask-AI, session, and learning-plan activity
  writers use the same persisted stream; their route writers now share the SSE
  bridge where they already use the common activity helper.
- Activity is forward-looking for teacher and assistant work. Previously
  completed conversations and tasks are not reconstructed into synthetic
  historical study dates.
- No new event stream, timer poller, queue, route, or database is introduced.
