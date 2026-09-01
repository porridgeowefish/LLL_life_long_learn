# Learning Project Structure

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: approved long-lived project types, flat storage shapes, map-backed sidebar folders, and navigation semantics.

## Core Principle

LLL is a folder-first local learning workbench. A project folder represents a
real product object, not every topic that AI happens to mention.

```text
overview content describes possibilities
project folders record learner commitments
```

## Project Types

| Type | User-facing name | Primary artifact | Learning zones |
|---|---|---|---|
| `discipline-map` | 学科地图 | 学科总览 | none |
| `system-learning` | 系统学习 | zone artifacts and summary | five fixed zones |

The learner explicitly chooses the type before creation. AI may recommend a
type, explain the trade-off, and prefill fields, but cannot submit the choice.

Existing projects without type metadata decode as `system-learning`.

## Discipline-Map Project

A discipline map is a long-lived organizing entry with one learner-facing
entry. The frontend label is `学科总览`; the physical filename is an internal
contract and must not leak into the UI.

Target layout:

```text
projects/<map-slug>/
  project.md
  state.json
  overview.md
  discipline-topics.json
  learning-plan.json
  memory/
  runs/
  assets/
```

Its `project.md` is a map brief, not a learning contract. It records the map
shape, overview goal, optional scope notes, and learner notes. It must not
contain current/target ability, completion criteria, `Active Phase`, or any
five-zone name.

The overview should establish:

```text
the discipline's purpose and boundary
major research areas and their relationships
methods and evidence forms
representative applications
possible learning routes
```

The encyclopedia Agent first plans the complete architecture, then writes H3
major chapters and H4 learnable topics to `overview.md` plus their machine-readable
boundaries to `discipline-topics.json`. It does not choose a
learning order. The learner adds H4 topics to the ordered `learning-plan.json`
task list, changes task status, and views that list in a second tab on the same
discipline-map page.

Areas listed in the overview do not create folders, pages, jobs, or sidebar
nodes. There is no branch-overview object in this model.

Other-project status is not written into overview prose. The overview changes
only after an explicit learner update action.

## System-Learning Project

A system-learning project is a focused deep dive with five fixed zones:

```text
Intro -> Explain -> Practice -> Extend -> Summary
```

Target layout:

```text
<learning-slug>/
  project.md
  state.json
  learning-scope.json
  memory/
  intro/
  explain/
  practice/
  extend/
  summary/
  progress/
  runs/
  assets/
```

Zone semantics and artifact protocols are owned by the agent charters and the
current iteration contracts. The five-zone rule applies only to
`system-learning`, not to every `WorkspaceProject`.

Its `project.md` retains the focused-learning contract: motivation, current
ability, target ability, completion standard, active phase, and learner notes.

Its `learning-scope.json` is the objective boundary consumed by every learning
Agent. A map-origin project receives a ready creation-time snapshot. A
standalone project starts draft and Intro finalizes it after calibration.
Map regeneration never mutates an existing project scope.

When Intro detects a prerequisite gap, `intro/assessment.json` includes a
concise summary of what the knowledge is and what understanding is missing. The
UI displays that summary directly and exposes no supplement action. A gap does
not create a new project. Only a separate learner-confirmed commitment creates
system learning.

## Creating A Deep Dive From A Map

The learner may choose a topic named in the map and request system learning.
The system must:

```text
place the action beside the corresponding H4 learnable-topic heading in the body
prefill the project title and learning context
send the map slug and canonical topic ID
show a confirmation form
allow the learner to edit the learning contract
write a project only after confirmation
copy the canonical topic boundary into learning-scope.json
```

The page may render a Word-style table of contents for navigation, but it must
not duplicate selectable topics into a top card index or detach one generic
deep-dive action from the concept being discussed.

All project directories remain physically flat:

```text
projects/
  物理学/
    overview.md
    state.json
  流体力学/
    project.md
    state.json
    intro/
    explain/
    practice/
    extend/
    summary/
```

No directory exists for Newtonian mechanics, thermodynamics, or quantum
mechanics merely because those names appear in the physics overview.

## Sidebar Contract

