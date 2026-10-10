# Domain Model

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
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
The active system-learning aggregate: one TeacherConversation, versioned
review/practice TeachingAssets, TeacherNotes and durable TeacherNoteJobs,
SourceMaterials, restricted AssistantTasks, and review annotations.

LearningActivity
One idempotent, learner-visible progress event. It records an activity source
and contribution for heatmap aggregation; it is not a second conversation or
task lifecycle.

TeacherConversation
The one canonical learner-teacher dialogue that forms a LearningUnit.

TeachingAsset
One editable, versioned accumulated Markdown asset: review or practice.
TeacherNote is a separate immutable Markdown entry for a batch of successful turns.

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
The low-latency API teaching role. It owns dialogue, one disclosed
delegation tool, and an optional bounded web-search tool, but never executes
CLI work or writes project files directly.

RuntimeSession
One live or completed native-CLI execution linked to a project and task target.

RunRecord
The raw execution trace for a session, including prompt and terminal output files.

LearningArtifact
Curated project output, such as a discipline overview, review/practice asset, or teacher note.

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

A prerequisite gap is transient teacher context, not a project object or a separate Intro module. A system-learning project exists only after
the learner confirms a durable learning commitment. The created project
remains physically flat; there is no map-parent field, subproject object, or
recursive ownership. Its ordinary folder membership may place it beneath the
map-backed folder in navigation.

ADR-0012 defines the canonical LearningUnit inside the same flat
SystemLearningProject root. One-time migration concatenates Intro/Body into review, converts legacy
Practice to Markdown, and preserves old annotation quotes and version references. Summary and Extend remain
preserved historical records only, outside the runtime data model. Project type,
map provenance, folder classification, and scope
snapshot identity do not change. LearningUnit objects are implemented contracts
owned by code and the current iteration documents.

Project memory is not an active domain object. New projects do not create a
`memory/` directory. Existing per-project memory files remain untouched as
legacy user data but are excluded from prompt assembly and frontend navigation.
ADR-0013 defines the replacement global preference boundary.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

LearningUnit 由教师会话、复习稿、练习题、笔记条目、资料与受限助教任务组成。TeacherNote/TeacherNoteJob 是直接 API 总结的成功回合批次；助教教学写入只拥有 review/practice，资料解析仍属于 SourceMaterial 基础设施。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

SourceMaterial 的解析拥有者为 sources 本地模块，状态与唯一 derived/content.md 仍属于原资料修订。AssistantTask 仅 consolidate，不能写资料或笔记。教师上下文在每次请求中自动快照 ready 资料摘录，不把资料文字当作工具授权；导入与教师生成生命周期独立。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
