# Data Design

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28

## Discipline Topic Catalog

Location: `projects/<map-slug>/discipline-topics.json`

The schema-v1 catalog is Agent-generated, validated on read, and paired with
`overview.md`. Topic IDs are stable references; titles remain learner-facing.
`ownedConcepts` has global one-owner semantics within the catalog.

## Learning Scope

Location: `projects/<system-learning-slug>/learning-scope.json`

Fields:

```text
schemaVersion
status: draft | ready
title
chapterTitle
goal
inScope[]
outOfScope[]
prerequisites[]
ownedConcepts[]
reusedConcepts[]
source: standalone | {type: discipline-map, mapSlug, topicId}
updatedAt
```

A map-origin scope is copied at creation. The map catalog is provenance, not a
live foreign key. A standalone scope starts as `draft`; Intro may make it
`ready` after learner evidence exists.

## Ownership

```text
Encyclopedia -> map topic boundaries
Backend -> canonical lookup, validation, snapshot persistence
Intro -> learner adaptation; standalone draft finalization only
Explain / Practice / Extend / Summary -> consume scope
Learner -> explicit project creation and any future scope expansion decision
```
