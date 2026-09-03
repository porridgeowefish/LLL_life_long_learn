# Iteration 13 Interface Contract

Status: implemented contract
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: iteration-13 HTTP, SSE, provider, teacher-tool, task, and CLI handoff contracts.

## 1. Contract Boundary

This document specifies new iteration-13 interfaces. Current route behavior
remains executable truth until implementation. New code must implement these
shapes without leaking provider-native events, absolute workspace paths, or CLI
flags into the browser or teacher model.

All JSON is UTF-8. Timestamps are RFC 3339 UTC. Stable product IDs are opaque,
type-prefixed ULIDs. Clients treat IDs as strings and never derive paths or time
from them.

New error responses use:

```json
{
  "error": {
    "code": "stable_machine_code",
    "message": "safe learner-facing message",
    "retryable": false,
    "details": {}
  }
}
```

`operationId` is a client- or provider-originated idempotency key for one
explicit action. Replaying the same operation with the same owning unit returns
the existing result. Reusing it with different content returns
`409 operation_conflict`.

## 2. Teacher Conversation HTTP

### 2.1 Read durable conversation

Initial and backward reading uses:

`GET /api/projects/{unitSlug}/conversation?beforeSeq={n}&limit={n}`

`beforeSeq=0` means the current tail. The interactive client requests at most
40 messages initially and supplies the returned `pageFromSeq` to load the next
older page. The server returns messages in chronological order. Forward event
reading remains available for compatibility:

`GET /api/projects/{unitSlug}/conversation?afterSeq={n}&limit={n}`

Backward message paging defaults to 40 and accepts at most 100. Forward event
paging defaults to 200 and accepts at most 500. The response is an ordered
durable projection, not raw provider frames:

```json
{
  "conversationId": "conv_01...",
  "unitId": "unit_01...",
  "latestSeq": 91,
  "pageFromSeq": 52,
  "pageThroughSeq": 91,
  "hasPrevious": true,
  "totalMessages": 126,
  "hasMore": true,
  "messages": [
    {
      "id": "msg_01...",
      "role": "learner",
      "status": "completed",
      "blocks": [{"id": "blk_01...", "type": "markdown", "source": "..."}],
      "createdAt": "2026-08-30T08:00:00Z"
    }
  ],
  "taskLinks": [
    {"messageId": "msg_01...", "taskId": "task_01..."}
  ]
}
```

Message roles are `learner`, `teacher`, or `system`. Persisted teacher message
status is `completed`, `interrupted`, or `failed`. A system message is reserved
for durable non-speaking notices and never impersonates the teacher.

Canonical block types are:

- `markdown`: Markdown source, including closed code and math syntax;
- `reasoning-summary`: provider-supplied summary source, never raw hidden
  reasoning;
- `attachment`: stable attachment or generated-artifact reference;
- `tool-call`: normalized call identity and safe logical arguments;
- `tool-result`: accepted, rejected, or failed application result.

Rendered HTML, Mermaid-generated SVG, decoded pixels, and syntax-highlighted
HTML are never returned as canonical history.

Opening a conversation never requires downloading its complete projection.
Older pages are prepended only after learner action and the viewport anchor is
preserved. Task links are selected by the messages in the page, so an inline
task card cannot be separated from its teacher message by event pagination.

Renderer syntax is explicit: fenced `mermaid` blocks produce diagrams, fenced
`svg` blocks produce sanitized isolated SVG previews, and `$...$` / `$$...$$`
produce LaTeX math through owned preprocessing before Markdown tokenization.
PNG/JPEG previews resolve only from an `attachment` block or an LLL-authorized
local artifact URL. Ordinary remote Markdown image URLs remain links and are
not fetched automatically.

### 2.2 Send a learner turn and stream the teacher response

`POST /api/projects/{unitSlug}/conversation/turns`

Request:

```json
{
  "operationId": "op_01...",
  "content": "请解释为什么……",
  "attachmentRefs": ["source_01..."],
  "replyToMessageId": null
}
```

