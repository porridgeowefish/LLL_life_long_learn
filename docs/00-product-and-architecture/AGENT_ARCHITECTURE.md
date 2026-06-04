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
prompt templates by study job type
agent role presets
artifact-specific pipelines
subagent orchestration for research, synthesis, and review
run history and replay
```

## Safety Principle

Agent output must stay inspectable:

```text
show raw terminal output
show rendered output
preserve the original prompt
record cancellation and failure states
```
