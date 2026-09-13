# Iteration 17 — Teacher Agent Refinements

Status: delivered; manual OCR/web-search smoke remains learner-run
Owner: project maintainer
Last reviewed: 2026-09-13

## Goal

Bring the teacher conversation to the interaction quality of a polished chat
agent: image sources become citable through a vision-model OCR step, learners
can type while the teacher streams (durable queue with edit and steering),
messages gain copy/regenerate/export affordances, the teacher can search the
public web mid-turn through a bounded tool loop, and the repository layout is
normalized so folders and scripts stop being misleading.

## Product Decisions

- OCR is implemented as a vision-model call, not a local engine. It reuses the
  `askAiProviders.bindings.ocr` binding and the existing cloud-disclosure gate
  at upload. Images with a configured OCR binding parse through this dedicated
  step and land in `derived/content.md`; without the binding the previous
  CLI-agent parse path is unchanged.
- Queued turns live in the conversation event log (`learner-queued` family).
  Editing, discarding, and promotion are events, so every tab and every reload
  sees the same queue. The server, not the page, drives auto-advance when a
  response finishes.
- Steering is explicit: a queued item's "steer now" action interrupts the
  active response (partial text is preserved as `interrupted`), promotes that
  item as a learner message with a steering marker, and immediately starts a
  new turn whose context explains the mid-generation guidance. Sending while
  streaming never silently discards work.
- Regeneration applies to the last teacher response only. The old response and
  message are superseded by an event (hidden from the projection, never
  deleted), and a fresh response runs for the same triggering learner message.
- Copy actions are reader-side: per-message markdown copy and a server-rendered
  `export.md` transcript download. Neither alters the event log.
- Web search is a teacher tool (`search_web`) backed by the Zhipu web-search
  API behind a `webSearch` config section. It is registered only when
  configured, needs no learner approval (read-only public web), and the
  provider loop is capped at two search rounds per turn. On the anthropic kind,
  thinking is disabled for turns that register tools because re-synthesized
  assistant messages cannot carry signed thinking blocks.
- Repository normalization: `agents/` is renamed to `learning-agents/` (one Go
  path constant changes), dev check scripts move from `scripts/` to
  `tools/check/`, manual QA scripts move to `tests/manual/`, and the runtime
  `folders.json` relocates under `projects/` with a one-time read-back
  migration. Public HTTP paths do not change.

## Documentation Impact

Delivered in the same task: ADR-0018 (conversation queue, steering,
regeneration), ADR-0019 (vision OCR, web search, repository normalization),
and ADR-0020 (bounded recovery of validated artifact-only promotion failures).
The latter synchronizes `DATA_MODEL.md`, `SYSTEM_ARCHITECTURE.md`,
`BACKEND_ARCHITECTURE.md`, the ADR index, and this iteration's data, API,
acceptance, test, and delivery records. Earlier delivery synchronized
`REPOSITORY_MAP.md`, `QUALITY_COMMANDS.md`, `AGENT_PRIMITIVES.md`,
`config/config.example.json`, and the iteration indexes (iteration 16 moved to
foundations).

## Contracts

- [API contract](./API_CONTRACT.md)
- [Data design](./DATA_DESIGN.md)
- [Acceptance criteria](./ACCEPTANCE_CRITERIA.md)
- [Test plan](./TEST_PLAN.md)
- [Implementation plan](./plans/2026-09-11-teacher-agent-refinements.md)
