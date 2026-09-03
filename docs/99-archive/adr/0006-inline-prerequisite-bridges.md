# ADR-0006 Inline Prerequisite Bridges

Status: superseded by ADR-0008
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: durable decision for prerequisite-gap handling and project-creation boundaries.

Supersession note: ADR-0008 removes the on-demand bridge and replaces it with
generated prerequisite summaries. This record remains unchanged below as
historical context.

## Context

Iteration 04 represented a weak or missing prerequisite as a prefilled child
project that the learner could confirm and enter. That converted a momentary
learning-rhythm problem into a project-management task. A learner who is
already studying usually wants enough context to continue, not a second
project and a context switch.

Iteration 07 establishes a clearer threshold: a real project records a
deliberate, longer-lived learning commitment. A temporary prerequisite bridge
does not meet that threshold.

## Decision

When a learning flow detects a prerequisite gap, the default action is an
inline, concise AI bridge inside the current project. The bridge:

```text
explains only what is needed for the current topic
uses one small example where useful
offers one lightweight understanding check
returns the learner to the interrupted learning path
```

The product must not present child-project creation as the primary response to
a prerequisite gap. A separate system-learning project may still be created
when the learner explicitly chooses a sustained deep dive, including from a
discipline map.

`subprojects/` represents confirmed organizational containment of real
projects. It is not the default representation of prerequisite dependency.

This decision supersedes iteration 04 only where it made confirmed prerequisite
child-project creation the primary action. Evidence-based prerequisite
diagnosis remains valid.

## Consequences

- Intro keeps prerequisite assessment cards, impact, and evidence.
- Weak or missing cards offer `快速补充` and render the answer inline.
- The bridge has no project-creation side effect.
- Discipline-map deep dives continue to use learner-confirmed system-learning projects.
- A future dependency graph must use an explicit relation rather than folder parentage.

## Alternatives Considered

- Keep child-project confirmation as the primary action.
- Automatically create prerequisite projects.
- Remove prerequisite diagnosis entirely.
- Model every prerequisite as a nested folder or sidebar node.
