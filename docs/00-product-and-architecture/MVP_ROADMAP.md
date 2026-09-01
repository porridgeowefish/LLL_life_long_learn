# MVP Roadmap

Status: active
Owner: project maintainer
Last reviewed: 2026-08-30
Source of truth: high-level delivery order and status summary; iteration directories own detailed scope.

## Delivery Order

| Iteration | Goal | Status |
|---|---|---|
| 01 | Core orchestrator foundation | superseded baseline |
| 02 | Local learning workbench foundation | delivered foundation |
| 03 | Five agents and learning loops | draft / partially aligned |
| 04 | Intro, Explain, and Practice refactor | implemented |
| 05 | Follow-up updates improve existing artifacts | active |
| 06 | Ask-AI inline help and live run progress | proposed |
| 07 | Discipline-map/system-learning types; prerequisite bridge later superseded | delivered |
| 08 | Learning rhythm, personalization, themes, and open-source entry | implemented |
| 09 | Learning deletion and generated prerequisite-gap summaries | implemented |
| 10 | Interaction feedback, greenhouse guidance, and contribution integrity | implemented |
| 11 | Hierarchical discipline maps and learner-owned task planning | implemented |
| 12 | Durable topic boundaries and unified learning-scope snapshots | implemented |
| 13 | Conversation-first API teacher, asynchronous CLI assistant, assets, and sources | planned |

Detailed scope, contracts, tests, and delivery evidence live under
`docs/01-iterations/` and override this summary when they differ.

## Current Product Transition

Iteration 07 changes the project model from one universal five-zone shape to:

```text
discipline map      -> one explicit overview row in its folder, may start ordinary learning projects classified inside that folder
system learning     -> five-zone learning workflow
```

Iteration 08 remains an already delivered slice despite its later number.
Iterations 09 and 10 are not delivery contracts.

Iteration 12 keeps one system-learning engine while adding an objective
`learning-scope.json`: maps preplan and snapshot the boundary; standalone
projects start draft; Intro calibrates the learner without broadening map scope.

Iteration 13 keeps both project types and flat storage but replaces the active
system-learning five-zone presentation after migration. One API teacher
conversation forms a learning unit; approved heavy work runs asynchronously in
the visible native CLI; accumulated learning enters versioned assets; learner
files enter an explicit source library. The current five-zone experience stays
executable until the cutover gate passes.

## Definition Of MVP

The MVP is complete when a learner can keep repeatable local learning workflows,
choose between orientation and deep study, inspect AI execution, and retain
editable artifacts without manually switching among terminal, notes, and output
readers.
