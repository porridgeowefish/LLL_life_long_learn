# API Contract Strategy

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: backend/frontend contract strategy for LLL.

## Current APIs

```text
GET  /api/health
GET  /api/tasks
POST /api/tasks
POST /api/tasks/:id/cancel
GET  /api/events
```

These APIs describe the current task-launcher implementation, not the target backend contract.

## Target Resource Groups

The backend contract should evolve toward resource-oriented groups:

```text
health
projects
zones
agents
sessions
turns
artifacts
memory
files
events
```

Illustrative surface:

```text
GET  /api/health
GET  /api/projects
GET  /api/projects/:id
GET  /api/projects/:id/tree
GET  /api/projects/:id/zones/:zone

GET  /api/agents
POST /api/agents/:id/invoke

GET  /api/sessions/:id
POST /api/sessions/:id/follow-up
POST /api/sessions/:id/cancel

POST /api/artifacts/promote
POST /api/memory/project/:id/refresh
GET  /api/events
```

## Contract Rule

For the current Node foundation and next slices:

```text
backend route behavior is the runtime truth
iteration API docs explain usage, payload shape, and edge cases
frontend should consume the documented JSON shape directly
```

## Next Contract Upgrade

When the API surface grows, move to:

```text
schema-defined request and response contracts
generated client types
versioned iteration contract updates
typed session and event payloads
```
