# Iteration Delivery

Status: active
Owner: project maintainer
Last reviewed: 2026-10-10
Source of truth: current implemented baseline, active planning slice, and routing index for delivered foundations.

Each iteration should be independently understandable and testable.

## Current Baseline And Active Delivery

| Iteration | Goal | Directory | State |
|---|---|---|---|
| 20 | native Windows application, MSI install/upgrade and unified icon | [iteration-20-native-desktop-msi](./iteration-20-native-desktop-msi/README.md) | implementing; native validation in progress |
| 19 | local document import, automatic source context, notes scrolling and installers | [iteration-19-document-import-installers](./iteration-19-document-import-installers/README.md) | implemented; Windows installation verified; native macOS/Linux CI pending |


Iterations 01–13 and 15–18 are retained under [foundations](./foundations/README.md).
Iteration 14 remains the architecture baseline. Select a foundation only when its
retained capability is relevant; current code and iteration 19 own present behavior.

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
