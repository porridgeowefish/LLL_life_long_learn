# Iteration Delivery

Status: active
Owner: project maintainer
Last reviewed: 2026-09-03
Source of truth: current implemented baseline, active planning slice, and routing index for delivered foundations.

Each iteration should be independently understandable and testable.

## Current Baseline

| Iteration | Goal | Directory |
|---|---|---|
| 14 | Enforced business-capability modular monolith with unified configuration and quality gates | [iteration-14-modular-monolith-architecture](./iteration-14-modular-monolith-architecture/README.md) |

## Active Delivery Slice

| Iteration | Goal | Directory | State |
|---|---|---|---|
| 14 | Reorganize LLL as an enforced business-capability modular monolith with unified configuration and quality gates | [iteration-14-modular-monolith-architecture](./iteration-14-modular-monolith-architecture/README.md) | delivered; native browser acceptance remains user-run |

Iteration 13 is retained under `foundations/`; iteration 14 is the current
implemented baseline.

Iterations 01–13 are retained as [delivered foundations](./foundations/README.md).
They explain reusable capabilities that still support the current architecture.
Read only the slice relevant to the task; active architecture and code decide
how that capability participates in the current teacher workspace.

## Required Files

```text
README.md
USER_STORIES.md
ACCEPTANCE_CRITERIA.md
TEST_PLAN.md
DELIVERY_NOTES.md

one contract:
API_CONTRACT.md / INTERFACE_CONTRACT.md / PIPELINE_CONTRACT.md

one data contract:
DATA_DESIGN.md for file-first/schema data
DATABASE_DESIGN.md only when a real database is part of the slice
```

When an iteration changes product shape, domain boundaries, runtime architecture,
or persistence, its README must include a `Documentation Impact` section listing
the ADR and synchronized long-lived fact sources. See the governance landing table.

When a newer iteration becomes the current baseline, move the previous delivery
directory into `docs/01-iterations/foundations/` if its implemented capability
is retained. Move it to `docs/99-archive/` only when it is fully abandoned or
replaced with no active contribution. Update the relevant index in the same change.
