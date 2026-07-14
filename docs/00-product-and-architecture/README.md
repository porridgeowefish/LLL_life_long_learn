# Product And Architecture

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15
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
- [MVP_ROADMAP.md](./MVP_ROADMAP.md)
- [DOCUMENTATION_STANDARD.md](./DOCUMENTATION_STANDARD.md)
- [agent-rules/README.md](./agent-rules/README.md)
- [ADR index](./ADR/README.md)

## Reference Attachments

These files are useful alignment inputs, but not core delivery contracts:

- [ALIGNMENT_MODE.md](./ALIGNMENT_MODE.md)
- [WORKBENCH_ALIGNMENT_REVIEW.html](./WORKBENCH_ALIGNMENT_REVIEW.html) — legacy visual reference
- [LEARNING_AGENTS_USER_STORY_ALIGNMENT.html](./LEARNING_AGENTS_USER_STORY_ALIGNMENT.html) — legacy visual reference

## Relationship To Iteration Docs

Long-lived docs define direction and boundaries.
Iteration docs define current delivery scope.

If they conflict:

```text
Current implementation follows docs/01-iterations/ for scope.
Long-term changes must be captured in ADR or updated architecture docs.
```
