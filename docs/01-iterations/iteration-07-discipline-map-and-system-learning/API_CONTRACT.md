# Iteration 07 API Contract

Status: active  
Owner: project maintainer  
Last reviewed: 2026-07-14  
Source of truth: iteration 07 HTTP contract; implemented Go request/response types supersede this document.

> Superseded delta: iteration 09 and ADR-0008 remove the quick prerequisite
> bridge endpoint. The remaining iteration 07 contracts stay active.

## Project Types

Canonical values are `discipline-map` and `system-learning`. Existing projects
without `projectType` decode as `system-learning`.

## AI Choice Advice

`POST /api/project-type-advice` accepts optional draft fields plus `messages`
(1–20 temporary `user` / `assistant` turns, ending with a user turn). It returns
`reply`, optional `recommendation`, optional reason/trade-off, and confidence.
When more context is required it omits `recommendation` and asks one clarifying
question. The server does not persist messages. Advice never creates a project
or changes the learner's selection. Provider output is parsed defensively: a
valid recommendation is optional, and recoverable malformed structured output
must still become a readable reply rather than raw JSON or a silent failure.

## Create A Project

`POST /api/projects` is the only project-creation endpoint.

| Field | Type | Required | Description |
|---|---|---:|---|
| `title` | string | yes | project title |
| `projectType` | enum string | new clients: yes | map or system learning |
| `why` | string | system learning: yes | learning motivation |
| `current` | string | system learning: UI default | current ability |
| `target` | string | system learning: UI default | target ability |
| `standard` | string | no | completion standard |

Starting from a discipline map submits the same system-learning request as any
other entry point. The request contains no parent relation.

Rules:

- New writes always use `projects/<globally-unique-slug>/`.
- Every project directory is physically flat.
- A discipline map is reconciled into a folder overview; system-learning projects remain movable between folders.
- A map project has no learner-facing five-zone artifacts.
- Project type is immutable through ordinary update APIs.

## Project List And Detail

Metadata includes `projectType` and `overviewAvailable`. The sidebar combines
returned objects with the existing folder layout. A map project's slug is
opened from its bound folder title and omitted from child rows. It does not
parse overview Markdown or infer project ownership.

## Folder Layout

`GET /api/folders` and `PUT /api/folders` preserve the existing layout shape and
the optional `mapProjectSlug`. The backend reconciles all indexed discipline
maps before returning: it promotes a same-name folder or creates a folder, and
removes map slugs from `slugOrder`.

## Discipline Overview

`GET /api/projects/{id}/discipline-overview` returns `{title, content}`.

`POST /api/projects/{id}/discipline-overview/generate` validates the map and
explicitly launches the registered `encyclopedia` agent through the selected
native Agent CLI. It reuses the normal session, run-directory, prompt-package,
visible-terminal, and SSE infrastructure and returns `201` with
`{session, runDir}` immediately after launch. It does not call the Ask-AI HTTP
provider and does not synchronously return generated overview content.

The versioned charter names root-level `overview.md` as the project-level output
artifact. A filesystem write emits `artifact-updated` with `zone: "overview"`;
the frontend then invalidates `GET /api/projects/{id}/discipline-overview`.
Its display label remains `学科总览`. Other project creation, progress,
movement, or deletion does not trigger regeneration.

## Prerequisite Presentation (superseded)

The delivered `POST /api/projects/{id}/prerequisite-bridge` endpoint was removed
in iteration 09. Intro now writes and renders concise prerequisite summaries as
part of `intro/assessment.json`.

## Errors

Errors use the existing `{ "error": "..." }` envelope. Current relevant
messages are:

| HTTP | Error message | Trigger |
|---:|---|---|
| 400 | `invalid_project_type` | unknown project type or an overview operation on a system-learning project |
| 404 | `project_not_found` | requested project does not exist |
| 409 | `project already exists: <slug>` | globally unique slug already exists |
| 503 | `<runtime> binary not available` | the selected native Agent CLI cannot be launched |
| 503 | `ai_not_configured` | the lightweight project-type advisor has no configured HTTP provider |

There is no project-type update endpoint; immutability is enforced by exposing
creation-time type selection only.

The legacy nested-create endpoint and nested-project compatibility are removed.
