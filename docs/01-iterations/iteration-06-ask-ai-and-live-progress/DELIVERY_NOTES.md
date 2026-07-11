# Iteration 06 Delivery Notes

Status: in progress (Phase C landed)
Last reviewed: 2026-07-08

Iteration 06 is approved as a design and phased A -> B -> C. Delivery has not
begun. Update this file per phase as work lands.

## Planned Phases

```text
Phase A (Theme 2 core)   fsnotify watcher -> kill polling; run-progress indeterminate bar; wire completed/failed
Phase B (Theme 1)        ask-ai config + settings + confusion-tied streaming endpoint + anchored floating window; multi-turn 持久化; 关闭自动生成 ≤250 字总结; 侧栏悬停查看; 调回只读; confusion -> 'asked'
Phase C (Theme 2 enhancement) Claude Code hooks -> granular progress + reliable completion
```

## Patch Round (2026-07-08) - compatibility, Ask-AI UX, and UI polish

### What landed

```text
backend-go/internal/agentruntime/        runtime resolution now falls back to WSL for Claude/Codex on Windows
backend-go/internal/claudelauncher/      interactive/headless launch paths understand WSL mode; WSL reads prompt.md
backend-go/internal/workspace/           corrupt JSON backup helper for local store recovery
confusionstore/folderstore/practicestore corrupt JSON now backs up to .corrupt-*.json and falls back safely
frontend Ask-AI                          selected text pre-fills the explain-selected-text prompt; browser search moved to the selection bar;
                                         floating panel is draggable/resizable and viewport-clamped
frontend Knowledge Flower                large glyph reduced to 75%; five dimension guide added beside the flower
frontend Home/Settings/Practice           denser spacing, smaller radii, less empty table-like space
```

### Follow-up patch (2026-07-08) - selection/Ask-AI placement and icons

```text
Explain selection toolbar              rendered through document.body and measured before final placement, so bottom-edge selections stay visible
Ask-AI active window                   opens beside the selected text when possible, avoids covering the source quote, focuses the prefilled prompt
Ask-AI review window                   opens centered and clamped inside the viewport instead of requiring a manual drag from a clipped edge
frontend icon sprite                   Icon now references #ico-* symbols; missing zone icons filled from the local SVG sprite
```

### Research basis

```text
Claude Code official docs: macOS/Linux/WSL install is the shell installer; Windows has a PowerShell installer;
interactive usage is `claude`, programmatic usage is `claude -p/--print`.
Codex CLI official docs: WSL install is the shell installer; Windows has a PowerShell installer;
interactive usage is `codex`, noninteractive usage is `codex exec`; WSL2 is recommended.
Microsoft WSL docs: Windows can launch Linux commands through wsl.exe and path ownership should match the toolchain.
```

### Validation

```text
go test ./backend-go/...        PASS
npm run test -- --run           PASS (existing React act warnings remain in ExplainInfographic tests)
npm run build                   PASS
headless visual smoke           PASS (Home, Settings, Extend, Practice at 1280px: no horizontal overflow)
follow-up frontend test/build    PASS (Vitest 25 files / 85 tests; production build)
git diff --check                 PASS (CRLF warnings only)
```

### Residual manual checks

```text
Real WSL launch was researched against official docs and implemented best-effort from Windows, but still needs
manual validation on a machine where Claude/Codex are installed only inside WSL.
Visual acceptance still needs a browser pass on the user's target screen sizes.
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
