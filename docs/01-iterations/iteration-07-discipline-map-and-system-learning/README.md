# Iteration 07: Discipline Map And System Learning

Status: delivered; automated acceptance repair verified
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: iteration 07 goal, scope, boundary, and documentation impact.

## Goal

```text
Let the learner explicitly choose a discipline map or system-learning project.
Provide optional low-cost AI choice advice without transferring the decision to AI.
Give discipline maps one coherent overview and system learning the existing five-zone workflow.
Reuse existing sidebar folders as the navigation model: a map is a folder overview, and confirmed deep dives are ordinary child rows by classification.
Keep prerequisite gaps inside the current learning flow instead of opening another project. Iteration 09 later replaces the on-demand bridge with generated summaries.
```

## Scope

### Included

```text
Explicit 学科地图 / 系统学习 choice during project creation.
Stateless, multi-turn AI choice advice with no creation or persistence side effect.
A discipline-map project with one 学科总览 and no five-zone navigation.
A registered encyclopedia agent launched explicitly through the selected native Agent CLI.
A system-learning project retaining Intro / Explain / Practice / Extend / Summary.
Word-style outline navigation derived from the agent-authored heading hierarchy.
Prefilled, learner-confirmed system-learning creation from an inline action beside each research-area heading.
Flat project creation that classifies a confirmed deep dive through the existing sidebar-folder move model.
Type-aware indexing and routing; discipline maps render as clickable folder overviews rather than project rows.
Explicit learner-triggered map updates; no automatic project creation from overview text.
Inline AI prerequisite bridges that explain only enough to continue the current topic.
```

### Excluded

```text
AI overriding the learner's product-type choice.
Folders, pages, jobs, or sidebar nodes for unselected map topics.
Branch-overview pages or a recursive virtual map browser.
Top-of-page topic card indexes or one generic deep-dive action detached from the concept being described.
Automatic project creation or automatic overview rewrites.
Changing project type in place.
Replacing the five-zone workflow inside system-learning projects.
Using project creation as the default response to a prerequisite gap.
Discipline protocols and local learning-lab execution.
```

## History

- [ADR-0005](../../00-product-and-architecture/ADR/0005-discipline-map-and-system-learning-project-types.md) owns the durable project-type decision.
- [ADR-0006](../../00-product-and-architecture/ADR/0006-inline-prerequisite-bridges.md) records the historical bridge decision.
- [ADR-0007](../../00-product-and-architecture/ADR/0007-flat-project-storage-and-sidebar-classification.md) owns flat storage and map-backed sidebar folders.
- [ADR-0008](../../00-product-and-architecture/ADR/0008-generated-prerequisite-gap-summaries.md) supersedes the bridge with generated summaries.
- [Learning Project Structure](../../00-product-and-architecture/LEARNING_PROJECT_STRUCTURE.md) owns the long-lived folder and navigation model.

## Delivery Documents

- [User stories](./USER_STORIES.md)
- [Acceptance criteria](./ACCEPTANCE_CRITERIA.md)
- [API contract](./API_CONTRACT.md)
- [Data design](./DATA_DESIGN.md)
- [Test plan](./TEST_PLAN.md)
- [Delivery notes](./DELIVERY_NOTES.md)

## Documentation Impact

Architecture decision:

- [ADR-0005: Discipline Map And System Learning Project Types](../../00-product-and-architecture/ADR/0005-discipline-map-and-system-learning-project-types.md)

Long-lived facts synchronized:

```text
PRD.md
DOMAIN_MODEL.md
LEARNING_PROJECT_STRUCTURE.md
SYSTEM_ARCHITECTURE.md
DATA_MODEL.md
BACKEND_ARCHITECTURE.md
AGENT_ARCHITECTURE.md
API_CONTRACT_STRATEGY.md
MVP_ROADMAP.md
DOCUMENTATION_STANDARD.md
agent-rules/10-documentation-governance.md
```

The delivered acceptance repair replaces the incorrect direct-provider overview generation
and top-of-page topic-card index with an explicit native Agent CLI session,
artifact-driven refresh, Word-style outline navigation, and inline deep-dive
actions beside the corresponding research-area headings.
