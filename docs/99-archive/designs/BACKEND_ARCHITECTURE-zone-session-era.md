# Backend Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: this document defines the current backend architecture for the local learning workbench.

## Intent

LLL backend should evolve from a simple task launcher into a local orchestration runtime for:

```text
project-based learning
real Claude Code terminal sessions
append-only follow-up history
role-based agent invocation
file-path-based context sharing
learner-owned global preference context
artifact indexing and retrieval
```

This backend is local-first and single-user.
It is not designed as a cloud service or multi-tenant platform.

## Design Principles

Use:

```text
filesystem as the canonical project store
real Claude Code terminal execution
append-only session history
explicit project and file boundaries
inspectable runs and prompts
separation between raw runs and curated learning artifacts
one learner-authored preference file as read-only AI context
```

Avoid:

```text
hidden in-memory-only state as the primary truth
opaque agent-to-agent messaging
summary overwrite by runtime output
cloud-first assumptions
account and permission system design
```

## Backend Responsibilities

Backend owns:

```text
workspace discovery
project tree indexing
workspace-global sidebar-folder persistence and discipline-map reconciliation
filesystem reads and writes
agent registry loading
prompt assembly
Claude Code process launch and PTY attachment
session and turn lifecycle
raw run capture
artifact materialization
global preference file reads and explicit learner writes
frontend API and event streaming
```

Frontend owns:

```text
navigation
reading and editing views
session display state
agent action affordances
rendering of Markdown and diagrams
user-facing follow-up entry
```

## Runtime Layers

The backend should be organized into the following layers.

### AgentExecution Domain Service

`backend-go/internal/agentexecution` is the sole application-facing boundary
for starting an Agent. Routes, teachers, assistant-task dispatchers, source
processors, and future features must not call `claudelauncher` directly.

The service exposes execution intents rather than terminal commands:

```text
StartProject  -> encyclopedia and project/zone agents
StartTask     -> teacher-delegated work in an isolated attempt workspace
Resume        -> reopen a supported interactive CLI conversation
StartHeadless -> explicitly non-interactive background work
```

`claudelauncher` is an infrastructure adapter below this boundary. All visible
executions converge on one interactive-terminal implementation that resolves
the selected runtime, enters the requested workspace, reads the durable UTF-8
prompt file, injects it as the initial turn, opens the real CLI, and reports
exit state. Domain-specific lifecycle markers are optional parameters; they do
not create a second launcher.

This means adding a future Agent capability requires choosing an execution
intent and supplying workspace/prompt contracts. It does not require copying
PowerShell, executable resolution, prompt injection, or terminal lifecycle
logic into a handler.

### 1. API Gateway Layer

Purpose:

```text
accept frontend requests
validate request shape
return JSON resources
stream live events
```

Examples of resource groups:

```text
health
projects
zones
agents
sessions
turns
artifacts
preferences
files
```

### 2. Workspace And Project Layer

Purpose:

```text
discover projects from the filesystem
index flat top-level project directories
read project metadata
resolve learning zones
provide canonical file paths to downstream services
```

This layer treats the filesystem as the durable source of truth for project structure.

### 3. Agent Registry Layer

Purpose:

```text
load agent identities
load agent charters and behavior rules
validate zone compatibility
declare allowed inputs and outputs
```

Each agent should be represented as a local definition, not as an implicit prompt fragment hidden in code.

### 4. Prompt Assembly Layer

Purpose:

```text
resolve predecessor nodes
collect one bounded global learner-preference snapshot
assemble the final Claude prompt
attach behavior rules and output contract
record the exact prompt used for audit
```

Inputs:

```text
project
zone
selected agent
user intent
follow-up text when present
predecessor file paths
global preference snapshot
output targets
```

Output:

```text
one fully assembled prompt package
```

### 5. Claude Session Runtime Layer

Purpose:

```text
launch a real Claude Code terminal session
attach a PTY or PTY-equivalent terminal stream
capture stdin, stdout, stderr, and lifecycle events
support live follow-up within the same session
```

Important rule:

```text
LLL does not simulate Claude execution inside the panel.
It launches and tracks the real terminal session.
```

### 6. Session Timeline Layer

Purpose:

```text
model sessions and turns
append follow-up history
separate user turns from assistant turns
index stream events into a stable timeline
support session replay and recovery
```

This layer replaces the current single `resultText` model.

### 7. Artifact Materialization Layer

Purpose:

```text
write generated outputs into project files
separate raw run outputs from curated zone outputs
promote selected session results into artifacts
index artifact metadata for the panel
```

Raw run outputs and curated project outputs must not be treated as the same object.

### 8. Global Preferences Boundary

Purpose:

