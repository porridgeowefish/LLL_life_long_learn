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
POST /api/projects/{id}/practice/attempts
POST /api/projects/{id}/practice/attempts/{attempt}/objective/{taskId}/check
POST /api/projects/{id}/practice/attempts/{attempt}/submit
GET  /api/projects/{id}/practice/evaluation?attempt=N
```

Objective check request:

```json
{"answer":"A"}
```

The answer may be a boolean, option ID, or option-ID array. The response includes correctness, correct answer, explanation, and awarded growth.

`POST /practice/submit` remains as a compatibility endpoint.

## Progress

`GET /api/projects/{id}/progress` returns:

```json
{"progress":{"total":12,"bySource":{"practice-submit":7},"recentEvents":[],"updatedAt":"..."}}
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
