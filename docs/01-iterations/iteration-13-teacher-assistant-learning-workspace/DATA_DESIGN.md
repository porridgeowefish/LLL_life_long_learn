# Iteration 13 Data Design

Status: implemented contract
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: file-first schemas, ownership, atomicity, migration, and recovery for iteration 13.

## 1. Data Principles

- The project filesystem is canonical; no database or external queue is added.
- JSON owns metadata, JSONL owns append-only semantic history, Markdown owns
  learner-editable prose, and binary originals retain exact uploaded bytes.
- One Go repository is the only writer for each canonical record family.
- Provider deltas, rendered HTML, decoded images, indexes, and compacted model
  context are projections and may be rebuilt.
- Stable IDs are opaque type-prefixed ULIDs. Paths are derived from validated
  project slugs and IDs, never from mutable titles.
- Formal writes use same-directory temporary files, flush, rename, and a
  recoverable journal when several paths must advance together.

## 2. Canonical Learning-Unit Layout

After migration, every system-learning project uses:

```text
projects/<unit-slug>/
  project.md
  state.json
  learning-scope.json
  unit.json
  conversation/
    conversation.json
    events.jsonl
    compact.json
  assets/
    intro/
      asset.json
      current.md
      versions/<version-id>/content.md
      versions/<version-id>/version.json
    body/
      asset.json
      current.md
      versions/<version-id>/content.md
      versions/<version-id>/version.json
      annotations.jsonl
    practice/
      asset.json
      current.md
      versions/<version-id>/content.md
      versions/<version-id>/version.json
    generated/<artifact-id>/
      artifact.json
      files/**
  sources/<source-id>/
    source.json
    revisions/<revision-id>/
      revision.json
      original/<original-file>
      derived/**
  assistant-tasks/<task-id>/
    task.json
    input-manifest.json
    attempts/<run-id>/
      envelope.json
      stdout.log
      stderr.log
      heartbeat.json
      workspace/
        scratch/**
        deliverables/<deliverable-key>/artifact.json
        deliverables/<deliverable-key>/files/**
        asset-updates/intro/**
        asset-updates/body/**
        asset-updates/practice/**
        source-updates/<revision-id>/**
        result-manifest.json
  runs/
  memory/
  migrations/iteration-13/
    migration.json
    journal.jsonl
    backup/<legacy-relative-tree>/**
```

`runs/` and `memory/` remain compatibility/runtime locations already owned by
the project. New assistant attempts are canonical below `assistant-tasks/`;
optional legacy run-index projections may point to them but do not own state.

Discipline-map projects retain their existing overview, topic catalog,
learning-plan, runs, memory, and asset layout. They do not gain a teacher
conversation merely by being a map.

## 3. Unit Metadata

`unit.json`:

```json
{
  "schemaVersion": 1,
  "unitId": "unit_01...",
  "conversationId": "conv_01...",
  "projectSlug": "linear-algebra",
  "discipline": {"mapSlug": "mathematics", "topicId": "linear-algebra"},
  "activeAssetKeys": ["intro", "body", "practice"],
  "migration": {"from": "legacy-five-zone", "completedAt": "..."},
  "createdAt": "...",
  "updatedAt": "..."
}
```

`discipline` is optional guidance and provenance. It does not impose one-topic
message validation, route blocking, or a second conversation.

## 4. Conversation Metadata And Events

`conversation/conversation.json`:

```json
{
  "schemaVersion": 1,
  "id": "conv_01...",
  "unitId": "unit_01...",
  "latestSeq": 91,
  "latestEventId": "event_01...",
  "eventCount": 91,
  "compactedThroughSeq": 40,
  "createdAt": "...",
  "updatedAt": "..."
}
```

`events.jsonl` contains one complete UTF-8 JSON event per line. Common envelope:

```json
{
  "schemaVersion": 1,
  "seq": 42,
  "eventId": "event_01...",
  "type": "message-recorded",
  "occurredAt": "...",
  "data": {}
}
```

Initial durable event types are:

| Type | Required data |
|---|---|
| `message-recorded` | message ID, role, normalized final blocks, status |
| `teacher-response-started` | response ID, teacher message ID, triggering learner message ID |
| `tool-call-requested` | call ID, response ID, logical name and safe arguments |
| `tool-result-recorded` | call ID, accepted/rejected/failed result |
| `teacher-response-finished` | response ID, message ID, completed/interrupted/failed, usage |
| `task-linked` | task ID, anchor message ID, tool call ID |

