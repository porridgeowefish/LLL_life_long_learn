# ADR-0003 Learning Artifact Protocols

Status: accepted
Date: 2026-06-14

## Context

Single Markdown outputs could not support evidence-based prerequisite diagnosis, navigable long explanations, private objective answers, or extensible motivation metrics.

## Decision

Adopt three explicit file protocols:

```text
Intro assessment: intro/assessment.json
Explain pages: explain/manifest.json + explain/pages/*.md
Practice separation: practice/tasks.json + practice/answer-key.json
```

Adopt `progress/events.jsonl` as an append-only project growth ledger, with `progress/summary.json` as derived data.

## Consequences

- Frontend rendering follows stable schemas rather than parsing headings.
- Follow-ups become immutable linked pages.
- The backend can judge objective questions without publishing answers.
- Growth sources can expand beyond practice without changing the current UI.
- Old single-file projects require explicit fallback behavior.
