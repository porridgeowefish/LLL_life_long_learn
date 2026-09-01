# Iteration 13: Teacher-Assistant Learning Workspace

Status: implemented; native provider/CLI smoke pending
Owner: project maintainer
Last reviewed: 2026-09-01
Source of truth: confirmed product and architecture decisions for the teacher-assistant learning-workspace transition.

## Goal

Replace the fixed five-zone product experience with one coherent learning
workspace centered on a real-time API teacher, asynchronous CLI assistants,
reusable learning assets, and learner-provided source materials.

The transition must reuse the current Go backend, file-first project storage,
CLI runtime, templates, and artifact infrastructure where their contracts
remain suitable. It is not a full-system rewrite.

## Target Product Shape

The product has three quiet top-level destinations: `教师`, `资产`, and `资料`.
`教师` is the default and visually dominant destination, built as one simple
continuous conversation. `资产` contains reusable outputs created through
teaching and assistant work. `资料` contains learner-provided books, documents,
PDFs, and other source files.

The existing knowledge-greenhouse presentation is removed from frontend
navigation during the cutover, but its files and backend behavior are not
deleted in this iteration merely because the old presentation is hidden.

## Confirmed Workstreams

1. Extract application orchestration from HTTP handlers before expanding the
   AI workflow. Handlers become transport adapters rather than owners of
   teaching, persistence, provider, and background-task decisions.
2. Add the real-time API teacher and a durable orchestration mechanism that
   turns bounded teaching needs into asynchronous CLI assistant work without
   exposing runtime or file-operation concerns to the teacher.
3. Add the teaching-asset and source-material libraries. Initial source-file
   parsing is performed through a controlled assistant task rather than a new
   in-process universal document parser.

## Delivery Scope

Iteration 13 delivers one coherent vertical transition rather than adding a
second experience beside the five zones. Included work is:

- extract teacher, annotation Q&A, task orchestration, asset commit, source,
  and provider concerns from HTTP handlers into application services;
- add one durable teacher conversation per system-learning unit, provider-
  neutral streaming, learning-aware context compaction, and safe rich-content
  rendering;
- add the disclosed-and-approved `delegate_learning_work` handoff, a local
  durable assistant queue, visible native-CLI execution, strict attempt
  workspaces, generic deliverables, and optional core-asset updates;
- replace the active five-zone presentation with `教师 / 资产 / 资料`, while
  preserving discipline overview, learning-plan, learning-scope, and folder
  classification behavior;
- migrate active legacy Intro, Explain, Practice, and Explain Ask-AI data into
  the canonical unit layout with a reversible pre-migration backup;
- retain annotation Ask AI as a lightweight service attached to body assets;
- keep old Summary, Extend, and knowledge-greenhouse data readable for
  rollback but absent from the new active experience.

Excluded work is multi-user or cloud sync, database introduction, hidden CLI
execution, automatic task generation, automatic Agent retry, UI or API task
cancellation, a general workflow engine, remote IDE behavior, raw hidden
chain-of-thought display, physical deletion of inactive legacy data, and
automatic migration of old Summary or Extend content into `正文`.

## Architecture Integration

The current Go server, filesystem workspace, native CLI launcher, React SPA,
TanStack Query, Zustand, and single global SSE connection remain the runtime
foundation. The architecture change is an extraction and contract boundary:

- HTTP handlers decode, authorize transport-level input, call one application
  operation, and encode HTTP or SSE output;
- `TeacherService` owns the main learning turn and provider-neutral tool loop;
- `AnnotationAskAIService` owns contextual Q&A on one asset annotation;
- `AssistantTaskService` and `TaskDispatcher` own durable admission, queueing,
  concurrency, recovery, and executor launch;
- `AssetService` and `AssetCommitService` own editable current files,
  immutable versions, merge, provenance, and transaction recovery;
- `SourceService` owns uploads, revisions, privacy decisions, and parse tasks;
- repository and provider adapters implement local files, normalized model
  streams, the native CLI, clocks, identifiers, and global SSE invalidation.

The filesystem remains canonical. No external message broker or database is
introduced. Existing handlers and stores may be adapted behind these services
incrementally, but new iteration-13 behavior must not be implemented directly
inside route files.

## Delivery Waves

1. **Application seam and compatibility readers.** Introduce service
   boundaries, repositories, stable IDs, migration inspection, and read-only
   compatibility without changing the default frontend.
2. **Teacher conversation.** Add canonical conversation persistence, provider
   gateway, streaming turn API, compact projection, safe Markdown/LaTeX/
   Mermaid/SVG/image rendering, and the single-chat teacher page.
3. **Assistant pipeline.** Add delegation authorization, durable tasks,
   dispatcher limits, visible CLI launch, run workspace, manifests, status
   projection, restart reconciliation, and manual CLI next-step guidance.
4. **Assets and annotations.** Add versioned `引入 / 正文 / 练习`, generic
   generated artifacts, learner-edit merge, Ask-AI migration, editing APIs,
   and provenance views.
5. **Sources.** Add upload, hashing, immutable revisions, explicit parsing,
   privacy disclosure, source references, deletion semantics, and previews.
6. **Migration and cutover.** Back up and migrate existing system-learning
   projects, enable `教师 / 资产 / 资料`, hide the five-zone and greenhouse
   surfaces, retain a rollback reader, and remove no legacy bytes.

Each wave has runnable acceptance paths in `TEST_PLAN.md`. The new frontend is
not the default until migration, recovery, streaming, task, asset, source, and
Ask-AI compatibility tests pass together.

## 2026-09-01 hardening alignment

- The three core asset documents are content-sized surfaces. Long structured
  Explain pages, tables, and practice material must expand their card border;
  a viewport-sized grid row must never let content escape the asset surface.
- Generic assistant deliverables are first-class assets, not task-log links.
  The Assets destination lists them under `助教成果`, presents title,
  description and file count, and renders a Markdown entry point in place.
  `artifact.json` remains an internal contract and is not the learner-facing
  representation.
- A visible CLI may finish writing after an observed process handle or an LLL
  server instance ends. A stable, valid late result manifest is authoritative:
  the dispatcher revalidates and commits it idempotently. A partial commit
  whose only missing pieces are generic deliverables is repaired on a later
  reconciliation pass without rerunning the Agent.
- Artifact descriptors accept canonical artifact-relative paths and bounded
  `deliverables/...` workspace-relative paths. Go repackages declared files
  under the generated-asset root; undeclared files and paths outside the
  attempt workspace remain rejected.
- Mermaid rendering is one frontend service, initialized once and serialized
  across readers. IDs are globally unique, Windows newlines and fence casing
  are normalized, and invalid syntax becomes a closed source fallback instead
  of breaking the page. The teacher is instructed to use a conservative
  flowchart subset and to choose SVG when Mermaid syntax is uncertain.

## Confirmed Decisions

### D13-001 — One classroom voice

The API teacher owns the learner-facing conversation. A CLI assistant never
inserts its own reply into the main teacher conversation.

### D13-002 — Assistant result boundary

A CLI assistant returns task state, produced artifacts, execution evidence,
failure details, and provenance to LLL as a complete task-result manifest. LLL
applies validated asset updates to the learner's editable asset workspace and
may expose a bounded result summary as context on a later teacher turn. This is
not a separate receipt product. The teacher receives neither asset-management
responsibility nor the complete production manifest merely because it expressed
the supporting-work need.

### D13-003 — Controlled template writes