```text
read `<WORKSPACE>/preferences.md` as bounded Markdown
provide read-only snapshots to teacher and assistant prompt assembly
allow writes only from the explicit learner preferences endpoint
never infer, merge, or auto-update preferences from AI output
```

Existing project `memory/` files are ignored legacy data. They are preserved on
disk but are not created, indexed, rendered, or supplied to AI.

## Core Domain Objects

The backend should revolve around these objects.

### WorkspaceProject

Represents one learning project folder.

Owns:

```text
project metadata
learning zones
system-learning projects created from any entry point
conversation, asset, source, task, and run files
assets
runs
```

### LearningZone

Represents one of the fixed project zones:

```text
Intro
Explain
Practice
Extend
Summary
```

Each zone has:

```text
known file paths
predecessor rules
allowed agents
artifact rules
```

### AgentRole

Represents a visible agent identity.

Owns:

```text
name
charter
allowed zones
behavior rules
default output targets
preference privacy and read-only rules
```

### ClaudeSession

Represents one live or completed Claude Code terminal session.

Owns:

```text
session id
project id
zone
agent id
runtime state
prompt package
terminal metadata
linked run directory
turn ids
```

### SessionTurn

Represents one append-only exchange unit.

Types:

```text
user turn
assistant turn
system turn
```

Properties:

```text
turn id
session id
ordinal
turn type
source
content reference
created time
```

### RunRecord

Represents raw runtime trace data for audit and replay.

Owns:

```text
prompt file
stdout log
stderr log
terminal events
result file
runtime metadata
```

### LearningArtifact

Represents curated knowledge written back into the project structure.

Examples:

```text
intro output
explain notes
practice tasks
extend prompts
summary draft
diagram file
```

### LearnerPreferencesSnapshot

Represents the bounded content and path of the one global preferences file.

It is read-only for AI execution and has no project-specific variant.

## Canonical Filesystem Model

Backend persistence should be file-first.

Recommended workspace layout:

```text
projects/
  <project-slug>/
    project.md
    state.json
    intro/
    explain/
    practice/
    extend/
    summary/
    runs/
    assets/

preferences.md

agents/
  registry/
  charters/
```

Important rule:

```text
filesystem content is the durable truth
in-memory indexes are runtime caches only
```

## Main Backend Flows

### Flow A: Invoke Agent On A Zone

```text
frontend requests agent invocation
-> API gateway validates project, zone, agent, and intent
-> workspace layer resolves project and predecessor file paths
-> agent registry resolves behavior rules
-> preferences store provides one read-only global snapshot
-> prompt assembly layer builds prompt package
-> Claude session runtime launches a real terminal session
-> session timeline creates a new session and initial system/user turns
-> run record directory is created
-> terminal events are streamed to frontend
-> final outputs are written to run files and zone targets
-> artifact materializer updates project artifact index
```

### Flow A2: Invoke A Project-Level Agent

The encyclopedia Agent is bound to `discipline-map`, not to a learning zone:

```text
frontend explicitly requests encyclopedia invocation
-> backend validates the project type and Agent `allowedProjectTypes`
-> project-level prompt assembly records `overview.md` and `discipline-topics.json` without inventing a zone
-> the normal Session and run directory are created
-> the selected native Agent CLI opens in a visible terminal
-> the CLI writes root `overview.md` and `discipline-topics.json`
-> learner actions atomically update root `learning-plan.json`
-> artifact watcher emits `artifact-updated` for `overview`, `discipline-topics`, or `learning-plan`
-> frontend invalidates and rereads the affected discipline-map view
```

Registered Agents never use the lightweight direct-provider path. That path is
reserved for explicitly non-Agent helpers such as Ask-AI.

### Flow A3: Create A Scoped System-Learning Project

```text
standalone create
-> workspace writes draft learning-scope.json
-> Intro finalizes it after learner calibration

map deep dive
-> client sends map slug + topic ID
-> backend validates the discipline map and topic catalog
-> backend copies the canonical topic boundary into ready learning-scope.json
-> later map regeneration does not mutate the copied scope

every zone invocation
-> prompt assembly reads and embeds learning-scope.json
-> Agent applies owned / reused / prerequisite / excluded semantics
```

### Flow B: User Follow-Up Inside The Same Session

```text
frontend sends follow-up text to active session
-> session timeline appends a new user turn
-> Claude session runtime writes follow-up into the live PTY
-> terminal output is captured
-> session timeline appends assistant turn when response completes
-> panel receives append-only updates
-> optional artifact promotion remains explicit
```

Important rule:

```text
follow-up appends to session history
it does not replace prior output
```

### Flow C: Promote Session Output Into Project Artifact

```text
user selects a session response or system promotes a declared output
-> artifact materializer maps output to target zone files
-> write operation updates the target file
-> artifact index records the linkage between session turn and project file
-> summary files remain protected from blind overwrite
```

### Flow D: Edit Global Preferences

