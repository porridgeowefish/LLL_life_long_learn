# Documentation Standard

Status: active  
Owner: project maintainer  
Last reviewed: 2026-06-04  
Source of truth: this document defines documentation rules for LLL.

## Core Principle

LLL uses:

```text
docs-as-code + iteration-driven delivery + single-source-of-truth discipline
```

Code and docs live in the same repository. Scope, API behavior, architecture, and delivery status must evolve together.

## Single Source Of Truth

Allowed:

```text
entry documents that index
summary documents that point to detailed owners
detailed documents that own the fact
```

Not allowed:

```text
copying the same long contract into README, AGENTS, iteration docs, and chats
parallel v1/v2 markdown files for the same fact
using chat history instead of ADR or iteration docs
```

## Documentation Layers

```text
docs/00-product-and-architecture/
Long-lived product and architecture facts.

docs/01-iterations/
Current delivery slices.

docs/99-archive/
Historical references only.
```

## Iteration Minimum

Each implementation iteration should maintain:

```text
README.md
USER_STORIES.md
API_CONTRACT.md
DATABASE_DESIGN.md
TEST_PLAN.md
ACCEPTANCE_CRITERIA.md
DELIVERY_NOTES.md
```
