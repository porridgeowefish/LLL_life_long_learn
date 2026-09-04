# System Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: current long-lived runtime boundaries; code owns implementation details.

## Current Runtime

LLL is a local-first web application:

```text
frontend/          React 18 + TypeScript SPA
backend-go/        Go HTTP/SSE server and application services
projects/          canonical learning-unit, conversation, asset, source, and task files
agents/            native CLI charters and reusable reasoning primitives
preferences.md     learner-owned global preferences, read-only to AI
```

The Go binary serves `frontend/dist/` in production. Filesystem content is
durable truth; in-memory indexes, dispatch slots, and SSE connections are
rebuildable runtime state. No database, external queue, cloud account, or
hidden-only agent runtime is required.

## Implemented Modular Monolith

ADR-0015 reorganized the runtime as a business-capability modular monolith in
Iteration 14. Deployment, public APIs, and canonical project files remain
unchanged.

```text
cmd/lll
-> app/bootstrap                         one composition root
-> transport/httpserver                 thin HTTP and SSE adapters
-> modules/{teacher,assistant,assets,sources,projects,learning,preferences}
-> platform/{config,filesystem,process,events,identity}
-> compatibility                        legacy reads and migration only
```

Business capabilities are the primary code boundary. Complex modules may use
domain, application, ports, and adapters internally; simple modules stay
compact. A module exposes one facade, owns its persisted data family, and keeps
stores, providers, executors, paths, and mutable entities private. Platform
code supplies technical mechanics and may not depend on a business module.

Cross-module calls use small consumer-defined ports connected in
`app/integration`. Only `app/bootstrap` sees concrete implementations across
the system. Automated import checks enforce these rules.

## Runtime Planes

| Plane | Responsibility |
|---|---|
| Experience | project tree, discipline overview, teacher conversation, assets, sources, preferences |
| Transport | thin HTTP routes, response-local teacher SSE, global invalidation SSE |
| Application | teacher, annotation Q&A, assistant task, dispatcher, asset, source, migration operations |
| Domain | conversation, task, run, asset version, source revision, annotation, and project rules |
| Infrastructure | file stores, provider adapters, visible native CLI, IDs, clock, event broadcast |

Route handlers decode, validate size, call one application operation, and encode
a stable response. Provider loops, queue admission, source privacy, and asset
commits belong behind application/domain services rather than route-local code.

## Project-Type Routing

### Discipline map

```text
explicit ××学科总览 row
-> overview.md + discipline-topics.json
-> learner-owned learning-plan.json
-> optional confirmed creation of an ordinary system-learning project
```

The encyclopedia Agent uses the shared visible native-CLI execution service. It
writes the overview and topic-boundary catalog but does not choose the learner's
order or create a learning unit without confirmation.

### System learning

```text
one project = one learning unit = one durable teacher conversation
-> 教师: fast provider-neutral streaming dialogue
-> 资产: versioned intro / body / practice plus generated deliverables
-> 资料: immutable uploaded revisions and assistant-derived files
-> 助教: disclosed, learner-authorized heavy work in a visible native CLI
```

A map-origin unit receives a canonical topic scope snapshot. A standalone unit
starts with a draft scope. Scope guides the teacher and assistant but does not
create extra conversations, nested projects, or forced single-topic policing.

## Teacher Flow

```text
learner sends a turn
-> teacher service assembles conversation, scope, active assets, disclosed sources, and read-only preferences
-> selected model provider streams reasoning summary (when supplied) and answer deltas
-> events append to the conversation log before they are projected to the UI
-> refresh reconnects to durable run state and recovers the final response
```

The teacher owns teaching only. It has one narrow tool,
`delegate_learning_work`. Before a task can be created, the teacher explains
the proposed command, inputs, output, and learning value; a later learner turn
must explicitly authorize that proposal.

Rich output uses a two-rate rendering boundary: provider deltas are coalesced,
Markdown/KaTeX/sanitization renders from a slower snapshot, collapsed reasoning
is not mounted, and Mermaid stays a stable placeholder until the message is
complete. Auto-scroll is throttled and historical messages are isolated from
live-message rerenders.

## Assistant Flow

```text
authorized teacher tool call
-> assistant task service validates proposal identity and creates durable task.json
-> dispatcher seals exact conversation/assets/sources/preferences into an attempt workspace
-> shared execution service opens the selected native CLI in the project folder and injects one prompt
-> CLI writes only declared attempt outputs
-> Go validates result.json and declared deliverables
-> valid candidates are versioned/merged into assets; generic outputs remain declared generated assets
-> task state and global invalidation events update the conversation UI
```

The queue is local and rebuildable from task files. Default running limits are
five globally and two per learning unit. Cancellation is the learner manually
ending the visible terminal process; LLL records the resulting state but does
not pretend to own a second remote cancellation mechanism.

## Persistence And Recovery

Canonical records live below the project root:

```text
unit.json
conversation/{conversation.json,events.jsonl,compact.json}
assets/{intro,body,practice,generated}
sources/<source-id>/revisions/<revision-id>/{original,derived}
assistant-tasks/<task-id>/{task.json,input-manifest.json,attempts}
migrations/iteration-13/{migration.json,journal.jsonl,backup}
```

REST reconstructs durable state after refresh; SSE only reduces latency. Formal
asset/source/conversation/task writes are owned by Go and use validated,
recoverable file operations. The CLI cannot write canonical records directly.

`<WORKSPACE>/preferences.md` is the only active preference document. Only the
explicit preferences editor or direct user file editing may write it. Teacher
and assistant prompts receive bounded read-only snapshots. Existing project
`memory/` folders are preserved as ignored legacy data and are not supplied to AI.

## Compatibility Boundary

Legacy Intro / Explain / Practice / Extend / Summary routes, files, registered
zone Agents, and readers remain only for old-project migration, rollback, and
Ask-AI/asset compatibility. They are not the active system-learning navigation
or the source for new assistant output contracts.

## Architectural Rules

```text
filesystem-first durable truth
one active teacher conversation per learning unit
explicit learner authorization before heavy assistant work
visible native CLI through one reusable execution service
raw execution separated from curated/versioned assets
REST recovery; SSE as notification/stream transport
learner-owned preferences; no AI-managed memory
legacy readers are compatibility, never fallback product direction
```
