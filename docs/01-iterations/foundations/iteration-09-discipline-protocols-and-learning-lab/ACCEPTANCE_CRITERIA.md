# Iteration 09 Acceptance Criteria

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15

- A system-learning row exposes a `删除学习` action.
- Selecting it opens exactly one confirmation that names the learning project
  and states that deletion is permanent and includes all content.
- Cancelling confirmation sends no delete request.
- Confirming calls `DELETE /api/projects/{id}` once.
- Successful deletion removes the canonical project root recursively.
- Folder membership or map binding references to the slug are removed while
  preserving the folder and unrelated members.
- Project index and completed in-memory sessions no longer expose the project.
- Known browser-local Markdown and Practice drafts for the slug are removed.
- A project with an active session returns HTTP `409` and remains intact.
- Invalid and missing slugs return `400` and `404` respectively.
- Every newly generated prerequisite item includes a concise `summary` of the
  knowledge and the learner's current gap.
- Legacy prerequisite items without `summary` display their `impact` as a
  fallback.
- Intro exposes no `快速补充` action and makes no prerequisite-helper request.
- Intro renders generated content first, prerequisite diagnosis second, and the
  survey form last.
