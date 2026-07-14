# ADR-0005 Discipline Map And System Learning Project Types

Status: accepted
Date: 2026-07-14
Last reviewed: 2026-07-14
Supersedes: ADR-0002 only where it requires five fixed zones for every project

## Context

ADR-0002 established the local folder-first workbench and a five-zone learning
workflow. That model works for a focused topic such as a theorem or technique,
but it forces broad disciplines into the same instructional sequence.

User feedback exposed two distinct jobs:

```text
orient inside a broad discipline before choosing where to invest
build capability through a committed, staged learning workflow
```

An AI classifier cannot reliably own this product decision from limited context.
Pre-generating folders for every area named in a map would also make the sidebar
represent AI possibilities rather than learner commitments.

## Decision

Adopt two immutable project types:

```text
discipline-map
system-learning
```

The learner explicitly chooses the type. An optional, stateless AI advisor may
hold a short clarification conversation, recommend a type, and explain the
trade-off, but must not persist the conversation, create a project, or silently
select a type.

A discipline map owns one learner-facing `学科总览` and no five-zone navigation.
Topics named in that overview are content only. They become real projects only
after learner confirmation.

The project brief follows the same boundary. A discipline-map `project.md`
contains map purpose and optional scope notes, never a current/target ability
ladder, completion gate, `Active Phase`, or five-zone name. Those fields belong
only to system-learning projects.

In navigation, the discipline map is the overview of a sidebar folder, not a
project row inside that folder. The folder uses the same membership and
expand/collapse model as existing folders. Confirmed deep dives are ordinary
system-learning projects classified inside it.

A system-learning project owns the existing Intro, Explain, Practice, Extend,
and Summary workflow. Its storage remains flat; sidebar folder membership is
classification rather than project ownership.

The sidebar renders actual indexed projects and persisted folder membership,
not headings parsed from overview Markdown. Existing projects without type
metadata decode as `system-learning`.

## Consequences

- The five-zone workflow remains scoped to system-learning projects.
- Project creation, indexing, routing, state, and frontend navigation are type-aware.
- Discipline maps use a dedicated single-overview generation/update contract.
- Project brief generation is type-aware; clients cannot inject system-learning
  context into a discipline map.
- A discipline map binds its overview slug to one existing sidebar folder shape.
- Confirmed deep dives reuse ordinary flat project creation and folder membership.
- Project type cannot be mutated in place during iteration 07.

## Alternatives Considered

- Let AI infer map versus system learning without explicit learner choice.
- Keep one universal five-zone project and make the overview page longer.
- Pre-create a recursive folder and overview page for every map branch.
- Treat map branches as virtual sidebar nodes before the learner creates projects.
- Render a discipline map as an uncategorized project row beside an unrelated folder system.