Existing reusable templates remain available to assistant tasks, except that
the knowledge-greenhouse presentation is not treated as a reusable target
template. A CLI assistant may write content into an allowed reusable template
only through a scoped task contract and approved output targets; this decision
does not grant arbitrary project-wide file-write authority.

### D13-004 — Preserve before migration

Hiding the knowledge-greenhouse frontend does not authorize deleting legacy
zone files, projects, or readers. Compatibility and rollback remain required
until a later explicit migration decision supersedes them.

### D13-005 — Template definition and project instance

Reusable template definitions remain read-only product contracts. A CLI
assistant may instantiate a template or update the resulting project-owned
asset within the task's allowed output targets; it does not modify the shared
template definition while completing a learner task.

### D13-006 — Conversation-grounded assistant work

CLI assistant work is derived from accumulated teacher-learner conversation
context and an explicit teacher task. The assistant no longer receives an open
brief to independently decide what learning content to generate. Each task must
identify the conversation boundary it was derived from, its learning purpose,
its supplied context, and its allowed operations and outputs.

This constraint does not reduce the assistant to mechanical text replacement.
It may use professional judgment to organize material, generate substantial
learning outputs, and present evidence inside the confirmed task and
learning-scope boundary.

### D13-007 — Strict teacher-assistant work boundary

The teacher teaches. Its functional surface is limited to conducting the
learning conversation, assessing learner understanding, adapting the next
teaching move, and expressing a bounded need for supporting work. It does not
operate the CLI, manage files or templates, track retries, or directly create
or modify formal teaching assets.

LLL interprets and authorizes supporting-work intents, dispatches durable
asynchronous tasks, and manages their lifecycle. CLI assistants perform the
authorized production work and are the only AI role that produces teaching-
asset candidates; Go alone commits formal asset files. Consequently, teacher
conversation and assistant production
remain asynchronous after an explicitly approved task starts.

### D13-008 — Non-blocking teaching with turn-boundary result delivery

Assistant execution never blocks the teacher's current API response. Teaching
may continue while work is queued or running. Completed assistant results
update task state and the asset workspace without interrupting an in-progress
teacher response; LLL supplies a compact result notification to the teacher at
a subsequent turn boundary.

When the next teaching move truly depends on unfinished work, the teacher may
declare a learning checkpoint and avoid advancing that dependent branch, but
the synchronous teacher request still does not wait on the CLI process. The
learner may inspect progress, wait, or manually interrupt the visible terminal.

### D13-009 — Draft-first assistant output

Superseded by D13-019. The earlier draft-first proposal introduced an
accept/reject/promotion workflow that was rejected as unnecessary product
complexity.

### D13-010 — Learner-owned asset curation

The learner owns direct editing of generated assets. The teacher does not
review generated content for retention, recommend versions, approve changes,
or participate in asset management. D13-019 supersedes the earlier
accept/edit/discard workflow: generated content is committed directly as an
editable asset version rather than waiting for learner promotion.

This ownership rule is separate from the still-open question of whether a
completed assistant result may be supplied to the teacher as context needed
for subsequent teaching. Context consumption, if allowed, does not confer any
asset-management authority.

### D13-011 — Separate teaching receipts from asset updates

Superseded by D13-063. Assistant completion has durable task results and asset
updates, but no separate teaching-receipt surface. On a later learner turn, LLL
may include a bounded task-result summary in teacher context when relevant. It
does not supply asset content for teacher review or management.

### D13-012 — Narrow teacher tool boundary and non-IDE assistant

The real-time API teacher has no general-purpose execution or file-operation
tool surface. It may use the single narrow assistant-handoff operation defined
by D13-016, but cannot execute the resulting work. It otherwise reasons from
the supplied conversation and learning context, teaches within that evidence
boundary, and explicitly avoids claiming that it ran code, inspected local
state, or verified an external result.

Small interactive actions such as running a learner's snippet, reproducing an
ordinary error, invoking a shell command, or performing an IDE-style debug loop
do not justify an assistant task. The learner uses their normal development
environment for those actions and may bring the observed result back into the
teaching conversation. CLI assistants are reserved for substantial,
asynchronous production work rather than acting as a remote IDE.

### D13-013 — Execution allowed inside substantial assistant work

A CLI assistant may execute code, use specialist tools, and perform multi-step
operations as internal implementation details of an authorized substantial
deliverable. This authority does not create a learner-facing remote code runner
and does not route small interactive debugging or ordinary IDE actions through
the assistant-task system.

The task contract remains responsible for bounding inputs, operations, output
targets, cost, time, and evidence. Internal execution authority does not widen
the learning scope or grant unrestricted project access.

### D13-014 — No generic compression of learning history

The complete teacher-learner conversation remains the canonical learning
record. A generic rolling summary, ordinary conversation compaction, or
teacher-produced per-turn summary must not replace it as the source used to
ground assistant work. Such compression can erase learner wording,
misconception transitions, explanation history, uncertainty, and source
dependencies that remain pedagogically significant.

LLL may build lossless structural indexes and references over the canonical
record. Any future lossy representation requires a learning-specific design,
explicit provenance back to source turns and materials, and evaluation against
defined pedagogical information-preservation criteria. Until that contract is
approved, derived summaries are navigation aids only and cannot be the sole
context supplied for a task.

### D13-015 — No learning-history compression in iteration 13

Superseded by D13-031. The canonical conversation is still retained in full,
but iteration 13 may maintain a small rebuildable auto-compacted projection for
teacher context-window management.

### D13-016 — Teacher handoff tool inside the learning conversation

The teacher may invoke one narrowly scoped handoff tool when its soft teaching
methodology identifies substantial verification, consolidation, reflection, or
material-production work suitable for the asynchronous CLI assistant. A
learner's explicit natural-language request may prompt the same tool use, but
is not the only valid origin. There is no dedicated consolidation button.

LLL does not infer tasks from turn counts, idle timers, or a mandatory teaching
state machine. The model makes an explicit tool call within the conversation;
the tool returns quickly after durable task acceptance and never waits for the
CLI work to finish.

The handoff tool does not expose the CLI, shell, file paths, templates, or task
lifecycle to the teacher. It records the learner's requested scope and desired
outcome; LLL resolves that request into an authorized asynchronous assistant
task. Other learner-facing controls may invoke the same LLL application
operation without changing the role boundary.

### D13-017 — Local conversation history under a path contract

The complete conversation history is stored locally as project-owned data and
remains the canonical input record for later assistant tasks. Its directory
and file layout follow the project's path-is-contract principle: stable paths,
identities, and ownership boundaries are part of the persistence contract and
must not be treated as incidental implementation details.

Assistant tasks reference a frozen conversation identity and exact source
scope through that contract. They do not begin by rediscovering, scraping, or
generically summarizing the current UI conversation. The concrete path and
serialization layout remain to be specified in the data contract.

### D13-018 — System-defined incremental scope and append-based assets

The learner does not select or highlight a conversation range when requesting
material generation. LLL computes one default incremental source range for the
target asset, ending at the learner's current request turn and beginning after
the last conversation position successfully incorporated into that asset.

Generation is cumulative rather than a fresh summary. The assistant task reads
both the current saved asset version and the exact new conversation range, then
produces a next version that adds the newly learned material to the prior
content. Each asset therefore owns its own incorporation cursor and provenance
ranges; there is no conversation-wide "already summarized" cursor shared by
unrelated assets. A successfully committed update advances that asset's cursor;
a failed task does not.

### D13-019 — Direct-to-editable asset updates

