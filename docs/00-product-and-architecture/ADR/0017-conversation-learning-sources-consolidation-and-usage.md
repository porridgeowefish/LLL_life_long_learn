# ADR-0017 — Conversation Learning Sources, Consolidation, And Usage

Status: accepted
Date: 2026-09-05

## Decision

The conversation-first learning unit uses a deliberately narrow source contract:
after learner-approved background parsing, a supported source has exactly one
canonical `derived/content.md`. The Sources page is a file/status ledger; only
the Teacher composer may preview and attach ready parsed text. A selected
source's Markdown becomes bounded teacher context for that turn.

Map topics may carry `teachingOutline`; the immutable scope snapshot preserves
it and the teacher receives it as soft teaching guidance.

Assistant consolidation is an approved asynchronous task, not a transcript or
a new Summary zone. It must update Intro (why the covered knowledge matters)
and Body (a self-contained teaching manuscript ending with critical thinking).
Practice changes only when the approved task explicitly requests questions.

Provider-reported teacher usage is appended per response under
`conversation/usage.jsonl` and displayed by durable conversation in the
separate Token Usage page. Assistant token accounting is intentionally deferred.
The Agent-management UI route is retired; task progress stays with the teacher
conversation.

## Consequences

- Parsing and citation are asynchronous but simple: one ready Markdown payload.
- Older map catalogs remain readable without `teachingOutline`; newly generated
  catalogs are instructed to produce it.
- Usage values are provider-reported, so absent provider fields are not guessed.
- The backend's compatibility agent/session routes remain until their callers
  are retired; only the standalone management screen is removed now.

## Supersedes

Extends ADR-0012 and ADR-0016 for the current conversation workflow.