Task progress is not copied into this log. It remains in `task.json` and is
joined into the conversation read projection by stable task link.

The repository serializes appends per conversation, assigns `seq`, appends and
flushes the event, then atomically advances conversation metadata. On startup,
it ignores one truncated final JSONL line, verifies monotonic sequence, and
repairs metadata from valid events. Corruption before the final line blocks new
writes and requires recovery; it is never skipped silently.

## 5. Normalized Message Blocks

One stored message contains:

```json
{
  "id": "msg_01...",
  "role": "teacher",
  "status": "completed",
  "blocks": [
    {"id":"blk_01...","type":"reasoning-summary","source":"..."},
    {"id":"blk_01...","type":"markdown","source":"..."},
    {"id":"blk_01...","type":"attachment","artifactRef":"artifact_01..."}
  ],
  "createdAt": "...",
  "completedAt": "..."
}
```

Markdown source is canonical. LaTeX, Mermaid, code, and inline sanitized SVG
remain embedded source syntaxes. Sanitized render caches, if added, are keyed
by source hash and renderer version outside canonical history.

Raw hidden chain of thought has no schema field. `reasoning-summary` may contain
only provider-designated summary output.

## 6. Learning-Aware Compact Projection

`compact.json` is rebuildable:

```json
{
  "schemaVersion": 1,
  "promptVersion": "teacher-1",
  "covered": {"fromSeq": 1, "throughSeq": 40, "eventHash": "sha256:..."},
  "recentExactStartsAtSeq": 41,
  "learningState": {
    "goals": [],
    "confirmedUnderstanding": [],
    "openQuestions": [],
    "misconceptions": [],
    "definitions": [],
    "methods": [],
    "examples": [],
    "verification": [],
    "decisions": [],
    "nextDirections": []
  },
  "evidence": [
    {"itemPath":"/openQuestions/0","eventIds":["event_01..."],"quote":"..."}
  ],
  "createdAt": "..."
}
```

Compaction triggers around an estimated 256K-token teacher context, or earlier
when the configured provider hard limit requires it. The target keeps roughly
the most recent 64K tokens exact. The algorithm must preserve distinctions
such as learner claim versus teacher claim and unresolved versus verified.
Critical formulas, definitions, code, questions, and wording-sensitive text
retain exact evidence excerpts.

If schema, prompt version, event range, or hash does not match, the projection
is stale and rebuilt from events. It never advances an asset cursor or creates
an assistant task.

## 7. Core Asset Metadata And Versions

Each `asset.json`:

```json
{
  "schemaVersion": 1,
  "assetId": "asset_01...",
  "key": "body",
  "title": "正文",
  "currentVersionId": "aver_01...",
  "editRevision": 18,
  "conversationCursor": 87,
  "contentHash": "sha256:...",
  "updatedAt": "..."
}
```

`current.md` is the ordinary learner-editable current file. A learner edit
atomically updates `current.md`, increments `editRevision`, and snapshots a new
immutable version. Direct external file edits are detected by hash before read
or assistant commit and imported as a learner-owned version before proceeding.

`version.json`:

```json
{
  "schemaVersion": 1,
  "versionId": "aver_01...",
  "assetId": "asset_01...",
  "parentVersionId": "aver_01...",
  "author": "learner",
  "taskId": null,
  "runId": null,
  "baseVersionId": "aver_01...",
  "conversationRange": {"fromSeq": 70, "throughSeq": 87},
  "sourceRevisionIds": [],
  "contentHash": "sha256:...",
  "changeSummary": "修正定义并补充例子",
  "createdAt": "..."
}
```

`author` is `learner`, `assistant`, or `migration`. Updated and unchanged
assistant results advance `conversationCursor`; only updated creates a new
content version. Failed and conflicted results do neither.

Assistant merge uses sealed base, current learner content, and full candidate.
Non-overlapping changes create a new assistant version. Overlap keeps current
content unchanged and records `edit-conflict` in the task result.

## 8. Body Annotations

`assets/body/annotations.jsonl` is append-only semantic annotation history. A
read projection groups events into threads.

Annotation identity records:

```json
{
  "annotationId": "ann_01...",
  "assetId": "asset_01...",
  "assetVersionId": "aver_01...",
  "quoteSnapshot": "...",
  "anchors": {"start": 120, "end": 147, "prefix": "...", "suffix": "..."},
  "status": "open",
  "createdAt": "..."
}
```

