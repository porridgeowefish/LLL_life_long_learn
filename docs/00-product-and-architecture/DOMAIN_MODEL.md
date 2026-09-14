# Domain Model

Status: active
Owner: project maintainer
Last reviewed: 2026-09-14
Source of truth: long-lived domain objects, relationships, and ownership boundaries for LLL.

## Core Objects

```text
WorkspaceProject
The durable local project object. It has exactly one project type.

DisciplineMap
A WorkspaceProject that owns one discipline overview plus one learner-owned task plan and is bound to a sidebar folder.

ProjectFolder
The single navigation/classification container. It may bind one DisciplineMap
as its overview and may reference zero or more SystemLearningProjects by slug.

SystemLearningProject
A WorkspaceProject that owns one conversation-first LearningUnit.

LearningZone
A legacy compatibility phase: Intro, Explain, or Practice. It remains readable
for migration and rollback but is not an active target object. Historical
Extend/Summary folders are deliberately not LearningZone values.

LearningUnit
The active system-learning aggregate: one TeacherConversation, three
core TeachingAssets, generated artifacts, SourceMaterials, AssistantTasks, and
asset annotations.

LearningActivity
One idempotent, learner-visible progress event. It records an activity source
and contribution for heatmap aggregation; it is not a second conversation or
task lifecycle.

TeacherConversation
The one canonical learner-teacher dialogue that forms a LearningUnit.

TeachingAsset
One editable, versioned accumulated asset: intro, body, practice, or a declared
generated artifact.

SourceMaterial
A learner-provided local source with immutable revisions, privacy decisions,
and exactly one ready canonical Markdown citation (`content.md`) after supported
background parsing.

AssistantTask
A durable approved supporting-work objective, separate from its CLI RunAttempts.

AssetAnnotation
A quote- and version-anchored note or Ask-AI thread on a TeachingAsset.

OverviewTopic
A subject named inside a discipline overview with one objective TopicBoundary.
It is content, not a project or sidebar node.

TopicBoundary
The map-owned goal, inclusion, exclusion, prerequisite, concept ownership, and
reuse contract for one actionable topic. It may include a natural-language
`teachingOutline` injected as soft teacher guidance.

LearningScope
The system-learning project's durable objective content boundary. It is either
a map-topic snapshot or a standalone draft/finalized boundary.

DisciplineLearningPlan
An ordered learner-owned list of selected overview topics and per-task status. It is not project membership and does not create projects.

AgentRole
A visible native-CLI role with a charter, allowed inputs, and declared outputs.

Teacher
The low-latency API teaching role. It owns dialogue and one disclosed
delegation tool, but never executes CLI work or writes project files directly.

RuntimeSession
One live or completed native-CLI execution linked to a project and task target.

RunRecord
The raw execution trace for a session, including prompt and terminal output files.

LearningArtifact
Curated project output, such as a discipline overview, teaching asset, or
declared generated deliverable.

LearnerPreferences
The single workspace-global, learner-authored `preferences.md`. AI services may
consume a read-only snapshot but cannot create, update, or infer persisted preferences.

Iteration
A scoped engineering delivery slice for the product itself.
```

## Relationships

```text
WorkspaceProject
├─ DisciplineMap
│  ├─ owns exactly one learner-facing discipline overview and task plan
│  └─ may bind one ProjectFolder and render as its explicit overview row
└─ SystemLearningProject
   └─ owns exactly one LearningUnit

ProjectFolder
├─ may expose one bound DisciplineMap as an explicit overview row
└─ classifies zero or more SystemLearningProjects without owning either project type

OverviewTopic
├─ is represented by an H4 learnable topic in the new overview contract
├─ owns exactly one TopicBoundary in the generated topic catalog
└─ may snapshot that boundary into a future SystemLearningProject after learner confirmation

SystemLearningProject
├─ owns exactly one LearningScope
└─ owns one LearningUnit
   ├─ owns one TeacherConversation
   ├─ owns versioned TeachingAssets and AssetAnnotations
   ├─ references SourceMaterials
   └─ owns zero or more AssistantTasks, each with RunAttempts
   └─ owns zero or more LearningActivities in the progress stream

DisciplineLearningPlan
├─ orders exact OverviewTopic titles according to learner choice
└─ records planned / in-progress / completed without creating or owning projects
```

An `OverviewTopic` never becomes a project merely because AI wrote its name.
Project identity begins only after an explicit creation action succeeds.
The Word-style table of contents is a derived navigation view of the same
heading hierarchy, not another topic collection or ownership tree.
Legacy overviews without H4 keep their existing H3 topics actionable.
They must regenerate before creating a map-scoped deep dive because no canonical
topic boundary exists to snapshot.

## Ownership Boundaries

```text
Frontend owns explicit type choice, navigation, reading, editing, and confirmation UI.
Backend owns project indexing, type validation, filesystem boundaries, sessions, and artifact writes.
Project files own durable local learning state.
The current iteration owns the active delivery contract.
Code and runnable schemas own current behavior.
```

## Compatibility

Projects without a persisted project type are interpreted as
`system-learning`. Project type is immutable; changing type
creates a new related project instead of mutating the existing object.

A prerequisite gap is transient learning context, not a project object. Intro
records a concise generated summary directly in its assessment; it does not
expose a second supplement action. A system-learning project exists only after
the learner confirms a durable learning commitment. The created project
remains physically flat; there is no map-parent field, subproject object, or
recursive ownership. Its ordinary folder membership may place it beneath the
map-backed folder in navigation.

ADR-0012 defines the canonical LearningUnit inside the same flat
SystemLearningProject root. Migration adapts Intro, Explain, Practice, and
Explain Ask-AI data to assets and annotations. Summary and Extend remain
preserved historical records only, outside the runtime data model. Project type,
map provenance, folder classification, and scope
snapshot identity do not change. LearningUnit objects are implemented contracts
owned by code and the current iteration documents.

Project memory is not an active domain object. New projects do not create a
`memory/` directory. Existing per-project memory files remain untouched as
legacy user data but are excluded from prompt assembly and frontend navigation.
ADR-0013 defines the replacement global preference boundary.
