# LLL Documentation Index

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: this file is the entry point for people and AI agents.

Read this file before implementation, then move to the linked domain documents.

## Documentation Layers

```text
00-product-and-architecture
Long-lived product, domain, architecture, data, and governance documents.

01-iterations
Delivery documents for each implementation slice. Each slice must be independently testable.

99-archive
Historical material and deprecated notes. Not a current implementation contract.
```

## Fact Priority

When sources conflict, use:

```text
1. Implemented code and runnable behavior
2. Current iteration documents in docs/01-iterations/
3. Long-lived documents in docs/00-product-and-architecture/
4. docs/99-archive/
5. research/raw/ materials
```

Raw material informs the product, but does not by itself expand the current implementation scope.

## Required Entry Points

- Agent entry: [AGENTS.md](../AGENTS.md)
- Documentation standard: [DOCUMENTATION_STANDARD.md](./00-product-and-architecture/DOCUMENTATION_STANDARD.md)
- Product and architecture overview: [00-product-and-architecture/README.md](./00-product-and-architecture/README.md)
- Iteration overview: [01-iterations/README.md](./01-iterations/README.md)
- Archive overview: [99-archive/README.md](./99-archive/README.md)

## Update Timing

```text
Before a new slice: define iteration README, user stories, API contract, and test plan.
When APIs change: update backend behavior and iteration API docs in the same task.
When architecture changes: add or update an ADR.
After delivery: update delivery notes and acceptance status.
```
