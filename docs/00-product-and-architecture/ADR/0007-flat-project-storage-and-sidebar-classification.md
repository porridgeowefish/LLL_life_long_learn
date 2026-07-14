# ADR-0007: Flat Project Storage And Map-backed Sidebar Folders

Status: accepted  
Owner: project maintainer  
Last reviewed: 2026-07-14  
Source of truth: durable decision for flat projects, map-backed folders, and removal of the subproject domain model.

## Context

A deep dive started from a discipline map is behaviorally identical to a
system-learning project created elsewhere. Physical nesting or a durable parent
relation would create two kinds of otherwise identical learning project.

The frontend already owns sidebar folders. Creating a second navigation model
for maps made a map appear as an uncategorized child row beside ordinary
projects, even though the intended product shape was for “物理学” itself to be
the folder-level overview above mechanics, thermodynamics, and fluid mechanics.

## Decision

Every real project is stored flat at:

```text
projects/<globally-unique-slug>/
```

There is no `Subproject` domain object, no new `subprojects/` directory, and no
durable map-parent field. Starting a deep dive from a map calls the same project
creation API and creates the same `system-learning` shape as direct creation.

Sidebar folders remain the single classification structure. A folder may bind
one discipline-map project through `mapProjectSlug`; that project's overview is
opened from the folder title and is never rendered as a child row. Creating a
map creates a folder or promotes a same-name existing folder. Confirmed
system-learning projects use the existing folder membership and move interaction.

The binding identifies the folder overview only. It is not a parent relation:
folder placement does not affect prompts, tokens, project files, or learning
behavior, and all project directories remain physically flat.

The pre-release nested test data is deleted. The runtime no longer discovers or
writes nested projects.

This supersedes the nested-project parts of ADR-0002, ADR-0005, and ADR-0006.

## Consequences

- Map origin for a learning project is transient; only ordinary folder membership is persisted.
- Discipline maps reuse folder headers; system-learning projects reuse child rows and the existing move control.
- No map maintains child IDs and no index synthesizes parent-child hierarchy.
- Token use depends on the learning project and invoked agent, never its folder.
- Removing a stale map binding leaves the folder and its learner-owned membership intact.

## Rejected Alternatives

- Persist a `parentMapId` solely to reproduce a second sidebar hierarchy.
- Disable folder movement for projects created from a map.
- Render a discipline map as an uncategorized project row while a separate folder structure exists above it.
