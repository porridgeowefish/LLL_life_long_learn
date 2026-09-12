# Data Model

Status: active
Owner: project maintainer
Last reviewed: 2026-09-12
Source of truth: long-lived conceptual file-first model; Go structs and persisted schemas own current runtime truth.

## Project

| Field | Type | Values / shape | Rule |
|---|---|---|---|
| `id` | string | filesystem-safe project id | required |
| `slug` | string | filesystem-safe slug | required and unique in its scope |
| `title` | string | learner-facing title | required |
| `projectType` | enum string | `discipline-map` or `system-learning` | missing legacy value decodes as `system-learning` |
| `status` | string | project lifecycle state | required |
| `activeZone` | string or null | legacy Intro/Explain/Practice name | accepted on old state files; not active teacher-workspace navigation |
| `createdAt` | RFC 3339 timestamp | timestamp | required |
| `updatedAt` | RFC 3339 timestamp | timestamp | required |

Project type is immutable.

`project.md` is type-specific learner context:

| Project type | Allowed context |
|---|---|
| `discipline-map` | project shape, overview goal, optional scope notes, learner notes |
| `system-learning` | motivation, current ability, target ability, completion standard, active phase, learner notes |

A discipline-map brief never stores a focused-learning ability ladder or
learning phase. The backend enforces this boundary even if a client submits
system-learning-only fields.

## Discipline Overview, Topic Catalog, And Learning Plan

| Field / artifact | Type | Rule |
|---|---|---|
| overview content | Markdown file | exactly one learner-facing overview per discipline map |
| topic catalog | `discipline-topics.json` | one validated boundary per actionable H4 topic; core concepts have one owner |
| learning plan | `learning-plan.json` | learner-selected ordered tasks and per-task status |
| display labels | string | `学科总览` and `学习计划`; physical filenames are internal |
| generation run | normal Session + `runs/<timestamp>-encyclopedia/` | selected native Agent CLI; project-level output paths are `overview.md` and `discipline-topics.json` |
| table of contents | derived from Markdown H2/H3/H4 headings | navigation only; not persisted as a second structure |
| inline topic action | derived from every H4 learnable topic | must not create a project until confirmation; H3 remains the legacy fallback when no H4 exists |
| folder overview binding | `folders.json.mapProjectSlug` | optional discipline-map slug rendered as the folder's explicit overview row; not `slugOrder` membership or project ownership |
| learning-project membership | `folders.json.slugOrder[]` | system-learning classification only; never contains a bound map slug |

An overview topic has no durable project identity, but its catalog entry has a
stable topic ID and an objective boundary. A project record begins only
after learner-confirmed creation.
The learning plan references exact overview topic titles. Array position is the
learner-selected order; each item stores `planned`, `in-progress`, or `completed`
plus timestamps. It stores no system-learning project ownership or sidebar membership.
Each distinct completion timestamp may append one idempotent
`learning-task-complete` activity event with zero growth value.

The backend reconciles indexed discipline maps into folder objects on folder
reads and writes. A same-name folder is promoted in place; otherwise one folder
is created. Clearing a stale map binding preserves the folder and its membership.

## System-Learning Data

Every system-learning root contains `learning-scope.json` with status
`draft|ready`, topic goal, inclusion, exclusion, prerequisites, owned concepts,
reused concepts, provenance, and update time. A map-origin scope is a copied
snapshot; `source.mapSlug/topicId` records provenance rather than live ownership.

The active system-learning model contains:

```text
one durable conversation and recoverable teacher runs
versioned intro, body, and practice assets
generic generated assistant deliverables
body annotations and Ask-AI history
versioned source originals and derived content
append-only provider-reported teacher usage by response
assistant tasks, sealed inputs, attempts, and run records
```

Legacy zone protocols (`intro/assessment.json`, Explain manifests/pages,
Practice tasks/answers/evaluations, and progress events) remain readable for
migration and compatibility. Historical Extend/Summary files are intentionally
excluded from runtime readers and migration inventory. None is the data contract
for new assistant outputs.

Exact fields remain owned by their code schemas and active iteration contracts.

## Global Learner Preferences

