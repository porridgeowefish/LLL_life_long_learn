# Iteration 04 API Contract

Status: active
Last reviewed: 2026-06-14

## Projects

`GET /api/projects/{id}` includes the derived field:

```json
{"project":{"generatedZones":["Intro","Explain"]}}
```

`generatedZones` is computed from valid, non-empty zone artifacts on disk and
is not persisted into `state.json`.

`POST /api/projects/{id}/subprojects` accepts:

```json
{"title":"...","slug":"...","why":"...","current":"...","target":"...","standard":"..."}
```

The child is addressable through `/api/projects/{childSlug}`.

## Agent Invoke

`POST /api/agents/{id}/invoke` adds optional:

```json
{"parentPageId":"p003"}
```

For Explain, this means append a follow-up page linked to that parent.

## Practice

```text
GET  /api/projects/{id}/practice/tasks
GET  /api/projects/{id}/practice/draft
PUT  /api/projects/{id}/practice/draft
POST /api/projects/{id}/practice/attempts
GET  /api/projects/{id}/practice/attempts/latest
POST /api/projects/{id}/practice/attempts/{attempt}/objective/{taskId}/check
POST /api/projects/{id}/practice/attempts/{attempt}/submit
GET  /api/projects/{id}/practice/evaluation?attempt=N
```

`GET /practice/draft` returns `{ "draft": null }` when no draft exists.
`PUT /practice/draft` persists the in-progress answers to `practice/draft.json`
and stamps `updatedAt`. The payload shape is:

```json
{
  "schemaVersion": 1,
  "setId": "2026-06-15",
  "generatedAt": "2026-06-15T12:00:00Z",
  "taskIds": ["q1", "q2"],
  "drafts": {
    "q1": {"answer": "text, boolean, or option array", "selfAssess": 3}
  },
  "attempt": 2
}
```

The backend validates `setId` and `generatedAt` against the current task set,
then normalizes `taskIds` from `tasks.json`. The frontend saves drafts both to
localStorage and this endpoint while answering, so refreshes and backend
restarts do not erase work.

Objective check request:

```json
{"answer":"A"}
```

The answer may be a boolean, option ID, or option-ID array. The response includes correctness, correct answer, explanation, and awarded growth.

`POST /practice/submit` remains as a compatibility endpoint.

`GET /practice/attempts/latest` returns the latest submitted attempt, including
submitted subjective answers and locked objective results. The frontend uses
it to restore a submitted attempt after refresh instead of clearing the work.
Newer empty or still-answering attempts do not hide the submitted result.

Practice Agent invocation accepts:

```json
{"practiceQuestionCount":7}
```

The generation prompt must produce exactly that number of questions.

After a successful one-time submission, the frontend requests background
evaluation without opening an interactive Agent window:

```text
POST /api/projects/{id}/practice/attempts/{attempt}/evaluation
```

The background evaluation writes:

```text
practice/evaluations/3.json
practice/evaluations/3.md
```

The evaluation JSON includes an overall `summary`; every subjective result
includes `feedback` and a direct `suggestedAnswer`.

## Progress

`GET /api/projects/{id}/progress` returns:

```json
{"progress":{"total":12,"bySource":{"practice-submit":7},"recentEvents":[],"updatedAt":"..."}}
```

## Summary Flashcards

```text
GET  /api/projects/{id}/summary/flashcards
POST /api/projects/{id}/summary/flashcards/grade
```

The read endpoint returns the cards from `summary/flashcards.json` together
with durable review progress. New files use a versioned envelope; legacy JSON
arrays remain readable:

```json
{
  "version": 1,
  "cards": [{
    "id": "fc-001",
    "front": "为什么 NFA 与 DFA 的表达能力相同？",
    "back": "子集构造把一组 NFA 状态映射为一个 DFA 状态，因此保持识别语言不变。",
    "category": "relationship",
    "sourceRefs": ["explain/pages/005-nfa-dfa.md"],
    "generatedReason": "检验自动机等价性的核心理解"
  }]
}
```

`front` and `back` support Markdown. Cards are concept-review material, not
an error log: they cover core concepts, relationships, boundaries,
misconceptions, and transfer.

New Summary Agent writes must use exactly the `version/cards/front/back`
protocol above, with raw JSON only. The backend reader is intentionally more
forgiving for existing files: it also accepts a top-level array, fenced JSON,
and common generated variants using `flashcards`/`items` or
`question`/`answer`, then normalizes them before returning the frontend DTO.

Grade request:

```json
{"cardId":"fc-001","grade":"forgot|fuzzy|got-it|easy"}
```

## File Privacy

`GET /files/projects/{id}/practice/answer-key.json` returns `403`.

## Explain Summaries

The existing `/confusions` route remains stable for file compatibility, while
the frontend presents these records as saved summaries.

- `charStart` and `charEnd` are offsets in the rendered page text.
- `quoteSnapshot` is used to recover the highlight when nearby text shifts.
- `DELETE /api/projects/{id}/confusions/{confusionId}` permanently removes the
  record; legacy records with `state: "deleted"` are omitted from unfiltered lists.
