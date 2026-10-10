# Repository Map

Status: active  
Owner: project maintainer  
Last reviewed: 2026-10-10
Source of truth: this file classifies repository content by ownership and trust level.

## Purpose

This file answers four practical questions:

```text
哪些内容是当前项目实现的一部分
哪些内容是文档合同
哪些内容只是参考附件 / mock / 对齐材料
哪些内容已经属于旧版或运行产物，不应继续当成兜底实现
```

## 1. Product Code

These directories are the current implementation surface:

```text
backend-go/        Go backend, API routes, session runtime, file stores
frontend/          Vite + React 18 + TypeScript SPA
learning-agents/   learning-runtime agent registry, charters, primitives
```

Use these as executable truth before trusting any design/mock document.
`learning-agents/` was renamed from `agents/` in iteration 17 (ADR-0019) to
stop colliding with the repo-maintenance "coding agents" concept governed by
AGENTS.md.

### Current Source Layout

Iteration 14 reorganized the internals; iteration 17 normalized the
top-level operator layout:

```text
backend-go/internal/app/              composition and cross-module glue
backend-go/internal/transport/        HTTP/SSE adapters
backend-go/internal/modules/          business-capability facades and private implementations
backend-go/internal/platform/         config and filesystem mechanics (SSE broadcast lives in internal/httpx, IDs in internal/idgen)
backend-go/internal/compatibility/    legacy reads and migration only

frontend/src/app/                     router, providers, app shell, single SSE mount
frontend/src/features/                business feature public entries and private implementation
frontend/src/shared/                  proven cross-feature UI and pure utilities

scripts/                              operator entry points and backend-invoked runtime scripts
                                      (Start/Stop/Install, gen_infographic.py — ADR-0023)
tests/                                cross-module contracts, fixtures, smoke, manual QA scripts
tests/manual/                         human-run one-off QA scripts (not in npm run check)
tools/archcheck/                      executable dependency policy
tools/check/                          developer check pipeline (run-check.js, check-coverage.js)
.artifacts/quality/                   ignored generated test and coverage evidence
.artifacts/                           ignored local-only evidence (quality gates, incident forensics)
config/                               committed schema and non-secret example
LICENSE                               repository license (MIT)
.gitattributes                        line-ending and binary normalization contract
```

This is the implemented source layout; `BACKEND_ARCHITECTURE.md` owns its
dependency and responsibility rules.

## 2. Project Runtime Data

These directories are runtime state or generated local data, not product contracts:

```text
workspace/          learner workspace root (ADR-0023): projects/ (incl. the
                    folders ledger), preferences.md, config.local.json,
                    workspace-local learning-agents/ and scripts/ overrides,
                    tmp/ diagnostics (teacher stream-shape log, launcher PID)
dist/               built desktop/backend artifacts
learning/           local learning artifacts / workspace data
repro-shell/        local repro helpers
```

The workspace root defaults to `<repo>/workspace/` and is overridable via
`--workspace`, `LLL_WORKSPACE_ROOT` / `WORKSPACE`, or `workspace.root`.
Learning agents and the infographic script resolve from the workspace first
and fall back to the repository's shipped copies.

A legacy workspace-root `folders.json` is migrated into `projects/` by a
one-time copy on first open; the legacy file is left in place.

Do not write long-lived product rules here.

`npm run check:repo` is the executable guard for this boundary. It fails when
a tracked path matches `.gitignore`, including an ignored file that was staged
with `git add -f`; the command runs inside every repository quality gate.

## 3. Source-Of-Truth Documents

These are the current contract layers:

```text
docs/INDEX.md
docs/00-product-and-architecture/
docs/01-iterations/ current delivery baseline only
docs/99-archive/   ← historical only, never current contract
AGENTS.md
```

Within `docs/`:

```text
00-product-and-architecture/   long-lived facts, ADRs, rules, repository map
01-iterations/                 current delivery scope, API contracts, tests, acceptance
99-archive/                    retired ADRs, iterations, designs, and references
```

## 4. Legacy / Archive / Superseded Material

These should be treated as archive or migration reference, not fallback product:

```text
docs/99-archive/               retired docs; excluded from default AI context
```

The archive is retained for traceability only and never acts as a product fallback.

## 5. Current Reality Check

### Database

Current backend persistence is:

```text
file-first
```

Observed stores:

```text
projects/<slug>/... files
explain/confusions.json
intro/assessment.json
explain/manifest.json + explain/pages/*.md
practice/tasks.json + practice/answer-key.json
practice/attempts/*.json
practice/submissions/*.json
practice/evaluations/*.json + *.md
progress/events.jsonl + progress/summary.json
runs/_index/*.json
preferences.md
```

`preferences.md` remains the only editable preference source. A recovery-only,
workspace-keyed mirror lives outside the repository under the operating system
user configuration directory so repository replacement does not erase the last
saved preference content.

There is no active SQLite / Postgres / MySQL / embedded DB integration in the current codebase.
Any future database is documented only as a possible acceleration/index layer, not present runtime truth.

### Current Learning Workspace Snapshot

Implemented in code today:

```text
Go backend + React frontend
conversation-first API teacher with durable response recovery
visible asynchronous CLI assistant tasks and sealed inputs
versioned review/practice assets and independent teacher notes
source upload, immutable revisions, and built-in local parsing
one ready `derived/content.md` source citation; separate teacher/note usage ledgers
single learner-owned global preferences file
one-time Intro/Explain/Practice migration; generic invoke/session infrastructure
review annotation CRUD and Ask AI
Markdown practice reader/editor; no structured practice compatibility in the active workspace
Markdown editor for learner-owned files
file-first project/runs persistence
```

Compatibility retained intentionally:

```text
old projects may still contain `memory/`, `summary/`, and `extend/` files
those files remain recoverable but are not active teacher assets, prompt memory,
file-API targets, migration input, or agent outputs
the `/memory` frontend route redirects to `/preferences`
the standalone Agent-management frontend route is retired; legacy invoke/session APIs remain compatibility-only
```

### Historical Iteration 04 Contribution (migration inputs only)

Implemented in code:

```text
evidence-based Intro assessment with generated prerequisite summaries
manifest-owned Explain pages with follow-up parent links
six Practice question types with private answer-key checking
project-local idempotent growth events
legacy Explain and Practice readers
```

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

发布压缩包含二进制、frontend/dist、learning-agents 与非敏感说明/配置样例；用户 workspace 不随包发布，默认落入 OS 用户目录。generated 与 intro/body 不再是目标正式资产；一次性迁移读取不得变成新的运行期 fallback。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

本地解析：sources/internal/parser、processing 与 store/parse.go；教师有界资料上下文：teacher/internal/service/source_context.go。安装工具：tools/installer 与 tools/release/build-installers.js，平台安装/卸载/烟测脚本；公开包含 THIRD_PARTY_NOTICES.txt。维护者 Start-LLL/desktop:install 保留但不进入发行包。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
