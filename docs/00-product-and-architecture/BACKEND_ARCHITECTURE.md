# Backend Architecture

Status: draft  
Owner: project maintainer  
Last reviewed: 2026-06-08  
Source of truth: this document defines the target backend architecture for the local learning workbench.

## Intent

LLL backend should evolve from a simple task launcher into a local orchestration runtime for:

```text
project-based learning
real Claude Code terminal sessions
append-only follow-up history
role-based agent invocation
file-path-based context sharing
project and learner memory growth
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
memory as assistive personalization, not hidden authority
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
filesystem reads and writes
agent registry loading
prompt assembly
Claude Code process launch and PTY attachment
session and turn lifecycle
raw run capture
artifact materialization
memory snapshot updates
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
memory
files
```

### 2. Workspace And Project Layer

Purpose:

```text
discover projects from the filesystem
index project trees and subprojects
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
collect project memory and learner memory context
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
memory snapshots
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

### 8. Memory Layer

Purpose:

```text
update project memory after meaningful runs
update learner-level memory with explicit rules
provide memory snapshots to prompt assembly
avoid silent mutation of canonical summary content
```

Memory writes should be controlled and inspectable.

## Core Domain Objects

The backend should revolve around these objects.

### WorkspaceProject

Represents one learning project folder.

Owns:

```text
project metadata
learning zones
subprojects
memory files
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
memory sensitivity rules
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

### MemorySnapshot

Represents a usable memory state for prompt assembly.

Types:

```text
learner memory snapshot
project memory snapshot
```

## Canonical Filesystem Model

Backend persistence should be file-first.

Recommended workspace layout:

```text
projects/
  <project-slug>/
    project.md
    state.json
    memory/
      project-memory.md
      project-state.json
    intro/
    explain/
    practice/
    extend/
    summary/
    runs/
    assets/
    subprojects/

memory/
  learner-profile.md
  learner-state.json

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
-> memory layer provides learner and project memory snapshots
-> prompt assembly layer builds prompt package
-> Claude session runtime launches a real terminal session
-> session timeline creates a new session and initial system/user turns
-> run record directory is created
-> terminal events are streamed to frontend
-> final outputs are written to run files and zone targets
-> artifact materializer updates project artifact index
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

### Flow D: Update Memory

```text
meaningful run completes
-> memory layer derives candidate memory updates
-> backend records memory delta
-> learner or project memory files are updated under explicit rules
-> new memory snapshot becomes available for later prompt assembly
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
memory-updated
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
memory writes are separated from summary writes
subproject writes stay inside the selected project subtree
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
memory update failure
session recovery failure
```

Each failure should expose:

```text
clear state
human-readable explanation
recoverability hint when possible
audit trace in run metadata
```

## Non-Goals

This backend design does not include:

```text
cloud deployment
user accounts
team collaboration
remote multi-device sync
provider abstraction beyond current Claude Code direction
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
memory growth as a controlled support system
```
