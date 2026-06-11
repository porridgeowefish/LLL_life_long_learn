# AGENTS.md: LLL Agent Entry

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: lightweight entry point for all AI coding agents.

## 0. Working Rule

Keep this file short. Read this file first, then:

```text
docs/INDEX.md
docs/00-product-and-architecture/agent-rules/README.md
docs/00-product-and-architecture/LESSONS_LEARNED.md   ← 必读：调试教训防再犯
```

Read only the rule files triggered by the current task. Do not preload the whole docs tree.

**LESSONS_LEARNED.md** contains 11 hard-won rules (Claude CLI modes, React
state sync, math preprocessing, etc.). Every agent MUST read it before
touching the launcher, the frontend, or any charter/primitive file.

## 1. Team Principles

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
Prefer incremental structure over speculative abstraction.
Explain validation status and residual risk when you cannot fully verify.
```

## 2. Progressive Disclosure

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

## 3. Frontend Stack (iter-02.3)

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
