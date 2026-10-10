# Learning Project Structure

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: current project types, flat storage, learning-unit shape, and navigation semantics.

## Core Principle

LLL is a folder-first local learning workbench. A project directory represents
a real learner commitment, not every topic AI happens to mention.

```text
overview content describes possibilities
project folders record learner commitments
one system-learning project owns one teacher conversation and becomes one learning unit
```

## Project Types

| Type | User-facing shape | Active surface |
|---|---|---|
| `discipline-map` | 学科地图 | 学科总览 + 学习计划 |
| `system-learning` | 学习单元 | 教师 + 资产 + 资料 |

The learner explicitly chooses the type before creation. AI may recommend and
prefill, but cannot submit the choice. Existing projects without type metadata
decode as `system-learning` for compatibility.

## Discipline-Map Project

A discipline map is a long-lived organizing entry. The frontend exposes an
explicit `××学科总览` row; physical filenames remain internal contracts.

```text
projects/<map-slug>/
  project.md
  state.json
  overview.md
  discipline-topics.json
  learning-plan.json
  runs/
  assets/
```

The encyclopedia Agent plans one coherent map, writes H3 major chapters and H4
learnable topics to `overview.md`, and writes machine-readable boundaries to
`discipline-topics.json`. The learner owns topic selection and order through
`learning-plan.json`. Mentioning a topic never creates a directory, page, job,
or sidebar node.

A map brief records map shape, overview goal, optional scope notes, and learner
notes. It does not contain a focused-learning ability ladder or phase.

## System-Learning Project

A system-learning project is one focused learning unit. Its active experience
is a teacher conversation with accumulated assets and source materials:

```text
projects/<learning-slug>/
  project.md
  state.json
  learning-scope.json
  unit.json
  conversation/
    conversation.json
    events.jsonl
    compact.json
  assets/
    review/
    practice/
    notes/{state.json,entries/,jobs/}
  sources/<source-id>/revisions/<revision-id>/
    original/
    derived/
  assistant-tasks/<task-id>/
    task.json
    input-manifest.json
    attempts/
  runs/
  progress/
  migrations/iteration-18.json
```

The skeleton also creates the legacy `intro/`, `explain/`, and `practice/`
zone directories: practice persistence, the learner-writable notes/survey
files, and the infographic pipeline still live there through the compatibility
layer. `progress/` holds the activity event stream and summary that feed the
home and usage heatmaps.

`project.md` stores the focused-learning brief. `learning-scope.json` stores the
objective topic boundary: a map-origin project receives a ready creation-time
snapshot; a standalone project begins with a draft boundary that teaching may
clarify. Map regeneration never mutates an existing unit's snapshot.

The scope is guidance, not a hard product gate. One conversation may follow the
learner across related themes, and auto-compact plus the teacher system prompt
manage long context. The conversation comes first; the learning unit is the
durable form that accumulates around it.

## Creating A Learning Unit From A Map

The learner may request a deep dive beside an H4 topic. The system:

```text
prefills title and learning context
sends map slug + canonical topic ID
shows a confirmation form
creates a flat system-learning project only after confirmation
copies the canonical topic boundary into learning-scope.json
opens the new unit's one teacher conversation
```

No physical parent/child project relation is created. The existing sidebar
folder is classification only.

## Sidebar Contract

```text
物理学                    文件夹标题：仅展开/收起
├─ 物理学学科总览         discipline-map
├─ 流体力学               system-learning
└─ 热力学                 system-learning
```

Overview headings never become sidebar nodes. A bound discipline map is derived
from `mapProjectSlug` and does not enter `slugOrder`; learning units remain
ordinary movable folder members.

## Assets, Sources, Runs, And Preferences

- `assets/review` and `assets/practice` are editable, versioned Markdown assets.
- `assets/notes` holds teacher API notes, durable batch jobs and counting state.
- Retired intro/body/generated assets and their public reading endpoints are removed.
- `sources/` holds learner-provided immutable revisions and, after supported
  background parsing, exactly one `derived/content.md` citation per ready revision.
