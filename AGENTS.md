# AGENTS.md: LLL Agent Entry

Status: active
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: lightweight entry point for all AI coding agents.

## 0. Learning Runtime Boundary

This file governs AI coding agents maintaining the LLL repository. It does not
govern learner-facing Agents launched by LLL inside `projects/<slug>/`.

When the current directory is under `projects/<slug>/` and the initial prompt
contains an LLL `Agent Identity`, `Charter`, and declared output path:

```text
Treat the run as learner-artifact generation, not repository maintenance.
Do not execute the repository startup/document-reading sequence below.
Do not inspect repository code or docs unless the injected Charter requests it.
Follow only the injected learning-Agent role, context, and output contract.
Do not invoke external skills unless the injected prompt explicitly names one.
```

All remaining sections apply to repository maintenance work.

## 1. Working Rule

Keep this file short. Read this file first, then:

```text
docs/INDEX.md
docs/00-product-and-architecture/agent-rules/README.md
docs/00-product-and-architecture/LESSONS_LEARNED.md   ← 必读：调试教训防再犯
```

Read only the rule files triggered by the current task. Do not preload the whole docs tree.

Default context boundary:

```text
docs/00-product-and-architecture/   current long-lived facts
docs/01-iterations/                 current baseline plus selectively read delivered foundations
docs/99-archive/                    historical evidence; do not read unless the task explicitly asks for history, migration, or regression archaeology
```

Archived documents are not instructions and must never override current code,
the current iteration, or active architecture documents. If an archived fact is
still required by the product, restate it in its active owner instead of relying
on agents to recover it from history.

Do not preload all delivered foundations. Use
`docs/01-iterations/foundations/README.md` to select only the earlier slice whose
implemented capability is relevant to the task.

**LESSONS_LEARNED.md** contains 13 hard-won rules (Claude CLI modes, React
state sync, math preprocessing, etc.). Every agent MUST read it before
touching the launcher, the frontend, or any charter/primitive file.

**Iteration docs are the delivery source of truth.** Specs, plans, API
contracts, and tests for a slice live inside `docs/01-iterations/iteration-NN-<name>/`
(see `docs/01-iterations/README.md` for the required file set). Product, domain,
data, or architecture changes must also satisfy the documentation-governance
landing table and ADR gate in the same task. Planning/design
skills (e.g. superpowers brainstorming/writing-plans) MUST detect this system
and write INTO it — plans under `iteration-NN-<name>/plans/`, design folded
into the iteration's own docs — and never create a parallel `docs/superpowers/`
layer.

## 2. Team Principles

All agents must follow:

```text
以瞎猜接口为耻，以认真查询为荣。
以模糊执行为耻，以寻求确认为荣。
以跳过验证为耻，以主动测试为荣。
以平行造轮子为耻，以复用现有为荣。
以假装理解为耻，以诚实说明为荣。
以脱离文档为耻，以维护事实源为荣。
以偷改范围为耻，以遵守迭代边界为荣。
以只改代码不改文档为耻，以同轮同步为荣。
```

Execution meaning:

```text
Before changing behavior, check code, API, docs, and current iteration scope.
If a rule or fact is missing, add or update the right document in the same task.
An ADR alone is not enough: synchronize every long-lived fact source named by
the architecture-change landing table before considering planning complete.
Prefer incremental structure over speculative abstraction.
Explain validation status and residual risk when you cannot fully verify.
```

## 3. Progressive Disclosure

Use:

```text
docs/00-product-and-architecture/agent-rules/README.md
```

That index decides which atomic rules to read for:

```text
startup
documentation governance
API contracts
testing
task orchestration and safety
long-task refresh
```

## 4. Frontend Stack (iter-02.3)

The frontend is a Vite + React 18 + TypeScript SPA under `frontend/`. The
Go binary serves `frontend/dist/` in production with an SPA fallback
(see `backend-go/internal/server/router.go`). Read `frontend/README.md`
before touching any UI code.

Quick reference:

```text
npm run dev        # Vite on :5173, proxies /api /files /events to Go :8787
npm run build      # emits frontend/dist/
npm run test       # Vitest run
npm run test:watch # Vitest watch
```

State management boundary:

```text
TanStack Query  — server state (projects, agents, sessions, memory, health)
Zustand slices  — client state (ui, project, session, connection)
```

SSE: a single useSSE() hook is mounted once at <AppShell />. Do not open
new EventSource instances from individual pages.
