# Iteration 17 — Data Design

Status: planned
Owner: project maintainer
Last reviewed: 2026-09-11

All persistence stays file-first under `projects/<slug>/`. The conversation
event log (`conversation/events.jsonl`) remains append-only; every new
capability below is a new event type, never a rewrite.

## Conversation events (new)

| event | data | writer |
|---|---|---|
| `learner-queued` | `{queueId, content, attachmentRefs, operationId}` | queue endpoint |
| `queue-item-edited` | `{queueId, content, attachmentRefs}` | edit endpoint |
| `queue-item-discarded` | `{queueId}` | discard endpoint |
| `queue-item-promoted` | `{queueId, learnerMessageId, mode}` mode ∈ `auto \| steer \| manual` | advance/steer |
| `steering-note` | `{learnerMessageId, responseId}` | steer (marks the promoted turn as mid-generation guidance) |
| `response-superseded` | `{responseId, newResponseId}` | regenerate |

### Queue projection rules

- A queue item exists from `learner-queued` until its `queue-item-promoted` or
  `queue-item-discarded` event; `queue-item-edited` replaces content/refs.
- `Read`/`ReadRecent` return the live queue in `Projection.queue` (ordered by
  queue seq).
- On server startup, reconciliation must not promote or discard queued items;
  they simply remain queued. (`ReconcileInterruptedResponses` gains no queue
  behavior; advance is triggered by turn completion and by the queue endpoint.)

### Supersession rules

- `response-superseded` marks the old `teacher-response-started` family as
  hidden: messages whose ID equals the superseded response's teacher message
  are excluded from projections.
- `ResponseForLearner` returns the newest non-superseded response for a
  learner message (regeneration creates a second `teacher-response-started`
  for the same trigger; the older one loses).
- Sealed assistant snapshots (`SnapshotThrough`) are unaffected: they capture
  through a cutoff seq and never read supersession markers.

### Steering context injection

The steering learner message is a normal `message-recorded` learner message.
The turn that follows it appends its system-prompt appendix:

```text
【生成中引导】学习者在你上一条回复未完成时插入了：<内容>。
上一条 interrupted 消息是中断稿：吸收其中仍然有效的部分，按引导方向继续教学。
```

## Tool-call protocol in provider payloads

The gateway message type gains optional tool fields so the search loop can
continue a conversation:

```go
Message{ Role, Content, ToolCalls []ToolCall, ToolCallID string } // ToolCallID set ⇒ role "tool"
```

- openai kind: assistant message carries `tool_calls` (provider call id +
  name + raw argument JSON); tool result is `{role: "tool", tool_call_id,
  content}`.
- anthropic kind: assistant message uses content blocks
  `[{type:"text"},{type:"tool_use",id,name,input}]`; tool result is a user
  message with `[{type:"tool_result",tool_use_id,content:[{type:"text"}]}]`.
- anthropic kind disables `thinking` on requests that carry tools this
  iteration (re-synthesized assistant messages cannot replay signed thinking
  blocks).

## webSearch config section

New optional top-level section in `config.local.json`:

```json
"webSearch": { "provider": "zhipu", "apiKey": "…", "engine": "search_std" }
```

- Schema lives in `platform/config` beside `ai`; secrets follow the existing
  inline/env-key conventions (`apiKeyEnv` supported).
- Zhipu endpoint: `POST {baseURL}/api/paas/v4/web_search` with
  `{"search_engine": engine, "search_query": query}`; response items
  `{title, link, content, media}` are re-packed as a compact text block for
  the provider. The client is an interface so a future provider swap is local.

## OCR artifacts

- Input: `sources/<sourceID>/revisions/<rev>/original/<file>` (image bytes).
- Output: the existing single derived file `derived/content.md` written via
  `CommitDerived` (same shape as document parsing — exactly one content.md).
- Binding: `askAiProviders.bindings.ocr` → any openai-kind vision provider
  (`Resolve("ocr")`); anthropic-kind vision is also accepted.
- No new persistent schema; failure marks the source failed via existing
  `SetStatus` codes (`ocr-failed`).

## Repository layout changes

| before | after | migration |
|---|---|---|
| `agents/` | `learning-agents/` | `git mv`; `platform/filesystem` `AGENTS_ROOT` constant updated; API paths unchanged |
| `scripts/run-check.js`, `scripts/check-coverage.js` | `tools/check/` | `git mv`; `package.json` script paths updated |
| `scripts/qa_*.py`, `scripts/gen_infographic.py` | `tests/manual/` | `git mv`; `tests/manual/README.md` documents usage |
| `<root>/folders.json` (runtime, gitignored) | `projects/folders.json` | one-time read-back: new path first, legacy root path fallback + copy; folder store owns the migration |
