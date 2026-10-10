# LLL Roadmap

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: delivered product slices and the next architecture work; iteration documents own exact acceptance contracts.

## Delivered Foundation

```text
Iterations 02–06   file-first Go backend, learning agents, rich readers, Ask AI
Iterations 07–12   project types, flat storage, discipline maps, plans, scope snapshots
Iteration 13       teacher conversation, async CLI assistant, assets, sources, migration
Iteration 14       enforced modular monolith, unified configuration, quality gates
Iteration 15       retired Summary/Extend/Knowledge Garden product surfaces
Iteration 16       source citations, teaching outlines, consolidation, teacher usage
Iteration 17       teacher queue with steering, regeneration/export, OCR, web search
Iteration 18       review/practice cutover, teacher notes, simplified UI, portable releases
```

The active system-learning experience is `教师 / 资产 / 资料`. The earlier
Intro / Explain / Practice formats are one-time migration sources only;
review/practice/notes are the current assets, with no old teaching reader. Summary, Extend, and
Knowledge Garden are retired product surfaces; their historical files remain on
disk only.

## Current Architecture Commitments

- API teacher owns responsive teaching dialogue, one disclosed delegation tool, and the bounded web-search tool when configured (ADR-0019).
- Visible native Agent CLI owns approved heavy work.
- Conversation, task, source, and asset files are durable truth.
- Rich stream rendering coalesces deltas and defers Mermaid until completion.
- One `<WORKSPACE>/preferences.md` is learner-owned and read-only to AI.
- There is no project-memory agent, automatic preference inference, or AI write path.

## Current Baseline

Iteration 18 is the current implemented baseline: review/practice, teacher notes,
project usage chart, sidebar navigation and portable releases. Iteration 17 delivered durable conversation queue
with steering, response regeneration and transcript export, vision-model OCR
for image sources, the bounded web-search tool, and repository normalization.
Iteration 14 remains the architecture baseline: it reorganized the existing
runtime into an enforced business-capability modular monolith and added one
typed configuration system plus standardized architecture, test, coverage,
build, browser, and Windows smoke gates.

The slice preserved learner-visible behavior, public APIs, and canonical
project schemas. It introduced no microservices, database, broker, or product
feature.

## Iteration 16 Delivered

Canonical asynchronous source parsing/citation, map teaching outlines,
approved Intro/Body consolidation with optional practice, and teacher-only
token usage are delivered. Browser E2E remains the learner's acceptance step.

## Later Candidates

These remain candidates after iteration 14 and are not part of its committed
scope:

1. Add browser performance instrumentation for long teacher conversations,
   including render counts, long tasks, and rich-block timing.
2. Virtualize very long completed transcripts without disturbing scroll follow
   or message anchors.
3. Add a user-visible import/export flow for `preferences.md` while retaining
   one canonical file.
4. Keep retired historical directories recoverable while avoiding new runtime
   readers, writers, or product interfaces for them.

## Permanently Out Of Scope

```text
Multi-user accounts
Cloud-first canonical storage
Database-first project truth
Hidden assistant execution
AI-controlled preference or memory writes
```

## Verification Baseline

```text
go test ./...
npm --prefix frontend test -- --run
npm --prefix frontend run build
```

Iteration 14 added repository-level `check:fast`, `check`, and `check:full`
gates with generated evidence below the ignored `.artifacts/quality/`
directory; the individual commands above remain runnable.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

迭代 18 已实现资产 cutover 与成果删除、教师自动笔记、主页/导航/Token/密钥/CLI 整理、六目标便携包与 OS matrix 验证。Windows 原生包完成本地验证；macOS/Linux 真实终端及远端 CI 验收仍待对应机器运行。未发布远端版本。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

迭代 19 实现本地文档解析、自动有界资料上下文、笔记滚动、始终可用的导入入口以及六目标安装分发。macOS PKG 依赖原生构建，跨编译不表示已在对应主机安装；实际验证与剩余缺口见本迭代交付记录。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
