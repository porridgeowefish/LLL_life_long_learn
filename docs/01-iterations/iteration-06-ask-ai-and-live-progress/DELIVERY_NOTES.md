# Iteration 06 Delivery Notes

Status: in progress (Phase C landed)
Last reviewed: 2026-07-06

Iteration 06 is approved as a design and phased A -> B -> C. Delivery has not
begun. Update this file per phase as work lands.

## Planned Phases

```text
Phase A (Theme 2 core)   fsnotify watcher -> kill polling; run-progress indeterminate bar; wire completed/failed
Phase B (Theme 1)        ask-ai config + settings + confusion-tied streaming endpoint + anchored floating window; multi-turn 持久化; 关闭自动生成 ≤250 字总结; 侧栏悬停查看; 调回只读; confusion -> 'asked'
Phase C (Theme 2 enhancement) Claude Code hooks -> granular progress + reliable completion
```

## Implementation Status

```text
Phase A   not started
Phase B   not started
Phase C   not started
```

## Validation Status

```text
not yet validated; tests and smoke checks defined in TEST_PLAN.md
```

## Deferred / Dropped

```text
none yet
```

## Phase C (Claude Code hooks) — landed 2026-07-06

### What landed

```text
backend-go/internal/runprogress/        new in-memory runId->Status store + per-run token auth (New/Register/Set/Get)
backend-go/internal/server/             POST /api/runs/{runId}/status handler (X-Run-Token auth; emits run-progress;
                                        on done:true -> sessions.SetFinished(completed) + session-completed)
                                        Server.runProgress field + init in New(); route registered
backend-go/internal/claudelauncher/     LaunchRequest.RunProgress field; writeHookSettings builder;
                                        --settings <file> injected into the Claude exec line (Claude runtime only)
frontend                                SSE_EVENTS.runProgress + fanout; useRunProgress subscribes run-progress
                                        (correlate runId == activeSessionId) and exposes pagesDone/pagesPlanned/activity;
                                        RunProgressBar renders determinate "N / M 页" when pagesPlanned > 0
```

### Real findings (resolved unknowns)

```text
1. claude CLI --settings flag: SUPPORTED.
   `claude --help` lists:
     --settings <file-or-json>   Path to a settings JSON file or a JSON string
                                 to load additional settings from
   So per-run hook injection via a run-scoped claude-settings.json + --settings
   is viable (no need for the one-time-global-hook fallback the plan allowed).

2. sessionstore "mark finished" signature: it is a METHOD on *Store, NOT a
   package-level function.
     func (s *Store) SetFinished(id string, state SessionState, exitCode int) bool
   DEVIATION from the plan: the handler calls sessions.SetFinished(runId,
   sessionstore.StateCompleted, 0) against the package-global *sessionstore.Store
   declared in routes_sessions.go (NOT the plan's assumed
   sessionstore.SetFinished(...)). For an unknown runId SetFinished is a no-op
   (returns false), so a hook firing after a server restart does not fail the
   request. State enum type is SessionState; constant is StateCompleted.
```

### PENDING live validation (manual; not auto-tested)

```text
1. Hook command + schema against the real Claude Code CLI on Windows.
   The writeHookSettings builder emits PostToolUse (matcher "Write|Edit") +
   Stop hooks that curl.exe POST to /api/runs/{runId}/status with X-Run-Token.
   The matcher regex, event-name casing, and the curl quoting MUST be confirmed
   against the current Claude Code hooks docs by running a real Explain
   generation and observing run-progress events on /api/events + the session
   reaching "completed" on Stop. Adjust the keys in writeHookSettings if the
   real schema differs.

2. Wiring the store into the launch call sites.
   routes_sessions.go does NOT yet pass RunProgress: s.runProgress into the
   claudelauncher.LaunchRequest (out of Phase C task scope — the plan's Task 3
   file list is launcher.go only). Until that 1-line wiring lands on both
   claudelauncher.Launch call sites, req.RunProgress stays nil, no settings
   file is written, and Claude runs keep the Phase-A indeterminate bar.
   This is the integration step for the live validation above.

3. Windows shell + UTF-8 hook command.
   Per the project's Windows + UTF-8 rule (CLAUDE.md §"git-bash 中文乱码"),
   only ASCII currently goes on the curl command line; Chinese activity text
   would be mangled via argv. Confirm on a real Windows machine whether the
   hook runs under cmd or PowerShell and whether richer activity payloads
   need a stdin/file channel instead of -d.
```

### Tests

```text
backend  go test ./backend-go/... PASS (runprogress + server run-status + claudelauncher writeHookSettings/--settings)
frontend npm run test  PASS (useRunProgress run-progress subscription + reset; RunProgressBar determinate "N / M 页")
         npm run build OK
```
