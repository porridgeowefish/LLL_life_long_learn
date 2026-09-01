# Iteration 11 Interface Contract

Status: active
Owner: project maintainer
Last reviewed: 2026-07-18
Source of truth: iteration 11 HTTP and heading-interface delta.

## Read Task Plan

`GET /api/projects/{id}/learning-plan`

Success `200` returns the schema in `DATA_DESIGN.md`. An older map without the
file returns an empty `items` array.

## Replace Task Plan

`PUT /api/projects/{id}/learning-plan` atomically replaces the ordered list.
The server validates at most 200 unique tasks, non-empty IDs and topic titles,
and status values `planned | in-progress | completed`. Array order is preserved.
Each completed item records one idempotent `learning-task-complete` activity
event keyed by task ID and `completedAt`; resaving the same completion does not
duplicate the check-in.

## Generate Overview

`POST /api/projects/{id}/discipline-overview/generate` keeps its response shape
and records only `overview.md`. The encyclopedia Agent does not write task state.

## Heading Compatibility

```text
new contract: H2 section -> H3 major chapter -> H4 actionable topic
legacy fallback: when no H4 exists, H3 remains actionable
```
