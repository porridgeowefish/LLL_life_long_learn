# Agent Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: rules for using AI runtimes as controlled study operators.

## Current Agent Role

LLL treats the configured native AI CLI as an external task runtime:

```text
The browser prepares a structured prompt.
The Go backend launches the selected runtime.
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
flat project-root resolution independent of sidebar classification
run history and replay
memory-aware prompt adaptation
```

## Execution Principle

LLL should not replace the real Claude Code execution surface.

Every capability registered and presented as an `Agent` must use the selected
native Agent CLI execution path. Direct HTTP model providers are reserved for
explicitly lightweight AI helpers such as Ask-AI; they must not impersonate a
registered Agent or bypass Session/run/terminal observability.

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

## Project-Type Boundary

The five learning-role agents and their zone contracts apply to
`system-learning` projects. A `discipline-map` has no learning zones.

Iteration 07 provides separate advisor and map-generation capabilities, but
neither capability becomes a sixth learning stage:

```text
choice advisor     -> recommends a project type; never creates
encyclopedia agent -> explicitly launches the selected native CLI and writes or updates one discipline overview
five learning agents -> operate only inside system-learning zones
```

Within a system-learning flow, Intro records a concise explanation of each
detected prerequisite gap in `intro/assessment.json`. The UI presents it as
diagnostic context without a second AI action. Explain then supplies necessary
background naturally. A gap does not become a project; only a separately confirmed system-learning commitment
has no durable map-parent relation; sidebar folders are not added to prompt context.

The registered `encyclopedia` agent is project-type-bound to `discipline-map`,
has no allowed learning zone, and owns a versioned charter that reserves
level-three headings for independently learnable key concepts or branches.
It is a project-level Agent, not a direct-provider shortcut: invocation still
creates the normal Session and run package and opens the real CLI terminal.

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
