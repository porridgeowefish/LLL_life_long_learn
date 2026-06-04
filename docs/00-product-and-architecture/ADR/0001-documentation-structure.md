# ADR-0001 Documentation Structure

Status: accepted  
Date: 2026-06-04

## Context

LLL needs a documentation system that stays lightweight at the root, separates long-lived architecture from slice delivery, and remains readable by both humans and AI coding agents.

## Decision

Use:

```text
AGENTS.md and CLAUDE.md as thin root entry files
docs/INDEX.md as the global entry point
docs/00-product-and-architecture/ for long-lived facts
docs/01-iterations/ for delivery slices
docs/99-archive/ for historical material
```

## Consequences

- Root files stay short.
- Iteration scope can evolve without rewriting long-lived docs.
- AI agents can use progressive disclosure instead of loading everything.

## Alternatives Considered

- A single monolithic project handbook
- Flat docs with no iteration separation
