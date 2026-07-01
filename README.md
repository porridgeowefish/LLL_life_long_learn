# LLL

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: this README is a project entry point; domain facts live in `docs/`.

LLL is a local AI learning lab for orchestrating study tasks, launching agent sessions, and turning model output into readable study artifacts.

## What Exists Today

- Backend runtime:
  `backend-go/`
- Frontend app:
  `frontend/`
- Agent definitions:
  `agents/`
- Documentation system:
  `docs/`
- Local runtime/project data:
  `projects/`

## Repository Map

```text
backend-go/                   Go backend runtime and API
frontend/                     Vite + React workbench
frontend/legacy/              Legacy static frontend, archive/reference only
frontend-designs/             Mock/design reference, not executable truth
agents/                       Agent registry, charters, primitives
docs/                         Product, architecture, iteration, and archive documents
projects/                     Local learner projects and run artifacts
research/                     Raw study / reference materials
AGENTS.md                     Shared instructions for AI coding agents
CLAUDE.md                     Thin Claude-specific loader
```

## Quick Start

```bash
cd D:\2_Study\LLL
npm run build
go run ./backend-go/cmd/lll
```

Then open:

```text
http://localhost:8787/
```

## Reading Order

1. `docs/INDEX.md`
2. `AGENTS.md`
3. `docs/00-product-and-architecture/README.md`
4. The current iteration under `docs/01-iterations/`

## Fact Priority

When project documents disagree, use this order:

```text
1. Implemented code and runnable behavior
2. Current iteration documents in docs/01-iterations/
3. Long-lived architecture documents in docs/00-product-and-architecture/
4. Historical material in docs/99-archive/
5. Raw study materials in research/raw/
```
