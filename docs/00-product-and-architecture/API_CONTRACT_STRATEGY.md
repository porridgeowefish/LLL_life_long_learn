# API Contract Strategy

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
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
overviews. The frontend renders each binding as an explicit `××学科总览` row,
while map slugs never enter child `slugOrder` membership.

Exact request, response, and error semantics are owned by the implemented Go
and TypeScript types and explained by iteration 07 `API_CONTRACT.md`.

The iteration 07 on-demand prerequisite bridge is removed by iteration 09 and
ADR-0008. Prerequisite summaries are generated as part of
`intro/assessment.json`; there is no separate prerequisite-helper endpoint.

## Iteration 11 Discipline Planning

Discipline-map generation records `overview.md` and `discipline-topics.json`; AI does not choose a task
order. `GET/PUT /api/projects/{id}/learning-plan` read and atomically replace the
learner-owned ordered task list in `learning-plan.json`. Older maps without the
file read as an empty list. Root-file artifact events independently refresh the
overview and plan queries. Completing a task records an idempotent activity event
for that completion timestamp. Exact behavior is owned by iteration 11.

## Iteration 12 Learning Scope

`GET /api/projects/{id}/discipline-topics` reads the validated topic-boundary
catalog. A system-learning create request may send `scopeSource` with
`discipline-map`, map slug, and topic ID. The backend resolves the canonical
topic and writes a creation-time `learning-scope.json` snapshot; the client
never supplies boundary content. Standalone creation writes a draft scope.
Exact shapes and stable errors are owned by iteration 12.

## Iteration 09 Project Deletion

`DELETE /api/projects/{id}` permanently removes a project resource after the
client obtains one explicit learner confirmation. The server rejects deletion
with `409 project_has_active_session` while an Agent may still write artifacts.
Successful deletion includes canonical project files plus workspace-global
folder references and in-memory session metadata. Exact semantics are owned by
iteration 09 `API_CONTRACT.md`.

## Iteration 13 Conversation And Task Boundary

Iteration 13 adds one provider-neutral teacher-turn stream, durable conversation
reads, task and asset reads, editable core assets, body annotations, and source
operations. The frontend consumes normalized block events and never consumes a
provider SDK stream directly.

The teacher sees one `delegate_learning_work` tool with logical type, objective,
source references, and proposal message identity. Application services inject
physical paths, approval identity, task/run IDs, input ranges, and executor
policy. CLI execution uses a sealed envelope and generic result manifest rather
than public HTTP payloads.

Global SSE carries identifier-only invalidation for conversation, task, asset,
and source changes. REST remains the recovery truth. There is intentionally no
public assistant-task create, cancel, or retry endpoint in the initial contract.
Legacy confusion routes adapt to canonical body annotations during migration.

Exact proposed shapes and stable errors are owned by iteration 13
`INTERFACE_CONTRACT.md`. Current Go routes remain executable truth until the
slice is implemented.

## Compatibility

Breaking public-shape changes require an ADR, an iteration contract, migration
behavior, synchronized client/server types, and tests covering older persisted
projects or requests.
