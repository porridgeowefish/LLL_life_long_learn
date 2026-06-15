# Iteration 04 Data Design

Status: active
Last reviewed: 2026-06-14

File remains the durable source of truth.

```text
intro/output.md
intro/assessment.json
explain/manifest.json
explain/pages/NNN-slug.md
practice/tasks.json
practice/answer-key.json
practice/attempts/<N>.json
practice/submissions/<N>.json
practice/evaluations/<N>.json
practice/evaluations/<N>.md
progress/events.jsonl
progress/summary.json
```

Compatibility:

- Missing task difficulty defaults to 3.
- Missing task set ID is synthesized from `generatedAt`.
- Missing explain manifest falls back to `explain/output.md`.

Growth event identity is deterministic:

```text
<sourceType>:<attemptId>:<sourceId>
```

This makes retries idempotent. `summary.json` is derived and rebuildable.
