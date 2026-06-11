# Domain Model

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: top-level domain boundaries for LLL.

## Core Objects

```text
WorkspaceProject
The durable local learning project folder.

LearningZone
One fixed phase inside a project: Intro, Explain, Practice, Extend, or Summary.

AgentRole
A visible role with a charter, behavior rules, allowed zones, and output targets.

ClaudeSession
One live or completed Claude Code terminal conversation linked to a project and zone.

SessionTurn
One append-only unit inside a session, such as a user turn or assistant turn.

RunRecord
The raw execution trace for a session, including prompt and terminal output files.

LearningArtifact
Curated project output written into zone files or assets.

MemorySnapshot
Learner-level or project-level memory used to improve later runs.

Iteration
A scoped engineering delivery slice for the product itself.
```

## Boundaries

```text
Frontend owns navigation, reading, editing, session display state, and rendering.
Backend owns project indexing, process launch, session lifecycle, artifact writes, memory updates, and filesystem boundaries.
Docs own product scope, architecture, and delivery contracts.
Raw research materials inform tasks but are not executable truth.
```