A successful teacher-handoff generation task commits its validated result as
the current editable asset version. The product does not expose draft inbox,
accept, reject, promotion, or regenerate actions. If the learner dislikes the
generated content, the normal response is to edit the asset directly.

Later generation reads the latest saved asset, including learner edits, and
appends the next unincorporated conversation range. Internal version history
keeps the preceding state recoverable, but it is a safety mechanism rather than
an approval workflow. There is no automatic regeneration loop.

### D13-020 — Generate in conversation, edit in assets

The primary learner interaction stays inside the teacher conversation. An
explicit natural-language request to organize recent learning may invoke the
narrow handoff operation; there is no dedicated consolidation button competing
with the ordinary send action.

After submission, the conversation shows a compact non-speaking inline status for
queued, running, succeeded, partial, failed, or cancelled state. The status is not a teacher or
assistant chat message and does not interrupt teaching. Successful updates are
opened and edited in the asset workspace, whose responsibility is viewing and
editing rather than accepting, rejecting, promoting, or regenerating output.

### D13-021 — SDL roles organize both teaching and generation

Learning behavior and generated material are organized around five named
roles: `引导人`, `澄清者`, `验证者`, `沉淀者`, and `复盘者`. The replacement
must not recreate the old fixed-zone model under generic asset labels such as
summary, explanation, practice, extension, and review.

The teacher requires explicit pedagogical methods corresponding to these roles
so that they shape how it guides the learner in conversation. D13-026
supersedes the earlier idea of managing them as separate role contracts or
plugins: they are sections of one unified teacher methodology prompt.

The role methods do not all have to remain inside the synchronous conversation.
When substantial work is pedagogically useful, methods such as verification,
consolidation, and reflection may lead the teacher to use its narrow handoff
tool and delegate that work to the asynchronous CLI assistant.

### D13-022 — Roles are dynamic methods, not learner-selected modes

All five SDL roles are available to the teacher as a standing pedagogical
method library. The teacher applies and transitions among them according to the
live teaching need; the learner is not required to select a current role or
switch a five-mode UI. The resulting role-shaped conversation provides the
pedagogical source for later assistant organization, but the roles do not
define five matching asset columns or five independently managed components.

### D13-023 — Three active asset destinations

For the current iteration, cumulative assistant output is organized into three
active destinations: `引入`, `正文`, and `练习`. `正文` is the ordinary editable
teaching body; it may naturally contain conclusions from the current learning
conversation, but legacy Summary and Extend content is never inserted into it.
The five SDL roles may shape content written across these
destinations and are not mapped one-to-one onto them.

The legacy extension and standalone-summary destinations are not part of this
active three-destination surface. Their preservation or removal state is the
next compatibility decision. If retained as inactive definitions, they belong
to an asset-template library, not the source-material library used for learner
uploads.

### D13-024 — Preserve disabled extension and standalone summary

The legacy `延伸` and standalone `总结` definitions are disabled for iteration
13 rather than physically deleted. They are hidden from the active frontend,
excluded from learner-triggered generation, and not created by default for new
projects. Existing project data remains readable for compatibility and
rollback.

Retained definitions belong to the inactive section of the asset-template
library. They do not belong to the source-material library, whose contract is
reserved for learner-provided books, PDFs, documents, and other source files.
A later explicit migration decision is required before physical deletion.

### D13-025 — One handoff, one CLI task, three asset checks

Narrowed by D13-061 and D13-062. The three-asset check applies to a
conversation-consolidation handoff, while verification and material-production
tasks may return only generic deliverables.

The teacher's narrow `tool_use`, exercised under D13-016, is the generation
gate. One successful handoff creates one CLI task;
LLL does not create a separate invocation for each asset destination.

That single task reads the incremental conversation range once and inspects the
current `引入`, `正文`, and `练习` assets. It may commit an update to each
destination that has a meaningful addition and leave the others unchanged. Its
result manifest reports the outcome per destination so partial work is visible
without multiplying context extraction or CLI startup cost. Other task types
may omit all core asset updates.

### D13-026 — One unified default teacher methodology prompt

The five SDL role definitions are authored and versioned together inside one
teacher system prompt. They are complementary principles used dynamically by
one teacher, not separate prompts, packages, plugins, modes, or runtime roles.
Iteration 13 therefore implements no per-role install, enable, disable,
selection, or lifecycle management.

The unified methodology is a soft default rather than a state machine or a
mandatory conversation ceremony. The teacher applies the five perspectives
adaptively to help the learner understand; LLL does not implement methodology
override state, per-turn bypass flags, workflow enforcement, or UI controls for
entering and leaving the roles.

The prompt may describe when substantial verification, consolidation, or
reflection is suitable for delegation. The teacher still uses only the narrow
handoff tool, while LLL resolves and executes the durable assistant task.

### D13-027 — Abstract task service between teacher API and CLI

The teacher API never invokes, configures, or monitors a CLI runtime directly.
Its handoff tool calls an application-level assistant-task service using a
logical task intent. That service resolves the durable task record, authorized
input references, output locations, versioned result format, notification
anchor, execution policy, and lifecycle.

CLI runtimes implement an executor abstraction behind the service. The current
native CLI launcher is one adapter; a future background API runtime or Agent
SDK may implement the same executor contract without changing the teacher tool
or product task model. Product tasks and CLI sessions remain distinct domain
objects linked by identifiers.

The executor writes only to its task-scoped output area. LLL validates the
versioned result manifest and performs versioned asset commits. Immediate tool
acceptance is returned through the teacher stream, while durable task state is
observable through query APIs and SSE notifications anchored to the originating
conversation turn.

### D13-028 — Conversation is a first-class project path boundary

Teacher conversations are stored under the canonical root
`projects/<unit-slug>/conversation/`. Because a conversation forms exactly one
learning unit, the unit does not contain a plural conversation collection or a
second nested conversation identifier. The conversation is not
embedded in legacy `explain/confusions.json`, project state, assistant-task
records, or CLI run directories.

The three durable concepts remain separate and reference one another by stable
identifiers: a conversation records teacher-learner interaction, an assistant
task records product work requested by a teacher tool call, and a run records
one executor attempt. Task and run directories do not copy the canonical
conversation history.

### D13-029 — Conversation metadata plus append-only event timeline

Each canonical conversation directory contains `conversation.json` for
versioned conversation metadata and `events.jsonl` for the complete append-only
timeline. Message bodies are retained in full; no conversation compression is
applied.

Timeline records use an increasing per-conversation sequence plus stable,
type-prefixed opaque identifiers. They represent messages, teacher-response
boundaries, tool calls, and task links. Provider-specific call identifiers are
external references rather than LLL identities. Streaming token deltas are not
durable events; an unmatched response-start record is reconciled as interrupted
after restart.

Task state is not duplicated into the conversation log. A `task-linked` event
stores the task identifier and message anchor, and the frontend projects the
inline task status by joining that reference with the durable assistant-task service.
Conversation Repository is the single serialized writer for each event log.

### D13-030 — Conversation-first, one-to-one learning units

A generated discipline overview remains useful as guidance, but its chapters,
topic catalog, and suggested boundaries do not impose a hard teaching
partition. A discipline-backed folder may contain multiple conversation-unit
pairs.

The conversation comes first: starting one teacher conversation forms exactly
one learning unit. The unit is the product projection around that conversation
and owns the assets, source materials, assistant tasks, and progress accumulated
through it. A unit does not contain multiple teacher conversations, and LLL
does not require a separately created learning-project shell before
conversation can begin.

