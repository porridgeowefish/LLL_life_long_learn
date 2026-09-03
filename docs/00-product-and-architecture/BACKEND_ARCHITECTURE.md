# Backend Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: current Go backend boundaries and write ownership; package code owns implementation details.

## Intent

The Go backend is a local application service, not a remote multi-tenant
platform. It keeps dialogue responsive, delegates substantial work to a visible
native CLI, and makes every durable learning object inspectable on disk.

```text
HTTP/SSE adapters
-> application/domain services
-> file repositories and external runtime adapters
```

Handlers must not own provider loops, queue admission, prompt injection,
filesystem path policy, source disclosure, or multi-file asset commits.

## Iteration-14 Target Package Model

The package map below describes current iteration-13 code. ADR-0015 approves a
behavior-preserving migration to this target during iteration 14:

```text
internal/
  app/{bootstrap,integration}
  transport/http
  modules/
    teacher
    assistant
    assets
    sources
    projects
    learning
    preferences
  platform/{config,filesystem,process,events,identity}
  compatibility
```

Each module root is its public facade. Private domain, application, port, and
adapter packages live below that module's nested `internal` directory as
needed. Go visibility plus `tools/archcheck` prevents transport or other
modules from bypassing the facade. `app/integration` converts bounded values
between facades but owns no business rule; `app/bootstrap` is the only concrete
composition root.

The migration does not introduce microservices, a database, a broker, new
public routes, or new project-file schemas. Current package names remain
runtime truth until their iteration-14 wave is delivered and verified.

## Current Package Map

| Area | Primary packages | Responsibility |
|---|---|---|
| Transport | `internal/server`, `internal/httpx` | decode, size-limit, call one operation, encode, stream |
| Teacher | `teacherservice`, `teachergateway`, `conversationstore` | context assembly, provider-neutral blocks, tool loop, durable recovery |
| Assistant | `assistanttask`, `agentexecution`, `claudelauncher` | authorization, task state, queue/leases, sealed input, visible CLI lifecycle |
| Learning assets | `assetstore`, `artifactwriter`, `artifactwatch`, `annotationstore` | versions, edits, merge/promotion, annotations, invalidation |
| Sources | `sourcestore` | upload limits, hashes, immutable revisions, disclosure, parse outputs, deletion |
| Projects | `workspace`, `projectindex`, `folderstore`, `learningscope` | project roots, types, flat classification, scope snapshots |
| Preferences | `preferencestore`, `promptassembly` | learner-owned global file and bounded read-only prompt snapshots |
| Migration | `iteration13migration` | inventory, backup, staged conversion, journal, cutover/rollback reads |
| Compatibility | `sessionstore`, `agentregistry`, `practicestore`, `flashcardstore`, `confusionstore`, `progressstore` | old-project readers and legacy routes only |

`agentexecution.Service` is the reusable domain boundary for opening a project
folder, selecting a native runtime, injecting a prompt, exposing the terminal,
and observing exit. Encyclopedia generation and teacher-delegated assistant
tasks use this same service with different input/output contracts.

## Teacher Application Flow

```text
POST teacher turn
-> validate model and message
-> append learner event with stable ID
-> assemble bounded conversation, scope, assets, disclosed sources, and preferences
-> stream normalized reasoning-summary / answer / tool-use blocks
-> append durable teacher events and run state
-> expose final conversation projection through REST
```

Provider adapters live behind `teachergateway`; the frontend never parses a
provider SDK stream. A refresh may disconnect the browser but does not cancel
the server-side run. REST and durable conversation/run records repair missed
stream events.

The teacher owns one tool: `delegate_learning_work`. Tool calls are accepted
only when they refer to the disclosed proposal message and a later explicit
learner authorization. A normal agreement word is not enough if the proposal
identity or complete task definition is missing.

## Assistant Task Flow

