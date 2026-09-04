# Iteration 15 Acceptance Criteria

1. The frontend contains no route, navigation item, rendered component, query
   hook, bundle import, unmounted Greenhouse page, or five-zone fallback branch
   for the retired Summary, Extend, or Knowledge Garden feature. A historical
   Extend/Summary URL falls back to Explain.
2. `GET /api/projects/{id}/summary/flashcards` and
   `POST /api/projects/{id}/summary/flashcards/grade` are not registered.
3. The generic legacy file endpoint no longer reads or writes `summary/` or
   `extend/`; it continues to support the remaining explicitly allowed legacy
   paths until their separate retirement.
4. `agents/registry/summary.json`, `agents/registry/extend.json`, their
   charters, and their dedicated primitive documents are removed. No active
   runtime resolves either ID.
5. Iteration-13 migration leaves existing project `summary/` and `extend/`
   directories untouched and does not inventory them as active migration input.
6. A fixture containing legacy Summary/Extend files still completes migration;
   those files remain present on disk after the migration.
7. A source search confirms no product-facing reference to Summary, Extend, or
   Knowledge Garden remains outside historical documentation, migration backup
   fixtures, and deliberately retained compatibility-reader notes.
8. Frontend unit tests, relevant Go tests, and repository quality gates pass.