`content` is required, trimmed, and limited to 64 KiB UTF-8. Attachment
references grant no source-content permission by themselves; source use still
follows the disclosed task contract. The learner message is durably appended
before the provider request starts.

Success returns `200 text/event-stream`. Each event has an SSE `id` that is a
response-local monotonically increasing frame number. Frames are transient and
may be replayed only when the adapter supports a provider cursor. Durable REST
recovery is always authoritative.

Frontend stream union:

| Type | Required data | Meaning |
|---|---|---|
| `turn-accepted` | `learnerMessageId`, `responseId` | learner event is durable |
| `message-started` | `teacherMessageId` | teacher response boundary is durable |
| `text-delta` | `blockId`, `delta` | append Markdown source |
| `reasoning-summary-delta` | `blockId`, `delta` | append disclosed provider summary |
| `attachment-reference` | `blockId`, `artifactRef` | add safe referenced media or file |
| `task-accepted` | `toolCallId`, `taskId`, `status` | durable task exists |
| `tool-rejected` | `toolCallId`, `code`, `existingTaskId?` | expected business refusal |
| `usage` | provider-neutral token counts | optional, diagnostic |
| `message-completed` | `messageId`, `latestSeq` | final normalized message is durable |
| `message-failed` | `messageId`, `code`, `partialPreserved` | provider or service failure |

The server emits no raw provider frame. The frontend concatenates deltas by
stable `blockId`; it never reparses a delta as a complete message. A repeated
frame ID is ignored.

Global workspace SSE uses domain-specific invalidation instead of reloading
the entire learning unit:

| Event | Invalidated projection |
|---|---|
| `assistant-task-updated` | assistant task list and inline task state |
| `generated-artifact-updated` | generated-material list |
| `learning-asset-updated` | three core asset projections |
| `source-updated` | source list and selected source detail |
| `annotation-updated` | body annotations |

Task completion remains durable in `assistant-tasks/<task-id>/task.json`. The
teacher page joins that state through `taskLinks`, announces active-to-terminal
transitions, and retains the terminal card after refresh. A transient browser
notification is never the source of truth.

### 2.2.1 Reconnect an active teacher response

`GET /api/projects/{unitSlug}/conversation/responses/active`

This endpoint returns `204 No Content` when the unit has no active response.
Otherwise it returns the same `text/event-stream` frame union as the turn
endpoint, replaying all frames already produced and continuing until the
response completes. The provider request is owned by the backend run, not by
the browser connection: closing or refreshing a page only closes that
subscriber. Only the explicit stop endpoint cancels the provider response.
After completion, the canonical `conversation/events.jsonl` projection is the
source of truth and the frontend invalidates its conversation query.

### 2.3 Stop generation

`POST /api/projects/{unitSlug}/conversation/responses/{responseId}/stop`

Body:

```json
{"operationId": "op_01..."}
```

This stops only the active teacher-provider response. It is not an assistant
task cancellation API. Success returns the final durable message with status
`interrupted`. If the provider cannot stop, the server returns
`409 provider_stop_unsupported` and continues persisting the response.

## 3. Provider Gateway

Every teacher-capable adapter implements the same ordered internal event union:

```text
response-started
reasoning-summary-delta
text-delta
attachment-reference
tool-call-ready
usage
response-completed
response-failed
```

`tool-call-ready` is emitted only after the adapter has buffered and parsed a
complete JSON argument object. It contains `toolName`, normalized arguments,
an LLL call ID, and an optional provider call ID. Provider IDs are diagnostic
external references, never product identity.

Application tool results are translated by the adapter into the provider's
continuation format. Provider SDK request and event types do not cross the
gateway. Bounded redacted diagnostic traces may be retained outside canonical
conversation history.

### 3.1 Provider service bindings

Iteration 13 introduces a service-oriented settings surface while preserving
the current Ask-AI settings adapter:

