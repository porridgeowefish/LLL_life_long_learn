# Iteration Delivery

Status: active
Owner: project maintainer
Last reviewed: 2026-08-27
Source of truth: iteration subdirectories define active delivery scope.

Each iteration should be independently understandable and testable.

## Active Order

| Iteration | Goal | Directory |
|---|---|---|
| 13 | Teacher-assistant learning workspace, assets, and source materials (planned) | [iteration-13-teacher-assistant-learning-workspace](./iteration-13-teacher-assistant-learning-workspace/README.md) |
| 12 | Durable map-topic boundaries and unified learning scope (implemented) | [iteration-12-learning-scope-contract](./iteration-12-learning-scope-contract/README.md) |
| 11 | Hierarchical discipline maps and learner-owned task planning (implemented) | [iteration-11-hierarchical-discipline-planning](./iteration-11-hierarchical-discipline-planning/README.md) |
| 10 | Interaction feedback and usability cleanup (implemented) | [iteration-10-interaction-feedback-and-polish](./iteration-10-interaction-feedback-and-polish/README.md) |
| 09 | Learning deletion and prerequisite-summary cleanup (active) | [iteration-09-discipline-protocols-and-learning-lab](./iteration-09-discipline-protocols-and-learning-lab/README.md) |
| 08 | Learning rhythm, growth aggregation, themes, and open-source entry (implemented) | [iteration-08-learning-rhythm-and-personalization](./iteration-08-learning-rhythm-and-personalization/README.md) |
| 07 | Discipline map and system-learning product types (delivered) | [iteration-07-discipline-map-and-system-learning](./iteration-07-discipline-map-and-system-learning/README.md) |
| 06 | Ask-AI inline help and live run progress (proposed) | [iteration-06-ask-ai-and-live-progress](./iteration-06-ask-ai-and-live-progress/README.md) |
| 05 | Follow-up update mechanism (active) | [iteration-05-follow-up-update-mechanism](./iteration-05-follow-up-update-mechanism/README.md) |
| 04 | Three-agent learning experience refactor (implemented) | [iteration-04-three-agent-learning-experience](./iteration-04-three-agent-learning-experience/README.md) |
| 03 | Five agents and learning loops (draft, scoping) | [iteration-03-five-agents-and-learning-loops](./iteration-03-five-agents-and-learning-loops/README.md) |
| 02 | Local learning workbench foundation | [iteration-02-local-learning-workbench-foundation](./iteration-02-local-learning-workbench-foundation/README.md) |
| 01 | Core orchestrator foundation (superseded baseline) | [iteration-01-core-orchestrator-foundation](./iteration-01-core-orchestrator-foundation/README.md) |

## Required Files

```text
README.md
USER_STORIES.md
ACCEPTANCE_CRITERIA.md
TEST_PLAN.md
DELIVERY_NOTES.md

one contract:
API_CONTRACT.md / INTERFACE_CONTRACT.md / PIPELINE_CONTRACT.md

one data contract:
DATA_DESIGN.md for file-first/schema data
DATABASE_DESIGN.md only when a real database is part of the slice
```

When an iteration changes product shape, domain boundaries, runtime architecture,
or persistence, its README must include a `Documentation Impact` section listing
the ADR and synchronized long-lived fact sources. See the governance landing table.