The teacher may naturally cross related topics inside the conversation. LLL
does not refuse teaching because more than one overview topic is involved and
does not force another conversation merely because the topic changes. Existing
scope metadata may guide context and provenance, but must not act as a hard
exclusion gate.

### D13-031 — Rebuildable auto-compaction for teacher context only

The full append-only conversation remains canonical. Auto-compaction is a
derived, rebuildable projection used only to fit the long-lived conversation
into the teacher model's context window. It never deletes, overwrites, or
becomes a replacement for source events.

Context assembly combines the unified teacher system prompt, a compacted view
of older covered events, and recent exact events. Compaction is triggered by
context pressure rather than every turn. Its preservation schema, threshold,
and regeneration rules remain part of the context contract to be aligned;
generic free-form summarization is not sufficient.

### D13-032 — Adapt existing learning projects in place

Existing system-learning project directories are adapted in place as the new
conversation-first learning-unit shape. They are not recreated and are not
required to regenerate their existing learning content. The existing project
identity, discipline-folder membership, progress, and run history remain valid,
while their persisted content is migrated into the iteration-13 canonical
layout. Permanent legacy-path compatibility is superseded by D13-033.

Old zone-generator JSON requirements must not become mandatory fields in the
new assistant-task or asset-append contracts merely because previous direct
generation used them. Obsolete field constraints and validators are removed
after a compatibility audit identifies their legacy readers. Standalone
summary generation is no longer an active pipeline. Conversation-grounded
consolidation may produce ordinary notes or conclusions in an appropriate
active asset, but it does not require writing them to body. Existing standalone
`summary` and `extend` outputs are not imported into the body asset.

### D13-033 — One canonical storage shape after migration

All learning units, including existing system-learning projects, converge on
the same iteration-13 physical layout and schemas. LLL does not keep legacy
zone paths and new asset paths as two permanent writable representations, and
does not make every future reader branch on project age.

Existing project identity and discipline-folder membership are preserved while
a versioned migration converts legacy artifacts into the canonical
conversation, asset, and source-material shape. Migration must be restart-safe,
auditable, and recoverable before legacy storage is retired. The exact
canonical directory layout and legacy-content landing rules are the next
decisions.

### D13-034 — Teaching assets share one extensible container

The three active cumulative teaching assets use the canonical physical paths
`assets/intro/`, `assets/body/`, and `assets/practice/` inside the learning
unit. `body` is the正文 area; there is no active standalone summary destination
and no automatic legacy-summary merge.

Assets are grouped under one container so future asset types can be introduced
as siblings without expanding the learning-unit root or reviving the former
five-zone layout. Legacy `intro`, `explain`, and `practice` content is migrated
into these destinations according to explicit landing rules that will be
defined before implementation.

### D13-035 — Flat learning-unit roots remain canonical

Every conversation-formed learning unit uses the stable flat root
`projects/<unit-slug>/`. Discipline folders remain navigation and
classification metadata; they do not become physical parent directories for
learning units.

Iteration 13 therefore changes the meaning and internal layout of the former
system-learning project directory without introducing a nested
`projects/<discipline>/<unit>/` hierarchy. Moving or renaming a discipline
classification does not move the unit, change its stable identity, or rewrite
conversation, asset, source, task, and run references. Existing projects can be
migrated within their current root directories.

The canonical top-level shape reserves `unit.json`, `conversation/`, `assets/`,
`sources/`, `assistant-tasks/`, and `runs/`. This decision freezes their root
placement only; the metadata schemas and the internal task, run, and source
contracts remain subject to their dedicated alignment areas.

### D13-036 — Metadata JSON, append-only JSONL, rebuildable compact projection

The canonical `conversation/` directory contains `conversation.json` for
versioned conversation metadata, `events.jsonl` for the complete ordered event
timeline, and `compact.json` for the derived learning-aware context projection.
The compact file is disposable and rebuildable; it never replaces or mutates
the event log.

`events.jsonl` stores one complete semantic event per UTF-8 line. New events are
appended instead of rewriting a growing JSON array. The Go Conversation
Repository is the only writer for a unit, assigns the monotonic sequence, and
detects a truncated final line during recovery. Provider token deltas are
transient transport data and are not persisted as canonical events.

This representation preserves path-as-contract portability while supporting
incremental reads, exact task and asset provenance, and bounded crash recovery.
Human-readable Markdown may be exported as a projection, but is not a source of
truth because tool calls, response boundaries, and stable references must not
be reconstructed from prose.

### D13-037 — Six durable semantic events and LLL-owned stable IDs

The canonical event vocabulary is limited initially to `message-recorded`,
`teacher-response-started`, `tool-call-requested`, `tool-result-recorded`,
`teacher-response-finished`, and `task-linked`. These events preserve complete
messages, response recovery boundaries, normalized tool interaction, and the
stable anchor needed to project inline task status. Task progress transitions remain
owned by the assistant-task repository and are not copied repeatedly into the
conversation log.

Every JSONL record has the common envelope `schemaVersion`, monotonic `seq`,
stable `eventId`, `type`, `occurredAt`, and type-specific `data`. LLL generates
opaque, type-prefixed ULIDs for `unit`, `conversation`, `event`, `message`,
`response`, `tool call`, `task`, and `run` identities. IDs never encode mutable
titles, disciplines, paths, roles, or statuses.

Provider-native response and call identifiers are optional external references
only. Provider token frames are transient, and provider event schemas are
normalized before persistence. This keeps stored conversations reconstructable
when a provider adapter changes and allows a started response without a matching
finished event to be reconciled as interrupted after restart.

### D13-038 — Inline Ask AI remains an asset-annotation capability

The conversation-first redesign retains the existing ability to select or
annotate content in a final teaching asset and ask AI a contextual question.
These exchanges are asset-annotation threads, not additional primary teacher
conversations and not separate learning units. The one-to-one relationship
between the main teacher conversation and the learning unit therefore remains
unchanged.

Legacy `explain/confusions.json` selections, annotations, questions, and
answers are migrated into the annotation storage owned by the corresponding
canonical `assets/body/` content. Their stable legacy identifiers, source
artifact references, quote snapshots, selection ranges where still resolvable,
message order, and timestamps are preserved. They are not merged into
`conversation/events.jsonl` and are not discarded when the old `explain/`
directory is retired.

The provider behavior, prompt mode, context relationship, and canonical
annotation schema remain to be aligned. This decision freezes only that inline
Ask AI survives as a distinct contextual interaction attached to assets.

### D13-039 — Teacher and annotation Q&A are separate application services

`TeacherService` and `AnnotationAskAIService` are separate application-level
capabilities. The teacher owns the primary learning conversation, the unified
five-method teaching prompt, and the narrowly scoped assistant-handoff tool.
Annotation Ask AI owns only contextual questions and follow-ups attached to a
selected teaching-asset passage.

Annotation Ask AI does not load the teacher methodology, primary conversation
history, or assistant tools. Its compact prompt is grounded in the selected
quote, nearby asset content, the owning asset version, and the current
annotation thread. It may use a different, faster model configuration from the
teacher.

The two services share only infrastructure through the provider gateway:
provider SDK adapters, normalized streaming transport, timeout and error
handling, credentials, and usage accounting. This avoids duplicating provider
integration without coupling localized Q&A to the richer teacher behavior.

### D13-040 — Active legacy content maps to three assets; summary and extend do not

