# Data Model

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
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
versioned review and practice assets
independent teacher note entries and jobs
review annotations and Ask-AI history
versioned source originals and derived content
append-only provider-reported teacher usage by response
append-only idempotent learning activity events for heatmaps
assistant tasks, captured inputs, attempts, and run records
```

Legacy zone protocols (`intro/assessment.json`, Explain manifests/pages,
Practice tasks/answers/evaluations) are one-time migration inputs only.
Progress events remain active; retired teaching formats have no runtime fallback. Historical Extend/Summary files are intentionally
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
restores the mirror; an explicit empty save writes both copies. The workspace
root defaults to `<repo>/workspace/` (ADR-0023); `--workspace`,
`LLL_WORKSPACE_ROOT` / `WORKSPACE`, and `workspace.root` relocate it. Runtime
workspace configuration is applied once to `WORKSPACE`, `PROJECTS_ROOT`, and
`AGENTS_ROOT` after CLI, environment, and config-file precedence is resolved.

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
versioned review/practice assets and durable note batches
review-owned annotation and Ask-AI history
versioned source originals and derived content
durable assistant tasks, captured inputs, and per-attempt workspaces
migration backup and recovery journals
```

### Assistant-accepted publication

`result-manifest.json` declares only review/practice Markdown candidates for
consolidation. Source-derived content.md is owned by the local sources parser, not an assistant output.
There is no deliverables/artifact.json publication or arbitrary generated reader.
Go retains version/merge and filesystem authorization; content acceptance remains
with the assistant. Retired caches and generated directories are removed during
cutover before dispatch. notes/state.json records the epoch, seen learner IDs,
pending turns and counts; durable jobs and entries are under assets/notes.
conversation/note-usage.jsonl records each API attempt separately from teacher usage.

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

### Learning activity

`progress/events.jsonl` is the append-only source for the home and usage
heatmaps. A `ProgressEvent` has a stable `id`, `sourceType`, `sourceId`,
optional outcome/growth fields, `activityDelta`, presentation title/detail, and
creation time. The stable ID prevents duplicate activity when a request or task
recovery is replayed.

Successful teacher responses write `teacher-response:<responseId>`; successful
assistant tasks that publish at least one review/practice asset
write `assistant-publication:<taskId>`. Both have `activityDelta: 1`. Failed,
interrupted, and no-output task completions have no event. Existing reading,
practice, Ask-AI, Agent session, and learning-plan events remain compatible
members of the same stream.

## Delivery State

The Go state model persists `projectType` for new projects. Missing type remains
the legacy compatibility signal and decodes as `system-learning`. Iterations 13
through 18 are implemented: iteration 13 established the active system-learning
model, iteration 14 the enforced modular-monolith architecture, iteration 15 the
retired legacy surfaces, iteration 16 source citations and consolidation, iteration 17 teacher queue/steering, OCR, and web search, and iteration 18
review cutover, independent notes and portable release infrastructure. Code and the current
iteration documents are product-data truth.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

存储为 assets/review、assets/practice，以及 assets/notes 的条目与持久任务；原 intro/body 一次性拼接 cutover，generated 文件与引用删除。成功回合按 learner message 去重计数，笔记独立记录 provider usage；不追溯升级前历史自动生成。

未开放工具使教师候选无效，不作为 completed 回合持久化；失败候选不计笔记。Anthropic 签名/不透明思考块是仅在当前工具续答内存中的传输状态，不进入 conversation blocks。既有 response-superseded 支持已确认异常回复撤回；所有新会话/助教输入投影排除已撤回消息，封存快照只应用截止 seq 之内的撤回。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

source.json 新增 parser/extractedBytes/extractionTruncated。原件不可变，最大 50MiB；派生文字最大 4MiB，ZIP 展开另有限额。processing/stored 在启动恢复，结果写入必须仍为同一 active revision，删除不能被晚到结果复活。旧原件/可读派生结果保留，source-processing 写入端口退役。安装清单拥有应用文件，用户数据独立于安装目录并在卸载后保留。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
