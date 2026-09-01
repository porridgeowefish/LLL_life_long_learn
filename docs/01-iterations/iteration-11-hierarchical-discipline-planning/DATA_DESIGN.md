# Iteration 11 Data Design

Status: active
Owner: project maintainer
Last reviewed: 2026-07-18
Source of truth: persisted task-plan schema and compatibility.

`projects/<map-slug>/learning-plan.json`:

```json
{
  "schemaVersion": 1,
  "items": [
    {
      "id": "stable-client-id",
      "topicTitle": "经典力学",
      "status": "planned",
      "addedAt": "2026-07-18T00:00:00Z",
      "startedAt": "",
      "completedAt": ""
    }
  ],
  "updatedAt": "2026-07-18T00:00:00Z"
}
```

Rules:

- array position is the learner-selected order;
- `status` is `planned`, `in-progress`, or `completed`;
- completing a task is the single-task check-in and records `completedAt`;
- one completion timestamp creates one activity-only `learning-task-complete` event;
- topic titles are copied from actionable overview headings;
- task state does not imply a system-learning project exists;
- missing legacy files decode as an empty plan.
