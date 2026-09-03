# Iteration 04 Data Design

Status: active
Last reviewed: 2026-06-14

File remains the durable source of truth.

```text
intro/output.md
intro/survey.json
intro/assessment.json
explain/manifest.json
explain/pages/NNN-slug.md
practice/tasks.json
practice/answer-key.json
practice/draft.json
practice/attempts/<N>.json
practice/submissions/<N>.json
practice/evaluations/<N>.json
practice/evaluations/<N>.md
summary/flashcards.json
summary/review-pack.md
summary/flashcard-progress.json
progress/events.jsonl
progress/summary.json
```

`practice/draft.json` is the durable in-progress answer buffer. It stores
`schemaVersion`, `setId`, `generatedAt`, ordered `taskIds`, per-task
`answer/selfAssess`, optional `attempt`, and `updatedAt`. It is overwritten
atomically during answering and is not an answer key or evaluation artifact.

Evaluation JSON keeps the submitted attempt number, overall score, generated
time, and an overall `summary`. Each subjective result stores `taskId`,
`score`, `feedback`, `suggestedAnswer`, `evidence`, and `passed`. Older
evaluation files without the two new text fields remain readable.

Compatibility:

- Missing task difficulty defaults to 3.
- Missing task set ID is synthesized from `generatedAt`.
- Missing explain manifest falls back to `explain/output.md`.

Growth event identity is deterministic:

```text
<sourceType>:<attemptId>:<sourceId>
```

This makes retries idempotent. `summary.json` is derived and rebuildable.

Summary flashcards use:

```json
{
  "version": 1,
  "cards": [{
    "id": "fc-001",
    "front": "question in Markdown",
    "back": "answer in Markdown",
    "category": "concept|relationship|boundary|misconception|transfer",
    "sourceRefs": ["explain/pages/003-core-concepts.md"],
    "generatedReason": "why this card matters"
  }]
}
```

The backend also accepts the legacy top-level card array, fenced JSON, and
common generated aliases (`flashcards`/`items`, `question`/`answer`) for
compatibility. New writes always use the versioned envelope. Card IDs must be
unique and both sides must be non-empty.
