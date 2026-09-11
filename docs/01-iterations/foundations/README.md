# Delivered Foundations

Status: active reference index
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: maps earlier delivery slices to capabilities reused by the current architecture.

These iterations are not discarded history. They record how reusable parts of
the current system were delivered. They are also not the primary current
contract: select one only when its capability is relevant, then confirm present
behavior in code and active architecture documents.

| Iteration | Retained contribution | Current landing area |
|---|---|---|
| [01](./iteration-01-core-orchestrator-foundation/README.md) | local task launch and observable execution baseline | `agentexecution`, launcher, run records |
| [02](./iteration-02-local-learning-workbench-foundation/README.md) | file-first projects, sessions, raw/curated separation | workspace/project/session compatibility stores |
| [03](./iteration-03-five-agents-and-learning-loops/README.md) | learning-role exploration and reusable prompt mechanisms | teacher soft methods, legacy charters, primitives |
| [04](./iteration-04-three-agent-learning-experience/README.md) | structured Intro/Explain/Practice readers and schemas | migration inputs, assets, Ask AI, practice compatibility |
| [05](./iteration-05-follow-up-update-mechanism/README.md) | questions improve accumulated content | body annotations and editable/versioned assets |
| [06](./iteration-06-ask-ai-and-live-progress/README.md) | Ask AI and live execution observation | annotation service, SSE/run progress, visible CLI |
| [07](./iteration-07-discipline-map-and-system-learning/README.md) | project types, flat storage, explicit map entry | workspace, folder store, discipline maps |
| [08](./iteration-08-learning-rhythm-and-personalization/README.md) | activity/rhythm, themes, contribution surfaces | progress store and appearance settings |
| [09](./iteration-09-discipline-protocols-and-learning-lab/README.md) | deletion safety and prerequisite-summary compatibility | project deletion and legacy Intro readers |
| [10](./iteration-10-interaction-feedback-and-polish/README.md) | navigation, confirmations, waiting/error feedback | current interaction conventions and compatibility UI |
| [11](./iteration-11-hierarchical-discipline-planning/README.md) | H3/H4 maps and learner-owned plans | overview/topic/learning-plan contracts |
| [12](./iteration-12-learning-scope-contract/README.md) | canonical topic boundaries and scope snapshots | learning-scope store and teacher/assistant context |
| [13](./iteration-13-teacher-assistant-learning-workspace/README.md) | teacher conversation, asynchronous assistance, assets, sources, and global preferences | current teacher/assistant/assets/sources/preferences modules |
| [16](./iteration-16-conversation-learning-workflow/README.md) | conversation sources, consolidation, teaching outlines, and teacher token usage | teacher conversation context, sources, usage store |

The current integrated product contract is [Iteration 14](../iteration-14-modular-monolith-architecture/README.md);
the current delivered product baseline is [Iteration 17](../iteration-17-teacher-agent-refinements/README.md).
