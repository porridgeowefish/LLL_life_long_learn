# Implementation Plan — Iteration 17 Teacher Agent Refinements

Status: delivered
Owner: project maintainer
Last reviewed: 2026-09-11

Ordered waves; each wave leaves `go test ./...` and frontend tests green.

## Wave 1 — Conversation queue, steering, regeneration (backend)

1. `conversation/store.go`: queue event family + `QueueItems()` projection +
   helpers (`AppendQueueItem`, `EditQueueItem`, `DiscardQueueItem`,
   `PromoteQueueItem`); `Projection.Queue`; supersession map +
   `ResponseForLearner` skip; projection message filtering. → store tests.
2. `service/service.go`: refactor `StreamTurn` to accept a prepared turn
   (existing learner message reuse for regenerate); add steering appendix
   injection; keep delegate authorization semantics byte-identical.
3. transport `routes_learning_workspace.go` + `router.go`: turn-latch mutex
   map; `runTeacherTurn` helper (start/goroutine/finish + advance hook);
   endpoints queue/edit/discard/steer/regenerate/export; SSE reattach for
   auto-advanced turns reuses the active-response endpoint. → route tests.
4. Startup: advance check after `ReconcileAllInterruptedResponses` (idle queue
   heads start once).

## Wave 2 — Web search tool loop

1. `platform/config`: `webSearch` section (provider/apiKey/apiKeyEnv/engine) +
   validate + example config.
2. `teacher/internal/websearch`: `Provider` interface + zhipu implementation +
   tests (httptest).
3. `gateway`: `Message.ToolCalls`/`ToolCallID`; openai + anthropic payload
   building; anthropic thinking suppression with tools. → gateway tests
   (fake SSE, field-path assertions per LESSONS 14/15 style).
4. `service`: `search_web` tool registration when configured; loop ≤ 2
   searches; frames `search-started/completed/failed`; system prompt
   appendix about search usage and source citation.

## Wave 3 — OCR

1. `sources/internal/ocr`: vision client (openai/anthropic image message),
   extraction prompt, markdown output; tests with fake provider.
2. `sources/api.go`: exported async entry `ProcessImage(slug, sourceID, rev)`.
3. Upload route: image + parse approved + ocr binding resolvable → OCR path
   (no assistant task); else legacy path untouched. Failure → `ocr-failed`.

## Wave 4 — Frontend

1. `learningWorkspace` API client: queue/edit/discard/steer/regenerate/export
   + conversation `queue` field + new frame types.
2. Shared icon component (inline SVG set).
3. `TeacherView`: always-on composer; queue chips with hover icon actions
   (edit/steer/discard); send button semantic switch; post-stream reattach to
   auto-advanced response; last-message hover actions (copy/regenerate);
   export button; search status line. CSS: hover reveal + `@media
   (hover:none)` always-visible + `focus-visible`.
4. Vitest coverage per TEST_PLAN.

## Wave 5 — Repository normalization + docs

1. `git mv agents learning-agents`; update `AGENTS_ROOT` + docs references.
2. `git mv scripts/run-check.js scripts/check-coverage.js tools/check/`;
   update `package.json` + internal path refs.
3. `git mv scripts/qa_*.py scripts/gen_infographic.py tests/manual/`; add
   `tests/manual/README.md`; update QUALITY_COMMANDS paths.
4. folderstore: path → `projects/folders.json` with legacy-root read-back
   migration + test.
5. ADR-0018, ADR-0019; sync `DATA_MODEL.md`, `BACKEND_ARCHITECTURE.md`,
   `REPOSITORY_MAP.md`, `QUALITY_COMMANDS.md`, ADR index, iteration index
   (17 → current), INDEX.md active delivery.
6. Final: `npm run check:fast`, `archcheck`, full `go test`, frontend tests,
   grep for stale paths.