The sidebar combines real project objects with the existing folder layout. A
discipline map is the bound overview of its folder and renders as one explicit
overview row. System-learning projects remain ordinary movable members:

```text
物理学                    ← 文件夹标题仅展开/收起
├─ 物理学学科总览         总览
├─ 流体力学               系统学习
└─ 热力学                 系统学习
```

It must not parse overview headings into navigation nodes. Selecting a
explicit `××学科总览` row opens the map; selecting a system-learning child row
opens its five-zone experience. The map row is derived from `mapProjectSlug`
and never enters `slugOrder`.

## Required Project State

All project state tracks:

```text
project id, globally unique slug, title, type, status, timestamps
```

System-learning state additionally tracks an active zone. Discipline maps do
not invent an active zone.

Project type is immutable in iteration 07. A learner who wants another shape
creates a related project instead of rewriting the existing folder in place.

## Runs, Artifacts, And Memory

```text
runs/ stores raw execution records
overview and zone files store curated learner-facing artifacts
memory/ stores project context used for later personalization
```

Raw runtime output never becomes the curated learning layer automatically.
Memory may guide recommendations and explanations but must not override explicit
learner choices.

## Iteration 13 Target Learning Unit

ADR-0012 preserves the physically flat `system-learning` project and changes
its active internal experience after migration. One project owns one teacher
conversation and becomes one learning unit. Its active top-level destinations
are `教师 / 资产 / 资料`; the five legacy zone directories remain only for
backup, compatibility reads, and rollback.

The canonical additions are:

```text
unit.json
conversation/{conversation.json,events.jsonl,compact.json}
assets/{intro,body,practice,generated}
sources/<source-id>/revisions/<revision-id>/{original,derived}
assistant-tasks/<task-id>/{task.json,input-manifest.json,attempts}
migrations/iteration-13/{migration.json,journal.jsonl,backup}
```

Intro, Explain, Practice, and Explain Ask-AI migrate into intro, body, practice,
and body annotations. Summary and Extend are not active assets and do not enter
new model context. A migration backup is created before canonical writes.

Discipline maps remain separate project objects. Their topic catalog and a
learning unit's scope snapshot remain provenance and initial guidance; neither
creates extra conversations or filesystem nesting.

## Delivery State

Iterations 07, 11, and 12 are delivered: project state, skeleton creation, indexing, API
routing, and frontend navigation are type-aware. The encyclopedia Agent runs
through the selected native CLI. Its H2/H3/H4 heading hierarchy renders as a
Word-style table of contents, every H4 learnable topic receives an adjacent body
action, and a separate top-level plan view presents the learner-owned task order.
Legacy H3-only overviews remain actionable. The confirmed project is classified
through the map-backed folder.
Each new map topic also has a validated structured boundary, and confirmed
deep dives persist a stable scope snapshot consumed by the five learning Agents.
Iteration 13 is an accepted target and is not yet implemented.

## Decision References

- [ADR-0002](./ADR/0002-local-learning-workbench-structure.md) established the folder-first workbench and five-zone learning workflow.
- [ADR-0005](./ADR/0005-discipline-map-and-system-learning-project-types.md) limits the five-zone workflow to system-learning projects and adds discipline maps.
- [ADR-0006](./ADR/0006-inline-prerequisite-bridges.md) is the superseded historical bridge decision.
- [ADR-0007](./ADR/0007-flat-project-storage-and-sidebar-classification.md) removes physical subprojects and defines flat storage plus map-backed reuse of existing sidebar folders.
- [ADR-0008](./ADR/0008-generated-prerequisite-gap-summaries.md) makes the generated assessment summary the current prerequisite response.
- [ADR-0010](./ADR/0010-hierarchical-discipline-maps-and-learning-plans.md) adds hierarchical maps and the learner-owned task plan.
- [ADR-0011](./ADR/0011-learning-scope-snapshots-and-intro-calibration.md) adds topic-boundary catalogs, scope snapshots, and the Intro calibration boundary.
- [ADR-0012](./ADR/0012-teacher-assistant-learning-workspace.md) replaces the active five-zone presentation with one teacher conversation, versioned assets, sources, and asynchronous CLI assistance.
