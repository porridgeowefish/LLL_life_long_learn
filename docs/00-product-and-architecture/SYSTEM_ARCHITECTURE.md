# System Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: current long-lived runtime boundaries; code owns implementation details.

## Current Runtime

LLL is a local-first web application:

```text
frontend/          React 18 + TypeScript SPA
backend-go/        Go HTTP/SSE server and application services
projects/          canonical learning-unit, conversation, asset, source, and task files
learning-agents/   native CLI charters and reusable reasoning primitives (renamed from agents/, ADR-0019)
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
-> platform/{config,filesystem}          technical mechanics only
-> compatibility                        legacy reads and migration only

root-level technical helpers: httpx (JSON + SSE broadcaster), idgen (IDs),
imageconfig, uiconfig. SSE broadcasting and ID generation live here, not in
platform.
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
-> 资产: versioned review / practice and independent teacher notes
-> 资料: immutable uploaded revisions and locally parsed text
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
-> nonzero provider-reported usage appends to conversation/usage.jsonl
-> a successful durable response appends one idempotent learning-activity event
-> refresh reconnects to durable run state and recovers the final response
```

The teacher owns teaching only. It has two bounded tools:
`delegate_learning_work` (registered only after an actual assistant proposal
and a later explicit learner approval; agreement to an ordinary teaching
question does not qualify) and `search_web`
(registered only when the optional webSearch configuration resolves a key; at
most two search rounds per response — ADR-0019).

Offering tools does not disable configured thinking. Signed/opaque Anthropic
thinking blocks remain in memory for the current tool round-trip and are never
persisted or exposed as answer text. An unoffered tool invalidates its entire
candidate: clear browser text/reasoning with `message-reset`, then retry the
original question once without tools. Repeated violations fail with no answer
blocks, no assistant task, and no successful note turn. Usage includes every
provider request, including recovery.

Rich output uses a two-rate rendering boundary: provider deltas are coalesced,
Markdown/KaTeX/sanitization renders from a slower snapshot, collapsed reasoning
is not mounted, and Mermaid stays a stable placeholder until the message is
complete. Auto-scroll is throttled and historical messages are isolated from
live-message rerenders.

## Assistant Flow

```text
authorized teacher tool call
-> assistant task service validates proposal identity and creates durable task.json
-> dispatcher captures conversation/assets/sources/preferences into an attempt workspace
-> shared execution service opens the selected native CLI in the project folder and injects one prompt
-> CLI accepts its own outputs and writes result-manifest.json plus output directories
-> Go versions/merges declared review/practice candidates; sources locally parses derived content.md independently
-> a publication I/O failure is terminal; the CLI is not rerun or server-side revalidated (ADR-0021)
-> task state and global invalidation events update the conversation UI
-> a succeeded task that actually publishes an asset appends one idempotent learning-activity event
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
assets/{review,practice,notes}
sources/<source-id>/revisions/<revision-id>/{original,derived}
conversation/usage.jsonl
assistant-tasks/<task-id>/{task.json,input-manifest.json,attempts}
progress/                                activity events and aggregated summary
migrations/iteration-18.json
conversation/note-usage.jsonl
assets/notes/{state.json,jobs,entries}
```

REST reconstructs durable state after refresh; SSE only reduces latency. Formal
asset/source/conversation/task writes are owned by Go and use validated,
recoverable file operations. The CLI cannot write canonical records directly.

The home and usage heatmaps both read aggregated `progress/events.jsonl` via
REST. The single AppShell SSE connection carries `learning-activity-updated`
after a newly persisted event, which invalidates cached activity summaries; it
does not carry a second heatmap state or replace REST recovery.

`<WORKSPACE>/preferences.md` is the only active preference document. Only the
explicit preferences editor or direct user file editing may write it. Teacher
and assistant prompts receive bounded read-only snapshots. Existing project
`memory/` folders are preserved as ignored legacy data and are not supplied to AI.
The server mirrors an existing canonical preference file to a workspace-keyed
recovery copy under the OS user configuration directory at startup and after
explicit saves. A missing canonical file can be read from that copy and is
restored when the learner opens the preferences editor. A non-empty recovery
copy also repairs an unexpectedly zero-byte canonical file; explicit empty
saves remain valid because they update both copies together.

## Compatibility Boundary

Legacy teaching files are one-time cutover input. The active reader uses only
review/practice/notes; retired intro/body/generated asset endpoints return 404.
Generic invoke/session infrastructure remains for discipline maps and explicit
registered Agents. Historical Summary/Extend/Knowledge Garden and project memory
files remain ignored. Old teaching files never provide runtime fallback.

## Architectural Rules

```text
filesystem-first durable truth
one active teacher conversation per learning unit
explicit learner authorization before heavy assistant work
visible native CLI through one reusable execution service
raw execution separated from curated/versioned assets
REST recovery; SSE as notification/stream transport
learner-owned preferences; no AI-managed memory
legacy teaching files are migration input, never runtime fallback
```

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

已新增独立教师笔记后台服务，按持久回合批次直接调用 API，经共享 SSE 刷新资产；不走原生 CLI。发布模式从可执行文件目录解析随包资源，使用 OS 用户数据工作区，绑定 loopback；三平台补齐实际可见终端启动。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。

Before mutable bootstrap, the executable holds a workspace OS file lock and reserves
the loopback listener. This prevents a second port or process from recovering live
jobs. Notes startup replays successes after its persisted epoch; historical turns
are seeded as seen. Explicit missing provider IDs fail instead of falling back.


## Iteration 19 — 资料解析与安装（已实现）

sources 本地串行解析 worker 在 app/bootstrap 启动，HTTP 只持久上传/状态并通知 worker，恢复 stored/processing。文件 50MiB、派生文本 4MiB、解压/解析/模型摘录各自限量。teacher 在每次请求开头快照 ready 资料，后续工具续答共享快照，新导入下次请求生效。纯 Go 解析编入程序；Windows Setup、macOS APP/PKG 和 Linux DEB/安装脚本重用完整资源包，用户数据与安装目录分离。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
