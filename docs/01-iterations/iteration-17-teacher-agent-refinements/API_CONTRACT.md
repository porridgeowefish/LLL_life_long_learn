# Iteration 17 — API Contract

Status: planned
Owner: project maintainer
Last reviewed: 2026-09-13

Base path: `/api/projects/{id}/...`. All new endpoints follow the existing
error envelope `{"error": {"code", "message"}}`.

## Existing endpoints that change

### GET /api/projects/{id}/conversation

Projection gains one field. No existing field changes.

```json
{
  "queue": [
    {
      "queueId": "q_xxx",
      "content": "...",
      "attachmentRefs": ["source_xxx"],
      "queuedAt": "2026-09-11T08:00:00Z"
    }
  ]
}
```

`messages` hides messages belonging to superseded responses (regeneration);
`taskLinks` unchanged.

## New endpoints

### POST /api/projects/{id}/conversation/turns/queue

Queue a learner turn. Works whether or not a response is active; when idle the
item is promoted immediately and a normal turn starts.

Request/Response:

```json
// request
{ "operationId": "uuid", "content": "text", "attachmentRefs": [], "providerId": "optional" }
// 202
{ "queueId": "q_xxx", "promoted": false }
```

`promoted: true` means no response was active and the item already became a
turn (clients should reattach via the existing active-response endpoint).
Validation matches `POST /turns` (content 1..64KiB, operationId required).
Conflict-free: queueing while active never returns 409.

### POST /api/projects/{id}/conversation/queue/{queueId}/edit

```json
{ "content": "new text", "attachmentRefs": [] }
// 200 { "queueId": "q_xxx" }
```

404 when the queue id is not queued. Content validation as above.

### POST /api/projects/{id}/conversation/queue/{queueId}/discard

```json
// 200 { "queueId": "q_xxx", "discarded": true }
```

### POST /api/projects/{id}/conversation/queue/{queueId}/steer

Interrupts the active response (partial text preserved as `interrupted`),
promotes this item immediately as a steering learner message, and starts the
next turn. When no response is active it degenerates to an immediate promote.

```json
// 202 { "queueId": "q_xxx", "responseId": "resp_xxx", "steered": true }
```

409 `steer_conflict` only if another lifecycle operation holds the turn latch.

### POST /api/projects/{id}/conversation/responses/{responseId}/regenerate

Regenerates the given (latest) teacher response for the same triggering
learner message. Old response/message become superseded.

```json
// 202 { "responseId": "resp_new", "supersededResponseId": "resp_old" }
```

Errors: 404 unknown response; 409 `response_active` when a response is
streaming; 409 `not_latest_response` when the target is not the newest teacher
response.

### GET /api/projects/{id}/conversation/export.md

Returns `text/markdown; charset=utf-8` attachment with the complete durable
transcript (all messages, no pagination, attachments as names, steering
noted, superseded responses excluded).

## New SSE frames (teacher stream)

| event | data | when |
|---|---|---|
| `queue-snapshot` | `{ "queue": [ {queueId, content, attachmentRefs, queuedAt} ] }` | after any queue mutation reaches a live stream (optional, clients may refetch conversation) |
| `search-started` | `{ "query": "..." }` | search_web accepted |
| `search-completed` | `{ "query": "...", "count": 5 }` | results returned to provider |
| `search-failed` | `{ "query": "...", "code": "websearch_unavailable" }` | search backend error |

All other frames (`turn-accepted`, `message-started`, `text-delta`,
`reasoning-summary-delta`, `usage`, `task-accepted`, `tool-rejected`,
`message-completed`, `message-failed`) are unchanged.

## Generated artifact promotion recovery

No HTTP request or response shape changes. The existing assistant-task
projection may move from `failed/commit-failed` to `succeeded` when the
dispatcher completes the bounded artifact-only promotion recovery in ADR-0020.
If that one recovery fails, the existing `failure.code` field is
`artifact-recovery-failed`; its `suggestion` identifies the uncommitted output
class. No CLI/model execution occurs during either transition.

## Config additions (config.local.json)

```json
{
  "askAiProviders": {
    "bindings": {
      "ocr": { "providerId": "glm", "model": "glm-4v-plus" }
    }
  },
  "webSearch": {
    "provider": "zhipu",
    "apiKey": "…",
    "engine": "search_std"
  }
}
```

- `askAiProviders.bindings.ocr` is optional; absent means images keep the
  CLI-agent parse path.
- The `webSearch` section is optional; absent or `apiKey == ""` disables the
  `search_web` tool. Only `provider: "zhipu"` exists in this iteration.
