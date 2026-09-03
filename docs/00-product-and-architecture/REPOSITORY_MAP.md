# Repository Map

Status: active  
Owner: project maintainer  
Last reviewed: 2026-09-03
Source of truth: this file classifies repository content by ownership and trust level.

## Purpose

This file answers four practical questions:

```text
哪些内容是当前项目实现的一部分
哪些内容是文档合同
哪些内容只是参考附件 / mock / 对齐材料
哪些内容已经属于旧版或运行产物，不应继续当成兜底实现
```

## 1. Product Code

These directories are the current implementation surface:

```text
backend-go/        Go backend, API routes, session runtime, file stores
frontend/          Vite + React 18 + TypeScript SPA
agents/            agent registry, charters, primitives
```

Use these as executable truth before trusting any design/mock document.

### Approved Iteration-14 Source Target

The current directories above remain executable truth. Iteration 14 will
reorganize their internals without moving the top-level `backend-go/`,
`frontend/`, or `agents/` roots and without moving runtime data:

```text
backend-go/internal/app/              composition and cross-module glue
backend-go/internal/transport/        HTTP/SSE adapters
backend-go/internal/modules/          business-capability facades and private implementations
backend-go/internal/platform/         config, filesystem, process, event, identity mechanics
backend-go/internal/compatibility/    legacy reads and migration only

frontend/src/app/                     router, providers, app shell, single SSE mount
frontend/src/features/                business feature public entries and private implementation
frontend/src/shared/                  proven cross-feature UI and pure utilities

tests/                                cross-module contracts, fixtures, recovery, smoke
tools/archcheck/                      executable dependency policy
.artifacts/quality/                   ignored generated test and coverage evidence
config/                               committed schema and non-secret example
```

The target becomes current only after iteration 14 passes its full gate. Until
then, use code and the current package map in `BACKEND_ARCHITECTURE.md` as
implemented truth.

## 2. Project Runtime Data

These directories are runtime state or generated local data, not product contracts:

```text
projects/          learner projects, conversations, assets, sources, tasks, runs
preferences.md     local learner-owned global preferences (gitignored)
dist/              built desktop/backend artifacts
learning/          local learning artifacts / workspace data
repro-shell/       local repro helpers
```

Do not write long-lived product rules here.

## 3. Source-Of-Truth Documents

These are the current contract layers:

```text
docs/INDEX.md
docs/00-product-and-architecture/
docs/01-iterations/ current delivery baseline only
docs/99-archive/   ← historical only, never current contract
AGENTS.md
```

Within `docs/`:

```text
00-product-and-architecture/   long-lived facts, ADRs, rules, repository map
01-iterations/                 current delivery scope, API contracts, tests, acceptance
99-archive/                    retired ADRs, iterations, designs, and references
```

## 4. Reference Attachments And Design Inputs

These are useful, but they are not executable truth and should not override code:

```text
frontend-designs/v3/           mock pages + DESIGN.md for alignment/reference
docs/00-product-and-architecture/ALIGNMENT_MODE.md
docs/99-archive/references/     legacy visual references; historical only
荔枝读书.png                    standalone visual reference
research/                      raw research / source material
```

Rule:

```text
reference attachments may explain intent
they do not by themselves prove a feature is implemented
new alignment handoff should use Markdown, not HTML
```

## 5. Legacy / Archive / Superseded Material

These should be treated as archive or migration reference, not fallback product:

```text
frontend/legacy/               old static frontend implementation
docs/99-archive/               retired docs; excluded from default AI context
```

Current stance:

```text
frontend/legacy/ is no longer a product fallback
```

If we keep them, we keep them for comparison or migration history, not as a second UI to maintain.

## 6. Current Reality Check

### Database

Current backend persistence is:

```text
file-first
```

Observed stores:

```text
projects/<slug>/... files
explain/confusions.json
intro/assessment.json
explain/manifest.json + explain/pages/*.md
practice/tasks.json + practice/answer-key.json
practice/attempts/*.json
practice/submissions/*.json
practice/evaluations/*.json + *.md
progress/events.jsonl + progress/summary.json
summary/flashcards.json
summary/flashcard-progress.json
runs/_index/*.json
preferences.md
```

There is no active SQLite / Postgres / MySQL / embedded DB integration in the current codebase.
Any future database is documented only as a possible acceleration/index layer, not present runtime truth.

### Current Learning Workspace Snapshot

Implemented in code today:

```text
Go backend + React frontend
conversation-first API teacher with durable response recovery
visible asynchronous CLI assistant tasks and sealed inputs
versioned intro/body/practice assets plus generated deliverables
source upload, immutable revisions, and assistant parsing
single learner-owned global preferences file
legacy five-zone and agent invoke/session compatibility
Explain confusion CRUD
Practice task reading + batch submit + evaluation reading
Summary flashcard reading + grading
Markdown editor for learner-owned files
file-first project/runs persistence
```

Compatibility retained intentionally:

```text
old projects may still contain `memory/`, Summary, and Extend files
those files remain recoverable but are not active teacher assets or prompt memory
the `/memory` frontend route redirects to `/preferences`
```

### Iteration 04 Additions

Implemented in code:

```text
evidence-based Intro assessment with generated prerequisite summaries
manifest-owned Explain pages with follow-up parent links
six Practice question types with private answer-key checking
project-local idempotent growth events
legacy Explain and Practice readers
```
