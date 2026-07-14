# Iteration 06 API Contract

Status: proposed
Last reviewed: 2026-07-14

## Existing Endpoints (unchanged unless noted)

The heavy agent generation endpoints are unchanged:

```text
POST /api/agents/{id}/invoke          launches the selected CLI runtime TUI
POST /api/projects/{id}/explain/resume resumes the selected CLI runtime
GET  /api/events                      the app-wide SSE bus (lifecycle events only)
POST /api/projects/{id}/confusions    reused to create the doubt that Ask-AI attaches to
```

## Runtime discovery patch (2026-07-08)

Runtime provider objects may include an optional `mode` field:

```jsonc
{
  "id": "codex",
  "name": "Codex CLI",
  "bin": "codex",
  "available": true,
  "mode": "wsl"                     // "native" | "wsl"; omitted by older clients
}
```

On Windows, if a native Claude/Codex probe fails, the backend probes WSL with
`wsl.exe -e sh -lc "command -v <bin> && <bin> --version"`. WSL-mode launches
convert project and prompt paths with `wslpath`; the Linux side reads
`prompt.md` directly, so Chinese prompt text does not cross the Windows argv or
environment boundary.

Codex launches always opt into its explicit unrestricted mode. Interactive and
headless invocations include `--yolo`; Explain resume runs
`codex resume --last --yolo` from the project root. Claude resume remains
`claude -c`. The resume endpoint checks the selected runtime's availability
rather than requiring Claude to be installed.

## Ask-AI Settings

```text
GET  /api/settings/ask-ai             -> { default, searchEngine, providers[] }
PUT  /api/settings/ask-ai             <- { default, searchEngine, providers[] }
POST /api/settings/ask-ai/probe       <- { providerId } | inline provider config
                                      -> { ok, latencyMs?, error? }
```

`providers[]` entries:

```jsonc
{
  "id": "deepseek",
  "kind": "openai",                 // "openai" | "anthropic"
  "name": "DeepSeek",
  "baseURL": "https://api.deepseek.com/v1",
  "apiKey": "sk-...",               // write-only; GET never returns plaintext
  "model": "deepseek-chat",
  "reasoning": false                // openai-compatible: parse reasoning_content
}
```

Anthropic entries use `"kind":"anthropic"`, no `/v1` suffix, and `"thinking":true`.

`PUT` persists via the raw-JSON merge used by `agentruntime.SaveSelected`
(`backend-go/internal/agentruntime/runtime.go:131-146`), so other
`config.local.json` keys are preserved.

`probe` sends a single 1-token completion to validate the key. It is only ever
called on explicit user action.

## Ask-AI conversation (persisted on a confusion)

An Ask-AI exchange is anchored to a confusion (the selected doubt). The flow:

```text
select passage + 问 AI
  -> POST /api/projects/{id}/confusions  (existing) creates the doubt, state 'open'
  -> open AskAiPanel tied to that confusion id
each follow-up
  -> POST /api/projects/{id}/confusions/{cid}/ask-stream  (streams the assistant reply)
close window
  -> POST /api/projects/{id}/confusions/{cid}/ask/summarize  (auto-summary)
  -> confusion state becomes 'asked'
reopen from sidebar
  -> read-only review of the stored messages (no ask-stream call)
```

### POST /api/projects/{id}/confusions/{cid}/ask-stream (request-scoped, NOT on the app-wide bus)

```text
Content-Type: application/json
Accept: text/event-stream

{
  "providerId": "deepseek",         // optional; falls back to default
  "content": "why does ... ?",      // the new user message
  "pageArtifactId": "explain/pages/02-overview.md"  // optional context
}
```

The backend appends the user message to `confusion.ask.messages`, optionally loads
`pageArtifactId` as system context, calls the provider with the full stored
history, streams the reply, and on completion persists the assistant message.

Response is `text/event-stream`, flushed per frame. Both OpenAI-compatible and
Anthropic streaming are normalized to one frame shape:

```text
data: {"type":"text","content":"..."}
data: {"type":"thinking","content":"..."}
data: {"type":"done"}
data: {"type":"error","content":"..."}
```

Normalization:

