# System Architecture

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: runtime structure for the current LLL foundation.

## Current Context

LLL is currently a local-first web application with two runtime layers:

```text
frontend/
Browser console for composing tasks, viewing logs, and reading rendered output.

backend/
Node HTTP service that launches Claude Code, streams events, and serves the frontend.
```

## Runtime Flow

```text
User input in frontend
-> POST /api/tasks
-> backend launches Claude Code with print + stream-json
-> backend parses stdout/stderr and broadcasts SSE updates
-> frontend task monitor updates terminal and rendered result views
```

## Near-Term Expansion

The architecture is expected to grow into:

```text
task persistence
session history
study artifact storage
provider abstraction beyond Claude Code
role-specific agents and reusable prompt pipelines
```
