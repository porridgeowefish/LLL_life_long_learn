# Backend Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
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

## Current Package Model

ADR-0015's behavior-preserving modular package model was implemented in
iteration 14:

```text
internal/
  app/{bootstrap,integration}
  transport/httpserver
  modules/
    teacher
    assistant
    assets
    sources
    projects
    learning
    preferences
  platform/{config,filesystem}
  compatibility
  httpx, idgen, imageconfig, uiconfig   root-level technical helpers
```

Each module root is its public facade. Private domain, application, port, and
adapter packages live below that module's nested `internal` directory as
needed. Go visibility plus `tools/archcheck` prevents transport or other
modules from bypassing the facade. `app/integration` converts bounded values
between facades but owns no business rule; `app/bootstrap` is the only concrete
composition root.

The migration introduced no microservices, database, broker, public routes, or
project-file schemas.

## Responsibility Map

| Area | Primary packages | Responsibility |
|---|---|---|
| Transport | `internal/transport/httpserver`, `internal/httpx` | decode, size-limit, call one operation, encode, stream |
| Teacher | `modules/teacher` | context assembly, provider-neutral blocks, tool loop, durable recovery |
| Assistant | `modules/assistant` | authorization, task state, queue/leases, captured input, visible CLI lifecycle |
| Assets and sources | `modules/assets`, `modules/sources` | versions, edits, annotations, immutable source revisions and disclosure |
| Projects and learning | `modules/projects`, `modules/learning` | project roots, folders, scope, activity, progress and live run state |
| Preferences | `modules/preferences` | learner-owned global file and bounded read-only prompt snapshots |
| Compatibility | `internal/compatibility` | old-project readers, artifact watch, migration, and legacy routes only |

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
-> append provider-reported nonzero token usage by response
-> report durable successful completion through the bootstrap activity seam
-> expose final conversation projection through REST
```

Provider adapters live behind `teachergateway`; the frontend never parses a
provider SDK stream. A refresh may disconnect the browser but does not cancel
the server-side run. REST and durable conversation/run records repair missed
stream events.

The teacher owns two bounded tools: `delegate_learning_work` and, when the
optional webSearch configuration resolves a key, `search_web` (at most two
rounds per response — ADR-0019). Delegation tool calls are accepted
only when they refer to the disclosed proposal message and a later explicit
learner authorization. A normal agreement word is not enough if the proposal
identity or complete task definition is missing.

The delegation tool itself is omitted from unapproved ordinary turns. The
pre-request gate and task creation share the adjacent assistant-proposal
resolver, including bounded retries after technical rejection. An unoffered
tool discards the candidate and triggers at most one tool-free recovery; it
cannot create a task or contaminate notes. Anthropic signed/opaque thinking is
replayed in memory across live search rounds, so registering a tool no longer
silently disables thinking. Externally supplied tool history without replayable
blocks keeps its non-thinking transport shape. Provider-reported usage is
summed per request, not overwritten by the last search/recovery round.

## Assistant Task Flow

```text
authorized tool call
-> assistanttask service performs atomic check-and-create
-> durable task enters queued state
-> dispatcher acquires global/unit slots and attempt lease
-> exact conversation range, asset bases, source revisions, scope, and preferences are copied into attempt inputs
-> agentexecution.Service opens the selected visible CLI in the learning-unit folder
-> CLI accepts its outputs and writes result-manifest.json plus output directories
-> Go versions/merges declared review/practice candidates; no arbitrary deliverable publication
-> a publication I/O failure is terminal; there is no validator or automatic retry (ADR-0021)
-> durable task state changes and global SSE invalidates affected resources
-> successful publication with actual output reports through the bootstrap activity seam
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
| task state, attempts, leases, captured input manifests | assistant task/dispatcher service |
| formal asset versions and merge journals | asset application/store |
| source revisions and tombstones | source application/store |
| teacher provider usage | teacher usage store |
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
  assets/{review,practice,notes}
  sources/<source-id>/revisions/<revision-id>/{original,derived}
  assistant-tasks/<task-id>/{task.json,input-manifest.json,attempts}
  runs/
  progress/
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
- `learning-activity-updated {projectSlug}` invalidates every cached activity
  summary after a new idempotent progress event;
- REST returns durable conversation, task, asset, and source projections after
  refresh or missed events;
- startup scans task/run files to reconcile queued, running, interrupted, and
  terminal states.

SSE is never the only copy of a state transition.

## Preferences Boundary

`<WORKSPACE>/preferences.md` is Markdown capped at 256 KiB. The preferences
page and direct learner file editing are the only write paths. Teacher context
and assistant captured input receive bounded read-only snapshots. Preferences do
not authorize tools, establish facts, or override explicit learner choices.
Startup and explicit saves maintain a recovery mirror below the operating
system user configuration directory, keyed by workspace. If the canonical file
is missing, AI reads use the mirror and the preferences editor restores it.
The mirror is not a second editable preference source.

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

Legacy teaching formats are read only during cutover to review/practice. The active
teaching reader has no compatibility renderer, old asset aliases or arbitrary
publication. Generic invoke/session routes remain infrastructure for registered
Agents; they are not the teacher workspace. Summary, Extend and Knowledge Garden
historical files remain outside runtime services. New behavior belongs to module
facades and the single composition root.

## Non-Goals

```text
cloud deployment or accounts
multi-tenant collaboration
external queue/broker
database-first canonical persistence
hidden assistant execution
AI-managed preference or memory writes
```

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

仍维持现有模块化单体边界：teacher 拥有笔记任务与调用，assets 拥有 review/practice，app/integration 组装跨模块 ports。独立 generated 发布流程退役；sources 本地解析独立拥有资料输出边界。平台层负责资源根、用户数据根及终端适配，不在业务模块猜 cwd 或 go.mod。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。

platform/filesystem owns release resource/user-data discovery and the workspace
OS lock. main reserves its listener before bootstrap; teacher owns note recovery,
strict provider selection, two-worker scheduling and per-project serialization.
platform/config writes sensitive configuration as 0600 on Unix.


## Iteration 19 — 资料解析与安装（已实现）

sources/internal/parser 拥有原生解析，sources/internal/processing 拥有恢复、串行调度和失败状态；store 在同一删除锁内校验 active revision 并发布本地派生文本。teacher/internal/service/source_context.go 拥有一次请求资料快照与相关片段预算；模型选择决定窗口，资料优先于历史重复正文。assistant 删除 sourceUpdate/CommitSourceDerived，旧 source-processing 不调度/不恢复发布。Go 构建要求 1.26，解析运行无需外部依赖。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