- `runs/` and task attempts hold execution evidence until the assistant declares
  an output for publication; scratch files remain attempt-local.
- `<WORKSPACE>/preferences.md` is one learner-owned global preference document,
  read-only to AI and never tool authorization or factual evidence.

Formal assets are written by Go after assistant acceptance and version/merge
handling. The visible CLI writes only its attempt workspace and result manifest.
Confirmed consolidation updates review; practice changes only if explicitly requested.

## One-Time Cutover Boundary

Legacy Intro/Explain/Practice files are read only during verified cutover. Canonical
intro/current.md precedes body/current.md in the new review; canonical empty files
stay empty. Old annotations retain quote snapshots and source version IDs. A failed
read, conflicting existing review or linked directory blocks that project's learning
APIs and keeps sources. After journal completion there is no runtime fallback.
Source originals/derived, conversations, maps, plans and preferences are retained.
Retired Summary/Extend and project memory remain ignored historical files.

## Required State

All projects track stable ID, globally unique slug, title, immutable type,
status, and timestamps. Legacy `activeZone` may remain in old state files but is
not active navigation state for the teacher workspace.

## Decision References

- [ADR-0005](./ADR/0005-discipline-map-and-system-learning-project-types.md) defines the retained project types.
- [ADR-0007](./ADR/0007-flat-project-storage-and-sidebar-classification.md) defines flat storage and map-backed folders.
- [ADR-0009](./ADR/0009-explicit-discipline-overview-entry.md) defines the explicit overview row.
- [ADR-0010](./ADR/0010-hierarchical-discipline-maps-and-learning-plans.md) defines hierarchical maps and learner-owned plans.
- [ADR-0011](./ADR/0011-learning-scope-snapshots-and-intro-calibration.md) defines topic-boundary snapshots.
- [ADR-0012](./ADR/0012-teacher-assistant-learning-workspace.md) defines the conversation-first learning unit.
- [ADR-0013](./ADR/0013-stable-stream-rendering-and-global-preferences.md) defines stable streaming and global preferences.
- [ADR-0016](./ADR/0016-retire-legacy-summary-extend-and-knowledge-garden.md) retires Summary, Extend, and Knowledge Garden product surfaces.

Earlier partially effective workbench and artifact decisions remain in the
[active ADR graph](./ADR/README.md). Only fully superseded decisions move to the
[historical ADR index](../99-archive/adr/README.md).

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

资产导航只展示复习稿、练习题、笔记；review 替代 intro/body，并保留迁移前批注的引用快照与旧来源标记。generated 成果目录及运行期读取不保留，source originals/derived、教师会话与学科地图不在清理范围。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。迭代 18 交付的历史基础仍保留；资料与安装的当前合同由迭代 19 更新。


## Iteration 19 — 资料解析与安装（已实现）

资料原件与派生文字继续位于 sources/<sourceId>/revisions/<revisionId>/，本地解析后台完成；教师/资产/资料顶部均提供导入入口。资料无需手动引用即可进入下次教师请求的有界摘录，显式引用仅提升优先级。笔记在资产页剩余空间独立滚动。公开安装目录不包含私有工作区，维护者仓库运行和安装运行分离。

参见 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md) 与 [ADR-0025](./ADR/0025-native-source-context-and-installers.md)。验收状态见交付记录。


## Iteration 20 — Native Windows application and MSI (implementing)

Windows targets a dedicated WebView2 application window with the Go services in the same process, private loopback, owned shutdown and a standard per-user MSI. Installation detects installed versions and asks before overwrite/upgrade, supports repair/uninstall and protects learner data. Native window/executable/installer/shortcut/UI share the project-owned icon. Verified duplicate script desktop entries are removed; source deployment commands remain maintainer tools. macOS/Linux currently retain their existing browser-based distributions; this Windows MSI change does not claim native shells on those platforms.

See [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md) and [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md); actual native/legacy-transition verification belongs to the delivery notes.
