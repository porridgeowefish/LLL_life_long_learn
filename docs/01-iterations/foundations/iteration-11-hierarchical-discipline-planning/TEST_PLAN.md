# Iteration 11 Test Plan

Status: active
Owner: project maintainer
Last reviewed: 2026-07-18
Source of truth: iteration 11 verification plan.

## Automated

```text
agent registry verifies planning-first H3/H4 charter terms and no AI ordering
outline parser verifies three-level numbering, H4 actions, and H3 fallback
discipline page verifies adding a topic and top-tab task rendering
workspace verifies the empty task-plan file on new maps
GET/PUT handlers verify task order and status persistence
artifact watcher and frontend refresh verify the learning-plan event
frontend full tests and production build
backend full Go tests
```

## Manual Smoke

Generate one broad map, add two H4 topics in a chosen order, reorder them, start
and complete one task, reopen it, and launch the system-learning confirmation
form. Reopen an old H3-only map and confirm its actions remain available.
