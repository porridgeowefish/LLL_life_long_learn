# Iteration 08 Data Design

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: delivered iteration 08 file-first data delta; persisted schemas own runtime truth.

Existing project-local `progress/events.jsonl` remains canonical. New optional
fields extend the existing event record:

```json
{
  "id": "practice-submit:2:q1",
  "sourceType": "practice-submit",
  "sourceId": "q1",
  "activityDelta": 3,
  "delta": 4,
  "title": "完成练习",
  "detail": "第 2 次练习",
  "createdAt": "2026-07-11T08:00:00Z"
}
```

- `activityDelta` contributes to heat and action history.
- legacy `delta` remains the growth value.
- the global view is derived by reading every project stream; no duplicate global event log exists.
- old Practice events are backfilled as activity during aggregation using their source type.

Appearance persists under `ui.theme` in `config.local.json`. Local storage is a
startup cache, not the durable source.
