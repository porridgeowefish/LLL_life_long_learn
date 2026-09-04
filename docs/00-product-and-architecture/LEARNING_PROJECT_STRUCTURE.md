# Learning Project Structure

Status: active
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: current project types, flat storage, learning-unit shape, and navigation semantics.

## Core Principle

LLL is a folder-first local learning workbench. A project directory represents
a real learner commitment, not every topic AI happens to mention.

```text
overview content describes possibilities
project folders record learner commitments
one system-learning project owns one teacher conversation and becomes one learning unit
```

## Project Types

| Type | User-facing shape | Active surface |
|---|---|---|
| `discipline-map` | 学科地图 | 学科总览 + 学习计划 |
| `system-learning` | 学习单元 | 教师 + 资产 + 资料 |

The learner explicitly chooses the type before creation. AI may recommend and
prefill, but cannot submit the choice. Existing projects without type metadata
decode as `system-learning` for compatibility.

## Discipline-Map Project

A discipline map is a long-lived organizing entry. The frontend exposes an
explicit `××学科总览` row; physical filenames remain internal contracts.

```text
projects/<map-slug>/
  project.md
  state.json
  overview.md
  discipline-topics.json
  learning-plan.json
  runs/
  assets/
```

The encyclopedia Agent plans one coherent map, writes H3 major chapters and H4
learnable topics to `overview.md`, and writes machine-readable boundaries to
`discipline-topics.json`. The learner owns topic selection and order through
`learning-plan.json`. Mentioning a topic never creates a directory, page, job,
or sidebar node.

A map brief records map shape, overview goal, optional scope notes, and learner
notes. It does not contain a focused-learning ability ladder or phase.

## System-Learning Project

A system-learning project is one focused learning unit. Its active experience
is a teacher conversation with accumulated assets and source materials:

```text
projects/<learning-slug>/
  project.md
  state.json
  learning-scope.json
  unit.json
  conversation/
    conversation.json
    events.jsonl
    compact.json
  assets/
    intro/
    body/
    practice/
    generated/
  sources/<source-id>/revisions/<revision-id>/
    original/
    derived/
  assistant-tasks/<task-id>/
    task.json
    input-manifest.json
    attempts/
  runs/
  migrations/iteration-13/
```

`project.md` stores the focused-learning brief. `learning-scope.json` stores the
objective topic boundary: a map-origin project receives a ready creation-time
snapshot; a standalone project begins with a draft boundary that teaching may
clarify. Map regeneration never mutates an existing unit's snapshot.

The scope is guidance, not a hard product gate. One conversation may follow the
learner across related themes, and auto-compact plus the teacher system prompt
manage long context. The conversation comes first; the learning unit is the
durable form that accumulates around it.

## Creating A Learning Unit From A Map

The learner may request a deep dive beside an H4 topic. The system:

```text
prefills title and learning context
sends map slug + canonical topic ID
shows a confirmation form
creates a flat system-learning project only after confirmation
copies the canonical topic boundary into learning-scope.json
opens the new unit's one teacher conversation
```

No physical parent/child project relation is created. The existing sidebar
folder is classification only.

## Sidebar Contract

```text
物理学                    文件夹标题：仅展开/收起
├─ 物理学学科总览         discipline-map
├─ 流体力学               system-learning
└─ 热力学                 system-learning
```

Overview headings never become sidebar nodes. A bound discipline map is derived
from `mapProjectSlug` and does not enter `slugOrder`; learning units remain
ordinary movable folder members.

## Assets, Sources, Runs, And Preferences

- `assets/intro`, `assets/body`, and `assets/practice` are editable,
  learner-facing teaching assets.
- `assets/generated` holds declared research, code, images, experiments, and
  other assistant deliverables that do not fit the three teaching sections.
- `sources/` holds learner-provided immutable revisions and, after supported
  background parsing, exactly one `derived/content.md` citation per ready revision.
- `runs/` and task attempts hold raw execution evidence; they never become
  curated assets merely because a CLI wrote them.
- `<WORKSPACE>/preferences.md` is one learner-owned global preference document,
  read-only to AI and never tool authorization or factual evidence.

Formal assets are written by Go after validation and version/merge handling.
The visible CLI writes only its attempt workspace and declared result manifest.
Confirmed consolidation always rewrites Intro and Body from sealed conversation
evidence; Practice changes only when the approved task explicitly requests it.

## Compatibility Boundary

Old projects may contain `intro/`, `explain/`, `practice/`, `extend/`,
`summary/`, `progress/`, and `memory/`. Migration may read Intro, Explain,
Practice, and Explain annotations into the current intro/body/practice model.
`extend/` and `summary/` are preserved historical files: they do not enter
teacher context, migration inventory, file APIs, watchers, navigation, or agent
registration. Project memory is likewise preserved but ignored.

## Required State

All projects track stable ID, globally unique slug, title, immutable type,
status, and timestamps. Legacy `activeZone` may remain in old state files but is
not active navigation state for the teacher workspace.

## Decision References

- [ADR-0005](./ADR/0005-discipline-map-and-system-learning-project-types.md) defines the retained project types.
- [ADR-0007](./ADR/0007-flat-project-storage-and-sidebar-classification.md) defines flat storage and map-backed folders.
- [ADR-0009](./ADR/0009-explicit-discipline-overview-entry.md) defines the explicit overview row.
- [ADR-0010](./ADR/0010-hierarchical-discipline-maps-and-learning-plans.md) defines hierarchical maps and learner-owned plans.
- [ADR-0011](./ADR/0011-learning-scope-snapshots-and-intro-calibration.md) defines topic-boundary snapshots.
- [ADR-0012](./ADR/0012-teacher-assistant-learning-workspace.md) defines the conversation-first learning unit.
- [ADR-0013](./ADR/0013-stable-stream-rendering-and-global-preferences.md) defines stable streaming and global preferences.
- [ADR-0016](./ADR/0016-retire-legacy-summary-extend-and-knowledge-garden.md) retires Summary, Extend, and Knowledge Garden product surfaces.

Earlier partially effective workbench and artifact decisions remain in the
[active ADR graph](./ADR/README.md). Only fully superseded decisions move to the
[historical ADR index](../99-archive/adr/README.md).
