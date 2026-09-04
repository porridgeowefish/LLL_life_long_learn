# ADR-0016: Retire Legacy Summary, Extend, and Knowledge Garden

Status: accepted

## Context

The conversation-first learning workspace superseded the old staged learning
experience, but its Summary and Extend implementations still exposed flashcard
APIs, zone types, agents, project scaffolding, file access, and an unmounted
Knowledge Garden. Those surfaces implied that the retired stages were active
and allowed new runtime behavior to accumulate around obsolete artifacts.

Existing learner projects may nevertheless contain `summary/**` and `extend/**`
files that are historical evidence and must not be deleted or rewritten.

## Decision

Retire Summary, Extend, and Knowledge Garden from the active product:

- keep only Intro, Explain, and Practice in legacy compatibility zone types and
  new project scaffolding;
- remove their frontend pages, hooks, unmounted greenhouse, APIs, roles,
  charters, primitives, watchers, and specialized write behavior;
- make historical Extend/Summary URLs fall back to Explain rather than render a
  retired UI;
- reject generic file API reads and writes for `summary/**` and `extend/**`;
- preserve old on-disk directories exactly, while excluding them from new
  migration inventory and backup generation.

This decision does not affect provider reasoning summaries, annotation
summaries, task-result summaries, or Explain critical-summary pages.

## Consequences

New projects no longer create Summary/Extend folders and no active runtime can
generate or consume their old contracts. Historical projects remain recoverable
from disk or an existing migration backup, but those files are not prompt
context or active assets. A future assistant material is a generic Assets
artifact, not a revived summary zone.

## Supersedes

This supersedes the Summary/Extend/Knowledge Garden portions of ADR-0002,
ADR-0003, and ADR-0004. Their retained file-first and compatibility concepts
remain in force.
