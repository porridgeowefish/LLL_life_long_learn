# ADR-0018 — Teacher Conversation Queue, Steering, And Regeneration

Status: accepted
Date: 2026-09-11

## Decision

The teacher conversation gains a durable waiting queue and explicit steering,
both stored as events in the append-only `conversation/events.jsonl` log — no
new store, no client-owned queue state.

Queue family: `learner-queued`, `queue-item-edited`, `queue-item-discarded`,
`queue-item-promoted {mode: auto|steer}`. The live queue is a replay
projection exposed as `queue` on every conversation read, so reloads and
multiple tabs agree by construction. The server, not the page, drives
auto-advance: when a response reaches a terminal state, the transport
promotes the queue head and starts the next turn; a queued item sent while
idle promotes immediately. Server restart never auto-sends queued items —
they stay visible and editable until the next lifecycle event.

Steering is an explicit user action on a queued item, because no supported
provider stream accepts mid-flight injection. `POST /queue/{id}/steer`
cancels the active response (partial text is preserved as an `interrupted`
message through the existing interruption path), promotes that item as a
learner message, and immediately starts a new turn whose system prompt gains
a 【生成中引导】 appendix instructing the teacher to absorb the interrupted
draft and follow the guidance.

Regeneration applies to the latest teacher response only:
`POST /responses/{id}/regenerate` appends `response-superseded {responseId,
newResponseId}` and re-runs the same triggering learner message. Superseded
messages are hidden from every projection (`Read`, `ReadRecent`,
`SequencedMessages`, provider context) but never deleted; `ResponseForLearner`
returns the newest non-superseded response so replay cannot resurrect an old
reply. Sealed assistant snapshots read through a historical cutoff seq and
are unaffected.

Transcript export is read-only: `GET /conversation/export.md` renders the
visible durable transcript (roles, timestamps, attachment names); it never
mutates the log.

Concurrency: a per-project turn-lifecycle mutex serializes start, steer,
regenerate, and advance. Streaming itself stays outside the lock;
`ActiveResponse.Done()` lets steer wait for the interrupted turn's
persistence before starting the follow-up.

## Consequences

- Queue durability equals conversation durability; no separate sync protocol.
- Steering cost: one cancelled provider request plus one fresh request — the
  same trade every major chat agent makes, now explicit and auditable.
- Regeneration is replay-safe and idempotent per learner message; old replies
  remain in the event log for audit.
- The frontend composer stays enabled during streaming; its send button
  becomes "queue" and queue chips offer edit / steer / discard with
  hover-revealed icon actions.

## Supersedes

Extends ADR-0017's conversation workflow; no decisions are reversed.