`assets/body/` is the canonical new `正文` area and the primary final-note
surface. Migration maps legacy `intro` content to `assets/intro/`, legacy
`explain` tutorials, pages, and media to `assets/body/`, and legacy `practice`
structures to `assets/practice/`. Existing `explain/confusions.json` content is
converted into body-owned annotation and Ask AI threads rather than primary
teacher-conversation events.

Legacy standalone `summary/` and `extend/` outputs are deliberately excluded
from this migration. They are not merged into body, shown as active assets,
read as teacher or assistant context, or accepted as new CLI destinations.
Iteration 13 does not need to physically delete or otherwise curate those
inactive files; physical archival or cleanup is outside this migration's
acceptance boundary.

The conversion is deterministic and must preserve structured practice records
and asset-owned media rather than flattening everything into Markdown or asking
AI to regenerate existing content.

### D13-041 — REST recovers durable history; SSE only accelerates delivery

The browser reads conversation metadata and ordered events through the Go
application API, never by accessing project files directly. The canonical read
surface provides conversation metadata plus bounded event reads using
`afterSeq` and `limit`, returning the observed `lastSeq` and pagination state.
There is no generic client-facing append-events endpoint; business operations
are the only path to repository writes.

SSE delivers newly committed events for low-latency UI updates but is not a
source of truth. After refresh, reconnect, or a detected sequence gap, the
frontend resumes from its last contiguous sequence through REST. A missed SSE
frame therefore changes latency, not durable state or correctness.

Conversation events restore messages and inline-status anchors. Current task status
is joined from the durable assistant-task read API instead of inferred from old
conversation events. `compact.json` remains an internal context-assembly
projection and is never returned as a replacement for learner-visible history.

### D13-042 — Background API compaction with Go-owned validated persistence

Learning-aware conversation compaction is performed by a dedicated
`ConversationCompactionService`, not by the teacher response, a CLI assistant,
or an agent runtime. After a teacher turn, the Go application evaluates context
pressure and may invoke an ordinary model through the shared provider gateway
as asynchronous internal maintenance work.

The model receives an explicit source sequence range and returns a typed compact
projection. It has no file tools. Go validates the response, verifies its source
range against the canonical event log, and atomically replaces `compact.json`.
The service may use a faster or cheaper model configuration independently of
the teacher.

Compaction never appears as a learner-visible assistant task. When a provider
hard-window guard requires a fresh projection, the next teacher request waits
for that projection instead of sending an oversized or silently truncated
context. Failure preserves the previous compact projection and the complete
event log. Main teacher context combines the unified system prompt, the
validated compact projection, and uncovered recent exact events. Asset Ask AI
annotation threads are outside this compaction stream.

### D13-043 — 256K-token compaction trigger with provider safety guard

The product-level automatic compaction trigger is an estimated assembled main
teacher context of `256K tokens`, not a turn count or a percentage of the
selected model. When the threshold is crossed after a teacher response, the
application starts background compaction and normally keeps approximately the
most recent `64K tokens` as exact events.

The absolute threshold assumes a provider context window large enough to carry
it. A provider-specific hard guard still applies before dispatch: reserved
output capacity, the system prompt, and the provider's actual input limit must
never be exceeded. If a configured model cannot safely accept the 256K product
threshold, LLL compacts earlier at its safe input boundary rather than sending
an invalid request or silently dropping events.

Compaction is not run on every turn. If the asynchronous projection is not
ready when the next request reaches the provider's hard guard, context assembly
waits for or immediately completes compaction and exposes a temporary
context-preparation state only when that wait becomes learner-visible. Raw
events are never truncated as a fallback.

### D13-044 — Compact projection preserves typed learning state and evidence

`compact.json` is a structured learning-state projection rather than a generic
narrative summary. It records the covered sequence range and source hash plus
typed collections for learning goals, concepts and current understanding,
misconceptions and corrections, open questions, verified conclusions,
important explanations and examples, learner preferences and prior knowledge,
decisions and commitments, and durable asset, source, task, or event references.

Material claims in the projection carry source event identifiers and an
epistemic status so that `discussed`, `learner-understood`, `corrected`,
`unverified`, and `verified` are not collapsed. Exact excerpts are retained for
critical learner questions, definitions, formulae, code, repeatedly referenced
examples, and wording whose meaning would be damaged by paraphrase.

The projection remains deliberately lossy and never replaces raw events. Its
range and hash make staleness detectable, its references make important items
auditable, and a schema or prompt revision can rebuild it from the complete
event log. Annotation Ask AI threads remain outside this main-dialogue compact
projection.

### D13-045 — Teacher handoff exposes objective and optional source references only

Partially superseded by D13-055: same-type active-task exclusion requires one
primary operational `taskType` in addition to `objective` and optional
`sourceRefs`. The rejection of pedagogical workflow kinds and all other
injected infrastructure fields still stands.

The model-facing `delegate_learning_work` tool accepts only a required natural-
language `objective` and optional stable logical `sourceRefs`. It does not
expose a `kind` enum for verification, consolidation, reflection, or material
production. One objective may legitimately combine all of those pedagogical
intentions.

LLL injects the unit, conversation, triggering message, asset cursors and
locations, exact conversation range, task identity, staging and result
contracts, executor policy, failure policy, and notification anchor. The teacher
cannot supply physical paths, commands, executors, models, output schemas,
conversation ranges, timeouts, or notification destinations.

All teacher-originated handoffs share the logical work class `learning-work`.
Verification, consolidation, reflection, and material production remain soft
methodological descriptions that may become derived observability tags, never
required tool arguments or workflow states. Upload parsing and other
system-originated work enter the assistant-task service through separate
application operations rather than this teacher tool.

### D13-046 — Provider adapters emit complete provider-neutral model events

All teacher-capable provider adapters normalize native streams into the same
internal event union: `text-delta`, `tool-call-ready`, `usage`,
`response-completed`, and `response-failed`. Provider-specific event objects do
not escape the adapter into `TeacherService`, conversation persistence, or task
orchestration.

Providers may stream a tool call name and arguments across multiple frames, but
the adapter buffers them and emits `tool-call-ready` only after the argument
document is complete JSON. The normalized event carries the tool name,
arguments, and optional provider call ID as an external reference.
`TeacherService` then validates the registered tool name and its application
schema before invoking any operation.

Adapters also translate the normalized application tool result back into the
provider's continuation format. This ordinary SDK adapter loop is sufficient;
iteration 13 does not require an Agent SDK. Provider-native frames may be kept
in bounded diagnostic traces but are neither durable conversation events nor
frontend SSE contracts.

### D13-047 — Durable acceptance, explicit rejection and failure, one task per response

`delegate_learning_work` returns exactly one of `accepted`, `rejected`, or
`failed`. `accepted` is emitted only after the assistant task has been durably
created and returns its stable `taskId`; it never implies that an executor has
started or that an asset has changed. `rejected` represents an expected
business refusal with a stable reason code and creates no task. `failed`
represents an infrastructure inability to create the task and declares whether
a new explicit attempt may be safe.

The teacher is instructed to describe accepted work as queued or started, not
completed; explain an actionable rejection; and state clearly that no task was
started after failure. Tool results expose no physical paths, executor details,
or speculative completion receipt.

One teacher model response may have at most one accepted learning-work handoff.
If a provider emits additional calls, they are rejected with
`one-task-per-response`. A complex objective remains one task and may combine
verification, consolidation, reflection, asset updates, and supporting
material. Provider or transport retries are resolved by the task service's
idempotency contract rather than creating another task.

### D13-048 — Local durable assistant queue, separate from executor runs

