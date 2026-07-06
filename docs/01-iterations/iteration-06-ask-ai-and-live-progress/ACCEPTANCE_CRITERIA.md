# Iteration 06 Acceptance Criteria

Status: proposed
Last reviewed: 2026-07-06

```text
Ask-AI
selecting a passage shows a 问 AI action alongside the existing 标注 actions
clicking 问 AI creates/reuses a confusion and opens a floating window anchored to the selection
the window flips or clamps at the viewport edge instead of overflowing
the answer streams token by token from the configured external provider
the model's thinking is shown in a collapsible section above the answer
the answer is fully rendered (GFM, KaTeX math, Mermaid diagrams, tables)
the learner can ask follow-up questions in the same window with context preserved
the learner can stop generation mid-stream
the learner can switch provider within the window
the conversation is persisted on its confusion: messages survive reload
the stream is request-scoped and is NOT broadcast on the app-wide SSE bus
on close the backend auto-generates a <=250-char summary from quote + conversation
summaryState transitions pending -> done and emits confusion-updated
summarize fails gracefully (no provider / error -> failed), conversation still saved
the confusion becomes 'asked' after summarize
the sidebar shows the doubt quote; hover shows the summary
hover shows 生成总结中... while pending and 总结生成失败 on failure
reopening from the sidebar is read-only review with no follow-up
a one-click browser search opens Google (or Bing when configured) for the quote
API keys are never returned in plaintext by GET /api/settings/ask-ai
config.local.json round-trips without clobbering other keys

multi-vendor
OpenAI-compatible providers work via custom baseURL (OpenAI/DeepSeek/Moonshot/Qwen)
Anthropic providers work natively with extended thinking
reasoning_content (OpenAI-compatible) and thinking_delta (Anthropic) both surface as thinking
both formats normalize to the same frame shape

Settings
the learner can add, edit, delete, and set-default Ask-AI providers
probe validates a key with a single minimal request
SettingsPage copy reflects the carve-out; an ADR documents it

Live refresh (Phase A)
explain/ pages appear within ~1s of being written, with no 5s poll
practice task/evaluation refresh is event-driven, not 3s polling
OutputViewer (Intro/Extend/Summary) updates without window refocus
health polling remains (it is a liveness check, not artifact refresh)
refresh works on every runtime via fsnotify, not only Claude

Progress (Phase C)
a RunProgressBar is visible during generation near the zone header
determinate "N / M 页" when a planned count is known
indeterminate bar + latest activity text otherwise
the bar hides on session-completed / session-failed
sessions reach "completed" on Claude runs (no longer leak as "running")
the run-status endpoint rejects requests without a valid run token

non-Claude runtimes
get fsnotify event-driven refresh
get an indeterminate progress bar with latest activity
are not broken by the absence of Claude Code hooks

quality gates
all Go tests pass
all frontend tests pass
production frontend build succeeds
terminal smoke and browser smoke pass
git diff --check passes
```
