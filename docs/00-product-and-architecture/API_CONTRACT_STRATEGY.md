# API Contract Strategy

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: long-lived backend/frontend contract strategy; Go handlers and TypeScript types own current shapes.

## Current Surface

The Go server currently exposes resource groups for:

```text
health and system lifecycle
settings and appearance
projects, project types, project lists, sidebar folders, zones, and activity
agents and invocation
sessions, follow-up, cancellation, and run status
project files
Explain confusions and Ask-AI
Practice tasks, attempts, checking, submission, and evaluation
Summary flashcards
progress, activity aggregation, infographics, and SSE events
```

`backend-go/internal/server/router.go` is the route truth. Go request/response
types and frontend TypeScript types own payload truth.

## Contract Rules

```text
iteration API docs describe only the current slice delta
complex bodies use typed field tables
errors include HTTP status and stable code in new contracts
backend and frontend types change in the same implementation task
legacy-compatible defaults are explicit
Markdown never overrides implemented route behavior
```

## Iteration 07 Direction

Project APIs become type-aware:

```text
discipline-map
system-learning
```

New clients send an explicit type; legacy projects and legacy create requests
default to `system-learning`. The optional AI advice operation accepts temporary
conversation turns, returns a reply and optional recommendation, and has no
persistence or project-creation side effect.

The lightweight type advisor may use the configured HTTP provider. The
registered encyclopedia Agent may not: its generation endpoint returns a normal
Session/run response after launching the selected native Agent CLI, and the
overview is refreshed from the filesystem artifact event.

Folder APIs preserve the existing folder/membership payload and optionally bind
`mapProjectSlug`. The backend reconciles indexed discipline maps into folder
overviews, while map slugs never enter child `slugOrder` membership.

Exact request, response, and error semantics are owned by the implemented Go
and TypeScript types and explained by iteration 07 `API_CONTRACT.md`.

The iteration 07 on-demand prerequisite bridge is removed by iteration 09 and
ADR-0008. Prerequisite summaries are generated as part of
`intro/assessment.json`; there is no separate prerequisite-helper endpoint.

## Iteration 09 Project Deletion

`DELETE /api/projects/{id}` permanently removes a project resource after the
client obtains one explicit learner confirmation. The server rejects deletion
with `409 project_has_active_session` while an Agent may still write artifacts.
Successful deletion includes canonical project files plus workspace-global
folder references and in-memory session metadata. Exact semantics are owned by
iteration 09 `API_CONTRACT.md`.

## Compatibility

Breaking public-shape changes require an ADR, an iteration contract, migration
behavior, synchronized client/server types, and tests covering older persisted
projects or requests.
