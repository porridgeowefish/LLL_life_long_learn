# ADR-0011: Learning Scope Snapshots And Intro Calibration

Status: accepted
Owner: project maintainer
Date: 2026-07-28
Last reviewed: 2026-07-28
Source of truth: durable topic-boundary ownership and Intro responsibility.
Extends: ADR-0005 and ADR-0010

## Context

An actionable map topic named A.1 did not carry a machine-readable boundary.
The later system-learning Agent inferred scope from the title and could teach
content owned by sibling A.2. Creating A.2 then produced repeated material.

Using a separate map-learning engine would fork behavior and leave standalone
system learning with a different contract. Letting Intro freely redefine scope
would also mix two questions: what the topic contains and how this learner
should study it.

## Decision

The encyclopedia Agent writes both the human overview and a structured topic
catalog. Every topic declares its goal, inclusion, exclusion, prerequisites,
owned concepts, and reused concepts. A core concept has one owner per map.

When a learner confirms a map deep dive, the client sends only the map slug and
topic ID. The backend resolves the canonical topic and copies it to the new
project's `learning-scope.json`. This is a creation-time snapshot, not a live
reference.

All system-learning projects use the same five zones and prompt assembly.
Standalone projects start with a draft scope. Intro may finalize that draft
after calibration. For a ready map-origin scope, Intro only calibrates learner
readiness, depth, examples, scaffolding, and practice difficulty; it cannot
broaden the objective boundary.

Every later Agent receives the same scope. Owned and included concepts may be
taught fully; prerequisites and reused concepts receive minimum support;
excluded concepts do not become core learning artifacts.

## Consequences

- Sibling boundaries are planned before any deep-dive project exists.
- A regenerated map cannot silently change active learning projects.
- Standalone and map-origin learning remain one runtime and one zone model.
- Legacy maps must regenerate before they can supply a scoped deep dive.
- Structural scope compliance is testable; semantic paragraph-level compliance
  remains an Agent-quality concern.

## Rejected Alternatives

- Infer every project boundary from its title at Explain time.
- Let Intro redefine map topics based on learner answers.
- Store a live map-topic foreign key and mutate child scope on regeneration.
- Create a separate map-specific learning workflow.
- Duplicate sibling concepts and rely on prose instructions alone.