`AssistantTaskService` persists each accepted product task under the owning
unit before returning `accepted`. A Go `TaskDispatcher` reconciles durable tasks
on creation and application startup, selects eligible queued work, acquires a
lease, seals its executor input, and launches a replaceable executor adapter.
Iteration 13 requires no Redis, Kafka, external broker, or separate queue
server.

A product task and an executor run are different identities and lifecycles. One
task expresses the learner-visible objective and may survive restart, be
cancelled, or own multiple explicitly initiated attempts. Each run represents one
specific CLI or future executor attempt. A failed or interrupted run therefore
does not imply creation of another product task.

Durable task files are the source of truth. Any workspace-wide scheduling index
is derived and rebuildable by scanning unit task repositories. Leases and
heartbeats prevent duplicate execution and allow an orphaned running attempt to
be reconciled after restart.

### D13-049 — Five global assistant slots, two per learning unit

Superseded by the user's simpler concurrency policy. The initial dispatcher has
a configurable global assistant concurrency limit with a default of `5` and a
per-learning-unit limit with a default of `2`. Tasks otherwise enter the normal
accepted-time queue when the applicable limit is full.

Iteration 13 does not implement a resource-lock compatibility matrix to decide
whether two different task types may coexist. The per-unit and global counters,
plus the same-type active-task exclusion in D13-054, are the complete initial
admission and scheduling policy.

### D13-050 — Two-phase input sealing at acceptance and first execution

Task acceptance freezes the learner-visible objective, the main-conversation
cutoff, explicit source references, and the authorized logical source scopes.
It does not immediately enumerate every file in a permitted collection or pin
the current asset versions while the task may still be queued.

When the first executor attempt enters `preparing`, LLL resolves the authorized
scope once, captures the then-current
base versions of `intro`, `body`, and `practice`, and writes an immutable input
manifest containing source revision identifiers and hashes. Files that become
available in an authorized collection before sealing are included; external
files created after sealing are not injected into the running task.

All retries reuse the same sealed manifest. Iteration 13 has no general task
dependency graph: if another task's output is required, the learner explicitly
creates the follow-up after that output commits. Accidental execution order is
not a dataflow contract. Files produced by the executor inside its own staging workspace may
be used as intermediates but are outputs of that attempt, not late external
inputs. Physical folder proximity alone never expands authorization.

### D13-051 — TaskRecord, sealed InputManifest, and per-attempt RunEnvelope

Assistant persistence separates three responsibilities. Mutable `task.json`
is the product TaskRecord and owns stable identity, origin, objective, lifecycle,
attempt references, timestamps, and learner-visible failure information.
Immutable `input-manifest.json` owns the concrete conversation range, asset base
versions and cursors, source revisions and hashes, and satisfied dependency
inputs sealed before the first attempt.

Each executor attempt owns `attempts/<run-id>/envelope.json`. Its RunEnvelope
contains the task and run identities, selected executor adapter, resolved input
manifest location, attempt workspace and staging locations, and result-contract
version. The initial contract launches one executor attempt for a task. Failed
and cancelled tasks remain terminal; repeating work creates a new explicit task.

Executors consume their RunEnvelope and write attempt-scoped output; they do not
mutate `task.json`, advance asset cursors, or determine product lifecycle state.
LLL validates executor output and owns task transitions. Run directories may
eventually follow a cleanup policy without erasing the durable product task,
input provenance, committed asset versions, or final result references.

### D13-052 — Explicit task creation and a deliberately small lifecycle

Iteration 13 persists only `queued`, `running`, `succeeded`, `partial`,
`failed`, and `cancelled` as product task statuses. While `running`, a small
internal phase may report `preparing`, `executing`, `validating`, or
`committing`; those phases do not create a workflow engine or additional
learner actions.

Assistant tasks are created only by an explicit teacher
`delegate_learning_work` tool call or an explicit learner operation such as the
fixed consolidation action or uploading material for parsing. Turn counts,
context length, inactivity, schedules, task completion, or a hidden teaching
state never create a CLI task. Conversation compaction remains separate
internal maintenance and is not an assistant task.

There is no automatic executor retry, exponential backoff workflow, or general
task dependency graph in the first implementation. Queued tasks resume after
restart; an interrupted running executor produces an explicit failed outcome.
Atomic commit recovery may finish or roll back an already-started filesystem
transaction without invoking the agent again.

### D13-053 — Failure is reported inline; a manual CLI next step is suggested

After an assistant task fails, its conversation-anchored system card displays
the failed stage and safe failure reason plus a human-readable suggested CLI
instruction. This is system UI below the related conversation item, not teacher
speech, a blocking modal, an automatic tool call, or a frontend retry button.

The learner may copy, inspect, edit, and manually publish that suggestion in a
visible CLI window. LLL does not interpret this action as a retry protocol and
does not automatically import its files. Failed and cancelled tasks remain
terminal. If the result must be committed into LLL, the teacher discloses it
again and the learner approves a new explicit task; the earlier diagnostics
remain preserved.

### D13-054 — Simple same-type active-task exclusion

Before creating a task, `AssistantTaskService` atomically checks active tasks
for the same learning unit and operational task type. A new call is rejected
and returns the existing task's identity, status, and safe result summary when
a same-type task is still queued or running. Terminal tasks never block a new
explicit call; there is no time-based cooldown.

Different task types may coexist up to the per-unit concurrency limit of two,
and all units share the global limit of five. No objective similarity model,
resource read/write graph, automatic task merging, or hidden follow-up is used.
The exact source and vocabulary of the operational task type remain the next
teacher-tool decision because D13-045 previously removed pedagogical `kind`
from the model-facing contract.

### D13-055 — One primary operational task type for active exclusion

`delegate_learning_work` adds a required primary `taskType` with exactly
`consolidate`, `verify`, or `produce-material`, alongside `objective` and
optional `sourceRefs`. Upload-triggered parsing is assigned the internal type
`source-processing` by LLL and is not a teacher tool value.

The type is an active-exclusion and presentation key, not a workflow, permission set,
asset target, or exclusive capability boundary. A task may perform supporting
actions associated with other methods; mixed work selects the primary requested
outcome. Reflection is not a separate type and normally remains part of
consolidation.

For one unit, a queued or running same-type task rejects the new call and
returns the existing task status or safe result. As soon as that task is
terminal, the next explicit same-type call may create a new task. Different
types may run together within the unit limit of two and global limit of five
without a resource-lock matrix.

### D13-056 — Cancellation exists only as a learner's manual terminal interrupt

Iteration 13 exposes no conversation-card cancel button, cancellation API,
teacher cancellation tool, dispatcher kill action, or automatic timeout
cancellation. A task can be cancelled only after its executor has been visibly
launched in a terminal and the learner manually interrupts or closes that CLI
execution.

The launcher observes the intentional terminal interruption and records the
run and product task as `cancelled`; an unexpected process loss remains
`failed`. Queued work cannot be cancelled through another product surface.
After the executor exits and LLL enters validation or commit, terminal
interruption is no longer available and the filesystem transaction follows its
normal validation and recovery rules.

Manual interruption commits no unvalidated staging output, does not delete the
TaskRecord, InputManifest, run logs, or diagnostics, and produces a system
status notice below the anchored conversation message rather than teacher speech.

### D13-057 — Teacher discloses the assistant instruction, output, and value first

Before emitting `delegate_learning_work`, the teacher tells the learner in
plain language the exact logical instruction it is about to delegate, what the
assistant is expected to inspect or produce, and why that output matters for
the current learning process. The disclosed instruction is the same
`objective` placed in the tool call; it is not a hidden shell command or a
teacher-authored physical path.

