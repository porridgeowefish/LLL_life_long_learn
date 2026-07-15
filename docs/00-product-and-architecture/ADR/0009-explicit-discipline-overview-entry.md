# ADR-0009: Explicit Discipline Overview Entry

Status: accepted
Owner: project maintainer
Last reviewed: 2026-07-15
Source of truth: discipline-map placement inside sidebar folder navigation.

## Context

ADR-0005 and ADR-0007 bound a discipline map to a sidebar folder and presented
the folder title itself as the overview link. In use, the same label therefore
acted both as an expand/collapse control and as a document entry. Learners could
not tell whether clicking it would reveal children or open content.

## Decision

Keep the flat project model and `folders.json.mapProjectSlug` binding. Change
only its sidebar presentation:

```text
物理学                         folder title: expand/collapse only
├─ 物理学学科总览              explicit discipline-map overview row
├─ 流体力学                    movable system-learning row
└─ 热力学                      movable system-learning row
```

The overview row opens the existing discipline-map route. The folder title is
not a content link. Overview headings remain document content and never become
sidebar nodes automatically.

## Consequences

- Overview intent is visible before navigation.
- Folder expansion and document opening no longer compete on one target.
- The map project gains an ordinary project-actions affordance for deletion,
  but it still never enters `slugOrder` and is not owned by the folder.
- Existing folder payloads and project storage require no migration.

This supersedes only the clickable-folder-title presentation clauses in
ADR-0005 and ADR-0007; their project types, flat storage, and binding decisions
remain accepted.
