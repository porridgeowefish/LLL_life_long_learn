# System Architecture

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: runtime structure for the current LLL foundation.

## Current Context

LLL is currently a local-first web application with two runtime layers:

```text
frontend/
Browser console for composing tasks, viewing logs, and reading rendered output.

backend/
Node HTTP service that launches Claude Code, streams events, and serves the frontend.
```

This is the implementation reality today, but it is no longer the target product shape.

## Target Runtime Shape

LLL should evolve into a local learning workbench with three interacting planes:

```text
Project plane
Filesystem-based learning projects, subprojects, zone files, assets, and memory.

Execution plane
Real Claude Code terminal sessions launched and tracked by the backend.

Experience plane
Frontend project tree, large learning panel, follow-up session view, and agent side rail.
```

## Target Runtime Layers

```text
frontend/
Project tree navigation, editing, session display, artifact reading, and agent controls.

backend/
Project indexing, prompt assembly, agent orchestration, PTY-backed Claude sessions, artifact writing, memory updates, and API/event delivery.

filesystem workspace/
Canonical local truth for projects, runs, artifacts, summaries, and memory.
```

## Target Runtime Flow

```text
User selects project + zone + agent
-> frontend requests agent invocation
-> backend resolves predecessor file paths and memory snapshots
-> backend assembles prompt package
-> backend launches a real Claude Code terminal session
-> backend captures session events and run files
-> frontend shows live session history and large reading/editing surface
-> backend writes curated outputs into project files
-> user may follow up in the same session
-> session turns append instead of replacing prior results
```

## Architectural Rules

Use:

```text
filesystem-first project truth
project-based learning units
append-only session history
real terminal visibility
file-path-based agent coordination
separation between raw runs and curated artifacts
```

Do not optimize for:

```text
cloud deployment
accounts
multi-tenant collaboration
hidden-only orchestration
single-buffer result replacement
```

## Near-Term Expansion

The architecture is expected to grow into:

```text
project persistence through filesystem structure
session and turn history
study artifact indexing
role-specific agent invocation
project and learner memory
interactive follow-up through PTY-backed sessions
```
