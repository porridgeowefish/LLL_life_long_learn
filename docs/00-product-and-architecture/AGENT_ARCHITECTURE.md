# Agent Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-08-31
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

All execution is mediated by the `agentexecution.Service` domain boundary.
The encyclopedia Agent and teacher-delegated assistant tasks therefore share
the same runtime selection, visible terminal, working-directory activation,
prompt injection, error reporting, and exit lifecycle. Their output contracts
differ, but their process-launch implementation does not.

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
encyclopedia agent -> explicitly launches the selected native CLI and writes or updates the overview plus topic-boundary catalog
five learning agents -> operate only inside system-learning zones
```

Within a system-learning flow, Intro records a concise explanation of each
detected prerequisite gap in `intro/assessment.json`. The UI presents it as
diagnostic context without a second AI action. Explain then supplies necessary
background naturally. A gap does not become a project; only a separately confirmed system-learning commitment
has no durable map-parent relation; sidebar folders are not added to prompt context.

The registered `encyclopedia` agent is project-type-bound to `discipline-map`,
has no allowed learning zone, and owns a versioned charter that first plans the
whole discipline architecture, and reserves H3 for major chapters and H4 for
independently learnable topics. It does not own the learner's task selection or order.
It is a project-level Agent, not a direct-provider shortcut: invocation still
creates the normal Session and run package and opens the real CLI terminal.

## Scope Ownership

The encyclopedia Agent owns objective topic boundaries before project creation:
goal, inclusion, exclusion, prerequisites, owned concepts, and reused concepts.
It writes these to `discipline-topics.json` alongside the overview.

Intro owns learner calibration. For a ready map-origin scope it may adapt
prerequisite support, explanation depth, examples, scaffolding, and practice
difficulty, but it cannot broaden the objective boundary. For a standalone
draft it finalizes `learning-scope.json` after survey evidence exists.

Explain, Practice, Extend, and Summary consume the same scope. Included and
owned concepts receive full treatment; prerequisites and reused concepts receive
minimum support; excluded sibling content does not become a core artifact.

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

## Iteration 13 Teacher And Assistant Roles

ADR-0012 separates product role from runtime mechanism:

- `teacher` is the real-time API classroom voice. It teaches with one unified
  prompt and five soft methods, and may use only `delegate_learning_work` after
  disclosing the work and receiving a later learner response;
- `assistant` is substantial asynchronous work executed by a visible native
  CLI. It consumes a sealed task envelope and writes only attempt staging;
- `LLL` owns authorization, paths, task state, queueing, validation, promotion,
  notifications, and recovery.

The API teacher is not a registered CLI Agent and does not violate the rule
that registered Agents use native CLI execution. Annotation Ask AI remains a
separate lightweight provider helper with no teacher methods or tools.

The active generated result contract changes from zone-bound direct writes to
generic declared deliverables plus optional intro/body/practice candidates.
Formal files are committed only by Go after validation and merge. Legacy Agent
charters remain compatibility inputs until migration; new assistant prompts are
grounded in the approved objective, exact conversation range, asset bases, and
authorized source revisions.

Raw hidden chain of thought is never part of the product contract. Only a
provider-designated reasoning summary may be shown or persisted.
