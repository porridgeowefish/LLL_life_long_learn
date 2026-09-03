# Iteration 12: Learning Scope Contract

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28
Source of truth: iteration 12 scope, boundary ownership, and delivery status.

## Goal

Prevent sibling discipline-map topics from silently teaching each other's core
content while keeping map-origin and standalone projects on one system-learning
workflow.

## Included

- encyclopedia-generated `discipline-topics.json`;
- explicit goal, inclusion, exclusion, prerequisite, ownership, and reuse fields;
- one-owner validation for core concepts;
- `GET /api/projects/{id}/discipline-topics`;
- canonical map-topic lookup and creation-time `learning-scope.json` snapshot;
- standalone `draft` scope finalized by Intro after calibration;
- scope injection into every zone Agent prompt;
- explicit split between objective scope and learner adaptation;
- artifact refresh, frontend topic references, compatibility, and tests.

## Excluded

- automatic semantic scoring of generated Markdown against the scope;
- live propagation from a regenerated map into existing learning projects;
- recursive map/project ownership;
- allowing Intro to broaden map-origin scope;
- migration of legacy maps without regeneration.

## Documentation Impact

Architecture decision: [ADR-0011](../../../00-product-and-architecture/ADR/0011-learning-scope-snapshots-and-intro-calibration.md).

Long-lived facts synchronized:

```text
PRD.md
DOMAIN_MODEL.md
LEARNING_PROJECT_STRUCTURE.md
MVP_ROADMAP.md
DATA_MODEL.md
SYSTEM_ARCHITECTURE.md
BACKEND_ARCHITECTURE.md
AGENT_ARCHITECTURE.md
API_CONTRACT_STRATEGY.md
ADR/README.md
docs/01-iterations/README.md
```
