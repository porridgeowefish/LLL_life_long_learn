# Incident Report: Long-Term Documentation Was Not Synchronized

Status: resolved at governance level
Owner: project maintainer
Last reviewed: 2026-07-14
Source of truth: raw record and extracted rule for the iteration 07 documentation-sync failure.

## Symptom

Iteration 07 established two project types and removed the assumption that every
project owns five learning zones, but the first documentation pass updated only
the iteration directory and numbering. Long-lived PRD, domain, structure,
architecture, data, roadmap, and ADR sources still described one universal
five-zone project.

## Root Cause

The documentation system had layers and a fact-priority ladder, but its
operational governance rule contained only broad reminders:

```text
update the owning file
update indexes
create an ADR when architectural
```

It lacked:

```text
a change-type to long-lived-document landing table
a mandatory Documentation Impact section in iteration README files
an ADR index and next-number gate
a completion rule saying an ADR alone is insufficient
a project-type-aware data-document convention
```

As a result, “write the iteration contract” appeared complete before the agent
was forced to enumerate downstream fact owners.

## Contributing Conditions

- `DOCUMENTATION_STANDARD.md` was only a short principle summary.
- ADR files had no index and two decisions used number 0003.
- Iteration minimums always required `DATABASE_DESIGN.md` despite file-first persistence.
- Several long-lived documents were stale enough to describe a Node backend or mostly in-memory persistence.
- No iteration README field made cross-layer synchronization visible in review.

## Corrective Actions

- Added the architecture change landing table and iteration impact gate.
- Added `ADR/README.md`, repaired duplicate numbering, and recorded ADR-0005.
- Updated affected long-lived product and architecture sources.
- Changed the data-document rule to `DATA_DESIGN.md` for file-first schemas.
- Added a Documentation Impact section to iteration 07.
- Added front matter, typed data contracts, traceable stories/acceptance, and verification commands.

## Long-Term Rule

```text
For every product, domain, data, or architecture decision:
classify the change -> create/supersede ADR -> satisfy every landing-table row
-> list synchronized long-lived documents in the iteration README
-> verify indexes, links, dates, and required files.

The task is incomplete until all five steps are satisfied.
```

## Residual Documentation Debt

- Thirty-six legacy iteration Markdown files do not yet use all four canonical front-matter keys. They were not mass-rewritten because several belong to unrelated active dirty work; the new standard requires normalization when each contract is next materially touched.
- ADR-0004 still records the earlier append-oriented Explain follow-up consequence, while iteration 05 now owns an update-in-place policy. A separate retrospective ADR should record that supersession rather than rewriting ADR-0004 history.
- Legacy HTML alignment artifacts retain old product assumptions by design; repository governance already classifies them as reference-only rather than current facts.