Thread messages use stable IDs, `learner` or `assistant` roles, source content,
status, and timestamps. Re-anchoring after asset edits uses quote, prefix,
suffix, and version provenance. Failure to re-anchor preserves the quote and
marks the selection detached; it never deletes the thread.

Legacy confusion IDs become stable external references on migrated annotation
records.

## 9. Generated Artifacts

`assets/generated/<artifact-id>/artifact.json`:

```json
{
  "schemaVersion": 1,
  "artifactId": "artifact_01...",
  "kind": "report",
  "title": "实验报告",
  "description": "...",
  "entryPoints": ["files/report.md"],
  "files": [
    {"path":"files/report.md","mediaType":"text/markdown","sha256":"...","bytes":1234}
  ],
  "provenance": {
    "taskId":"task_01...",
    "runId":"run_01...",
    "conversationRange":{"fromSeq":1,"throughSeq":87},
    "sourceRevisionIds":[]
  },
  "createdAt": "..."
}
```

The artifact tree is immutable after promotion in iteration 13. A later task
creates a new artifact identity or an explicitly related revision; it does not
edit an old promoted tree in place.

## 10. Source Records And Revisions

`source.json`:

```json
{
  "schemaVersion": 1,
  "sourceId": "source_01...",
  "displayName": "教材",
  "status": "ready",
  "currentRevisionId": "srev_01...",
  "tombstonedAt": null,
  "permanentlyDeletedAt": null,
  "createdAt": "...",
  "updatedAt": "..."
}
```

Source status is `stored`, `processing`, `ready`, `opaque`, `failed`,
`tombstoned`, or `deleted`.

`revision.json`:

```json
{
  "schemaVersion": 1,
  "revisionId": "srev_01...",
  "sourceId": "source_01...",
  "original": {
    "filename":"book.pdf",
    "mediaType":"application/pdf",
    "bytes":1234,
    "sha256":"sha256:..."
  },
  "derivedFiles": [
    {"key":"text","path":"derived/content.md","mediaType":"text/markdown","sha256":"sha256:..."}
  ],
  "parseTaskId": "task_01...",
  "privacy": {"cloudDisclosureAccepted":true,"acceptedAt":"..."},
  "createdAt": "..."
}
```

Uploads accept one regular file, default maximum 100 MiB. A unit retains a
default maximum 1 GiB of source original and derived bytes. Archives are stored
opaque; no automatic extraction occurs. Executables are never run by upload or
preview behavior.

Permanent deletion overwrites no historical hash or ID. It removes byte files,
clears filenames that may contain private content if configured, removes
derived paths, and records a deletion marker. Active sealed task references
block permanent deletion until terminal.

## 11. Task, Input, And Run Records

`task.json` is mutable product state:

```json
{
  "schemaVersion": 1,
  "id": "task_01...",
  "unitId": "unit_01...",
  "type": "consolidate",
  "objective": "...",
  "sourceRefs": [],
  "status": "queued",
  "phase": null,
  "origin": {
    "kind":"teacher-tool",
    "operationId":"op_01...",
    "proposalMessageId":"msg_01...",
    "approvalMessageId":"msg_01...",
    "toolCallId":"call_01..."
  },
  "conversationCutoffSeq": 87,
  "attemptIds": [],
  "result": null,
  "failure": null,
  "lease": null,
  "createdAt": "...",
  "updatedAt": "..."
}
```

Internal `source-processing` tasks have origin kind `source-upload` and a
source revision ID instead of proposal messages.

`input-manifest.json` is written once before the first run and then immutable.
It pins conversation range and hash, asset base versions and cursors, exact
source revisions and file hashes, authorized logical scopes, and schema
versions. A manifest hash is stored in every run envelope.

Each attempt owns envelope, heartbeat, stdout, stderr, workspace, and manifest.
The CLI never writes `task.json`. Go derives the result after validation.
An internal `source-processing` attempt may declare one update for its exact
sealed source revision under `source-updates/<revision-id>/`; Go validates and
promotes it to that revision's `derived/` tree. Teacher-originated tasks cannot
write source updates. The attempt receives no retry capability or hidden
control token; failed and cancelled records remain terminal.

