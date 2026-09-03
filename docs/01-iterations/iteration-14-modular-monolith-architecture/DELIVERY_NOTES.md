# Iteration 14 Delivery Notes

Status: design approved; implementation pending
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: implementation progress, verification evidence, and residual risk for iteration 14.

## Current State

The architecture design was approved on 2026-09-03. ADR-0015 and the complete
iteration contract set were created before implementation. No source module,
public API, product data, or local user configuration has been migrated yet.

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
| 1 — config and quality foundation | pending | none |
| 2 — bootstrap and transport | pending | none |
| 3 — preferences, sources, assets | pending | none |
| 4 — teacher | pending | none |
| 5 — assistant and CLI | pending | none |
| 6 — projects, learning, compatibility | pending | none |
| 7 — frontend feature boundaries | pending | none |
| 8 — cleanup and final cutover | pending | none |

## Residual Risk

The highest expected risks are assistant restart/idempotency regressions,
Windows prompt delivery, route-to-store bypasses surviving behind forwarding
facades, frontend query/SSE behavior changing during moves, and old project
fixtures failing after compatibility code is isolated. None is considered
closed until its planned automated and native smoke evidence is recorded here.