```text
GET /api/settings/ai-services
PUT /api/settings/ai-services
POST /api/settings/ai-services/probe
```

The resource stores masked provider definitions and explicit bindings for
`teacher`, `annotationAskAI`, and `conversationCompaction`. A binding selects a
provider ID and model plus supported reasoning-summary and tool-use features.
The compaction binding may reuse the teacher provider. API keys are write-only;
masked values round-trip without replacing stored secrets. Existing
`/api/settings/ask-ai` routes adapt to the annotation binding during migration.

```json
{
  "providers": [
    {
      "id": "primary",
      "kind": "openai-compatible",
      "baseUrl": "https://provider.example/v1",
      "apiKey": "••••"
    }
  ],
  "bindings": {
    "teacher": {"providerId":"primary","model":"model-name"},
    "annotationAskAI": {"providerId":"primary","model":"fast-model"},
    "conversationCompaction": {"providerId":"primary","model":"model-name"}
  }
}
```

Probe accepts one binding name plus either a saved provider ID or an inline
unsaved provider definition and returns only capability flags, latency, and a
safe error. It does not persist settings.

## 4. Teacher Tool

The only iteration-13 teacher tool is:

```json
{
  "name": "delegate_learning_work",
  "arguments": {
    "taskType": "consolidate",
    "objective": "根据目前的讨论更新引入与正文，并保留练习不变。",
    "sourceRefs": ["source_01..."]
  }
}
```

`taskType` is `consolidate`, `verify`, or `produce-material`. It controls
same-type active exclusion and presentation only. `objective` is required,
plain language, at most 8 KiB, and must match the disclosed proposal in
meaning. `sourceRefs` contains stable logical source IDs, not paths.

LLL injects and validates:

- owning unit and conversation;
- adjacent prior teacher proposal as `proposalMessageId`;
- current approving learner message as `approvalMessageId`;
- normalized tool call and origin operation IDs;
- conversation cutoff and permitted logical source scope;
- asset cursors, executor policy, staging paths, and result contract.

The proposal must be an earlier teacher message in the same conversation. The
approval must be a later learner message. A task is not created when either is
missing, belongs to another unit, or the proposal was superseded.

Tool result union:

```json
{"status":"accepted","taskId":"task_01...","taskStatus":"queued"}
```

```json
{
  "status":"rejected",
  "code":"same_type_active",
  "existingTaskId":"task_01...",
  "existingStatus":"running"
}
```

```json
{"status":"failed","code":"task_persistence_failed","retryable":true}
```

One teacher response may receive at most one accepted result. The tool has no
path, command, executor, model, timeout, asset target, notification target, or
cancellation argument.

## 5. Assistant Task HTTP And Events

### 5.1 Read tasks

```text
GET /api/projects/{unitSlug}/assistant-tasks?status={status}&cursor={cursor}
GET /api/projects/{unitSlug}/assistant-tasks/{taskId}
```

Task detail returns:

```json
{
  "id": "task_01...",
  "unitId": "unit_01...",
  "type": "verify",
  "objective": "核查……",
  "status": "running",
  "phase": "executing",
  "origin": {
    "kind": "teacher-tool",
    "operationId": "op_01...",
    "proposalMessageId": "msg_01...",
    "approvalMessageId": "msg_01...",
    "toolCallId": "call_01..."
  },
  "attemptIds": ["run_01..."],
  "result": null,
  "failure": null,
  "createdAt": "2026-08-30T08:00:00Z",
  "updatedAt": "2026-08-30T08:00:10Z"
}
```

Product statuses are `queued`, `running`, `succeeded`, `partial`, `failed`, and
`cancelled`. Running phases are `preparing`, `executing`, `validating`, and
`committing`. Phase is null outside running.

Allowed transitions:

```text
queued -> running
running -> succeeded | partial | failed | cancelled
```

