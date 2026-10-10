# API Contract Strategy

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: long-lived backend/frontend contract strategy; Go handlers and TypeScript types own current shapes.

## Current Surface

The Go server currently exposes resource groups for:

```text
health and system lifecycle
settings and appearance
projects, project types, project lists, sidebar folders, zones, and activity
agents and invocation
sessions, follow-up, cancellation, and run status
project files
Explain confusions and Ask-AI
Practice tasks, attempts, checking, submission, and evaluation
progress, activity aggregation, infographics, and SSE events
teacher conversation and response recovery
assistant task projection, assets, annotations, sources, and preferences
```

`backend-go/internal/transport/httpserver/router.go` is the route truth. Go request/response
types and frontend TypeScript types own payload truth.

## Contract Rules

```text
iteration API docs describe only the current slice delta
complex bodies use typed field tables
errors include HTTP status and stable code in new contracts
backend and frontend types change in the same implementation task
legacy-compatible defaults are explicit
Markdown never overrides implemented route behavior
```

## Iteration-14 Compatibility Freeze

The modular-monolith refactor changes internal ownership, not the public
product contract. Existing HTTP paths, status behavior, request and response
shapes, normalized teacher stream blocks, and global invalidation events are
frozen inputs to iteration 14.

Backend capabilities expose small Go facades; frontend features expose one
`index.ts`. These internal entry points replace direct imports of stores,
providers, executors, components, hooks, or API implementations. They do not
create a second public HTTP API.

Contract tests run against the new composition root during migration. Any
discovered need for a public shape change is a separate scoped decision that
must update the active iteration contract, Go and TypeScript types, migration
behavior, and tests before implementation.

## Project Types And Discipline Maps

Project APIs become type-aware:

```text
discipline-map
system-learning
```

New clients send an explicit type; legacy projects and legacy create requests
default to `system-learning`. The optional AI advice operation accepts temporary
conversation turns, returns a reply and optional recommendation, and has no
persistence or project-creation side effect.

The lightweight type advisor may use the configured HTTP provider. The
registered encyclopedia Agent may not: its generation endpoint returns a normal
Session/run response after launching the selected native Agent CLI, and the
overview is refreshed from the filesystem artifact event.

Folder APIs preserve the existing folder/membership payload and optionally bind
`mapProjectSlug`. The backend reconciles indexed discipline maps into folder
overviews. The frontend renders each binding as an explicit `××学科总览` row,
while map slugs never enter child `slugOrder` membership.

Exact request, response, and error semantics are owned by the implemented Go
and TypeScript types. Historical delivery rationale is indexed in the archive.

The on-demand prerequisite bridge is removed. Legacy prerequisite summaries are generated as part of
`intro/assessment.json`; there is no separate prerequisite-helper endpoint.

## Discipline Planning

Discipline-map generation records `overview.md` and `discipline-topics.json`; AI does not choose a task
order. `GET/PUT /api/projects/{id}/learning-plan` read and atomically replace the
learner-owned ordered task list in `learning-plan.json`. Older maps without the
file read as an empty list. Root-file artifact events independently refresh the
overview and plan queries. Completing a task records an idempotent activity event
for that completion timestamp. Exact behavior is owned by Go routes, stores, and
the matching TypeScript client types.

## Learning Scope

`GET /api/projects/{id}/discipline-topics` reads the validated topic-boundary
catalog. A system-learning create request may send `scopeSource` with
`discipline-map`, map slug, and topic ID. The backend resolves the canonical
topic and writes a creation-time `learning-scope.json` snapshot; the client
never supplies boundary content. Standalone creation writes a draft scope.
Exact shapes and stable errors are owned by the implemented request types and
learning-scope store.

## Project Deletion

`DELETE /api/projects/{id}` permanently removes a project resource after the
client obtains one explicit learner confirmation. The server rejects deletion
with `409 project_has_active_session` while an Agent may still write artifacts.
Successful deletion includes canonical project files plus workspace-global
folder references and in-memory session metadata. Exact semantics are owned by
the project route, workspace operation, and their tests.

## Conversation And Task Boundary

The active API includes one provider-neutral teacher-turn stream, durable conversation
reads, task and asset reads, editable core assets, review annotations, and source
operations. The frontend consumes normalized block events and never consumes a
provider SDK stream directly.

Ready sources expose one teacher-citation Markdown endpoint; the source ledger
does not preview it. `GET /api/usage/teacher` returns paginated, durable
conversation-level teacher usage with per-response rows. These values are
provider-reported only and do not include assistant usage.

Only a turn explicitly approving an actual adjacent assistant proposal sees
`delegate_learning_work`, with logical type, objective, source references and
optional practice intent. Proposal identity is resolved by the service, not the
model. The teacher also sees the bounded `search_web`
tool when the optional webSearch configuration resolves a key (ADR-0019).
Application services inject
physical paths, approval identity, task/run IDs, input ranges, and executor
policy. CLI execution uses a sealed envelope and generic result manifest rather
than public HTTP payloads.

Teacher response SSE can emit `message-reset` to retract an invalid candidate
before one tool-free recovery. Clients must clear pending and rendered answer,
reasoning and tool notices; only the final valid candidate is persisted. Signed
provider thinking used for tool continuation is private in-memory transport
state and never part of this SSE or conversation schema.

Global SSE carries identifier-only invalidation for conversation, task, asset,
source, activity and teacher-note changes. `learning-activity-updated` has payload
`{projectSlug}` and causes all cached `GET /api/activity` summaries to refetch.
REST remains the recovery truth. There is intentionally no
public assistant-task create, cancel, or retry endpoint in the initial contract.
Retired intro/body/generated asset routes return 404. Review annotations retain their canonical resource; legacy confusion APIs are not the teaching workspace flow.

Exact stable shapes and errors are documented by the current iteration
`API_CONTRACT.md`; implemented Go routes and TypeScript types remain
executable truth.

## Compatibility

Breaking public-shape changes require an ADR, an iteration contract, migration
behavior, synchronized client/server types, and tests covering older persisted
projects or requests.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

前后端同步升级为 review/practice 与 notes 资源；intro/body/generated 接口退役为 404，不保留旧端点别名。配置敏感资源返回完整密钥并限制本地同源/no-store，诊断保持脱敏；笔记配置、重试与项目用量聚合合同由迭代 18 拥有。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

POST sources 自动本地解析，不需要 parseApproved/cloudDisclosureAccepted；返回 source/revision/disposition/reason，不返回助教 task。50MiB 超限 413。POST sources/{sourceId}/retry 重试原修订，失败/opaque 状态可提交，202 返回 source。source 新增解析器/字节数/截断标记；source-updated 继续走唯一 AppShell SSE。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
