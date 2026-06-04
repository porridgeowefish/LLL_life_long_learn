# Domain Model

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: top-level domain boundaries for LLL.

## Core Objects

```text
StudyTopic
The user-defined subject being learned.

TaskRun
A single execution request sent to an AI runtime.

TaskLog
An ordered event stream emitted while a task runs.

StudyArtifact
The resulting Markdown, Mermaid, notes, flashcards, or plans produced by a run.

Iteration
A scoped engineering delivery slice for the product itself.
```

## Boundaries

```text
Frontend owns interaction, display state, and rendering.
Backend owns process launch, streaming, task lifecycle, and filesystem boundaries.
Docs own product scope, architecture, and delivery contracts.
Raw research materials inform tasks but are not executable truth.
```