There is no public create, cancel, or retry HTTP endpoint. Teacher tools and
explicit source operations create tasks through application services. Failed
and cancelled tasks are terminal. Their inline notice may suggest a
human-readable instruction the learner can manually publish in a visible CLI,
but that suggestion is not a command contract, API, state transition, or
automatic asset-import path. Work that must re-enter LLL is disclosed and
approved as a new explicit task.

### 5.2 Global SSE invalidation

The existing single `GET /api/events` connection adds:

| Event | Data |
|---|---|
| `conversation.changed` | `unitId`, `conversationId`, `latestSeq` |
| `assistant-task.changed` | `unitId`, `taskId`, `status`, `phase`, `updatedAt` |
| `asset.changed` | `unitId`, `assetId`, `versionId` |
| `source.changed` | `unitId`, `sourceId`, `revisionId`, `status` |

Events contain no message body, source bytes, CLI stdout, provider frames, or
full task result. They invalidate TanStack Query keys. Missing or out-of-order
SSE is repaired by REST.

## 6. Asset HTTP

```text
GET /api/projects/{unitSlug}/assets
GET /api/projects/{unitSlug}/assets/{assetKey}
PUT /api/projects/{unitSlug}/assets/{assetKey}
GET /api/projects/{unitSlug}/assets/{assetKey}/versions
GET /api/projects/{unitSlug}/assets/{assetKey}/versions/{versionId}
GET /api/projects/{unitSlug}/generated-assets/{artifactId}
```

Core `assetKey` is `intro`, `body`, or `practice`. Read returns current Markdown,
metadata, current version, edit revision, provenance summary, and a stable
`contentKind` presentation discriminator:

```text
markdown       render the canonical asset Markdown
explain-pages  render the preserved Explain manifest/pages contract without exposing file paths
practice-set   render the preserved tasks/answer-key contract; answer keys remain server-private
```

`explain-pages` and `practice-set` are compatibility projections only for the
active migration-authored version. A later learner or assistant asset version
becomes `markdown`, so stale legacy files can never shadow new canonical
content. The frontend must not render `manifest.json`, `tasks.json`,
`answer-key.json`, local paths, or delivery manifests as learner-facing
content. Update body:

```json
{
  "operationId": "op_01...",
  "baseEditRevision": 17,
  "content": "# 当前正文\n..."
}
```

The update is an atomic learner edit. A stale base returns
`409 asset_edit_conflict` with the current revision; the server never silently
overwrites a newer edit. Generic generated assets are read-only through this
API in iteration 13; their declared files may be opened or downloaded through
authorized file references.

## 7. Annotation Ask AI

```text
GET /api/projects/{unitSlug}/assets/body/annotations
POST /api/projects/{unitSlug}/assets/body/annotations
PATCH /api/projects/{unitSlug}/assets/body/annotations/{annotationId}
DELETE /api/projects/{unitSlug}/assets/body/annotations/{annotationId}
POST /api/projects/{unitSlug}/assets/body/annotations/{annotationId}/ask-stream
```

Create request records asset version, quote snapshot, optional selection
anchors, and learner note. Ask-stream keeps the current lightweight provider
stream semantics but is owned by `AnnotationAskAIService`. It accepts one
annotation-thread message and cannot emit tools.

Legacy `/confusions` routes remain compatibility adapters during migration and
must resolve to the same canonical body annotation IDs. They are not used by
the new frontend and may be removed only in a later contracted iteration.

## 8. Source HTTP

### 8.1 List and read

```text
GET /api/projects/{unitSlug}/sources
GET /api/projects/{unitSlug}/sources/{sourceId}
GET /api/projects/{unitSlug}/sources/{sourceId}/revisions/{revisionId}/files/{fileKey}
```

File reads require a declared revision file key; clients never send a local
path. Inline preview is allowed only for validated safe media or extracted
text. Other files download with attachment disposition.

### 8.2 Upload and parse

`POST /api/projects/{unitSlug}/sources` uses multipart form data:

