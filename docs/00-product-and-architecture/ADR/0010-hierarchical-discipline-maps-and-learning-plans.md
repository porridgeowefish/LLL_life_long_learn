# ADR-0010: Hierarchical Discipline Maps And Learner-Owned Task Plans

Status: accepted
Owner: project maintainer
Date: 2026-07-18
Last reviewed: 2026-07-18
Source of truth: durable discipline-map hierarchy, planning artifact, and compatibility decision.
Supersedes: ADR-0005 only where a discipline overview is treated as one flat H3 topic layer

## Context

The original discipline overview used H2 sections and one H3 layer of actionable
topics. Broad disciplines need stable major chapters plus finer learning units.
The same flat outline also gives the learner no way to select topics, decide an
order, and track one learning task at a time before creating deep-dive projects.

## Decision

The encyclopedia Agent plans one coherent knowledge architecture before writing
and produces one curated overview artifact:

```text
overview.md       H2 document sections, H3 major chapters, H4 learnable topics
```

The discipline-map page exposes `学科总览` and `学习计划` as top-level tabs.
Only H4 headings are actionable in the new contract. Existing overview files
without any H4 retain their H3 deep-dive actions, so regeneration is optional.

The encyclopedia Agent does not choose a learning order. The learner adds exact
H4 topic titles to `learning-plan.json`, and array order is the learner's chosen
order. Each task is `planned`, `in-progress`, or `completed`; completion is the
single-task check-in. The list does not create a system-learning project until
the learner separately confirms the existing deep-dive form.

## Consequences

- The table of contents supports H2/H3/H4 numbering.
- `learning-plan.json` is learner-owned task state with read/write endpoints and
  a filesystem refresh event.
- No recursive project, sidebar node, or map-parent relation is introduced.
- Reordering, starting, completing, reopening, and removing tasks update the
  ordered task list without asking AI to rewrite the overview.

## Rejected Alternatives

- Keep one flat H3 list and encode chapters in prose.
- Ask AI to choose a recommended order and generate a read-only plan.
- Append task state to the overview Markdown, mixing learner state with Agent output.
- Turn every planned topic into a project or sidebar item before confirmation.
