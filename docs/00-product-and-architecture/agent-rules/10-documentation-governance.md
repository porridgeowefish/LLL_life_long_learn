# Documentation Governance

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: operational routing and forward gates for LLL documentation changes.

## Entry

```text
AGENTS.md
docs/INDEX.md
docs/00-product-and-architecture/DOCUMENTATION_STANDARD.md
docs/00-product-and-architecture/agent-rules/README.md
```

## New Rule Or Fact Workflow

```text
1. classify as behavior rule, product fact, iteration contract, or historical record
2. search the repository for the existing owner and conflicts
3. resolve conflicts using docs/INDEX.md fact priority
4. update the owning file rather than creating a duplicate
5. create an ADR for durable architecture decisions
6. use the landing table to synchronize affected long-lived facts
7. update indexes and Last reviewed dates
8. verify links, required documents, and diff formatting
```

## Architecture Change Landing Table

| Change type | ADR | Required long-lived sync |
|---|---:|---|
| Product shape, project type, or primary user journey | yes | `PRD.md`, `DOMAIN_MODEL.md`, `LEARNING_PROJECT_STRUCTURE.md`, `MVP_ROADMAP.md` |
| Domain object or boundary | yes | `DOMAIN_MODEL.md`, `DATA_MODEL.md`, relevant structure document |
| Runtime component or critical workflow | yes | `SYSTEM_ARCHITECTURE.md`, `BACKEND_ARCHITECTURE.md` or `AGENT_ARCHITECTURE.md` as applicable |
| Persisted file/schema or database boundary | yes | `DATA_MODEL.md`, iteration data contract, migration readers/writers |
| Public API/interface compatibility | when durable/strategic | `API_CONTRACT_STRATEGY.md`, iteration contract, client/server types |
| Security, privacy, permission, or migration policy | yes | affected architecture document plus task-safety rule when operational |
| Iteration renumbering or supersession | no, unless scope changes | `docs/01-iterations/README.md`, `MVP_ROADMAP.md`, cross-references, delivery notes |
| ADR creation or renumber repair | no extra ADR | `ADR/README.md`, product-and-architecture README |

Rules:

```text
The table is additive: satisfy every matching row.
An ADR alone does not complete the change.
An iteration README must list which long-lived documents were synchronized.
If implementation is pending, long-lived docs must label target versus current reality.
```

## Iteration Gate

Before marking iteration planning complete:

```text
confirm core five + one contract + one data document
use DATA_DESIGN for file-first schemas; DATABASE_DESIGN only for an actual database
fill Documentation Impact in the iteration README
record unresolved implementation decisions in delivery notes, not formal contracts
```

Before marking delivery complete, additionally record verification commands,
results, and residual risks.
