# Iteration 17 — Delivery Notes

Status: delivered
Owner: project maintainer
Last reviewed: 2026-09-12

## Implementation decisions (settled)

- Auto-advance fires immediately after `teacher-response-finished` in the
  same goroutine chain; no delay/backoff for v1.
- Queue items carry the providerId chosen at enqueue time so promoted and
  steered turns reuse the learner's model selection.

## Verification

- [x] go test ./backend-go/... ./tools/... — all packages pass
- [x] npm --prefix frontend run test — 45 files / 163 tests pass
- [x] npm run archcheck — pass (via check:fast)
- [x] npm run check:fast — pass (gofmt/vet/build/archcheck/short Go/lint/frontend)
- [ ] manual smoke with real keys: GLM-4V OCR quality, Zhipu web search
      results, queue persistence across restart, exported .md review
      (learner-run; automated coverage exists via httptest fakes)

## Residual risks

- Zhipu web-search request/response field names were implemented from the
  published API shape and spec-tested with a fake server, but not yet
  exercised against the live endpoint (learner smoke pending).
- anthropic-kind teachers lose thinking summaries on turns that register the
  search tool (signed thinking blocks cannot be re-synthesized); documented
  in ADR-0019.
- Server restart does not auto-advance a queue left idle by the crash; items
  stay queued and editable by design — advance resumes on the next
  lifecycle event (send/steer/turn end).
- `ProjectPage` migration frontend test showed rare load-sensitive flakiness
  when the full check pipeline runs on a busy machine; passes reliably in
  isolation and on rerun. Pre-existing pattern, not introduced by this
  iteration's changes.
