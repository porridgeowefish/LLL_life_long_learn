# Iteration 02: Local Learning Workbench Foundation

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: this directory defines the second LLL delivery slice.

## Goal

Upgrade LLL from a task-oriented Claude Code orchestrator into the minimum viable skeleton of a local learning workbench.

This slice should prove that:

```text
learning is organized by project rather than one-off task
projects support subprojects and fixed learning zones
an agent can launch a real Claude Code terminal session
session history supports append-only follow-up
raw runs and curated project outputs are stored separately
project memory can begin to accumulate locally
```

## Scope

Included:

```text
project and subproject filesystem structure
fixed learning zones: Intro, Explain, Practice, Extend, Summary
project tree backend and frontend skeleton
Explain Agent as the first production agent role
real Claude Code terminal launch for agent invocation
append-only session and turn model
follow-up support within the same session model
runs/ raw trace storage
explain/ curated output storage
editable summary file protection
project memory initial implementation
```

Excluded:

```text
cloud deployment
user accounts and permissions
multi-tenant collaboration
provider abstraction beyond Claude Code
full multi-agent automation across every zone
advanced learner-memory inference
database-first persistence
```

## Success Shape

By the end of this iteration, a user should be able to:

```text
create a learning project
create a subproject under it
open the Explain zone
invoke Explain Agent
see a real Claude Code terminal session launch
review session history in append-only form
promote generated output into explain/output.md
edit summary/summary.md without automatic overwrite
reopen the project and still see project structure, runs, and outputs
```
