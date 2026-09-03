# Iteration 09 Delivery Notes

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-15

Iteration 09 is now scoped to permanent learning-project deletion. The earlier
discipline-protocol/local-learning-lab discovery notes remain deferred in
`BACKLOG.md` and do not expand this slice.

## Delivered

- Sidebar `删除学习` action with exactly one permanent-deletion confirmation.
- `DELETE /api/projects/{id}` with active-session conflict protection.
- Recursive project artifact deletion plus folder, cache, session, query-cache,
  selected-project, and browser-draft cleanup.
- Intro prerequisite cards now contain generated summaries with no supplement
  button or follow-up helper endpoint.
- Intro layout is generated content, diagnosis, then survey form at the bottom.

## Verification

```text
go test ./...                                                    PASS
npm run test -- --run                                           PASS (33 files, 104 tests)
npm run test -- --run Sidebar.test.tsx projectDeletion.test.ts  PASS (4 tests)
npm run build                                                    PASS
```

The full frontend run emits pre-existing React test `act(...)` and React Router
future-flag warnings; no tests fail. Manual browser smoke testing confirmed that
the supplement action is absent and the survey remains the final page section.
