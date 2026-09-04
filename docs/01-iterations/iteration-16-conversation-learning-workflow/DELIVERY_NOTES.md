# Iteration 16 Delivery Notes

Status: delivered; browser E2E remains learner-run.

Implemented:

- Canonical, background source parsing to one `content.md`, teacher-only
  citation preview/attachment, and a source-file ledger without generated assets.
- `teachingOutline` scope snapshots and teacher prompt injection.
- Confirmed conversation-evidence consolidation into Intro/Body, with optional
  Practice only via explicit `practiceRequested`.
- Provider-reported teacher token persistence and `/usage` management page;
  standalone Agent management navigation was removed.

Verification: `go test ./...` passed; frontend Vitest passed (46 files, 153
tests); `npm run build` passed. Browser E2E is intentionally left to the learner.