Expected output is described honestly as intended inspection or possible
updates because the assistant may return `unchanged` for any of `intro`,
`body`, and `practice`. The teacher does not promise completion, claim that the
CLI has already run, or hide the cost and scope of an asynchronous invocation.
The confirmation boundary is completed by D13-058.

### D13-058 — Every teacher delegation waits for learner feedback

An explicit learner request for consolidation, verification, or material
production starts proposal formation; it does not authorize immediate tool use.
The teacher first discloses the concrete logical instruction, expected output,
and learning value, then waits for a subsequent learner response before
emitting `delegate_learning_work`.

Learner feedback may approve, revise, or decline. Approval applies only to the
displayed proposal version. A revision causes the teacher to present the
changed instruction and wait again; decline creates no task. The original
request, a fixed consolidation action, and a teacher-originated suggestion all
follow this same feedback boundary, with no same-turn disclosure and execution.

This is conversation-native approval rather than a blocking modal. No executor
is launched, no assistant slot is consumed, and no TaskRecord is created while
the proposal awaits feedback. The durable proposal identity and its connection
to the final tool call remain the next contract decision.

### D13-059 — Conversation messages are the delegation authorization record

Iteration 13 adds no proposal tool, approval modal, confirmation button,
proposal file, or separate proposal state machine. The teacher's durable
proposal message and the learner's subsequent durable feedback message are the
authorization evidence for a teacher-originated assistant task.

`delegate_learning_work` therefore carries only the final `taskType`,
`objective`, and optional `sourceRefs`. LLL derives the adjacent prior teacher
proposal and current learner approval from durable history, then records both
message IDs plus the normalized tool call ID in the TaskRecord origin. Internal
IDs never need to enter model-visible content. LLL validates unit ownership,
author roles, and event ordering before durable acceptance.

The teacher model decides whether natural-language feedback is sufficiently
affirmative and must continue clarifying when it is ambiguous. If feedback
revises the instruction, the teacher discloses the revised objective and waits
for another learner response before tool use. Decline creates neither a tool
call nor a TaskRecord. The persisted message chain makes the disclosed scope,
learner feedback, and final objective auditable without another AI consent
classifier.

### D13-060 — Operation identity and deterministic restart reconciliation

Task identity is deduplicated by the originating explicit operation, never by
natural-language similarity. Replayed provider calls, client requests, upload
revisions, or replayed calls with the same operation ID return the existing
Task or Run. A later explicit operation may create new work even when
its objective text matches an earlier task, subject only to active same-type
exclusion.

On application startup, queued authorized work remains queued and may launch in
a visible terminal when a slot is available. A recorded running executor whose
process and heartbeat are still alive is observed again; a missing executor is
marked failed with `executor-interrupted` and receives the normal inline failure
notice and manual CLI next-step suggestion. LLL never starts another Agent attempt
automatically.

Deterministic Go-owned phases may resume without re-invoking the Agent:
validation rereads the sealed attempt result, while commit recovery completes
or rolls back its recorded filesystem transaction. Terminal task states remain
unchanged. Startup rebuilds global and per-unit concurrency counts from the
reconciled task and process state.

### D13-061 — Extensible deliverables inside a strict documented workspace contract

CLI work is not limited to the three cumulative teaching assets. Research,
code, experiments, datasets, logs intended as evidence, images, diagrams,
documents, and composite directory trees are valid deliverables. The five soft
roles are `引导人`, `澄清者`, `验证者`, `沉淀者`, and `复盘者`; especially the
last three may delegate heavy work whose useful result is broader than an asset
update.

Freedom of deliverable type does not permit arbitrary placement. Every attempt
uses the versioned path contract below:

```text
attempts/<run-id>/
├─ envelope.json
├─ stdout.log
├─ stderr.log
└─ workspace/
   ├─ scratch/                         disposable executor work
   ├─ deliverables/
   │  └─ <deliverable-key>/
   │     ├─ artifact.json              descriptor and entry points
   │     └─ files/                     arbitrary internal file tree
   ├─ asset-updates/
   │  ├─ intro/                        optional candidate
   │  ├─ body/                         optional candidate
   │  └─ practice/                     optional candidate
   ├─ source-updates/                  internal source-processing output only
   └─ result-manifest.json             declared handoff to LLL
```

The executor may organize any safe tree below a declared deliverable's
`files/` and may use `scratch/` freely. It may not invent other attempt-level
destinations, write formal assets directly, use absolute or parent-traversing
paths, or rely on undeclared files being promoted. `result-manifest.json` lists
deliverable keys and optional asset updates; it constrains the handoff, not the
content or complexity of a deliverable.

`source-updates/` is valid only for an LLL-created `source-processing` task and
only for the exact sealed source revision. It is not a teacher-selectable
destination or a general way to write the source library.

LLL validates descriptors, relative paths, declared entry points, file
existence, and policy limits. Scratch and undeclared output are not committed.
Valid generic deliverables receive stable artifact identities and are promoted
under `assets/generated/<artifact-id>/` while preserving their internal tree.
Core `intro`, `body`, and `practice` candidates follow their separate versioned
asset commit rules. The exact descriptor and manifest schemas are published as
the versioned pipeline contract rather than left to prompt convention.

### D13-062 — Generic handoff manifest plus optional versioned asset updates

`result-manifest.json` is a small executor-to-LLL delivery manifest, not a
teaching-content template. It may declare any number of generic deliverable
bundles and zero or more optional updates to `intro`, `body`, and `practice`.
A task can succeed solely by delivering valid research, code, experimental
data, diagrams, or another declared artifact; it need not force those files
into a core teaching asset.

Each core target reports `updated`, `unchanged`, or `failed`. Updated and
unchanged results advance that target's incorporated conversation cursor;
failed or edit-conflict results do not. Cumulative update semantics mean the
executor reads the sealed base and produces a complete next-version candidate,
not an uncontrolled append to the formal current file.

LLL validates the manifest and candidate tree before promotion. If the learner
has edited an asset since the sealed base, LLL performs a deterministic
base/current/candidate merge. Non-overlapping changes may commit; an overlap
preserves the learner's current content, reports `edit-conflict`, and commits no
candidate for that asset. No draft acceptance or rejection UI is introduced.

Each successful asset promotion creates an immutable version with task, run,
base version, conversation range, source revision, timestamp, and change-summary
provenance, then atomically advances its current pointer. Assets commit
independently under a task commit journal, so successful targets remain valid
when another target fails and the product task becomes `partial`. Crash recovery
finishes or rolls back recorded filesystem operations without rerunning the
Agent. Body annotation threads retain version and quote-snapshot provenance when
selection positions can no longer be resolved.

### D13-063 — Inline conversation status is the sole asynchronous result surface

Every accepted assistant task is projected as a compact non-speaking system
status inside the same conversation flow, anchored below the relevant message.
The status moves through queued and running to succeeded, partial, failed, or
cancelled. It may expand just enough to expose deliverables, core asset
outcomes, and deterministic failure or manual CLI next-step guidance. It is not a
separate task-card module, teacher message, side panel, dashboard, or
notification-center product.

SSE carries only small invalidation and status events such as task, source, or
asset identifiers and their latest state. Durable REST reads return complete
TaskRecord, result, deliverable, source, and asset details and recover the UI
after refresh, reconnect, or a detected sequence gap. On the learner's next
turn, teacher context may include a bounded task-result summary so the teacher
can respond naturally; completion itself never injects unsolicited teacher
speech.

