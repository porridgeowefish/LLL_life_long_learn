# Iteration 10 Interface Contract

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: changed HTTP behavior and frontend request shape for iteration 10.

## Flower Save Side Effect

`POST /files/projects/{id}/extend/flower.json` keeps its existing request and
response shape. After a successful atomic file write, it awards an activity
event:

```json
{
  "id": "extend-flower:<first-12-sha256-bytes-as-hex>",
  "sourceType": "extend-flower",
  "sourceId": "extend/flower.json",
  "activityDelta": 1,
  "title": "编辑知识花朵"
}
```

The content-derived ID makes identical repeated saves idempotent while allowing
distinct edits to contribute. If recording the event fails, the endpoint
returns `500 record flower activity: ...` after the file write; retrying is safe.

## Project-Type Advisor

`POST /api/project-type-advice` continues to accept optional draft fields. The
frontend sends only fields already present when the advisor opens. The backend
removes empty draft values before prompt assembly and explicitly treats missing
values as unknown.