`<WORKSPACE>/preferences.md` is the only active preference-memory file. It is
Markdown, capped at 256 KiB, and edited only through the explicit learner UI or
direct file editing. Teacher prompts receive its content as bounded read-only
context. Assistant attempts receive an immutable snapshot at
`workspace/inputs/preferences.md`. No AI path writes the canonical file.
An existing canonical file is mirrored at startup and after explicit saves to
`<USER_CONFIG>/LifeLongLearn/backups/<workspace-id>/preferences.md`. The mirror
is recovery-only: reads fall back to it when the workspace file is missing, and
opening the editor restores the canonical workspace file. If the canonical file
is unexpectedly empty while the mirror is non-empty, startup or editor access
restores the mirror; an explicit empty save writes both copies. Runtime workspace
configuration is applied once to `WORKSPACE`, `PROJECTS_ROOT`, and `AGENTS_ROOT`
after CLI, environment, and config-file precedence is resolved.

Legacy `projects/<slug>/memory/` files are preserved but ignored. New project
skeletons do not create them, and the generic project file API rejects reads or
writes to `memory/`.

## Application Configuration And Quality Evidence

Iteration 14 centralizes local application configuration without changing
learner project data. `config.local.json` remains gitignored and read-compatible
with current flat keys. The implemented configuration groups server, workspace, AI,
assistant, image, and UI values, with precedence:

```text
compiled defaults < local JSON < LLL_* environment < explicit CLI flags
```

Loading normalizes values in memory and never rewrites the learner's file.
Secrets are supplied through environment variables by default and are redacted
from diagnostics.

Generated test, coverage, and dependency evidence lives below the ignored
`.artifacts/quality/` root. It is build evidence, not product state, and must
not contain secrets, full prompts, learner preferences, uploaded source
contents, or unrestricted absolute paths. Exact iteration-14 shapes and
compatibility fields are owned by its `DATA_DESIGN.md`.

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

## Teacher-Assistant Learning-Unit Model

ADR-0012 retains the flat project root and adds canonical file-first records
for one learning unit:

```text
unit metadata
append-only conversation events plus rebuildable compact context
versioned intro, body, and practice assets plus generated artifacts
body-owned annotation and Ask-AI history
versioned source originals and derived content
durable assistant tasks, sealed inputs, and per-attempt workspaces
migration backup and recovery journals
```

Product Task and executor Run are separate identities. Conversation, task,
asset, source, and version IDs are opaque stable ULIDs. In-memory queue and SSE
state are derived. Provider deltas, rendered rich content, and compact context
are projections. Exact product paths and schemas remain owned by the current
iteration `DATA_DESIGN.md`; code is current truth. A ready source revision has
exactly one canonical `content.md` — for images with a configured
`askAiProviders.bindings.ocr`, that file is produced by the vision-model OCR
step (ADR-0019); `conversation/usage.jsonl` records only nonzero,
provider-reported teacher token usage. Assistant usage is not yet modelled.

### Conversation queue, steering, and regeneration (ADR-0018)

The event log remains append-only. New event families, all replayed into the
live `queue` projection:

```text
learner-queued {queueId, content, attachmentRefs, operationId, providerId}
queue-item-edited {queueId, content, attachmentRefs}
queue-item-discarded {queueId}
queue-item-promoted {queueId, learnerMessageId, mode: auto|steer}
steering-note {learnerMessageId, responseId}
response-superseded {responseId, newResponseId}
```

Superseded teacher messages are hidden from every projection and from provider
context but never deleted. `projects/folders.json` is the folder ledger's
canonical location; a legacy workspace-root `folders.json` is copied forward
once on first open (ADR-0019). The optional `webSearch` config section
(`provider`, `apiKey|apiKeyEnv`, `engine`) enables the teacher `search_web`
tool; absent means disabled.

## Delivery State

The Go state model persists `projectType` for new projects. Missing type remains
the legacy compatibility signal and decodes as `system-learning`. Iteration 13
is implemented; its file schemas and migration compatibility are the active
system-learning model. Iteration 14 is an approved source-architecture target
and does not become product-data truth until delivered.
