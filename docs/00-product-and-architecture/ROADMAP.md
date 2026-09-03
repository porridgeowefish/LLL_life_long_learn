# LLL Roadmap

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: delivered product slices and the next architecture work; iteration documents own exact acceptance contracts.

## Delivered Foundation

```text
Iterations 02–06   file-first Go backend, learning agents, rich readers, Ask AI
Iterations 07–12   project types, flat storage, discipline maps, plans, scope snapshots
Iteration 13       teacher conversation, async CLI assistant, assets, sources, migration
2026-09 hardening  stable rich streaming and one global learner preferences file
```

The active system-learning experience is `教师 / 资产 / 资料`. The earlier
Intro / Explain / Practice / Extend / Summary implementation remains a
compatibility reader and migration source, not the product's primary workflow.

## Current Architecture Commitments

- API teacher owns responsive teaching dialogue and one disclosed delegation tool.
- Visible native Agent CLI owns approved heavy work.
- Conversation, task, source, and asset files are durable truth.
- Rich stream rendering coalesces deltas and defers Mermaid until completion.
- One `<WORKSPACE>/preferences.md` is learner-owned and read-only to AI.
- There is no project-memory agent, automatic preference inference, or AI write path.

## Committed Next Slice

Iteration 14 is approved for implementation. It reorganizes the existing
runtime into an enforced business-capability modular monolith and adds one
typed configuration system plus standardized architecture, test, coverage,
build, browser, and Windows smoke gates.

The slice preserves learner-visible behavior, public APIs, and canonical
project schemas. It introduces no microservices, database, broker, or product
feature. Delivery proceeds in verified waves on one branch and merges only
after the complete compatibility gate passes.

## Later Candidates

These remain candidates after iteration 14 and are not part of its committed
scope:

1. Add browser performance instrumentation for long teacher conversations,
   including render counts, long tasks, and rich-block timing.
2. Virtualize very long completed transcripts without disturbing scroll follow
   or message anchors.
3. Add a user-visible import/export flow for `preferences.md` while retaining
   one canonical file.
4. Continue removing inactive five-zone UI/code only after compatibility usage
   is measured and a recovery export exists.

## Permanently Out Of Scope

```text
Multi-user accounts
Cloud-first canonical storage
Database-first project truth
Hidden assistant execution
AI-controlled preference or memory writes
```

## Verification Baseline

```text
go test ./...
npm --prefix frontend test -- --run
npm --prefix frontend run build
```

Iteration 14 will retain these individual commands and add repository-level
`check:fast`, `check`, and `check:full` gates with generated evidence below the
ignored `.artifacts/quality/` directory.