| Field | Rule |
|---|---|
| `operationId` | required idempotency key |
| `file` | one regular file, maximum 100 MiB by default |
| `displayName` | optional learner-facing name |
| `parseApproved` | must be `true` to create parsing work |
| `cloudDisclosureAccepted` | required only when selected parser may send bytes to a cloud model |

The server saves, hashes, and identifies the original before returning. If
parsing is approved, the response includes the durable `source-processing`
task. The unit retains at most 1 GiB of source original and derived bytes by
default.

```json
{
  "sourceId": "source_01...",
  "revisionId": "srev_01...",
  "status": "processing",
  "parseTaskId": "task_01..."
}
```

Replacement is explicit:

`POST /api/projects/{unitSlug}/sources/{sourceId}/revisions`

It uses the same multipart contract and creates an immutable revision.

### 8.3 Deletion

`DELETE /api/projects/{unitSlug}/sources/{sourceId}` tombstones and hides the
source while retaining content and provenance.

`POST /api/projects/{unitSlug}/sources/{sourceId}/permanent-delete` requires:

```json
{"operationId":"op_01...","confirmation":"PERMANENT_DELETE"}
```

It removes original and derived bytes after checking that no active task has
sealed that revision. Historical provenance retains only IDs, hashes,
timestamps, and deleted markers. An active reference returns
`409 source_revision_in_use`.

## 9. CLI Input Contract

An accepted task is not the executor input. On first execution, LLL seals
`input-manifest.json`, then writes one attempt `envelope.json`:

```json
{
  "schemaVersion": 1,
  "taskId": "task_01...",
  "runId": "run_01...",
  "taskType": "verify",
  "objective": "核查……",
  "executor": {"kind": "native-cli", "profile": "default"},
  "inputManifest": "../../input-manifest.json",
  "workspace": "workspace",
  "resultManifest": "workspace/result-manifest.json",
  "policy": {
    "allowNetwork": true,
    "formalAssetWrites": false,
    "absoluteOutputPaths": false
  },
  "control": {
    "loopbackBaseUrl": "http://127.0.0.1:8787",
    "attemptTokenEnv": "LLL_ATTEMPT_TOKEN"
  }
}
```

All physical paths in the envelope are attempt-relative. The prompt may
explain absolute paths to the local CLI wrapper internally, but those paths do
not enter product or provider contracts.
The token is injected only into the launched process environment, is redacted
from logs, is not persisted in the project, and is invalidated when used or
when the task reaches a new terminal result.

The sealed input manifest declares:

- exact conversation ID, event cutoff, range, and hash;
- permitted compact projection and exact recent-message references;
- core asset base versions, edit revisions, and conversation cursors;
- source revision IDs, hashes, declared file keys, and logical roles;
- result schema version and policy limits.

Files created after sealing are not external inputs. All manual retries reuse
the original sealed manifest.

## 10. CLI Result Contract

Each attempt writes exactly one `workspace/result-manifest.json`:

```json
{
  "schemaVersion": 1,
  "taskId": "task_01...",
  "runId": "run_01...",
  "summary": "完成两组实验并形成对照图。",
  "deliverables": [
    {
      "key": "experiment-report",
      "descriptor": "deliverables/experiment-report/artifact.json"
    }
  ],
  "assetUpdates": {
    "intro": {"status": "unchanged", "candidate": null},
    "body": {"status": "updated", "candidate": "asset-updates/body/current.md"},
    "practice": {"status": "failed", "candidate": null, "code": "insufficient-evidence"}
  },
  "sourceUpdate": null,
  "warnings": []
}
```

After the file has remained stable for the settlement window, its presence is
the executor-to-LLL completion signal. LLL must validate and collect it even if
the interactive CLI stays open at a new prompt. An `exit.json` record still
reports terminal exit or manual interruption, but successful collection never
waits for that record.

Each deliverable descriptor is:

