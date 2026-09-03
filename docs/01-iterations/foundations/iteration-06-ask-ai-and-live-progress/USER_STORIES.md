# Iteration 06 User Stories

Status: proposed
Last reviewed: 2026-07-06

## Ask-AI

- As a learner, I want to select a confusing passage and immediately ask an
  external AI about it, so I do not have to leave the reader or open a separate
  tool.
- As a learner, I want the answer to stream in token by token with the model's
  thinking visible, so the interaction feels like a real chat.
- As a learner, I want the answer rendered fully (formulas, diagrams, tables),
  not as plain text.
- As a learner, I want to ask follow-up questions in the same window to dig
  deeper.
- As a learner, I want my question and the AI's answer saved on the passage, so I
  can come back to them later in the sidebar.
- As a learner, I want a short summary generated when I close the window, so I can
  recall the resolution without rereading the whole exchange.
- As a learner, I want to see my saved doubt in the sidebar and hover to read its
  summary (or see that the summary is still generating).
- As a learner, I want to reopen a past Ask-AI exchange to review it, without
  accidentally starting a new follow-up.
- As a learner, I want to stop a long answer mid-stream.
- As a learner, I want to pick a fast or cheap model for quick questions and a
  stronger one when I need depth.
- As a learner, I want the answer to take the surrounding page into account, so
  basic-concept questions are answered in context.

## Live Progress

- As a learner, I want the reader to refresh the instant a new page is written,
  not on a timer.
- As a learner, I want to see what the agent is currently doing while it works in
  the terminal window.
- As a learner, I want a progress bar that reflects pages written, and an honest
  "still working" state when the total is unknown.
- As a learner, I want the progress bar to disappear when the run is actually
  done.
- As a learner, I want Intro / Extend / Summary output to appear without
  refocusing the window.

## Settings and Config

- As a learner, I want to add my own OpenAI-compatible or Anthropic provider with
  a base URL, model, and key.
- As a learner, I want to verify a key works before relying on it.
- As a learner, I want to choose the default provider for Ask-AI.
- As a learner, I want to switch the browser-search engine between Google and
  Bing.

## Maintainer

- As a maintainer, I want refresh to be runtime-agnostic (fsnotify), so non-Claude
  runtimes also benefit.
- As a maintainer, I want Claude runs to report completion reliably, so sessions
  stop leaking as "running".
- As a maintainer, I want the run-status endpoint authenticated, so only the
  launched run can report progress.
- As a maintainer, I want the Ask-AI carve-out from the CLI-only model principle
  documented as an ADR.
- As a maintainer, I want the new streaming, persistence, and progress code
  covered by automated tests using simulated provider SSE, the confusions store,
  and tempdir file watches.
