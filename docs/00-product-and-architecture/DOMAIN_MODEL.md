# Domain Model

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: long-lived domain objects, relationships, and ownership boundaries for LLL.

## Core Objects

```text
WorkspaceProject
The durable local project object. It has exactly one project type.

DisciplineMap
A WorkspaceProject that owns one discipline overview and is bound to a sidebar folder.

ProjectFolder
The single navigation/classification container. It may bind one DisciplineMap
as its overview and may reference zero or more SystemLearningProjects by slug.

SystemLearningProject
A WorkspaceProject that owns the five-zone learning workflow.

LearningZone
One fixed phase inside a SystemLearningProject: Intro, Explain, Practice, Extend, or Summary.

OverviewTopic
A subject named inside a discipline overview. It is content, not a project or sidebar node.

AgentRole
A visible role with a charter, behavior rules, allowed execution targets, and output targets.

RuntimeSession
One live or completed local AI runtime conversation linked to a project and execution target.

RunRecord
The raw execution trace for a session, including prompt and terminal output files.

LearningArtifact
Curated project output, such as a discipline overview or a zone-owned artifact.

MemorySnapshot
Learner-level or project-level memory used to improve later runs.

Iteration
A scoped engineering delivery slice for the product itself.
```

## Relationships

```text
WorkspaceProject
├─ DisciplineMap
│  ├─ owns exactly one learner-facing discipline overview
│  └─ may bind one ProjectFolder and render as its explicit overview row
└─ SystemLearningProject
   └─ owns Intro / Explain / Practice / Extend / Summary

ProjectFolder
├─ may expose one bound DisciplineMap as an explicit overview row
└─ classifies zero or more SystemLearningProjects without owning either project type

OverviewTopic
├─ is represented by an H3 key concept or branch in the overview body
└─ may prefill a future SystemLearningProject from its adjacent action after learner confirmation
```

An `OverviewTopic` never becomes a project merely because AI wrote its name.
Project identity begins only after an explicit creation action succeeds.
The Word-style table of contents is a derived navigation view of the same
heading hierarchy, not another topic collection or ownership tree.

## Ownership Boundaries

```text
Frontend owns explicit type choice, navigation, reading, editing, and confirmation UI.
Backend owns project indexing, type validation, filesystem boundaries, sessions, and artifact writes.
Project files own durable local learning state.
Iteration documents own not-yet-implemented delivery contracts.
Code and runnable schemas own current behavior.
```

## Compatibility

Projects without a persisted project type are interpreted as
`system-learning`. Project type is immutable in iteration 07; changing type
creates a new related project instead of mutating the existing object.

A prerequisite gap is transient learning context, not a project object. Intro
records a concise generated summary directly in its assessment; it does not
expose a second supplement action. A system-learning project exists only after
the learner confirms a durable learning commitment. The created project
remains physically flat; there is no map-parent field, subproject object, or
recursive ownership. Its ordinary folder membership may place it beneath the
map-backed folder in navigation.
