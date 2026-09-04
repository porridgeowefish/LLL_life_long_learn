# Iteration 15 Test Plan

## Frontend

- Public-barrel and domain tests verify no Summary/Extend/Knowledge Garden
  export or five-zone type remains. The legacy project fallback is covered by
  the frontend typecheck and build.
- Typecheck and Vitest verify deleting retired files leaves the active
  teacher/Assets/Sources and Intro/Explain/Practice compatibility surface buildable.

## Backend

- Router contract test asserts both former flashcard paths return the normal
  unmatched-route behavior, not a registered handler response.
- File whitelist test rejects reads and writes under `summary/` and `extend/`.
- Migration test creates both historical directories, runs Iteration-13
  migration, and asserts their files still exist afterwards.
- Workspace test verifies new system-learning projects create neither retired
  directory and reject both retired zone names.

## Repository verification

```text
npm --prefix frontend test -- --run
npm --prefix frontend run build
go test ./backend-go/...
npm run check:full
```

Run Windows native-runtime smoke checks only when the Iter14 quality gate
invokes them; record any local cgo toolchain limitation separately.
