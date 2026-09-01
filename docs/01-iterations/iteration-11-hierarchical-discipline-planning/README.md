# Iteration 11: Hierarchical Discipline Planning

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-18
Source of truth: iteration 11 scope and delivery boundary.

## Goal

Give broad disciplines a real chapter/topic hierarchy, then let the learner
choose topics, own their order, and check off one learning task at a time.

## Included

- H3 major chapters with H4 learnable topics;
- H2/H3/H4 Word-style outline and H4 actions;
- legacy H3-only action compatibility;
- learner-owned ordered task list in `learning-plan.json`;
- overview and learning-plan tabs on the same discipline map;
- add, reorder, start, complete, reopen, remove, and launch-system-learning actions;
- independent API reads/writes and artifact refresh for the task list.

## Excluded

- AI-generated or AI-selected learning order;
- automatic project creation from a task;
- calendar scheduling, recurring daily check-ins, or spaced repetition;
- recursive sidebar nodes or persisted topic objects;
- migration or forced regeneration of existing maps.

## Documentation Impact

Architecture decision: [ADR-0010](../../00-product-and-architecture/ADR/0010-hierarchical-discipline-maps-and-learning-plans.md).

Long-lived facts synchronized:

```text
PRD.md
DOMAIN_MODEL.md
LEARNING_PROJECT_STRUCTURE.md
AGENT_ARCHITECTURE.md
DATA_MODEL.md
SYSTEM_ARCHITECTURE.md
BACKEND_ARCHITECTURE.md
API_CONTRACT_STRATEGY.md
MVP_ROADMAP.md
ADR/README.md
docs/01-iterations/README.md
```
