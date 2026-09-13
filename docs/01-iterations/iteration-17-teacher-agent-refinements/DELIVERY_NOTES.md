# Iteration 17 — Delivery Notes

Status: delivered
Owner: project maintainer
Last reviewed: 2026-09-13

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

## Hotfix (2026-09-12, same iteration)

The folders.json relocation shipped with two structural flaws that together
caused learner-visible data loss; both are fixed and lesson-logged
(LESSONS_LEARNED 16/17):

- folderstore resolved its path from `paths.WORKSPACE` while every other
  store used the workspace projects-root override — a divergent process
  (test or partial startup) could persist a foreign/empty layout over the
  canonical ledger. The store now resolves through `workspace.ProjectsRoot()`
  and the migration copy hard-fails instead of silently continuing empty.
  User data was restored from the intact root snapshot (18 folders, 20
  memberships); the damaged file is kept at
  `tmp/folder-recovery/folders-damaged-backup-1246.json`.
- The homepage rhythm heatmap collapsed to 2px (border-only) because its
  section was the lone shrinkable flex child of a fixed-height scrolling
  column with `overflow:hidden`; the unclassified-project flood triggered
  it. `flex-shrink: 0` added; Playwright-verified at 509px/181 cells with
  the restored folders visible.

Also added in the hotfix round: OCR binding UI on the models settings page
(provider + model override), and a learning-investment heatmap on the usage
page (same activity source and heat scale as the homepage).

## Hotfix (2026-09-13, same iteration)

- Markdown `body` assets render through `BodyAnnotations` for paging and
  annotation support. Its stylesheet had omitted the shared document-table
  treatment, so table rows appeared as unstructured text. It now applies the
  same collapsed border, cell padding, and header background as other Markdown
  assets; a Playwright route smoke checks the computed production styles.
- A generated Tencent Cloud review package was valid in its sealed attempt but
  had a terminal generic `commit-failed` record, so it was absent from both the
  teacher task projection and `GET /generated`. ADR-0020 adds one bounded,
  idempotent recovery for this artifact-only case. It never reruns the native
  CLI or model, and a second failure becomes `artifact-recovery-failed`.
- Task commit failures now name the output class that failed to enter its
  durable store instead of leaving only a generic prompt/path suggestion.

Verification for this hotfix: `npm run check` passed (format, vet, build,
architecture graph, short and full Go tests with coverage, frontend lint/tests,
and production frontend/backend builds); the focused Playwright body-table
smoke passed; after restarting the production binary, the Tencent Cloud task
`task_01M2B1G0VRMHX92DZKNQ0HY9VN` converged to `succeeded` and
`GET /generated` returned its review artifact.

## Reasoning/body mixing investigation (2026-09-12 evening)

Three turns (14:26–14:34, GLM anthropic-compat) persisted thinking text as
正文 with no reasoning block. 34 controlled turns through the live backend
(short/long-context/cancel-race) plus 13 raw endpoint experiments could not
reproduce; adjacent turns on the same provider separated correctly. The
suspected mechanism is provider-side intermittent mislabeling (thinking
deltas riding summary/thinking_delta inside text-typed blocks — the exact
ambiguity e81b62d traded against). No classification change without
evidence; the anthropic gateway now fingerprints every stream's block/delta
labeling to tmp/teacher-stream-shapes.log so the next occurrence pins the
raw shape.

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
