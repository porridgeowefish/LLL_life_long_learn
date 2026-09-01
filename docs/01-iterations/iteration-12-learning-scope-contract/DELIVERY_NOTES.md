# Delivery Notes

Status: implemented
Owner: project maintainer
Last reviewed: 2026-07-28

## Delivered

- shared Go scope/catalog model and validation;
- map catalog generation contract and read API;
- canonical topic reference from frontend to backend;
- system-learning scope snapshot/draft creation;
- universal prompt injection and Intro/Explain charter boundaries;
- root artifact SSE refresh;
- backend and frontend automated coverage.

## Compatibility Notes

- Standalone and map-origin projects use the same five-zone engine.
- Legacy system-learning projects without a scope file still launch with a
  conservative prompt fallback.
- Legacy maps require explicit encyclopedia regeneration before scoped deep-dive
  creation; the UI explains this instead of silently creating an unscoped map
  child.
- Existing title-only learning-plan items remain readable and editable.
