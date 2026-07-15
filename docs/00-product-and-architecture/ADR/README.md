# Architecture Decision Records

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: ADR numbering, status, and supersession index for LLL.

## Rules

```text
Use the next unused four-digit number.
Register every new ADR here in the same change.
Accepted ADRs are historical records and are not rewritten to hide old decisions.
New decisions supersede earlier decisions explicitly.
Architecture work is incomplete until the governance landing table is satisfied.
```

## Index

| ADR | Title | Status | Relationship |
|---|---|---|---|
| [0001](./0001-documentation-structure.md) | Documentation Structure | accepted | establishes docs layers |
| [0002](./0002-local-learning-workbench-structure.md) | Local Learning Workbench Structure | accepted, partially superseded | universal five-zone rule narrowed by 0005; nested subprojects removed by 0007 |
| [0003](./0003-backend-session-and-project-runtime.md) | Backend Session And Project Runtime | accepted | runtime foundation |
| [0004](./0004-learning-artifact-protocols.md) | Learning Artifact Protocols | accepted | artifact schemas |
| [0005](./0005-discipline-map-and-system-learning-project-types.md) | Discipline Map And System Learning Project Types | accepted, partially superseded | project types retained; physical subprojects removed by 0007; folder-title navigation replaced by 0009 |
| [0006](./0006-inline-prerequisite-bridges.md) | Inline Prerequisite Bridges | superseded | on-demand bridge replaced by generated summaries in 0008 |
| [0007](./0007-flat-project-storage-and-sidebar-classification.md) | Flat Project Storage And Sidebar Classification | accepted, partially superseded | flat storage retained; folder-title navigation replaced by 0009 |
| [0008](./0008-generated-prerequisite-gap-summaries.md) | Generated Prerequisite Gap Summaries | accepted | supersedes the on-demand bridge in 0006 |
| [0009](./0009-explicit-discipline-overview-entry.md) | Explicit Discipline Overview Entry | accepted | supersedes only the clickable-folder-title presentation in 0005 and 0007 |

## Numbering Repair

`Learning Artifact Protocols` originally duplicated number 0003. On 2026-07-14
its filename and heading were corrected to 0004 without changing the decision.
