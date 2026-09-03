# Acceptance Criteria

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28

## Encyclopedia

- One `discipline-topics.json` topic exists for every actionable H4 heading.
- Every topic has a stable ID, goal, non-empty `inScope`, non-empty
  `ownedConcepts`, explicit exclusions, prerequisites, and reused concepts.
- Two sibling topics cannot own the same normalized core concept.

## Project Creation

- A map deep dive submits only map slug and topic ID as its scope source.
- The backend reads the canonical map catalog and rejects unknown topics.
- The created project's `learning-scope.json` is `ready` and contains a copied
  snapshot; later map edits do not mutate it.
- A standalone system-learning project receives a `draft` standalone scope.

## Agent Behavior

- Every zone prompt embeds the current scope and enforcement rules.
- Intro preserves a ready map scope and only calibrates prerequisite readiness,
  depth, examples, scaffolding, and practice difficulty.
- Intro finalizes a standalone draft after survey answers are present.
- Explain treats exclusions as non-core and prerequisites/reused concepts as
  minimum support only.

## Compatibility

- Legacy projects without `learning-scope.json` remain invokable with a
  conservative fallback.
- Existing maps must regenerate before a map topic can create a scoped deep dive.
- Existing learning-plan items without `topicId` remain readable.
