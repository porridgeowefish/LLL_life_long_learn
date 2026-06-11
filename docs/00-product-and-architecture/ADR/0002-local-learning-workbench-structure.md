# ADR-0002 Local Learning Workbench Structure

Status: accepted  
Date: 2026-06-08

## Context

LLL started as a local orchestrator for launching Claude Code tasks and rendering results.

That shape is not enough for the intended product direction.
The product is not a cloud service and should not be optimized around accounts, remote deployment, or hidden orchestration.

The desired direction is a local learning workbench where:

```text
one learning task becomes one project
Claude Code remains the authentic execution surface
LLL provides visual organization, observation, editing, and memory
multiple agents collaborate through file paths
subprojects support recursive topic deep dives
```

## Decision

Adopt the following long-lived structure:

```text
project folder as the primary learning unit
five fixed learning zones: Intro, Explain, Practice, Extend, Summary
real Claude Code terminal invocation for agent execution
LLL panel as the observation and knowledge organization surface
file paths as the primary interface between agents and project context
global learner memory plus project memory
subprojects as first-class nested learning projects
```

When an agent is invoked:

```text
LLL resolves predecessor files
LLL composes the prompt with file paths, role rules, and output rules
LLL launches a real Claude Code terminal
LLL injects the prepared prompt
LLL indexes outputs back into the project panel
```

## Consequences

- The product architecture shifts from task monitor to local learning workbench.
- The UI should visualize project trees, zones, agents, runs, assets, and memory.
- Agent design should favor explicit roles and file outputs over opaque internal orchestration.
- Summaries become curated editable knowledge instead of raw terminal artifacts.
- Practice and Extend zones require behavior constraints, not just output generation.
- Long-term memory becomes a core product capability instead of an optional add-on.

## Alternatives Considered

- Keeping LLL as a single-task prompt console
- Replacing the real Claude terminal with a fully embedded fake terminal
- Building multi-agent communication around a custom RPC bus first
- Modeling the product as a cloud-first SaaS learning platform
