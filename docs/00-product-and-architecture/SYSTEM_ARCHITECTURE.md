# System Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
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
  -> switch between one discipline overview and one learning plan
  -> render as an explicit overview row in an existing sidebar folder object
  -> no five-zone navigation
  -> may start creation of an ordinary system-learning project

system-learning
  -> open Intro / Explain / Practice / Extend / Summary
  -> load one objective learning-scope.json into every Agent prompt
  -> invoke zone-bound learning agents
  -> render generated prerequisite-gap summaries without a second AI request or new project
```

The sidebar combines indexed project objects with the persisted folder layout.
A bound discipline map opens from the folder title and is omitted from child
rows; system-learning projects remain ordinary movable folder members. It never
derives navigation nodes or projects from overview Markdown. The map page derives
a Word-style table of contents from the H2/H3/H4 hierarchy and attaches a prefill
action beside each contracted H4 topic in the explanatory body. A second top-level
tab renders the learner-owned ordered task list.

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
-> runtime writes the overview artifact
-> runtime also writes the validated discipline topic-boundary catalog
-> learner adds selected H4 topics to the ordered task-plan API
-> filesystem watcher emits artifact updates and the active page refetches
-> other project creation and progress changes do not trigger regeneration
```

The encyclopedia agent is project-type-bound and has no learning zone; no sixth
learning zone is implied.

Map deep-dive creation sends only map slug and topic ID. The backend resolves
the canonical catalog entry and persists a ready scope snapshot. Standalone
creation persists a draft scope. Intro changes learner adaptation, not a ready
map boundary; all later zones consume the same scope.

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

## Iteration 13 Target Runtime

ADR-0012 adds an application seam inside the existing Go process:

```text
Experience: one React teacher chat plus focused asset and source routes
Transport: thin HTTP/SSE handlers and the existing single global SSE stream
Application: teacher, annotation Q&A, task, dispatcher, asset, source, migration
Domain: conversation, task, run, asset, version, source, annotation rules
Infrastructure: provider adapters, file repositories, visible native CLI, IDs
```

The API teacher streams normal teaching and owns one narrow disclosed handoff
tool. The visible native CLI performs approved substantial work asynchronously.
LLL persists and mediates between them; neither runtime writes through the
other's private interface.

Conversation events, task files, source revisions, and asset versions remain
filesystem truth. A local dispatcher rebuilds its queue by scanning task files;
no external broker or database is introduced. Global SSE emits small
invalidation events, while REST recovers durable state.

This target is planned. The current project-type and five-zone execution flows
remain executable until iteration-13 migration and cutover tests pass.

## Delivery State

The two-type routing, encyclopedia-agent generation, flat project storage,
stateless type-advice conversation, and map-backed folder navigation are
implemented by iteration 07. Hierarchical overview planning and the separate
learning-plan view are implemented by iteration 11. Durable scope catalogs,
snapshots, and prompt enforcement are implemented by iteration 12.
