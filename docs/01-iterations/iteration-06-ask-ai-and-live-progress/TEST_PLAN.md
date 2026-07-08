# Iteration 06 Test Plan

Status: proposed
Last reviewed: 2026-07-08

## Automated

### Ask-AI config and streaming (`askaiconfig`, `askaiprovider`, frontend)

- `askaiconfig.Load` reads `askAiProviders`, tolerates a missing section (feature
  off), and preserves unrelated keys on save.
- `askaiprovider` OpenAI-compatible client: feed a simulated OpenAI SSE stream
  (including `reasoning_content`), assert normalized frames
  `{text, thinking, done}`.
- `askaiprovider` Anthropic client: feed a simulated Anthropic event stream
  (`text_delta`, `thinking_delta`, `message_stop`), assert the same normalized
  frames.
- `askaiprovider` non-streaming summarize: given quote + messages, returns a
  string; respects a <=250-char instruction.
- `/api/projects/{id}/confusions/{cid}/ask-stream` handler: given a stub
  provider, writes `text/event-stream` frames with correct flushing, persists the
  assistant message on `done`, and emits an error frame on provider failure.
- `parseAskAiStream` (frontend): splits a `fetch` ReadableStream on `\n\n`,
  parses `data:` JSON, handles partial chunks across reads.
- `floatingPosition` (frontend): flips/clamps the anchor rect at viewport edges.
- `AskAiPanel` (frontend): opening from a selected quote pre-fills the input
  with `请你解释「<quote>」`; browser search is not rendered inside the panel.
- `useMarkdown` on streamed fragments continues to render GFM + KaTeX + Mermaid
  (extend existing `useMarkdown.test.ts` cases).

### Ask-AI persistence and summary (`confusionstore`, frontend)

- An Ask-AI exchange is stored on its confusion: messages survive a store
  save/reload.
- On `done`, the assistant message is appended to `ask.messages`; on abort, the
  partial assistant turn is not stored (the user message is).
- `POST .../ask/summarize` sets `summaryState` pending, generates the summary from
  `quoteSnapshot` + `ask.messages`, stores it, sets state done, and emits
  `confusion-updated`.
- Summarize failure (no provider / provider error) sets `summaryState = failed`
  without losing the stored conversation.
- The confusion transitions to `'asked'` after summarize.
- ConfusionPanel hover: shows summary when done, "生成总结中..." when pending,
  "总结生成失败" when failed, quote only otherwise.
- Reopen from sidebar renders in read-only review mode (no input, no ask-stream).

### Live progress (`artifactwatch`, `runs/status`, frontend)

- `artifactwatch`: write a file into a tempdir-watched zone folder, assert a
  debounced `artifact-updated` event is emitted exactly once per stable write.
- `artifactwatch`: ignore writes outside watched zones.
- `/api/runs/{runId}/status`: rejects requests with missing/wrong `X-Run-Token`.
- `/api/runs/{runId}/status` with `done:true`: transitions the session to
  `completed` and emits `session-completed`.
- `RunProgressBar`: determinate when `pagesPlanned` known, indeterminate +
  activity text otherwise, hidden on completed/failed.

### General

- Run all Go tests, Vitest, production frontend build, and `git diff --check`.

## Terminal Smoke

- Configure one OpenAI-compatible provider and one Anthropic provider in
  SettingsPage; click probe for each and confirm success.
- Select a passage in an Explain page, click "问 AI", confirm a floating window
  opens anchored to the selection and the answer streams token by token.
- Confirm the thinking section is collapsible and the answer renders markdown.
- Ask a follow-up in the same window and confirm multi-turn context is kept.
- Close the window; confirm the sidebar entry appears and, after a moment, hover
  shows the <=250-char summary.
- Click stop-generation mid-stream and confirm it aborts.
- Reopen the sidebar entry and confirm the exchange is read-only with no input.
- Click "在浏览器搜索" and confirm the system browser opens a Google search for
  the quote (and Bing when configured).
- Trigger an Explain generation; confirm new pages appear in the reader within
  ~1s of being written, with no 5s poll lag.
- Confirm the progress bar appears during generation and disappears on
  completion (Claude run, Phase C).

## Browser Smoke

- Confirm selecting text shows the three-button bar (保存摘要 / 问 AI / 取消).
- Confirm the Ask-AI window is anchored near the selection and flips at the
  viewport edge rather than overflowing.
- Confirm a closed Ask-AI exchange appears in the sidebar with a hover summary
  (and "生成总结中..." while pending).
- Confirm Explain/Practice/Intro/Extend/Summary zones refresh on file write
  without a manual reload or window refocus.
- Confirm `OutputViewer` (Intro/Extend/Summary) updates without refocusing.
- Confirm the progress bar shows latest activity and hides when done.

## Manual Review

- Read a streamed Ask-AI answer end to end and verify it feels like a real chat
  model interaction (streaming cadence, visible thinking, full rendering).
- Verify the Ask-AI window does not obscure the passage it was asked about.
- Verify the auto-summary actually reflects the user's doubt and the exchange,
  not a generic restatement.
- Verify the sidebar hover and read-only reopen feel like a stable record, not a
  second live session.
- Watch a generation and verify the progress bar is honest: real activity text,
  no fake percentage when the total is unknown.
- Verify a non-Claude runtime still gets event-driven refresh (fsnotify) and an
  indeterminate progress bar.
- Verify SettingsPage copy no longer claims the platform never asks for model
  parameters, and the ADR explains the carve-out.

## Patch Smoke (2026-07-08)

- Select text in Explain and confirm the selection bar exposes annotation,
  Ask-AI, and browser search as same-level actions.
- Open Ask-AI from selected text and confirm the input is pre-filled as
  `请你解释「<selected text>」`.
- Confirm browser search opens Google/Bing directly from the selection bar and
  no longer appears inside the Ask-AI chat footer.
- Drag and resize Ask-AI; confirm it remains clamped to the viewport and source
  text can still be reached for copying.
- On a Windows host where Claude/Codex are installed only inside WSL, confirm
  runtime detection reports WSL mode and launch opens the CLI in the project
  root. The prompt must be read from `prompt.md`, not passed through argv/env.
- Corrupt `confusions.json`, `folders.json`, or a practice `draft.json` in a
  disposable project; confirm a `.corrupt-*.json` backup is written and the app
  falls back to an empty safe state.
- Visually review Knowledge Flower: large glyph is 75% of the prior size and
  the five dimension descriptions use the freed space.
- Visually review Home, AI config, and Practice generation pages for denser
  spacing, lower-radius cards, and less empty table-like space.
