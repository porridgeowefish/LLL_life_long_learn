# ADR-0014: Current Documentation Surface And Archive Boundary

Status: accepted
Owner: project maintainer
Date: 2026-09-02
Last reviewed: 2026-09-02
Source of truth: durable boundary between current documentation and historical evidence.
Extends: ADR-0001

## Context

The default documentation surface accumulated every delivery slice, retired
HTML alignment artifact, superseded ADR, and an orphaned design directory. Some
historical files still labelled themselves `active` or `proposed`. A person
could reconstruct chronology, but an AI reading by filename or broad search
could mistake an old five-zone, memory, or agent workflow for a current rule.

Deleting history would remove useful migration and regression evidence. Keeping
all history beside current contracts makes the present architecture needlessly
large and ambiguous.

## Decision

Use a strict current/history boundary:

```text
docs/00-product-and-architecture/   active long-lived facts and active ADR index
docs/01-iterations/                 one current baseline plus indexed delivered foundations
docs/99-archive/                    fully retired ADRs, designs, roadmaps, and references
```

`AGENTS.md` and `docs/INDEX.md` route ordinary work through active architecture
and the current iteration. Earlier delivered slices remain under a foundation
index and are read selectively when their retained capability is relevant. AI
agents may read the archive only for an explicit history, migration,
compatibility, or regression task.

An ADR moves to the historical index only when it is fully superseded. Partially
effective ADRs remain in the active decision graph. A delivered iteration moves
to `foundations/` when its implementation remains part of the product, and to
the archive only when it has no active contribution.

## Consequences

- Default context is smaller and has one unambiguous current delivery baseline.
- Old decisions remain auditable without silently directing new work.
- Broad repository search still finds history, so scoped `AGENTS.md` rules and
  explicit archive warnings remain necessary.
- Moving a document requires link validation and an index update in the same change.

## Migration

- Iterations 01–12 moved to `docs/01-iterations/foundations/` and received an
  explicit contribution-to-current-architecture map.
- Partially effective ADRs 0002, 0003, 0004, and 0008 remain in the active ADR
  graph. Only fully superseded ADR-0006 moved to `docs/99-archive/adr/`.
- Legacy HTML alignment files moved to `docs/99-archive/references/`.
- The orphaned `docs/superpowers/` design moved to `docs/99-archive/designs/`.
- The redundant `MVP_ROADMAP.md` moved to `docs/99-archive/roadmaps/`; the
  active `ROADMAP.md` is the only current delivery summary.
- The mixed zone/session-era backend design moved to `docs/99-archive/designs/`;
  the active backend architecture now describes the implemented service layers.
