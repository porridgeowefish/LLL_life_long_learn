# API Contract Strategy

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: backend/frontend contract strategy for LLL.

## Current APIs

```text
GET  /api/health
GET  /api/tasks
POST /api/tasks
POST /api/tasks/:id/cancel
GET  /api/events
```

## Contract Rule

For the current Node foundation:

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
```
