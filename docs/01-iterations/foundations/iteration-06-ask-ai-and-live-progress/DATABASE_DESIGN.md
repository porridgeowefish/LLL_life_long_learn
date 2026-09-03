# Iteration 06 Data Design

Status: proposed
Last reviewed: 2026-07-06

File remains the durable source of truth. No database is introduced.

## config.local.json

One new top-level section, `askAiProviders`, loaded by a new package
`backend-go/internal/askaiconfig/` (mirroring `imageconfig`):

```jsonc
{
  "askAiProviders": {
    "default": "deepseek",
    "searchEngine": "google",        // "google" | "bing"
    "providers": [
      {
        "id": "deepseek",
        "kind": "openai",
        "name": "DeepSeek",
        "baseURL": "https://api.deepseek.com/v1",
        "apiKey": "sk-...",
        "model": "deepseek-chat",
        "reasoning": false
      },
      {
        "id": "claude",
        "kind": "anthropic",
        "name": "Claude",
        "baseURL": "https://api.anthropic.com",
        "apiKey": "sk-ant-...",
        "model": "claude-sonnet-5",
        "thinking": true
      }
    ]
  }
}
```

`config.local.json` is gitignored and already holds `imageApiKey`, so storing
`apiKey` here is consistent with existing practice. `GET /api/settings/ask-ai`
never returns plaintext keys.

Writers use the raw-JSON merge pattern from `agentruntime.SaveSelected`
(`runtime.go:131-146`) so saving `askAiProviders` never clobbers other keys.

## Ask-AI Conversations (persisted on a confusion)

Ask-AI exchanges are NOT ephemeral. They are persisted as an optional `ask`
field on the confusion they were triggered from. The existing confusions store
(`backend-go/internal/confusionstore/store.go`, file
`<projectRoot>/explain/confusions.json`, atomic write via `workspace.AtomicWriteFile`)
is extended to hold it.

```text
confusion.ask.messages   [{ id, role: user|assistant, content, createdAt }]
confusion.ask.summary    <=250-char Chinese summary, set on close
confusion.ask.summaryState  idle | pending | done | failed
confusion.ask.providerId which provider produced the exchange
confusion.ask.updatedAt
confusion.state          'open' during active ask -> 'asked' after summarize
```

Readers tolerate a missing `ask`. Nothing about the existing confusion fields
(quoteSnapshot, charStart/End, notes, state) changes meaning; the `'asked'`
state gains a real use.

## Sidebar / ConfusionPanel

The existing sidebar already lists confusions. Iteration 06 adds hover behavior:

```text
confusion with ask.summaryState == 'done'    hover shows ask.summary
confusion with ask.summaryState == 'pending' hover shows "生成总结中..."
confusion with ask.summaryState == 'failed'  hover shows "总结生成失败"
confusion with ask.summaryState == 'idle' or no ask   existing behavior (quote only)
```

Clicking an `'asked'` confusion reopens the exchange in read-only review mode
(no input, no ask-stream call).

## Run Progress

Latest status per `runId` is kept in memory (a server-side map). It may
optionally mirror to `runs/<ts>-<agentId>/progress.json` for debugging, but the
durable source is not required: progress is transient and disposable once the run
ends.

## Hook Settings

Per-run, transient: `runs/<ts>-<agentId>/claude-settings.json`, written by the
launcher and pointed at via `claude --settings <file>` (injection method pending
validation, see ALIGNMENT.md). Contains only the hook commands and the run token
reference for that run.

## Artifact watching

No new persistent data. The fsnotify watcher observes existing zone folders:

```text
projects/<slug>/explain/
projects/<slug>/explain/pages/
projects/<slug>/intro/
projects/<slug>/extend/
projects/<slug>/summary/
projects/<slug>/practice/
```

Watcher events are debounced and emitted as `artifact-updated`; nothing is
written to disk by the watcher itself.

## No Migration

```text
no database migration
no manifest schema change
confusions.json gains an optional ask field on entries (readers tolerate absence)
all other new state is config (config.local.json) or transient (in-memory / run dir)
```