```json
{
  "schemaVersion": 1,
  "kind": "report",
  "title": "实验对照报告",
  "description": "...",
  "entryPoints": ["files/report.md", "files/chart.svg"],
  "mediaTypes": {"files/report.md": "text/markdown", "files/chart.svg": "image/svg+xml"},
  "provenance": {"sourceRefs": ["source_01..."]}
}
```

Only an internal `source-processing` task may instead report:

```json
{
  "sourceUpdate": {
    "sourceId": "source_01...",
    "revisionId": "srev_01...",
    "status": "ready",
    "derivedDescriptor": "source-updates/srev_01.../derived.json"
  }
}
```

`status` is `ready`, `opaque`, or `failed`. The descriptor declares normalized
relative derived files, media types, hashes, extraction warnings, and an
optional text entry point. LLL accepts it only when source and revision match
the sealed input; the files promote below that revision's `derived/` path,
never to `assets/generated`. Teacher-originated tasks set `sourceUpdate` to
null.

Rules:

- all referenced paths are normalized relative paths below the declared
  deliverable or asset-update root;
- `..`, drive prefixes, UNC paths, symlinks escaping the workspace, device
  files, sockets, and undeclared files are invalid;
- `updated` requires one complete next-version candidate;
- `unchanged` has no candidate and advances the target conversation cursor;
- `failed` has no candidate, requires a safe code, and does not advance cursor;
- valid deliverables may exist with zero core asset updates;
- source-processing output may update only its sealed source revision;
- executors never edit task state, asset current files, source records, or
  conversation events.

LLL validates, sanitizes active media, computes hashes, promotes each generic
deliverable, merges each asset independently, records a commit journal, and
then derives `succeeded`, `partial`, or `failed`.

## 11. Stable Error Codes

| Code | Status | Meaning |
|---|---:|---|
| `project_not_found` | 404 | unit does not exist |
| `conversation_unavailable` | 409 | migration or recovery required |
| `operation_conflict` | 409 | idempotency key reused with different input |
| `teacher_provider_unavailable` | 503 | configured provider cannot start |
| `provider_stop_unsupported` | 409 | provider cannot stop active response |
| `delegation_not_approved` | 409 | proposal/approval chain invalid |
| `same_type_active` | 409 | queued or running same-type task exists |
| `one_task_per_response` | 409 | response already accepted one handoff |
| `asset_edit_conflict` | 409 | learner edit base is stale |
| `source_too_large` | 413 | file or unit quota exceeded |
| `source_revision_in_use` | 409 | active sealed task references revision |
| `unsafe_deliverable_path` | 422 | declared output violates workspace policy |
| `invalid_result_manifest` | 422 | result is missing or schema-invalid |

Unexpected internal errors expose a correlation ID and safe message, not
credentials, prompts containing private source bytes, absolute paths, provider
payloads, or raw CLI command lines.

## 12. Compatibility

- Existing project and discipline-map APIs remain available during iteration
  13 unless this contract explicitly adds a replacement.
- Old system-learning projects must migrate before new conversation writes.
- Legacy reads may adapt zones and confusions into new assets and annotations;
  new writes use only the canonical iteration-13 paths.
- Existing global SSE remains a single connection mounted at the app shell.
- A future contract may remove compatibility routes only after migration data
  and rollback support are no longer required.

## 13. Global Learner Preferences

```http
GET /api/preferences
PUT /api/preferences
Content-Type: text/markdown; charset=utf-8
```

GET returns `{path:"preferences.md", content, maxBytes}` and creates the
documented default file when absent. PUT atomically replaces at most 256 KiB of
learner-authored Markdown. No project ID is accepted because the file is
workspace-global. Teacher and assistant services have read-only store access;
no AI tool, task result, generated asset, or prompt may call the write path.

The retired `/files/projects/{slug}/memory/*` surface returns forbidden. The
frontend `/memory` route redirects to `/preferences` for bookmarks only.
