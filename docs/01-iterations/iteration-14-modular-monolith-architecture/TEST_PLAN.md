# Iteration 14 Test Plan

Status: approved design; implementation not started
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: verification strategy and delivery gates for the modular-monolith refactor.

## Strategy

This is a behavior-preserving architecture change. Verification therefore
combines existing tests, black-box contract freezing, architecture rules,
real-file integration, browser paths, and Windows-native smoke checks. Mock-only
tests cannot prove filesystem, process, encoding, or recovery compatibility.

## Baseline

Before source moves, record:

- Go package and import graph;
- existing public routes and normalized SSE event names;
- current and legacy persisted fixtures;
- configuration fields, defaults, and precedence;
- Go and frontend coverage;
- full Go, frontend test, and production-build results.

## Module Tests

Domain tests cover state transitions and invariants without filesystem, HTTP,
provider, or process dependencies. Application tests cover orchestration and
error classification through small fakes. Adapter tests use real temporary
directories, malformed records, interrupted writes, and platform-specific
fixtures where applicable.

Each migrated capability must retain or improve coverage for its previous
behavior before its old call path is removed.

## Architecture Tests

Positive cases prove the approved graph builds. Negative fixtures prove the
checker rejects:

- transport importing a private store or provider;
- one module importing another module's nested `internal` package;
- platform importing a business module;
- frontend features importing another feature's internal file;
- new writes through the compatibility boundary.

## Contract And Compatibility Tests

- run existing route tests against the new composition root;
- compare frozen HTTP/JSON/SSE behavior;
- load current iteration-13 projects from copied fixtures;
- load legacy Intro, Explain, Practice, Extend, Summary, and memory-era data;
- verify reads do not mutate fixtures;
- verify new operations do not create retired file formats;
- verify old and sectioned configuration resolve equivalently;
- verify corrupt data returns a classified diagnostic and remains untouched.

## Recovery Tests

The complete suite retains coverage for:

- teacher response refresh recovery;
- queued and running task restart reconciliation;
- late valid result manifests;
- partial asset commit repair;
- duplicate completion and idempotency;
- source revision and asset atomic writes;
- missed SSE followed by REST recovery.

## Frontend Tests

- move component, hook, API, and state tests with their owning feature;
- retain route and query-key behavior;
- assert one global SSE subscription;
- retain the `/memory` to `/preferences` compatibility redirect;
- run critical Playwright paths for project creation, teacher dialogue, task
  authorization, asset reading/editing, source upload/reference, and preferences.

## Windows-Native Smoke

Run on a real supported Windows environment:

1. create or copy a project under a Chinese path;
2. generate and consume the PowerShell wrapper with required UTF-8 BOM;
3. deliver a long prompt containing ASCII double quotes and verify the native
   process receives it completely;
4. launch the selected CLI visibly;
5. exercise interruption or server restart and verify durable reconciliation.

## Standard Commands

Target commands after wave 1:

```text
npm run check:fast
npm run check
npm run check:full
```

Underlying baseline commands remain individually runnable:

```text
go test ./...
npm --prefix frontend run lint
npm --prefix frontend run test
npm --prefix frontend run build
```

## Per-Wave Gate

Every migration commit runs formatting, architecture checks, affected module
tests, complete Go tests, frontend tests when relevant, and the appropriate
build. A failing gate stops the wave; later moves do not accumulate on an
unverified boundary.

## Final Gate

The final merge additionally requires:

- complete contract and compatibility suites;
- coverage comparison against the recorded baseline;
- Playwright critical paths;
- real Windows-native smoke evidence;
- generated reports checked for secret or learner-content leakage;
- documentation links and status verification;
- clean Git worktree.

Commands, results, environment, and residual risks are recorded in
`DELIVERY_NOTES.md`.

