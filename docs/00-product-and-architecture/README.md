# Product And Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: `docs/INDEX.md` routes readers here; each linked document owns its topic.

This directory stores long-lived documents that answer:

```text
Why LLL exists
What problem it solves
Where the domain boundaries sit
Why the architecture is shaped this way
Which rules guide implementation and delivery
```

## Core Documents

- [PRD.md](./PRD.md)
- [REPOSITORY_MAP.md](./REPOSITORY_MAP.md)
- [SYSTEM_ARCHITECTURE.md](./SYSTEM_ARCHITECTURE.md)
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md)
- [DATA_MODEL.md](./DATA_MODEL.md)
- [BACKEND_ARCHITECTURE.md](./BACKEND_ARCHITECTURE.md)
- [AGENT_ARCHITECTURE.md](./AGENT_ARCHITECTURE.md)
- [LEARNING_PROJECT_STRUCTURE.md](./LEARNING_PROJECT_STRUCTURE.md)
- [API_CONTRACT_STRATEGY.md](./API_CONTRACT_STRATEGY.md)
- [ROADMAP.md](./ROADMAP.md)
- [DOCUMENTATION_STANDARD.md](./DOCUMENTATION_STANDARD.md)
- [QUALITY_COMMANDS.md](./QUALITY_COMMANDS.md)
- [LESSONS_LEARNED.md](./LESSONS_LEARNED.md)
- [agent-rules/README.md](./agent-rules/README.md)
- [ADR index](./ADR/README.md)

## Supporting Current Documents

This file is useful for active collaboration, but does not override the core
contracts above:

- [ALIGNMENT_MODE.md](./ALIGNMENT_MODE.md)

Retired alignment artifacts are indexed under
[`docs/99-archive/references/`](../99-archive/references/README.md) and are not
part of the default reading path.

## Relationship To Iteration Docs

Long-lived docs define direction and boundaries.
Iteration docs define current delivery scope.

If they conflict:

```text
Current implementation follows docs/01-iterations/ for scope.
Long-term changes must be captured in ADR or updated architecture docs.
```

The active architectural baseline is
[ADR-0012](./ADR/0012-teacher-assistant-learning-workspace.md): one API teacher
conversation per system-learning unit, asynchronous visible-CLI assistance,
versioned teaching assets, and explicit source material. It is extended by
[ADR-0013](./ADR/0013-stable-stream-rendering-and-global-preferences.md), which
defines stable rich-stream rendering and one learner-owned global preferences file.
[ADR-0014](./ADR/0014-current-documentation-surface-and-archive-boundary.md)
defines which documents belong to the current AI context and which are historical only.
[ADR-0015](./ADR/0015-business-modular-monolith-boundaries.md) defines the
implemented iteration-14 architecture: business-capability modules with private
implementations, selective hexagonal boundaries, one composition root, one
typed configuration system, and executable architecture checks.
[ADR-0022](./ADR/0022-durable-learning-activity-and-sse-refresh.md) defines
the durable activity event boundary and shared-SSE refresh path for both
learning heatmaps.

## Iteration 18 Current Contract

Status: implemented; platform-native and remote workflow validation limits are recorded in delivery notes.

迭代 18 与 accepted ADR-0024 定义当前合同，包括 learn-with-ai 便携发布、review/practice 收敛、教师自动笔记及独立 generated 成果退役；详细合同由迭代文档拥有。

参见 [Iteration 18](../01-iterations/foundations/iteration-18-learn-with-ai/README.md) 与 [ADR-0024](./ADR/0024-review-notes-and-portable-release.md)。当前合同以迭代 18 和可运行代码为准。


迭代 19 实施本地资料解析、自动资料上下文与安装包，见 [ADR-0025](./ADR/0025-native-source-context-and-installers.md) 和 [Iteration 19](../01-iterations/iteration-19-document-import-installers/README.md)。


Windows native app/MSI upgrade is tracked by [ADR-0026](./ADR/0026-native-windows-desktop-and-msi.md) and [Iteration 20](../01-iterations/iteration-20-native-desktop-msi/README.md).
