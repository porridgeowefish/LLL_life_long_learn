# Iteration 06: Ask-AI Inline Help and Live Run Progress

Status: proposed
Owner: project maintainer
Last reviewed: 2026-07-06
Source of truth: this directory defines the sixth LLL delivery slice.

Editable alignment handoff: [ALIGNMENT.md](./ALIGNMENT.md)

Related design and iterations:

- `docs/superpowers/specs/2026-06-17-progressive-explain-generation-design.md` — charter-side progressive generation. That spec explicitly deferred the fsnotify/SSE "observation layer" as optional polish. Iteration 06 lands it.
- `docs/01-iterations/iteration-05-follow-up-update-mechanism/` — no overlap (prompt contract + manifest metadata only).

## Goal

Improve the felt learning experience along two independent axes that share
infrastructure (the SSE bus, `config.local.json`, SettingsPage):

```text
Ask-AI inline help
  select a passage -> floating window answers quick "small / basic concept" questions
  powered by a user-configured external model (multi-vendor), direct HTTP streaming
  shows thinking, full markdown, and a one-click browser search
  the exchange is persisted on a confusion; on close a short summary is generated
  and shown in the sidebar; the learner can reopen it read-only

Live run progress
  replace file polling with event-driven refresh (write a page -> notify frontend)
  surface a run progress bar driven by Claude Code hooks
  the learner is never waiting without feedback
```

The Ask-AI path is a separate lightweight channel. The heavy agent generation
runs (Explain, Intro, Practice, Summary) keep using the `claude` CLI unchanged.

## Background

Exploration confirmed the current state:

```text
SSE bus exists end to end, but 6 of 7 event types have zero frontend subscribers
the frontend refreshes by polling files (explain manifest 5s, practice 3s, etc.)
OutputViewer for Intro/Extend/Summary does not poll at all
the backend launches Claude fire-and-forget and never learns when a page is written
session-completed is never emitted; sessions stay "running" forever
there is no in-LLM HTTP model provider; Claude is invoked only via the claude CLI
there is no streaming or thinking renderer in the frontend (a prior design choice)
```

Iteration 06 fixes the refresh root cause (file watching + hooks), and adds the
Ask-AI channel as a scoped exception to the "model config stays in the CLI"
principle. That exception is recorded as an ADR under
`docs/00-product-and-architecture/` (to be added in Phase B).

## Included

```text
multi-vendor external model config (OpenAI-compatible + Anthropic native)
ask-ai provider config section in config.local.json
SettingsPage Ask-AI provider management with key probe
POST /api/projects/{id}/confusions/{cid}/ask-stream request-scoped streaming endpoint
ask-ai floating window anchored to the selection (positioning + viewport flip)
streaming token rendering with collapsible thinking, full markdown (GFM/KaTeX/Mermaid)
multi-turn conversation persisted on the triggering confusion (not ephemeral)
on close, backend auto-summarizes (<=250 chars, from quote + conversation); confusion -> state 'asked'
sidebar shows the doubt quote; hover shows the summary (生成总结中... while pending)
reopen from sidebar in read-only review mode (no follow-up after close)
provider switcher, stop-generation, one-click browser search (Google, switchable to Bing)
```

## Excluded

```text
in-app terminal output mirroring (prior principle stands)
in-window AI web search / Perplexity-style retrieval (only one-click browser search this iteration)
follow-up on a reopened (closed) Ask-AI session (review is read-only)
injecting Ask-AI summaries into subsequent agent generation prompts (out of scope)
floating window dragging, resizing, or multiple concurrent windows
non-Claude runtime equivalents of Claude Code hooks (they use the fsnotify fallback path)
moving the heavy agent generation runs to HTTP providers (still CLI)
image generation provider changes
database introduction (file-first stands)
```

## Key Decisions

- Ask-AI is a separate, lightweight, direct-HTTP channel using user-configured
  external keys. It does not go through the `claude` CLI. Heavy agent runs are
  untouched.
- Multi-vendor scope is OpenAI-compatible (custom baseURL covers OpenAI, DeepSeek,
  Moonshot, Qwen, and most vendors) plus Anthropic native (first-class extended
  thinking). Two streaming formats, normalized to one frame shape.
- The Ask-AI stream is request-scoped (`text/event-stream` on the response body),
  not broadcast on the app-wide SSE bus. Chat tokens must not reach every client.
- Ask-AI conversations are persisted on the triggering confusion (file-first, via
  the existing confusions store), so the learner's doubts and their AI-summarized
  resolution stay visible in the sidebar. Multi-turn is allowed while the window is
  open; closing triggers an auto-summary (<=250 chars, from quote + conversation)
  and sets the confusion to 'asked'. Reopening is read-only review, no follow-up.
- Refresh signaling is hybrid: fsnotify for universal artifact-level refresh
  (runtime-agnostic, replaces polling), plus Claude Code hooks for granular
  progress and reliable completion on Claude runs. Non-Claude runtimes degrade to
  fsnotify page-level progress with an indeterminate bar.
- The progress concept is named `run-progress` to avoid collision with the
  existing `progress` (gamification XP in `progressstore`).
- The Ask-AI window is anchored to the selection (not a centered modal) so the
  learner can read and ask without losing place. Viewport edge flipping in v1;
  drag/resize deferred.
- Determinate progress is opportunistic: shown only when a planned page count is
  known (hook-reported or declared in the manifest). The default is an
  indeterminate bar plus latest activity text, because Claude run duration is
  unpredictable. This is an honest progress model.
- Delivery is phased A -> B -> C: fsnotify refresh, then Ask-AI, then hooks. Each
  phase is independently verifiable.

## Product Principle

```text
quick questions should not break the reading flow
the answer should arrive the way a real chat model delivers it: streamed, with thinking visible
a doubt and its resolution should outlive the chat: saved, summarized, and reviewable
the learner should never stare at a static screen while work happens in another window
refresh should be instant when an artifact is ready, not on a timer
progress should be honest: activity and counts, not fake percentages
external model config is opt-in and scoped to the quick-help channel; deep work still uses the CLI
```
