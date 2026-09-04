# Architecture Decision Records

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: active ADR numbering, status, and supersession index for LLL.

## Rules

```text
Use the next unused four-digit number.
Register every new ADR here in the same change.
Accepted ADRs are historical records and are not rewritten to hide old decisions.
New decisions supersede earlier decisions explicitly.
Architecture work is incomplete until the governance landing table is satisfied.
Move an ADR out of this active index only after every surviving current fact has
an active owner and the historical index records the move.
```

## Active Index

| ADR | Title | Status | Relationship |
|---|---|---|---|
| [0001](./0001-documentation-structure.md) | Documentation Structure | accepted | establishes docs layers |
| [0002](./0002-local-learning-workbench-structure.md) | Local Learning Workbench Structure | accepted, partially superseded | folder-first and visible CLI foundations retained; fixed zones, nested subprojects, and memory superseded |
| [0003](./0003-backend-session-and-project-runtime.md) | Backend Session And Project Runtime | accepted, partially superseded | file-first runtime and run separation retained; memory and route-owned orchestration superseded |
| [0004](./0004-learning-artifact-protocols.md) | Learning Artifact Protocols | accepted, compatibility contribution | legacy artifact readers and migration inputs retained; not the new assistant output contract |
| [0005](./0005-discipline-map-and-system-learning-project-types.md) | Discipline Map And System Learning Project Types | accepted, partially superseded | project types retained; physical subprojects removed by 0007; folder-title navigation replaced by 0009 |
| [0007](./0007-flat-project-storage-and-sidebar-classification.md) | Flat Project Storage And Sidebar Classification | accepted, partially superseded | flat storage retained; folder-title navigation replaced by 0009 |
| [0008](./0008-generated-prerequisite-gap-summaries.md) | Generated Prerequisite Gap Summaries | accepted, compatibility contribution | legacy prerequisite readers retained; active teaching moved to 0012 |
| [0009](./0009-explicit-discipline-overview-entry.md) | Explicit Discipline Overview Entry | accepted | supersedes only the clickable-folder-title presentation in 0005 and 0007 |
| [0010](./0010-hierarchical-discipline-maps-and-learning-plans.md) | Hierarchical Discipline Maps And Learner-Owned Task Plans | accepted | supersedes the flat H3-only overview contract in 0005 |
| [0011](./0011-learning-scope-snapshots-and-intro-calibration.md) | Learning Scope Snapshots And Intro Calibration | accepted | extends 0005 and 0010 with durable topic boundaries |
| [0012](./0012-teacher-assistant-learning-workspace.md) | Teacher-Assistant Learning Workspace | accepted | supersedes the active five-zone presentation and direct-generation workflow while retaining maps, flat storage, scope provenance, and visible native CLI execution |
| [0013](./0013-stable-stream-rendering-and-global-preferences.md) | Stable Stream Rendering And Global Learner Preferences | accepted | extends 0012 rendering and supersedes project/learner memory growth in 0002 and 0003 |
| [0014](./0014-current-documentation-surface-and-archive-boundary.md) | Current Documentation Surface And Archive Boundary | accepted | keeps only current decisions and delivery in the default AI reading surface |
| [0015](./0015-business-modular-monolith-boundaries.md) | Business-Capability Modular Monolith Boundaries | accepted; implementation planned | defines enforceable capability modules, selective hexagonal responsibilities, centralized configuration, and architecture quality gates for iteration 14 |
| [0016](./0016-retire-legacy-summary-extend-and-knowledge-garden.md) | Retire Legacy Summary, Extend, and Knowledge Garden | accepted | removes product code and public interfaces while preserving historical project files |
| [0017](./0017-conversation-learning-sources-consolidation-and-usage.md) | Conversation Learning Sources, Consolidation, And Usage | accepted | extends 0012 and 0016 with canonical source citations, teaching outlines, approved consolidation, and teacher usage |

## Historical Decisions

ADR-0006 is fully superseded and preserved in the
[historical ADR index](../../99-archive/adr/README.md). Partially superseded ADRs
remain above because they still explain capabilities reused by the current architecture.

## Numbering Repair

`Learning Artifact Protocols` originally duplicated number 0003. On 2026-07-14
its filename and heading were corrected to 0004 without changing the decision.
