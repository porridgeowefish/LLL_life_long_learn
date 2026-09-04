# Iteration 14 Delivery Notes

Status: implementation in progress
Owner: project maintainer
Last reviewed: 2026-09-04
Source of truth: implementation progress, verification evidence, and residual risk for iteration 14.

## Current State

The architecture design was approved on 2026-09-03. ADR-0015 and the complete
iteration contract set were created before implementation. Waves 0 through 7
are delivered on the iteration branch; public API and product-data contracts
remain unchanged.

Work is isolated on:

```text
codex/repository-architecture-reset
```

## Pre-Iteration Evidence

The immediately preceding iteration-13 hardening snapshot is commit
`69db3d0`. Fresh verification before that snapshot commit reported:

```text
go test ./...                         PASS
npm --prefix frontend run test       PASS: 44 files, 149 tests
npm --prefix frontend run build      PASS
```

This evidence confirms the starting snapshot, not iteration-14 delivery.
Coverage, contract fixtures, architecture graph, lint, E2E, and real Windows
native-provider smoke remain to be established or rerun by the implementation
plan.

## Progress

| Wave | State | Evidence |
|---|---|---|
| 0 — contract and baseline freeze | delivered | `tests/fixtures/` synthetic families (canonical/legacy-zones/memory-era/corrupt) generated through production stores via `backend-go/internal/testfixtures`; `contract_freeze_test.go` pins 89 frozen routes, key JSON shapes, SSE hello/event names, error paths, and fixture non-mutation; baseline `tests/baseline.json` (Go 29.35%, frontend 40.24%); `.gitignore` gains `.artifacts/` |
| 1 — config and quality foundation | delivered | `backend-go/internal/platform/config` typed loader with flat-key compat, LLL_* env, `--port/--workspace/--config` flags, redaction, provenance, and a no-rewrite guarantee (12 loader tests incl. flat↔sectioned equivalence); `tools/archcheck` R1–R4 with negative fixtures and allowlist; root `check:fast`/`check`/`check:full` via `scripts/run-check.js` + `scripts/check-coverage.js` floors; frontend ESLint 9 flat config (lint now actually runs: 3 stale disable comments and 1 useless-escape fixed); `config/config.example.json` + `config/README.md`; `.github/workflows/ci.yml` (windows-latest, `npm run check`); QUALITY_COMMANDS.md documented. Gate: `node scripts/run-check.js complete` PASSED (go 32.60% ≥ floor) |
| 2 — bootstrap and transport | delivered | `backend-go/internal/app/bootstrap` is the concrete composition root; HTTP/SSE files moved from `internal/server` to `internal/transport/httpserver`; broadcaster, registry, sessions, project cache, and job maps are instance fields rather than package globals; transport construction now accepts explicit dependencies and bootstrap owns probes, recovery, watcher/dispatcher startup, and integration callbacks. `npm run check` PASSED: archcheck 37 packages/219 edges, Go coverage 32.61% ≥ 28.35% floor, frontend 44 files/149 tests, frontend and backend production builds. The Windows fsnotify debounce test was stabilized at a realistic test window and passed 20 repeated runs. |
| 3 — preferences, sources, assets | delivered | `preferencestore`, `sourcestore`, `assetstore`, `annotationstore`, and `artifactwriter` implementations moved below `modules/{preferences,sources,assets}/internal`; callers now use module-root facades and the legacy annotation DTO is local to assets. The architecture scanner was corrected to evaluate direct Go imports rather than inherited transitive dependencies, with a regression test proving facade internals do not become transport edges. `npm run check` PASSED: archcheck 40 packages/125 direct edges, Go coverage 32.57% ≥ 28.35% floor, frontend 44 files/149 tests, and both production builds. |
| 4 — teacher | delivered | Conversation storage, teacher service/context, provider gateway, normalized providers, and AI configuration moved below `modules/teacher/internal` behind the root `teacher` facade. Teacher-owned active-response buffering moved out of transport. Teacher task creation now uses the consumer-owned `TaskAuthorizer` port implemented by `app/integration`, removing the teacher→assistant dependency; provider settings load/save delegates to the section-preserving platform config owner. `go test ./... -count=1` PASSED and archcheck reports 42 packages/127 direct edges. A first frontend gate run encountered a transient Windows Vitest cache `EBUSY`; immediate isolated rerun passed all 44 files/149 tests. |
| 5 — assistant and CLI | delivered | Task storage/dispatch, execution, registry, runtime selection, prompt assembly, and CLI launch implementations live below `modules/assistant/internal` behind one facade; launcher-only artifact promotion moved out of assets. The 1,370-line dispatcher is split into queue/manifest/execute/validate/commit/reconcile/artifacts files, and the launcher into types/exec/terminal/wrappers. Dispatcher access to conversation/preferences/assets/sources now uses consumer-owned ports wired by `app/integration`; runtime configuration delegates to platform config. A real PowerShell wrapper smoke builds `tests/smoke/fake-agent`, verifies UTF-8 BOM, a Chinese workspace, and one intact 4,422-character prompt argument after the documented straight-to-curly quote normalization. `check:full` first exposed and fixed a Windows shell-pipe bug in its contract command; direct reruns of the complete HTTP contract package and Windows fake-CLI smoke PASS. Full pre-contract layers passed with Go 32.15% ≥ 28.35%, frontend 44 files/149 tests, and both production builds. |
| 6 — projects, learning, compatibility | delivered | Project workspace/state/slug/index/folder implementations moved below `modules/projects/internal` behind the `projects` facade; learning scope, progress/activity, and live run status moved below `modules/learning/internal` behind the `learning` facade. The project skeleton accepts a serializable learning-scope contract so projects does not depend back on learning, while learning resolves project storage through the projects facade. Session/practice/confusion/flashcard legacy stores, artifact watching, and iteration-13 migration are quarantined under `internal/compatibility` with unchanged persisted formats. `check:complete` PASSED: archcheck 44 packages/122 direct edges, all Go tests, frontend 44 files/149 tests, Go coverage 32.15% ≥ 28.35%, and both production builds. |
| 7 — frontend feature boundaries | delivered | The SPA is organized into `app/`, `features/{learning,projects,agents,legacy-zones,preferences,settings}`, and `shared/`. Every feature has an `index.ts` public surface, and all 24 initially detected cross-feature private imports were redirected through those surfaces. Query-key strings, routes, the single app-level SSE mount, legacy components, and API behavior were preserved. Frontend architecture check, ESLint, all 44 files/149 tests, and the production TypeScript/Vite build PASS. |
| 8 — cleanup and final cutover | pending | none |

## Residual Risk

The highest expected risks are assistant restart/idempotency regressions,
Windows prompt delivery, route-to-store bypasses surviving behind forwarding
facades, frontend query/SSE behavior changing during moves, and old project
fixtures failing after compatibility code is isolated. None is considered
closed until its planned automated and native smoke evidence is recorded here.
