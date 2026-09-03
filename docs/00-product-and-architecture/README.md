# Product And Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: `docs/INDEX.md` routes readers here; each linked document owns its topic.

This directory stores long-lived documents that answer:

```text
Why LLL exists
What problem it solves
Where the domain boundaries sit
Why the architecture is shaped this way
Which rules guide implementation and delivery
```

## Core Documents

- [PRD.md](./PRD.md)
- [REPOSITORY_MAP.md](./REPOSITORY_MAP.md)
- [SYSTEM_ARCHITECTURE.md](./SYSTEM_ARCHITECTURE.md)
- [DOMAIN_MODEL.md](./DOMAIN_MODEL.md)
- [DATA_MODEL.md](./DATA_MODEL.md)
- [BACKEND_ARCHITECTURE.md](./BACKEND_ARCHITECTURE.md)
- [AGENT_ARCHITECTURE.md](./AGENT_ARCHITECTURE.md)
- [LEARNING_PROJECT_STRUCTURE.md](./LEARNING_PROJECT_STRUCTURE.md)
- [API_CONTRACT_STRATEGY.md](./API_CONTRACT_STRATEGY.md)
- [ROADMAP.md](./ROADMAP.md)
- [DOCUMENTATION_STANDARD.md](./DOCUMENTATION_STANDARD.md)
- [agent-rules/README.md](./agent-rules/README.md)
- [ADR index](./ADR/README.md)

## Supporting Current Documents

This file is useful for active collaboration, but does not override the core
contracts above:

- [ALIGNMENT_MODE.md](./ALIGNMENT_MODE.md)

Retired alignment artifacts are indexed under
[`docs/99-archive/references/`](../99-archive/references/README.md) and are not
part of the default reading path.

## Relationship To Iteration Docs

Long-lived docs define direction and boundaries.
Iteration docs define current delivery scope.

If they conflict:

```text
Current implementation follows docs/01-iterations/ for scope.
Long-term changes must be captured in ADR or updated architecture docs.
```

The active architectural baseline is
[ADR-0012](./ADR/0012-teacher-assistant-learning-workspace.md): one API teacher
conversation per system-learning unit, asynchronous visible-CLI assistance,
versioned teaching assets, and explicit source material. It is extended by
[ADR-0013](./ADR/0013-stable-stream-rendering-and-global-preferences.md), which
defines stable rich-stream rendering and one learner-owned global preferences file.
[ADR-0014](./ADR/0014-current-documentation-surface-and-archive-boundary.md)
defines which documents belong to the current AI context and which are historical only.
[ADR-0015](./ADR/0015-business-modular-monolith-boundaries.md) defines the
approved iteration-14 target: business-capability modules with private
implementations, selective hexagonal boundaries, one composition root, one
typed configuration system, and executable architecture checks. It is a target
architecture until iteration 14 delivery is complete; iteration-13 code remains
the implemented runtime truth during migration.
