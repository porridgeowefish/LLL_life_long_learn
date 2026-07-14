# Iteration 09 Test Plan

Status: active
Owner: project maintainer
Last reviewed: 2026-07-15

## Automated

- Workspace test: recursive project-root deletion includes nested run artifacts.
- Folder-store test: membership and map binding cleanup preserves other data.
- Session-store test: active detection and project-scoped removal.
- HTTP tests: successful cascade cleanup and `409` active-session protection.
- Frontend unit test: exactly one confirmation precedes the delete mutation.
- Browser-storage unit test: only matching project drafts are removed.
- Intro UI test: summary is visible, no supplement action exists, and the survey
  follows generated content and diagnosis.
- Agent registry test: Intro charter retains the assessment contract and
  requires concise summaries.
- Full Go test suite, frontend Vitest suite, and production frontend build.

## Manual smoke

Create a disposable learning project, add artifacts/drafts, delete it from the
sidebar, confirm once, verify navigation to the overview and absence after
reload. Cancel once in a second disposable project and verify it remains. Open
an Intro assessment and verify the survey is the final section at desktop and
mobile widths.