```text
learner opens the preferences editor
-> GET /api/preferences reads or creates preferences.md
-> learner explicitly edits and saves
-> PUT /api/preferences atomically writes the bounded file
-> later teacher and assistant runs receive a read-only snapshot
```

## Session And Turn Design

The current task model is too small.
Backend should adopt:

```text
Project -> Zone -> Session -> Turn -> Artifact
```

Meaning:

```text
project is the learning container
zone is the learning phase
session is one Claude terminal conversation
turn is one append-only conversational unit
artifact is curated project knowledge
```

Required session states:

```text
preparing
launching
running
awaiting-follow-up
completed
cancelled
failed
```

Required turn properties:

```text
append-only
ordered
traceable to source
separable from artifact files
```

## PTY And Terminal Strategy

Backend should move from plain spawned process pipes toward a PTY-backed session model.

Needed capabilities:

```text
interactive stdin support
live terminal stream capture
follow-up injection
session continuity while terminal stays alive
better fidelity with real Claude Code behavior
```

If a PTY implementation is not available on the first slice, the temporary fallback may be:

```text
launch external terminal windows for execution
track run files and result files
support follow-up only after PTY integration lands
```

But the target design remains PTY-backed session control.

## Event Streaming Model

Frontend should not consume only task snapshots.
Backend should stream typed session events.

Suggested event categories:

```text
session-created
session-state
turn-created
terminal-output
artifact-updated
preferences-updated
session-completed
session-failed
```

This keeps append-only history and panel updates aligned.

## Write Safety Rules

Backend file writes should obey:

```text
project files are written only through explicit target mapping
summary files are not silently replaced by raw runtime output
raw runs always go to runs/
AI execution has no preferences write capability
all new project writes stay inside their own top-level project directory
```

## Failure Model

The backend must treat failure states as first-class.

Expected bad-path handling:

```text
Claude binary missing
local auth missing or expired
terminal launch failure
port conflict
invalid project path
invalid agent or zone combination
missing predecessor file
artifact write failure
preferences read or explicit-save failure
session recovery failure
```

Each failure should expose:

```text
clear state
human-readable explanation
recoverability hint when possible
audit trace in run metadata
```

## Teacher-Assistant Application Architecture

ADR-0012 changes how new behavior is placed in the existing Go process. Route
files remain adapters and must not own provider loops, queue admission, source
privacy, or multi-file asset commits.

Current responsibilities:

| Layer | Responsibility |
|---|---|
| HTTP/SSE transport | decode, size-limit, call one application operation, encode stable result |
| Teacher application | prompt/context assembly, provider-neutral streaming, tool loop, stop generation |
| Annotation application | quote-grounded Ask AI without teacher prompt or tools |
| Assistant application | authorization, idempotent task creation, same-type exclusion, query projection |
| Dispatcher | rebuildable queue, global/per-unit slots, leases, input sealing, visible executor launch, restart reconciliation |
| Asset application | learner edits, external-edit import, versions, three-way merge, per-asset commit journal |
| Source application | upload, hash, revision, disclosure, parse task, tombstone, permanent deletion |
| Migration application | inventory, backup, staged conversion, journal, cutover and rollback reader |
| Infrastructure | file repositories, provider SDK adapters, native CLI adapter, clock, ULID, SSE broadcaster |

The first package extraction may preserve existing stores behind repository
adapters. New iteration-13 routes may not call `os`, provider packages, CLI
launchers, or global session stores directly. Application services receive
interfaces and are testable with deterministic fakes.

The local task queue has no daemon or broker. Durable `task.json` files are
scanned on startup. A workspace-scoped admission lock protects same-unit,
same-type check-and-create. Default running limits are five globally and two
per unit. Task identity and executor-run identity remain separate.

Teacher streaming and global invalidation are different channels. A teacher
turn returns response-local SSE content frames; the existing app-shell SSE
connection publishes only small durable-resource invalidations. REST reads
repair reconnects and missed events.

Visible native-CLI execution remains the heavy-work surface. Executors receive
one immutable sealed input and write only their attempt workspace. Go validates
the generic manifest and owns every formal asset, source, conversation, and
task-state write.

These application services are implemented. Legacy session and five-zone routes
remain executable only behind compatibility readers; they are not the placement
model for new teacher, assistant, source, asset, or preference behavior.

## Non-Goals

This backend design does not include:

```text
cloud deployment
user accounts
team collaboration
remote multi-device sync
cloud multi-tenant provider administration
database-first persistence as the canonical truth
```

## Decision Summary

LLL backend should become:

```text
a local orchestration runtime
with filesystem-first project truth
PTY-backed Claude sessions
append-only session and turn history
agent invocation from explicit file-path context
artifact promotion into project files
one learner-owned global preference file as read-only context
```
