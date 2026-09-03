# LLL Documentation Index

Status: active
Owner: project maintainer
Last reviewed: 2026-09-02
Source of truth: this file is the entry point for people and AI agents.

Read this file before implementation, then move to the linked domain documents.

## Documentation Layers

```text
00-product-and-architecture
Long-lived product, domain, architecture, data, and governance documents.

01-iterations
The current delivery baseline plus an indexed set of delivered foundations.

99-archive
Fully retired material and deprecated notes. Excluded from default AI context.
```

## Fact Priority

When sources conflict, use:

```text
1. Implemented code and runnable behavior
2. Current iteration documents in docs/01-iterations/
3. Long-lived documents and effective ADRs in docs/00-product-and-architecture/
4. Task-relevant delivered foundations selected through docs/01-iterations/foundations/README.md
5. Explicitly requested historical evidence in docs/99-archive/
6. research/raw/ materials
```

Raw material informs the product, but does not by itself expand the current implementation scope.

## Default Reading Boundary

For ordinary implementation work, start with the current iteration and
task-specific long-lived documents. If the task touches an inherited capability,
use the foundation index to select its originating slice; do not load all twelve
iterations. Do not scan `docs/99-archive/` to discover requirements. Open it only
for an explicit history, migration, compatibility, or regression question.

## Required Entry Points

- Agent entry: [AGENTS.md](../AGENTS.md)
- Documentation standard: [DOCUMENTATION_STANDARD.md](./00-product-and-architecture/DOCUMENTATION_STANDARD.md)
- Repository map: [REPOSITORY_MAP.md](./00-product-and-architecture/REPOSITORY_MAP.md)
- Product and architecture overview: [00-product-and-architecture/README.md](./00-product-and-architecture/README.md)
- Iteration overview: [01-iterations/README.md](./01-iterations/README.md)
- Historical index (read only when needed): [99-archive/README.md](./99-archive/README.md)

## Update Timing

```text
Before a new slice: define iteration README, user stories, API contract, and test plan.
When APIs change: update backend behavior and iteration API docs in the same task.
When architecture changes: add or update an ADR.
After delivery: update delivery notes and acceptance status.
```

Architecture completion additionally requires the forward gate in
[`agent-rules/10-documentation-governance.md`](./00-product-and-architecture/agent-rules/10-documentation-governance.md): classify the change, create or supersede an ADR, and synchronize every long-lived landing document in the same task.
