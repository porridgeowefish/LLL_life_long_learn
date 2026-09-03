# ADR-0013: Stable Stream Rendering And Global Learner Preferences

Status: accepted
Owner: project maintainer
Date: 2026-09-02
Last reviewed: 2026-09-02
Source of truth: teacher rich-stream performance boundary and active learner-preference persistence.
Supersedes: project and learner memory growth decisions in ADR-0002 and ADR-0003
Extends: ADR-0012

## Context

Teacher deltas previously updated React state about every 32 ms. Each update
reparsed the full growing answer through Marked, KaTeX, DOMPurify, and Mermaid.
After a Mermaid fence first became syntactically closed, every later text delta
cancelled and requeued diagram work, so the UI alternated between placeholder,
success, and failure while repeatedly forcing layout and auto-scroll.

The earlier architecture also created project-level memory files and exposed a
project-memory editor. The current conversation and versioned assets already
own project learning state. A second inferred memory layer duplicates truth and
creates an unclear AI-write boundary.

## Decision

### Rich-stream rendering

- Provider deltas are coalesced before React state updates.
- Streaming Markdown renders from a 120 ms snapshot rather than every delta.
- Mermaid is never executed for an active message. A stable placeholder is
  shown, and the final complete message renders the diagram once.
- Collapsed reasoning is not parsed or mounted until the learner expands it.
- Historical messages are memoized; live updates do not rerender their rich
  content when references and task state are unchanged.
- Auto-scroll is throttled and performs at most one layout write per scheduled
  frame while the learner remains near the bottom.
- The same streaming Markdown boundary is reusable by annotation Ask AI.

### Preferences instead of memory

- `<WORKSPACE>/preferences.md` is the only active preference-memory file.
- It is learner-owned Markdown with a 256 KiB limit.
- The preferences page and direct file editing are the only write paths.
- The API teacher receives bounded content as read-only system context.
- Each assistant attempt receives `workspace/inputs/preferences.md`, a sealed
  read-only snapshot recorded in its input manifest.
- AI services cannot edit the canonical file, infer persisted preferences, or
  use preferences as tool authorization or factual evidence.
- New projects do not create `memory/`. Generic project file routes reject it.
- Existing project memory directories are preserved but ignored; migration is
  non-destructive.

## Consequences

Streaming remains visibly responsive while expensive rendering occurs at a
bounded rate. Mermaid no longer oscillates on partial syntax, and scrolling
causes fewer forced layouts. Final Markdown output and safety sanitization are
unchanged.

Personalization becomes simpler and auditable: one file, one explicit editor,
one read-only AI contract. Automatic preference learning is intentionally out
of scope until a future decision defines review, provenance, and consent.
