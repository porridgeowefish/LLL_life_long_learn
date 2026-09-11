# Iteration 16 — Conversation Learning Workflow

Status: delivered; browser E2E remains learner-run
Owner: project maintainer
Last reviewed: 2026-09-04

## Goal

Make the conversation-first learning unit complete: sources parse asynchronously
to one selectable Markdown reference, map-topic prose bounds the teacher, a
confirmed assistant consolidation writes Intro and Body (and optional Practice),
and teacher token usage is visible per learning conversation.

## Product Decisions

- The Sources page lists learner files and status only. It never previews
  derived parse content or assistant deliverables.
- A supported uploaded source starts background parsing after the existing
  cloud-processing disclosure is accepted. Its only retained derived output is
  `content.md`; files stay unavailable for citation until it is ready.
- The Teacher composer owns source citation. Its compact reference picker may
  preview parsed text and selects a source for the next message; selected file
  chips sit at the left of the composer.
- A map topic's narrative description becomes `teachingOutline` in the learning
  scope snapshot and is injected into teacher context as a teaching boundary.
- A confirmed `consolidate` task reads the sealed real conversation and selected
  sources. It rewrites `assets/intro` with significance/value and
  `assets/body` with a self-contained teaching manuscript ending in critical
  thinking. It updates `assets/practice` only when the confirmed teacher task
  explicitly requests questions.
- Token management is a separate page. It records only teacher-provider usage,
  aggregates it by the one durable conversation in each learning unit, and
  exposes paginated per-turn rows. Assistant token usage is intentionally not
  shown.
- The old Agent management page is removed; assistant tasks remain visible only
  inside the teacher conversation.

## Documentation Impact

This changes product journeys, source/task persistence, teacher runtime context,
and public API. The delivery adds ADR-0017 and synchronizes the product,
domain, data, backend, API, repository, roadmap, and ADR/iteration indexes.

## Contracts

- [Interface contract](./INTERFACE_CONTRACT.md)
- [Data design](./DATA_DESIGN.md)
- [Acceptance criteria](./ACCEPTANCE_CRITERIA.md)
- [Test plan](./TEST_PLAN.md)
- [Implementation plan](./plans/2026-09-04-conversation-learning-workflow.md)