Lease fields contain run ID, dispatcher instance ID, acquisition time, and last
observed heartbeat. A lease is not proof a process is alive; startup reconciles
it with the visible executor process and heartbeat. Conversely, losing the
observed process is not proof that an interactive terminal can no longer write
its promised result. Failed `executor-state-lost` and
`invalid-result-manifest` attempts remain eligible for late-result
reconciliation when a stable manifest subsequently appears and passes the
same sealed-input and output validation.

## 12. Queue And Idempotency Indexes

There is no canonical global queue file. Startup scans
`projects/*/assistant-tasks/*/task.json`, validates ownership, and rebuilds:

- eligible queued order by creation ULID;
- global running count, default maximum five;
- per-unit running count, default maximum two;
- active `(unitId, taskType)` exclusion;
- origin operation ID to task or run lookup.

To make admission atomic across concurrent HTTP turns, a workspace-scoped lock
guards the check-and-create transaction. The lock file contains diagnostics but
is not task truth. A crash releases the OS lock; scanning restores indexes.

## 13. Commit Journal

Each validating attempt may create an append-only commit journal beside the run
workspace. Journal steps include validation complete, generated artifact
promotion prepared, each core asset version prepared, current-file replacement,
metadata advance, task-result write, and final commit.

Before every step, hashes and expected base revisions are rechecked. Recovery
either completes an already-durable prepared step or restores recorded old
files. It never asks the Agent to regenerate and never changes a terminal task.

Core assets commit independently. If body commits and practice conflicts, body
remains committed and the task is `partial`. If core updates succeeded but a
generated artifact promotion was interrupted, the dispatcher retries only the
missing idempotent artifact promotion and may advance the task from `partial`
to `succeeded`; it never reruns the CLI for this repair.

## 14. Migration

Migration applies only to system-learning projects without `unit.json`.

Before frontend cutover, a startup preflight inventories all legacy
system-learning projects. The new experience is enabled only when every
eligible project is either already canonical or has a completed migration.
Any failed project keeps the legacy experience active for the whole workspace;
the product does not mix active five-zone and new teacher interfaces silently.

Before writes:

1. acquire the project migration lock;
2. inventory relevant legacy files and hashes;
3. copy the relevant legacy tree into
   `migrations/iteration-13/backup/` without following external symlinks;
4. write and flush `migration.json` with status `prepared`;
5. create canonical unit, conversation, asset, annotation, source, and task
   directories in staging;
6. convert active content and validate all new records;
7. atomically promote the canonical paths and mark `completed`.

Mapping:

| Legacy | Canonical target |
|---|---|
| Intro learner-facing content | `assets/intro/current.md` migration version |
| Explain pages/output/media | `assets/body/current.md` plus generated artifacts where needed |
| Practice learner-facing content | `assets/practice/current.md` migration version |
| Explain confusions and Ask-AI | body annotation history |
| Summary | backup only; inactive |
| Extend | backup only; inactive |
| Greenhouse presentation data | remains legacy; hidden |

Migration creates no artificial teacher messages from old zone prose. The new
conversation begins empty unless real prior learner-teacher turns are available
in a durable legacy source.

Failure before promotion removes staging and leaves the legacy reader active.
Failure during promotion replays the journal to complete or restore. Rollback
switches the frontend compatibility reader and never deletes new unit data,
new conversations, or new asset versions created after migration.

`migration.json` status is `prepared`, `converting`, `completed`, `failed`, or
`rolled-back`, with inventory hash, backup path, journal tail, failure code,
and timestamps. Repeating preflight is idempotent.

## 15. Retention And Privacy

- Canonical conversations, committed assets, source revisions, task records,
  run diagnostics, and migration backups have no automatic deletion in
  iteration 13.
- Scratch content may be removed after terminal validation only when no
  declared deliverable or diagnostic reference depends on it; the first
  implementation may retain it entirely.
- Credentials remain in existing configuration stores, never project files.
- Logs redact provider authorization headers and must not duplicate uploaded
  bytes or full private prompts unnecessarily.
- Remote images are not fetched by rendering. Authorized source and artifact
  bytes are served through local validated references.

## 16. Schema Evolution

Every JSON/JSONL record has `schemaVersion`. Readers reject unsupported future
major versions without rewriting them. Additive optional fields preserve the
same version when old readers can safely ignore them. Breaking semantics require
a new version, a migration function, fixtures for the previous version, and an
iteration contract update.