```text
authorized tool call
-> assistanttask service performs atomic check-and-create
-> durable task enters queued state
-> dispatcher acquires global/unit slots and attempt lease
-> exact conversation range, asset bases, source revisions, scope, and preferences are copied into sealed inputs
-> agentexecution.Service opens the selected visible CLI in the learning-unit folder
-> CLI writes declared attempt outputs and result.json only
-> Go validates paths, hashes, manifest, and candidates
-> Go versions/merges formal assets or registers generic generated deliverables
-> durable task state changes and global SSE invalidates affected resources
```

The local queue is reconstructed by scanning task files after restart. The
default concurrency limit is five running tasks globally and two per learning
unit. Task identity and executor-run identity are separate. There is no hidden
public create/retry/cancel API; the learner cancels by ending the visible
terminal, and LLL records the exit result.

## Write Authority

| Data | Canonical writer |
|---|---|
| conversation events and teacher run state | teacher application service |
| task state, attempts, leases, sealed manifests | assistant task/dispatcher service |
| formal asset versions and merge journals | asset application/store |
| source revisions and tombstones | source application/store |
| global preferences | explicit learner preference endpoint only |
| CLI attempt files | selected native CLI, inside its declared attempt workspace |

The CLI never writes canonical conversations, tasks, assets, sources, or
preferences. Raw runtime output never becomes learner-facing merely because a
process created it. Go validates every declared path stays within its task
workspace before promotion.

## Persistence Layout

```text
projects/<slug>/
  unit.json
  conversation/{conversation.json,events.jsonl,compact.json}
  assets/{intro,body,practice,generated}
  sources/<source-id>/revisions/<revision-id>/{original,derived}
  assistant-tasks/<task-id>/{task.json,input-manifest.json,attempts}
  runs/
  migrations/iteration-13/{migration.json,journal.jsonl,backup}

<WORKSPACE>/preferences.md
```

Files are canonical; indexes, queue state, and SSE subscribers are projections.
Writes use temp/staging files, validation, and atomic replacement where the
contract spans one canonical file. Multi-file promotion records a journal and
recoverable version state.

## Event And Recovery Model

Teacher streaming and global invalidation are separate channels:

- the teacher response carries response-local normalized content blocks;
- the app-shell SSE connection carries small identifier-only invalidations;
- REST returns durable conversation, task, asset, and source projections after
  refresh or missed events;
- startup scans task/run files to reconcile queued, running, interrupted, and
  terminal states.

SSE is never the only copy of a state transition.

## Preferences Boundary

`<WORKSPACE>/preferences.md` is Markdown capped at 256 KiB. The preferences
page and direct learner file editing are the only write paths. Teacher context
and assistant sealed input receive bounded read-only snapshots. Preferences do
not authorize tools, establish facts, or override explicit learner choices.

There is no active project/learner memory service. Existing project `memory/`
directories remain untouched but are not indexed, exposed, or supplied to AI.

## Failure Model

Stable failures distinguish at least:

```text
invalid request or project path
provider configuration/stream failure
conversation conflict or stale proposal identity
assistant capacity or duplicate-active-type conflict
visible CLI launch/exit failure
invalid or undeclared assistant output
asset version/merge conflict
source size/type/privacy failure
preferences read or explicit-save failure
restart reconciliation failure
```

Every failure exposes a clear durable state, learner-facing explanation, and a
recoverability hint where possible. Failed assistant tasks appear in the
conversation; the UI may suggest a manual terminal command but does not invent
a one-click retry contract.

## Compatibility Boundary

Legacy zone/session routes and stores remain for old projects, migration,
rollback, and existing Ask-AI/practice/flashcard readers. New teacher,
assistant, asset, source, and preference behavior must use the application
services above and must not be added to legacy route-local orchestration.

## Non-Goals

```text
cloud deployment or accounts
multi-tenant collaboration
external queue/broker
database-first canonical persistence
hidden assistant execution
AI-managed preference or memory writes
```
