# Documentation Standard

Status: active
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: documentation structure, required contracts, and completion rules for LLL.

## Core Principle

```text
docs-as-code + iteration-driven delivery + single-source-of-truth discipline
```

Code and docs live together. Product shape, domain boundaries, interfaces, data,
architecture, tests, and delivery status change in the same task.

## Documentation Layers

```text
docs/00-product-and-architecture/   long-lived product, domain, architecture, ADR, governance
docs/01-iterations/                 one independently testable delivery increment
docs/99-archive/                    historical reference only
```

Entry documents index; detailed documents own facts. Do not copy one long
contract across AGENTS, README, iteration documents, and chat.

## Fact Priority

The canonical priority is defined once in `docs/INDEX.md`:

```text
implemented code and runnable schemas
current iteration contracts
long-lived architecture and accepted ADRs
archive
raw research and vision inputs
```

Proposed long-lived target documents must state when code has not implemented
the target yet.

## Required Front Matter

Every new or materially edited Markdown contract contains:

```text
Status
Owner
Last reviewed
Source of truth
```

Edit content, bump `Last reviewed`. Historical documents may be normalized when
next touched; do not invent review dates for untouched files.

## Architecture Impact Gate

Before completing an iteration plan or architecture task, classify whether it
changes:

```text
product shape
domain objects or boundaries
system/runtime architecture
data model or persistence
API/interface contracts
security, compatibility, or migration behavior
```

If any answer is yes:

```text
1. create or supersede an ADR
2. use the documentation-governance landing table
3. update every affected long-lived fact source in the same task
4. list the synchronized documents in the iteration README
5. update indexes and Last reviewed dates
```

An iteration document set is incomplete until this gate is satisfied. “Create
an ADR when architectural” without the landing-table sync is not sufficient.

## ADR Rules

```text
Use the next unused number from ADR/README.md.
Register the ADR in ADR/README.md in the same change.
Do not rewrite accepted history to hide a superseded decision.
State supersession explicitly in the new ADR and index.
```

## Iteration Document Set

Each delivery iteration contains core five plus one contract and one data file:

```text
README.md
USER_STORIES.md
ACCEPTANCE_CRITERIA.md
TEST_PLAN.md
DELIVERY_NOTES.md
API_CONTRACT.md or INTERFACE_CONTRACT.md or PIPELINE_CONTRACT.md
DATA_DESIGN.md for file/schema data, or DATABASE_DESIGN.md only for a real database
```

Delivered historical iterations may retain a legacy data-document filename until
that contract is materially revised; new file-first iterations use
`DATA_DESIGN.md`.

Recommended filling order:

```text
README -> USER_STORIES -> ACCEPTANCE -> CONTRACT -> DATA -> TEST_PLAN -> DELIVERY_NOTES
```

The iteration README must state target, included scope, excluded scope, delivery
status, and documentation-impact sync when long-lived facts change.

## Content Ownership

| Document | Owns | Does not own |
|---|---|---|
| User stories | user, behavior, value | fields, APIs, tests |
| Acceptance | black-box observable behavior | internal names and algorithms |
| Contract | interfaces, typed IO, errors | design rationale |
| Data design | typed structures, constraints, migration/sync points | product philosophy |
| Test plan | condition-to-result cases and commands | long design debate |
| Delivery notes | actual delivery, test results, residual risk, short-lived alignment | formal contract |
| ADR | durable architecture decision and trade-offs | daily progress |

## Completion Definition

An iteration is complete only when:

```text
user stories and black-box acceptance are aligned
implemented contracts and data docs match code or clearly remain proposed
test commands and results are recorded
delivery notes state what shipped and residual risk
architecture impact gate is satisfied
indexes and links are valid
```

## Supersession And Archive

Superseded delivery documents remain for traceability and point to the replacing
iteration. Archive content is never a fallback product contract.
