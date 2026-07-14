# Repository Map

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-11  
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

## 2. Project Runtime Data

These directories are runtime state or generated local data, not product contracts:

```text
projects/          learner projects, zone files, runs/, memory, outputs
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
docs/01-iterations/
docs/99-archive/   ← historical only, never current contract
AGENTS.md
```

Within `docs/`:

```text
00-product-and-architecture/   long-lived facts, ADRs, rules, repository map
01-iterations/                 delivery scope, API contracts, test plans, acceptance
99-archive/                    retired material
```

## 4. Reference Attachments And Design Inputs

These are useful, but they are not executable truth and should not override code:

```text
frontend-designs/v3/           mock pages + DESIGN.md for alignment/reference
docs/00-product-and-architecture/ALIGNMENT_MODE.md
docs/00-product-and-architecture/WORKBENCH_ALIGNMENT_REVIEW.html          legacy visual reference
docs/00-product-and-architecture/LEARNING_AGENTS_USER_STORY_ALIGNMENT.html legacy visual reference
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
frontend-designs/v2/           older mock set
docs/99-archive/               retired docs
```

Current stance:

```text
frontend/legacy/ is no longer a product fallback
frontend-designs/v2/ is historical reference only
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
```

There is no active SQLite / Postgres / MySQL / embedded DB integration in the current codebase.
Any future database is documented only as a possible acceleration/index layer, not present runtime truth.

### Iteration 03 Alignment Snapshot

Implemented in code today:

```text
Go backend + React frontend
five zones and agent registry
agent invoke / session / follow-up flow
Explain confusion CRUD
Practice task reading + batch submit + evaluation reading
Summary flashcard reading + grading
Markdown editor for learner-owned files
file-first project/runs persistence
```

Not fully aligned with iter-03 contract yet:

```text
project creation still uses why/current/target/standard, not the new 4-field contract
iter-03 API contract contains endpoints/shapes not yet matched by code
Explain "统一提问" is not yet wired end-to-end into one invoke with sourceRefs from the UI
some docs still describe legacy/fallback material as if it were active
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
