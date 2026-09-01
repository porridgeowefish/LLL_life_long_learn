# Data Model

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: long-lived conceptual file-first model; Go structs and persisted schemas own current runtime truth.

## Project

| Field | Type | Values / shape | Rule |
|---|---|---|---|
| `id` | string | filesystem-safe project id | required |
| `slug` | string | filesystem-safe slug | required and unique in its scope |
| `title` | string | learner-facing title | required |
| `projectType` | enum string | `discipline-map` or `system-learning` | missing legacy value decodes as `system-learning` |
| `status` | string | project lifecycle state | required |
| `activeZone` | string or null | five-zone name | only meaningful for `system-learning` |
| `createdAt` | RFC 3339 timestamp | timestamp | required |
| `updatedAt` | RFC 3339 timestamp | timestamp | required |

Project type is immutable in iteration 07.

`project.md` is type-specific learner context:

| Project type | Allowed context |
|---|---|
| `discipline-map` | project shape, overview goal, optional scope notes, learner notes |
| `system-learning` | motivation, current ability, target ability, completion standard, active phase, learner notes |

A discipline-map brief never stores an ability ladder, completion gate, or
`Intro`/five-zone phase. The backend enforces this boundary even if a client
submits system-learning-only fields.

## Discipline Overview, Topic Catalog, And Learning Plan

| Field / artifact | Type | Rule |
|---|---|---|
| overview content | Markdown file | exactly one learner-facing overview per discipline map |
| topic catalog | `discipline-topics.json` | one validated boundary per actionable H4 topic; core concepts have one owner |
| learning plan | `learning-plan.json` | learner-selected ordered tasks and per-task status |
| display labels | string | `学科总览` and `学习计划`; physical filenames are internal |
| generation run | normal Session + `runs/<timestamp>-encyclopedia/` | selected native Agent CLI; project-level output paths are `overview.md` and `discipline-topics.json` |
| table of contents | derived from Markdown H2/H3/H4 headings | navigation only; not persisted as a second structure |
| inline topic action | derived from every H4 learnable topic | must not create a project until confirmation; H3 remains the legacy fallback when no H4 exists |
| folder overview binding | `folders.json.mapProjectSlug` | optional discipline-map slug rendered as the folder's explicit overview row; not `slugOrder` membership or project ownership |
| learning-project membership | `folders.json.slugOrder[]` | system-learning classification only; never contains a bound map slug |

An overview topic has no durable project identity, but its catalog entry has a
stable topic ID and an objective boundary. A project record begins only
after learner-confirmed creation.
The learning plan references exact overview topic titles. Array position is the
learner-selected order; each item stores `planned`, `in-progress`, or `completed`
plus timestamps. It stores no system-learning project ownership or sidebar membership.
Each distinct completion timestamp may append one idempotent
`learning-task-complete` activity event with zero growth value.

The backend reconciles indexed discipline maps into folder objects on folder
reads and writes. A same-name folder is promoted in place; otherwise one folder
is created. Clearing a stale map binding preserves the folder and its membership.

## System-Learning Data

Every system-learning root contains `learning-scope.json` with status
`draft|ready`, topic goal, inclusion, exclusion, prerequisites, owned concepts,
reused concepts, provenance, and update time. A map-origin scope is a copied
snapshot; `source.mapSlug/topicId` records provenance rather than live ownership.

System-learning projects retain the existing zone-owned protocols, including:

```text
intro assessment and output
explain manifest and pages
practice tasks, answer keys, submissions, and evaluations
extend artifacts
summary and flashcards
progress events
memory and run records
```

Each `intro/assessment.json` prerequisite may include `summary`, a 45-100
Chinese-character introduction to the knowledge and the learner's current gap.
New Intro output writes it; legacy records without it remain readable by
falling back to `impact`.

Exact fields remain owned by their code schemas and active iteration contracts.

## Persistence Rules

```text
filesystem files are canonical truth
in-memory indexes are acceleration only
project indexes are rebuilt from real state files
overview headings never create index records
raw run folders and curated artifacts remain separate
project deletion removes the whole canonical project root and prunes global folder references
active Agent sessions block deletion to prevent post-delete artifact writes
```

## Iteration 13 Target Model

ADR-0012 retains the flat project root and adds canonical file-first records
for one learning unit:

```text
unit metadata
append-only conversation events plus rebuildable compact context
versioned intro, body, and practice assets plus generated artifacts
body-owned annotation and Ask-AI history
versioned source originals and derived content
durable assistant tasks, sealed inputs, and per-attempt workspaces
migration backup and recovery journals
```

Product Task and executor Run are separate identities. Conversation, task,
asset, source, and version IDs are opaque stable ULIDs. In-memory queue and SSE
state are derived. Provider deltas, rendered rich content, and compact context
are projections. Exact paths and schemas are owned by iteration 13
`DATA_DESIGN.md`; code remains current truth until delivery.

## Delivery State

The Go state model persists `projectType` for new projects. Missing type remains
the legacy compatibility signal and decodes as `system-learning`. Iteration 13
is planned, not yet implemented.
