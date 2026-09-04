# Iteration 15 — Retire Legacy Summary, Extend, and Knowledge Garden

Status: delivered
Owner: project maintainer
Last reviewed: 2026-09-04

## Goal

Remove the inactive learner-facing Summary and Extend zones, including the
Knowledge Garden, while preserving existing project files as historical local
evidence. The active system-learning surface remains `教师 / 资产 / 资料`.

## Scope

- Remove retired Summary and Extend frontend code, routes, public APIs,
  registered roles, charters, primitives, and tests.
- Remove the unused Knowledge Garden (`extend/flower.json`) implementation.
- Stop new migration code from treating `summary/` and `extend/` as active
  compatibility inputs.
- Preserve every existing project's `summary/`, `extend/`, and migration backup
  directories without moving, editing, or deleting their files.

## Non-goals

- Do not remove provider reasoning summaries, Ask-AI annotation summaries,
  task completion summaries, or Explain's final critical-thinking page.
- Do not define a new special Summary zone, `summary.md` path, or summary
  generation template. A future user-authorized material may be a generic
  generated artifact in Assets.
- Do not retire Intro, Explain, Practice, the internal encyclopedia launcher,
  or project history unrelated to the retired zones.

## Documentation Impact

This is a product-surface, domain-boundary, migration-policy, and public API
change. The implementation must add ADR-0016 and synchronize:

- `PRD.md`
- `DOMAIN_MODEL.md`
- `LEARNING_PROJECT_STRUCTURE.md`
- `ROADMAP.md`
- `DATA_MODEL.md`
- `SYSTEM_ARCHITECTURE.md`
- `BACKEND_ARCHITECTURE.md`
- `AGENT_ARCHITECTURE.md`
- `API_CONTRACT_STRATEGY.md`
- `REPOSITORY_MAP.md`
- `AGENT_PRIMITIVES.md`
- ADR index and iteration index

The detailed executable plan is at
[`plans/2026-09-04-retire-legacy-summary-extend.md`](./plans/2026-09-04-retire-legacy-summary-extend.md).