```text
OpenAI-compatible
  data: {"choices":[{"delta":{"content":"...","reasoning_content":"..."}}]}
  data: [DONE]
  -> content                          => type:text
     reasoning_content (reasoning=true) => type:thinking
     [DONE]                           => type:done

Anthropic Messages
  event: content_block_delta  data: {"delta":{"type":"text_delta","text":"..."}}
  event: content_block_delta  data: {"delta":{"type":"thinking_delta","thinking":"..."}}
  event: message_stop
  -> text_delta                       => type:text
     thinking_delta                    => type:thinking
     message_stop                      => type:done
```

If the client aborts mid-stream, the user message is kept but the partial
assistant turn is discarded (the learner can re-ask). On successful `done`, the
full assistant message is persisted.

### POST /api/projects/{id}/confusions/{cid}/ask/summarize

```text
no body
-> 202 Accepted immediately; sets confusion.ask.summaryState = "pending"
```

The backend generates a Chinese summary (<=250 chars) from the stored
`quoteSnapshot` + `ask.messages` via the configured provider, stores it as
`ask.summary`, sets `summaryState = "done"` (or `"failed"`), and emits
`confusion-updated` so the sidebar refreshes. Idempotent: calling again
regenerates. If no provider is configured or the call fails, `summaryState`
becomes `"failed"` and the conversation is still saved.

### Confusion shape (new optional `ask` field)

```jsonc
{
  "id": "c012",
  "sourceArtifactId": "explain/pages/02-overview.md",
  "quoteSnapshot": "selected passage...",
  "charStart": 120, "charEnd": 180,
  "state": "asked",                  // 'open' during active ask, 'asked' after summarize
  "ask": {
    "messages": [
      { "id": "m1", "role": "user", "content": "...", "createdAt": "..." },
      { "id": "m2", "role": "assistant", "content": "...", "createdAt": "..." }
    ],
    "summary": "结合疑问点的一句话总结（≤250 字）",
    "summaryState": "done",          // "idle" | "pending" | "done" | "failed"
    "providerId": "deepseek",
    "updatedAt": "..."
  }
}
```

Readers tolerate a missing `ask`. The existing `'asked'` state
(`frontend/src/api/confusions.ts:10`, `confusionstore/store.go:24`) is now used
for real: a confusion with a summarized Ask-AI exchange is `'asked'`.

## Run Status (Claude Code hooks, Phase C)

```text
POST /api/runs/{runId}/status
Header: X-Run-Token: <per-run random token>

{
  "phase": "running",               // optional
  "activity": "wrote pages/02-overview.md",  // optional, latest activity
  "pagesDone": 2,                   // optional
  "pagesPlanned": 5,                // optional
  "done": false                     // optional; true marks the run completed
}
```

The token is generated by the launcher per run, stored server-side, and written
into the injected hook config. Requests with a missing or wrong token are
rejected, so only the launched run can report.

`done:true` transitions the session to `completed`, emits `session-completed`,
and hides the progress bar (this fixes the "sessions never complete" bug for
Claude runs).

## SSE Events

The app-wide bus carries low-volume lifecycle state only.

```text
artifact-updated   { projectSlug, zone, path }     reused, now generalized to all zones
run-progress       { runId, phase?, activity?, pagesDone?, pagesPlanned? }   NEW
session-completed  { runId }                       reused, now subscribed
session-failed     { runId, error? }               reused, now subscribed
confusion-updated  { projectSlug, confusionId }    reused; now also fires on summary completion
```

`artifact-updated` is emitted by the fsnotify watcher on a debounced stable
write to a zone folder. `run-progress` is emitted on each accepted
`POST /api/runs/{runId}/status` and on watcher-derived page counts for non-Claude
runs. `confusion-updated` fires when an Ask-AI summary finishes so the sidebar
hover updates from "生成总结中..." to the summary text.

## Claude Code Hook Contract (Phase C)

The launcher injects hooks (via `claude --settings <run-scoped file>` if
supported; otherwise a documented one-time global install). The hooks report to
`POST /api/runs/{runId}/status`:

```text
PostToolUse  matching Write|Edit on **/explain/pages/*.md (and other zone files)
             -> { "activity": "wrote <path>", "pagesDone": <n> }

Stop         -> { "done": true }
```

Hook command form on Windows is pending validation (see ALIGNMENT.md): likely
`curl.exe` or PowerShell `Invoke-RestMethod`, with the run token in a header and
any Chinese text passed via stdin or a temp file, never argv (per project
Windows + shell + UTF-8 conventions).
