# ADR-0012: Teacher-Assistant Learning Workspace

Status: accepted
Owner: project maintainer
Date: 2026-08-30
Last reviewed: 2026-08-30
Source of truth: durable product, runtime, persistence, and migration decision for the conversation-first learning workspace.
Supersedes: the active five-zone system-learning presentation and direct-generation workflow in ADR-0002, ADR-0004, and ADR-0005
Extends: ADR-0003, ADR-0007, ADR-0009, ADR-0010, and ADR-0011

## Context

LLL currently turns a learning request into fixed Intro, Explain, Practice,
Extend, and Summary outputs. The model makes local artifacts durable and keeps
native Agent execution inspectable, but it makes the learner consume a content
pipeline rather than learn through a responsive teacher. Follow-up questions,
verification, asset production, and source material also cross several route,
zone, session, and file contracts.

Using the native CLI for every teacher response would preserve tool power but
make ordinary dialogue slow. Using a fast API teacher for all work would make
dialogue responsive but give the model unsafe and unreliable file/process
responsibility. Adding an API chat beside the five-zone UI would preserve both
implementations but split the product and leave the learner unsure which one
owns learning progress.

The backend also places substantial orchestration in HTTP handlers. A durable
teacher-tool-assistant loop, restart recovery, source privacy, and versioned
asset merge cannot be safely added as more route-local branching.

## Decision

### Conversation-first product

A system-learning unit is centered on one durable teacher conversation. The
active experience has three destinations: `教师`, `资产`, and `资料`. The teacher
surface is one restrained continuous chat, not a visible workflow of learning
roles or task modules.

The unified teacher prompt may dynamically use five soft methods:
`引导人 / 澄清者 / 验证者 / 沉淀者 / 复盘者`. They are teaching behavior, not
project objects, state-machine stages, plugins, or UI modes.

The existing discipline-map project type, overview, learner-owned plan, topic
catalog, flat storage, folder classification, and learning-scope provenance are
retained. Scope guides the teacher and assistant but does not forbid natural
cross-topic questions. One teacher conversation forms one learning unit; a
discipline map may organize multiple units.

### Teacher and assistant separation

The API teacher is the only classroom voice. It owns dialogue, assessment,
adaptation, and one narrow `delegate_learning_work` tool. It does not receive
shell, file, code-execution, task-control, or general computer tools.

The native CLI is an asynchronous assistant for substantial verification,
consolidation, research, experiments, diagrams, documents, source parsing, and
material production. CLI work remains visibly launched and inspectable. It
never speaks as the teacher and does not choose a new learning agenda.

LLL is the mediator. Application services validate a conversation-native
proposal and later learner approval, create durable tasks, enforce five global
and two per-unit execution slots plus same-type active exclusion, seal input,
launch the CLI, validate output, commit assets, and report compact inline
status. No hidden task, time cooldown, automatic Agent retry, workflow graph,
or product cancellation API is added. Running work is cancelled only by a
learner interrupt in the visible terminal.

### Durable local contracts

The complete conversation is canonical local JSONL semantic history with
stable IDs. Provider deltas are transient. A rebuildable learning-aware compact
projection may manage model context at approximately 256K tokens while keeping
recent exact events and evidence references; it never replaces original events.

The active cumulative assets are `引入`, `正文`, and `练习`, plus extensible
generated artifacts. Core assets remain ordinary editable files with immutable
versions, conversation cursors, provenance, deterministic three-way merge, and
learner-content priority on overlap. The CLI writes only attempt staging and a
generic manifest; Go validates and promotes. Useful deliverables are not
restricted to the three core asset types.

Source material retains immutable local originals and revisions. Upload and
parsing are explicit, source content does not automatically enter teacher
context, and cloud processing requires disclosure. Tombstone and permanent byte
deletion are separate operations.

Annotation Ask AI remains attached to body selections as a separate lightweight
application service. It shares provider infrastructure but not the teacher
prompt, main conversation, or tools.

### Provider and frontend boundary

Provider adapters normalize streaming text, provider-supplied reasoning
summary, tool, usage, completion, and error events. Raw provider events and raw
hidden chain of thought do not enter application or frontend contracts.

Teacher messages safely render Markdown, code, LaTeX, Mermaid, sanitized
isolated SVG, and authorized PNG/JPEG references. Rich output remains inside the
single conversation rather than creating another workbench surface.

### Go application seam

New iteration-13 routes are transport adapters. Teacher, annotation Q&A, task,
dispatcher, asset commit, source, migration, and provider behavior live behind
application/domain interfaces and file repositories. The current Go server,
React SPA, global SSE connection, native CLI launcher, and filesystem-first
runtime remain; no database, message broker, or Agent SDK is required.

### Migration

Existing system-learning projects migrate in place to one canonical new layout
after a versioned backup and journal. Intro, Explain, Practice, and Explain
Ask-AI/confusion data map to intro, body, practice, and body annotations.
Summary and Extend remain inactive legacy data and are not inserted into body or
new model context. Knowledge-greenhouse and five-zone navigation are hidden,
not physically deleted. Migration failure restores a readable legacy project.

## Consequences

- Ordinary teaching can use low-latency API streaming while heavy work retains
  native CLI power and observability.
- The learner sees one coherent classroom voice and one conversation instead of
  coordinating multiple role modules.
- The backend gains more application contracts and recovery code, but route
  handlers become smaller and behavior becomes independently testable.
- Filesystem task scanning is sufficient for the local product, though it is
  not a distributed queue and must remain single-workspace coordinated.
- Versioned assets and attempt workspaces consume more disk; iteration 13
  chooses recoverability over automatic cleanup.
- Sanitized rich rendering and source processing add security test obligations.
- Old zone code and files coexist temporarily for migration and rollback, so
  the implementation must prevent dual writes after cutover.
- Discipline boundaries become pedagogical guidance rather than mechanical
  conversation restrictions; objective asset and task provenance still record
  what was actually used.

## Rejected Alternatives

- Keep the five-zone UI and add an unrelated API chat beside it.
- Use native CLI execution for every teacher token.
- Give the API teacher shell and unrestricted file tools.
- Let CLI output write formal project files directly.
- Run one hidden CLI extraction for every teacher turn.
- Summarize and discard old conversation events.
- Create five selectable role Agents or role-specific task types.
- Add Redis, Kafka, a database, or an Agent SDK before local contracts require
  distributed execution.
- Automatically retry, cancel, merge, or chain assistant tasks.
- Require learner acceptance or rejection of every generated asset draft.
- Move old Summary or Extend content into body merely to avoid data loss.

## Compatibility

ADR-0003 remains the foundation for native runtime inspection and raw run
records, but iteration-13 assistant Task and Run identities are separate and
durable. ADR-0007 flat physical project storage and ADR-0009 explicit discipline
overview navigation remain. ADR-0010 map hierarchy and learner-owned planning
remain. ADR-0011 scope snapshots remain provenance and starting guidance.

ADR-0002, ADR-0004, and ADR-0005 remain historical truth for legacy projects,
but their active five-zone system-learning presentation and direct-generation
artifact workflow are superseded after the iteration-13 migration cutover.
