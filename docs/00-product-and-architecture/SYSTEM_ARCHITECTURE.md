# System Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: long-lived runtime boundaries and approved target architecture; code owns current implementation details.

## Current Runtime

LLL is a local-first web application:

```text
frontend/          Vite + React 18 + TypeScript SPA
backend-go/        Go HTTP server, project runtime, file stores, and agent launcher
projects/          canonical local project and artifact data
agents/            registry, charters, and reasoning primitives
```

The Go binary serves `frontend/dist/` in production. The file system is durable
truth; in-memory indexes accelerate reads and can be rebuilt.

## Runtime Planes

```text
Project plane
Flat typed project directories, artifacts, runs, progress, and memory.

Execution plane
Real local AI runtime sessions launched through explicit project contracts.

Experience plane
Frontend project tree, discipline overview, five-zone learning surfaces, and controls.
```

## Project-Type Routing

```text
discipline-map
  -> open one discipline overview
  -> render as an explicit overview row in an existing sidebar folder object
  -> no five-zone navigation
  -> may start creation of an ordinary system-learning project

system-learning
  -> open Intro / Explain / Practice / Extend / Summary
  -> invoke zone-bound learning agents
  -> render generated prerequisite-gap summaries without a second AI request or new project
```

The sidebar combines indexed project objects with the persisted folder layout.
A bound discipline map opens from the folder title and is omitted from child
rows; system-learning projects remain ordinary movable folder members. It never
derives navigation nodes or projects from overview Markdown. The map page derives
a Word-style table of contents from the heading hierarchy and attaches a prefill
action beside each contracted research-area heading in the explanatory body.

## Execution Flow

System-learning invocation:

```text
learner selects project + zone + agent
-> backend validates system-learning type and zone compatibility
-> backend resolves predecessor files and memory
-> backend assembles and launches the local AI runtime
-> run files remain raw; curated outputs land in zone contracts
```

Discipline-map generation or update:

```text
learner explicitly requests generation/update
-> backend validates discipline-map type
-> registered encyclopedia agent supplies its versioned charter
-> normal session and prompt package are created
-> selected native Agent CLI opens in a visible terminal
-> runtime writes the single overview artifact
-> filesystem watcher emits an overview artifact update and the page refetches
-> other project creation and progress changes do not trigger regeneration
```

The encyclopedia agent is project-type-bound and has no learning zone; no sixth
learning zone is implied.

## Architectural Rules

```text
filesystem-first truth
explicit project types
learner-owned product-shape choice
real project objects only in navigation
separation of raw runs and curated artifacts
append-only execution history where supported
missing legacy project type defaults to system learning
```

Do not optimize for cloud accounts, multi-tenant collaboration, hidden-only
orchestration, or database-first project modeling.

## Delivery State

The two-type routing, encyclopedia-agent overview generation, flat project
storage, stateless type-advice conversation, and map-backed folder navigation
are implemented by iteration 07.
