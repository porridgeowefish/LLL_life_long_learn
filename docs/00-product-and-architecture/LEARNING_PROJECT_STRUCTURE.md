# Learning Project Structure

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
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

When Intro detects a prerequisite gap, `intro/assessment.json` includes a
concise summary of what the knowledge is and what understanding is missing. The
UI displays that summary directly and exposes no supplement action. A gap does
not create a new project. Only a separate learner-confirmed commitment creates
system learning.

## Creating A Deep Dive From A Map

The learner may choose a topic named in the map and request system learning.
The system must:

```text
place the action beside the corresponding concept or branch heading in the body
prefill the project title and learning context
show a confirmation form
allow the learner to edit the learning contract
write a project only after confirmation
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
discipline map is the clickable overview of its folder; it is not duplicated as
a child project row. System-learning projects remain ordinary movable members:

```text
物理学                    ← 点击文件夹标题打开学科总览
├─ 流体力学               系统学习
└─ 热力学                 系统学习
```

It must not parse overview headings into navigation nodes. Selecting a
map-backed folder title opens `学科总览`; selecting a system-learning child row
opens its five-zone experience.

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

## Delivery State

Iteration 07 is delivered: project state, skeleton creation, indexing, API
routing, and frontend navigation are type-aware. The encyclopedia Agent runs
through the selected native CLI. Its heading hierarchy renders as a Word-style
table of contents, and every H3 key concept or branch receives an adjacent body
action that opens the ordinary system-learning form with an editable prefilled
title. The confirmed project is classified through the map-backed folder.

## Decision References

- [ADR-0002](./ADR/0002-local-learning-workbench-structure.md) established the folder-first workbench and five-zone learning workflow.
- [ADR-0005](./ADR/0005-discipline-map-and-system-learning-project-types.md) limits the five-zone workflow to system-learning projects and adds discipline maps.
- [ADR-0006](./ADR/0006-inline-prerequisite-bridges.md) is the superseded historical bridge decision.
- [ADR-0007](./ADR/0007-flat-project-storage-and-sidebar-classification.md) removes physical subprojects and defines flat storage plus map-backed reuse of existing sidebar folders.
- [ADR-0008](./ADR/0008-generated-prerequisite-gap-summaries.md) makes the generated assessment summary the current prerequisite response.
