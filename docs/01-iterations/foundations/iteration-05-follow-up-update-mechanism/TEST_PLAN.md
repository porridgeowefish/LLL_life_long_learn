# Iteration 05 Test Plan

Status: active
Last reviewed: 2026-07-05

## Automated

- Prompt assembly treats `parentPageId` as optional context, not an append-page
  instruction.
- Explain follow-up prompt says the default is to update the current page, not
  append a new page.
- Explain follow-up prompt defines the independent-module boundary for creating
  a new page.
- Explain follow-up prompt allows structure-level rewrites when the learner's
  question reveals a better logical frame.
- Explain follow-up prompt requires preserving learner hypotheses, challenges,
  and useful partial models.
- Explain follow-up prompt forbids turning every learner question into a
  standard-answer page.
- Updated manifest entries increment `revisionCount`, stamp `updatedAt`, and
  record `lastUpdateReason`.
- Creating a module adds `kind=module` and `derivedFromPageId`.
- Structure rewrites increment `structureRevision` on affected pages.
- Existing `kind=followup` manifests remain readable.
- Existing legacy `explain/output.md` projects remain readable.
- Summary and Practice readers continue to accept source refs pointing at
  revised Explain pages.
- Run all Go tests, Vitest, production frontend build, and `git diff --check`.

## Terminal Smoke

- Resume or open an Explain Agent terminal, ask a clarifying follow-up, and
  confirm the current page updates without adding a new table-of-contents entry.
- Ask a correction follow-up in the terminal and confirm the corrected sentence
  lands in the relevant page section.
- Ask a challenge framed as a learner hypothesis in the terminal and confirm
  the revised page preserves the hypothesis plus its limits.
- Ask for a genuinely separate module in the terminal and confirm a new
  `kind=module` page appears in the navigation.
- Ask a question that exposes a better organizing frame and confirm the agent
  updates page titles/order or splits/merges pages instead of only appending an
  answer.

## Browser Smoke

- Confirm the frontend renders updated page content after file refresh.
- Confirm the frontend does not expose a new follow-up question box or
  update/create decision control for this slice.
- Confirm legacy `kind=followup` pages still show in the reader.

## Manual Review

- Read one revised page end to end and verify it feels like a coherent study
  page, not an appended chat answer.
- Compare the before/after git diff or file content and verify the update
  improves local clarity without deleting important prior context.
- Check that the AI's tone encourages independent reasoning while still making
  factual corrections.
- Check that a structure rewrite improves the learning path and does not merely
  reshuffle pages for cosmetic reasons.
