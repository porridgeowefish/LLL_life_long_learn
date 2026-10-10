# learn-with-ai PRD

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: long-lived product intent and product-object boundaries for LLL.

## Problem

Serious learning with AI fails in two different ways:

```text
learning context, execution, notes, and review are scattered across tools
broad disciplines and focused learning goals are forced into the same content shape
```

The first causes weak continuity. The second causes premature depth: a learner
who needs a field map receives a tutorial, while a learner ready to build skill
needs a committed learning workflow.

## Product Goal

learn-with-ai is a local-first learning workbench that:

```text
keeps learning context and artifacts in durable local project folders
lets the learner choose the product shape instead of hiding that decision in AI inference
uses AI as an advisor and controlled study operator, not an autonomous product owner
supports broad orientation and focused skill-building without conflating them
keeps execution inspectable through real local AI runtime sessions
```

## Primary User

The primary user is a power learner who is comfortable with local files and
terminal-based AI runtimes and wants a durable thinking environment rather than
a disposable chat answer.

## Product Objects

### Discipline map

A long-lived organizing project with learner-facing `学科总览` and `学习计划` views. It presents
the discipline's major questions, research areas, methods, relationships,
boundaries, applications, a multi-level chapter/topic architecture, and possible learning routes.

In the sidebar it is shown as an explicit `××学科总览` row inside its bound
discipline folder. The folder title only expands or collapses the folder.

Topics named in the overview are possibilities, not project objects. They do
not enter the sidebar or filesystem until the learner creates a project.

### System-learning project

A committed deep-dive learning unit. Since iteration 13 the delivered active
presentation is one teacher conversation plus editable assets and source
material. Legacy Intro/Explain/Practice is one-time migration input only;
there is no active compatibility reader for retired teaching assets.

It is a physically flat project regardless of whether creation started from a
map. When started from a map, existing folder membership places it beneath that
map-backed folder in navigation.

Every system-learning project owns one explicit content boundary. A map-origin
project snapshots its selected topic's planned boundary; a standalone project
starts with a draft boundary. This boundary guides
the teacher and assistant without mechanically rejecting natural cross-topic
questions.

## Product Principles

```text
The learner explicitly chooses discipline map or system learning.
AI may hold a temporary clarification conversation, recommend, explain trade-offs, and prefill; it does not persist, silently choose, or create.
Every registered Agent, including the encyclopedia Agent, is explicitly launched through the selected native Agent CLI; lightweight AI helpers are the only direct-provider exception.
The overview represents the possibility space; the sidebar represents actual commitments.
The encyclopedia Agent plans the whole architecture before writing; the overview uses H3 major chapters and H4 learnable topics. A separate top-level learning-plan view lets the learner choose topics, own their order, and check off one task at a time; AI does not choose the sequence.
Each H4 topic also declares what it owns, reuses, requires, and excludes. The backend snapshots that canonical boundary when the learner confirms a deep dive.
Each new-contract H4 topic has its learning action beside that body heading; legacy H3-only overviews remain actionable until regenerated.
System-learning projects use the conversation-first teacher workspace.
Legacy Intro/Explain/Practice may be read during one-time cutover only;
retired Summary and Extend files stay recoverable but have no product interface.
Existing projects retain their type; verified intro/body content moves into review.
The teacher discusses prerequisite gaps directly; creating a project records a deliberate commitment.
The teacher calibrates depth, examples, scaffolding, and practice difficulty within the scope snapshot.
All project directories are flat. Existing frontend folders are the only navigation/classification structure: an explicit `××学科总览` row opens the bound map, while system-learning projects remain movable child rows. Folder placement never changes learning behavior or prompt context.
```

## Current Delivery Boundary

The two project types, flat map navigation, scope snapshots, and the teacher
workspace are implemented. `教师` is one low-latency API
conversation, `资产` is versioned editable accumulated learning, and `资料` is
explicit private source material. A visible native-CLI assistant performs
learner-authorized substantial work asynchronously. The five teaching roles are
soft methods in one teacher prompt, not product modules. Current runnable code,
the current iteration, and active architecture documents are the delivery truth.
Iteration 16 delivered ready source citations in the teacher composer, map-topic
teaching outlines as soft guidance, confirmed conversation consolidation into
Intro/Body, and teacher-only token usage (now superseded by review and project usage). Iteration 17
delivered the durable conversation queue with steering, response regeneration
and transcript export, vision-model OCR for image sources, and the bounded
web-search tool.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

产品名称为 learn-with-ai，主页与侧栏承担简化后的导航。系统学习资产为复习稿、练习题、教师自动笔记；助教不再提供任意生成成果。便携发布保持本地 Go 服务与浏览器界面，提供 Windows/macOS/Linux 下载包。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

学习单元任意标签页均可导入 PDF、Markdown、PPTX、DOCX、XLSX、CSV，教师回复期间也可导入。单文件 50MiB，本地自动提取文字，不再调用助教解析。教师每次请求自动使用有界相关摘录，扫描件明确提示无可提取文字。公开分发提供 Windows 每用户安装程序、macOS APP/PKG、Linux DEB/用户安装包；原源码部署与仓库桌面快捷方式是维护者快速验证入口。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
