# Iteration 15 Delivery Notes

Status: delivered.

## Delivered

- Removed the Summary/Extend frontend pages, flashcard and flower hooks, and
  the unmounted Knowledge Garden (`GreenhousePage`).
- Reduced compatibility zones and new-project scaffolding to
  Intro/Explain/Practice. Historical Extend/Summary URLs fall back to Explain.
- Removed flashcard HTTP routes, flashcard storage, registered roles, charters,
  dedicated primitives, runtime protection branches, and artifact watching for
  retired directories.
- Generic legacy file APIs now reject `summary/**` and `extend/**` reads and
  writes. Iteration-13 migration leaves those directories byte-for-byte intact
  and excludes them from its inventory and backup.
- Added ADR-0016 and synchronized active product, data, API, architecture,
  repository-map, and primitive documentation.

## Verification

- RED then GREEN: public frontend export/type tests, legacy URL fallback test,
  retired file-API test, new-project skeleton test, and migration-preservation
  test.
- `go test ./backend-go/...` — pass.
- `npm --prefix frontend test -- --run` — 45 files, 150 tests pass.
- `npm --prefix frontend run build` — pass.
- `npm run check:full` — pass: gofmt, vet, build, architecture check
  (43 packages / 120 edges), Go coverage 31.97% (floor 28.35%), frontend lint,
  tests, production builds, contract freeze, and Windows-native fake CLI smoke.

The first full-gate attempt exposed one timing-sensitive
`TestWatcherDebouncesBursts` failure (2 emits rather than 1). A 30-run focused
reproduction passed, and the fresh complete gate passed without changing
watcher debounce behavior; no causal link to the retirement change was found.

Existing historical project data remains explicitly out of deletion scope.
