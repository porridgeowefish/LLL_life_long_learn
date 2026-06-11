# ADR-0003 Backend Session And Project Runtime

Status: accepted  
Date: 2026-06-08

## Context

LLL started with a minimal backend that launches Claude Code once per task and streams a combined result buffer.

That is insufficient for the intended product direction because the product now requires:

```text
project-based learning containers
real terminal visibility
interactive follow-up
append-only session history
file-path-based multi-agent coordination
project and learner memory growth
```

## Decision

Adopt a backend runtime shaped around:

```text
filesystem-first project truth
Project -> Zone -> Session -> Turn -> Artifact domain structure
real Claude Code terminal execution
PTY-backed session control as the target interaction model
append-only session history instead of single result replacement
explicit artifact promotion into project files
separate raw run data from curated learning artifacts
separate project memory from learner memory
```

When an agent is invoked:

```text
backend resolves the selected project and zone
backend resolves predecessor file paths
backend resolves agent rules and memory snapshots
backend assembles a full prompt package
backend launches a real Claude Code terminal session
backend captures run files and session turns
backend writes curated outputs back into project files
```

## Consequences

- The backend ceases to be only a task runner and becomes a local learning orchestration runtime.
- PTY integration becomes a strategic requirement for full follow-up support.
- The frontend must move from task snapshots to session- and turn-oriented views.
- Persistence strategy becomes file-first rather than database-first.
- Summary content becomes protected curated knowledge rather than a direct terminal sink.

## Alternatives Considered

- Keeping the single task plus resultText model
- Embedding a simulated terminal instead of launching the real Claude terminal
- Building agent coordination around an opaque message bus before file-path contracts
- Treating raw run output and project knowledge files as the same persistence layer
