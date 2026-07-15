# Data Model

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
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

## Discipline Overview

| Field / artifact | Type | Rule |
|---|---|---|
| overview content | Markdown file | exactly one learner-facing overview per discipline map |
| display label | string | always `学科总览`; physical filename is internal |
| generation run | normal Session + `runs/<timestamp>-encyclopedia/` | selected native Agent CLI; project-level output path is `overview.md` |
| table of contents | derived from Markdown H2/H3 headings | navigation only; not persisted as a second structure |
| inline topic action | derived from every H3 key concept or branch | must not create a project until confirmation |
| folder overview binding | `folders.json.mapProjectSlug` | optional discipline-map slug rendered as the folder's explicit overview row; not `slugOrder` membership or project ownership |
| learning-project membership | `folders.json.slugOrder[]` | system-learning classification only; never contains a bound map slug |

An overview topic has no durable project identity. A project record begins only
after learner-confirmed creation.

The backend reconciles indexed discipline maps into folder objects on folder
reads and writes. A same-name folder is promoted in place; otherwise one folder
is created. Clearing a stale map binding preserves the folder and its membership.

## System-Learning Data

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

## Delivery State

The Go state model persists `projectType` for new projects. Missing type remains
the legacy compatibility signal and decodes as `system-learning`.
