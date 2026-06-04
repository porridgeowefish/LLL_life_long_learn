# LLL

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: this README is a project entry point; domain facts live in `docs/`.

LLL is a local AI learning lab for orchestrating study tasks, dispatching Claude Code work, monitoring task execution, and turning model output into readable study artifacts with Markdown and Mermaid rendering.

## What Exists Today

- Backend task orchestrator:
  `backend/server.js`
- Frontend operator console:
  `frontend/index.html`, `frontend/app.js`, `frontend/styles.css`
- Study source materials:
  `research/raw/`
- Documentation system:
  `docs/`

## Repository Map

```text
backend/                      Node backend that launches and monitors Claude Code tasks
frontend/                     Browser console for composing prompts and watching runs
docs/                         Product, architecture, iteration, and archive documents
research/raw/                 Local raw study materials, not tracked by git
AGENTS.md                     Shared instructions for AI coding agents
CLAUDE.md                     Thin Claude-specific loader
```

## Quick Start

```bash
cd D:\2_Study\LLL
npm start
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