### D13-064 — Local versioned source library with explicit parse and disclosure

Source material uses the path contract
`sources/<source-id>/{source.json,revisions/<revision-id>/{original/,derived/}}`.
Original user bytes are retained under `original/`; validated CLI extraction,
structure, and attachments enter `derived/`. A replacement upload creates a new
immutable revision rather than overwriting earlier provenance.

Initial automatic parsing covers PDF, DOCX, PPTX, XLSX, CSV, text, Markdown,
HTML, common images, and common source-text formats. Other regular files may be
stored as opaque material without a false ready state. Upload rejects non-regular
or unsafe traversal inputs, does not follow symbolic links, and neither
auto-extracts archives nor executes uploaded programs.

The learner sees the parse action and privacy disclosure and explicitly
confirms upload and parsing. LLL atomically saves and hashes the original before
creating a visible `source-processing` assistant task. Parsing follows the same
terminal, workspace, manifest, validation, concurrency, and failure-notice
contracts as other assistant work. Failure preserves the original.

Adding a source never places its content in teacher context. The teacher sees
safe metadata by default; content is readable by a task only after learner
feedback approves an objective that references the stable source. If the
executor uses a cloud model, the UI discloses that referenced file content may
leave the device before approval.

Normal deletion tombstones and hides a source while retaining immutable IDs,
revision IDs, hashes, and historical provenance. A separate explicit permanent
deletion removes original and derived bytes; historical records retain only the
non-content identity, hash, and deleted marker. Generated deliverables remain
assets rather than being relabeled as uploaded sources.

### D13-065 — One restrained, OpenAI-style teacher conversation surface

The teacher frontend is one continuous chat surface, following the restrained
interaction model of a simple OpenAI conversation: a centered readable message
column, generous whitespace, minimal navigation, and a persistent composer at
the bottom. Teacher disclosure, learner approval, ordinary teaching, and
follow-up discussion are all normal messages in that single stream.

The interface does not visually split `教师消息`, `用户反馈`, `助教方案说明`, or
`系统任务卡` into product modules. The five teaching roles remain invisible
system-prompt methodology; they are not tabs, role selectors, chips, stages, or
columns. Assistant state appears only as the compact inline system status
defined by D13-063.

The visual language is deliberately quiet: neutral surfaces, a comfortable
reading width, clear typography, subtle separators, and restrained status
color. There is no right-hand workflow rail, multi-pane classroom, task
dashboard, decorative gradient, or grid of cards. `资产` and `资料` remain
focused top-level destinations reached through minimal navigation and do not
crowd the teacher conversation.

### D13-066 — Streaming teacher messages with safe rich-content rendering

Teacher responses stream into the existing conversation message rather than
waiting for a complete response or creating a separate output surface. The UI
must preserve readable autoscroll, allow the learner to scroll away without
being pulled back, expose stop-generation where the provider supports it, and
persist the finalized message through the canonical conversation event model.
A reconnect recovers durable content through REST and may resume only from a
provider cursor that the normalized streaming contract explicitly supports.

The message renderer supports GitHub-flavored Markdown, syntax-highlighted
code fences, LaTeX math, Mermaid diagrams, sanitized inline SVG, and referenced
PNG or JPEG images. These are content capabilities of an ordinary teacher
message, not separate applications. Text and math render incrementally when
their syntax is complete enough; incomplete code fences, math delimiters,
Mermaid blocks, and SVG remain a lightweight source placeholder until closed,
then replace themselves in place. A rendering failure preserves the source and
offers a source view instead of losing the teacher response.

Mermaid and SVG use closed fenced blocks (`mermaid` and `svg`); raw Markdown
HTML remains disabled. PNG/JPEG preview uses an authorized attachment or local
artifact reference rather than an arbitrary remote Markdown image fetch.

Markdown does not enable arbitrary raw HTML. Links, code, Mermaid, and SVG pass
through explicit sanitization and content-security policy. Mermaid runs in a
strict configuration without script execution or network loading. SVG is
treated as active content: scripts, event handlers, `foreignObject`, external
resources, unsafe URLs, and document-level navigation are removed, and the
result renders in an isolated surface. PNG and JPEG previews use stable local
or authorized attachment references, bounded decoding and dimensions, lazy
loading, zoom, and an explicit file-open or download action; remote tracking
URLs are not fetched implicitly.

The learner may expand a compact `思考` disclosure associated with a streaming
teacher message. Its portable contract is a provider-supplied reasoning
summary plus coarse progress states, when available. Raw hidden chain-of-thought
is neither requested nor stored nor exposed. Providers that offer no reasoning
summary show only honest progress state and omit the disclosure after
completion rather than fabricating an explanation.

At the provider boundary, LLL normalizes response events into ordered content
blocks such as text delta, reasoning-summary delta, attachment reference,
tool-use, completion, and error. The frontend never consumes a provider-native
stream directly. Conversation persistence stores the final normalized message,
render-source text, safe attachment references, and stable block identities;
derived HTML, rendered Mermaid SVG, image pixels, and transient progress labels
are rebuildable projections rather than canonical history.

## Runtime Roles

```text
API teacher
  owns teaching dialogue, understanding assessment, immediate feedback,
  teaching adaptation, and one narrow learner-requested assistant-handoff
  operation

CLI assistant
  owns substantial conversation-grounded asynchronous production work such as
  source-material processing, material organization, image generation, and
  creation or updating of project assets from reusable templates; it may use
  code execution and specialist tools internally when the substantial task
  requires them

LLL
  translates supporting-work needs into task contracts and owns dispatch,
  permissions, durable state, validation, versioned asset commits, event
  delivery, cancellation, and failure reporting
```

Role and runtime remain separate concepts. `teacher` and `assistant` describe
product responsibility; `realtime-api`, `background-api`, and `native-cli`
describe execution methods.

## Planning State

Planning is frozen and approved for implementation. The formal product stories,
black-box acceptance criteria, interface contract, data design, and test plan
live beside this README. Implementation has not started merely because planning
is complete; `DELIVERY_NOTES.md` owns actual delivery and verification status.

Document set:

- [User stories](./USER_STORIES.md)
- [Acceptance criteria](./ACCEPTANCE_CRITERIA.md)
- [Interface contract](./INTERFACE_CONTRACT.md)
- [Data design](./DATA_DESIGN.md)
- [Test plan](./TEST_PLAN.md)
- [Delivery notes](./DELIVERY_NOTES.md)
- [ADR-0012](../../00-product-and-architecture/ADR/0012-teacher-assistant-learning-workspace.md)

## Documentation Impact

This iteration changes the primary user journey, domain objects, runtime
workflow, persistence contracts, public interfaces, privacy behavior, and
migration policy. Planning synchronizes:

- `ADR/0012-teacher-assistant-learning-workspace.md`, which explicitly
  supersedes affected five-zone and terminal-only assumptions without
  rewriting accepted history;
- `PRD.md`, `DOMAIN_MODEL.md`, `DATA_MODEL.md`,
  `LEARNING_PROJECT_STRUCTURE.md`, `SYSTEM_ARCHITECTURE.md`,
  `BACKEND_ARCHITECTURE.md`, `AGENT_ARCHITECTURE.md`,
  `API_CONTRACT_STRATEGY.md`, and `MVP_ROADMAP.md`;
- `docs/01-iterations/README.md`, `ADR/README.md`, and the complete iteration
  document set required by the documentation standard.
