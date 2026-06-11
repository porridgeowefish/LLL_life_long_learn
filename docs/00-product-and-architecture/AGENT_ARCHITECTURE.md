# Agent Architecture

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: rules for using AI runtimes as controlled study operators.

## Current Agent Role

Today LLL treats Claude Code as an external task runtime:

```text
The browser prepares a structured prompt.
The backend executes the runtime.
The UI observes status, logs, and final study output.
```

## Desired Direction

Future slices should support:

```text
visible agent identities by learning role
direct Claude Code terminal launch instead of hidden-only execution
prompt assembly from predecessor project file paths
zone-specific behavior rules and output contracts
file-path-based coordination between agents
subproject-aware agent invocation
run history and replay
memory-aware prompt adaptation
```

## Execution Principle

LLL should not replace the real Claude Code execution surface.

Use:

```text
Claude Code terminal as the live execution surface
LLL panel as the orchestration, observation, indexing, and editing surface
```

When an agent is invoked:

```text
LLL resolves the active project context
LLL resolves predecessor node file paths
LLL builds a role-specific prompt with behavior rules and output rules
LLL launches a real Claude Code terminal session
LLL auto-injects that prompt
LLL captures run artifacts and updates the project panel
```

## Agent Interface Principle

The primary interface between agents should be files and file paths.

Prefer:

```text
shared project folders
well-known target files
readable run artifacts
editable summary documents
```

Avoid early reliance on:

```text
opaque in-memory message buses
hidden agent-to-agent state
non-inspectable orchestration
```

## Safety Principle

Agent output must stay inspectable:

```text
show raw terminal output
show rendered output
preserve the original prompt
record cancellation and failure states
show which predecessor files were supplied
show which output files were written
```
