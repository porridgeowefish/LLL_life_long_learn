# Agent Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: rules for using AI runtimes as controlled study operators.

## Current Roles

LLL separates teaching from heavy execution:

```text
teacher       fast API conversation; teaches and may propose one disclosed delegation tool
assistant     visible native CLI; performs learner-authorized substantial work
encyclopedia visible native CLI; generates or updates a discipline map
Ask AI        lightweight quote-grounded provider helper; no teacher methods or tools
LLL           owns authorization, paths, task state, atomic publication, and notification
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

When a native CLI role is invoked:

```text
LLL resolves the active project context
LLL captures conversation, asset, source, scope, and preference inputs
LLL builds a role-specific prompt with declared output rules
LLL launches the selected native Agent CLI through agentexecution.Service
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
declared learning artifacts
```

Avoid early reliance on:

```text
opaque in-memory message buses
hidden agent-to-agent state
non-inspectable orchestration
```

## Project-Type Boundary

```text
discipline-map  -> encyclopedia CLI + overview/topic/plan contracts
system-learning -> one API teacher conversation + optional assistant CLI tasks
```

A choice advisor may recommend a type but never creates. A discipline map has
no teacher conversation or learning-stage navigation. A system-learning unit
does not run legacy zone Agents as its active flow; Intro, Explain, and Practice
compatibility registrations remain migration inputs only. Summary and Extend
registrations are retired.

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

Explain and Practice consume the same scope. Included and
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

## Teacher And Assistant Roles

ADR-0012 separates product role from runtime mechanism:

- `teacher` is the real-time API classroom voice. It teaches with one unified
  prompt and five soft methods, and may use only `delegate_learning_work` after
  disclosing the work and receiving a later learner response;
- `assistant` is substantial asynchronous work executed by a visible native
  CLI. It consumes a captured task envelope and writes only attempt staging;
- `LLL` owns authorization, paths, task state, queueing, atomic publication,
  and notifications; the assistant owns output acceptance.

The API teacher is not a registered CLI Agent and does not violate the rule
that registered Agents use native CLI execution. Annotation Ask AI remains a
separate lightweight provider helper with no teacher methods or tools.

The active generated result contract changes from zone-bound direct writes to
generic declared deliverables plus optional intro/body/practice candidates.
Formal files are atomically published by Go after the assistant has accepted
them. Go retains version/merge handling for core assets but does not inspect
task conformance, file manifests, hashes, or generated content. Legacy Agent
charters remain compatibility inputs until migration; new assistant prompts are
grounded in the approved objective, exact conversation range, asset bases, and
authorized source revisions.

For confirmed `consolidate` work, the assistant uses captured conversation and
source evidence to update Intro for significance and Body for a self-contained
teaching manuscript plus critical-thinking conclusion. Practice is untouched
unless the approved task explicitly requests questions.

Raw hidden chain of thought is never part of the product contract. Only a
provider-designated reasoning summary may be shown or persisted.
