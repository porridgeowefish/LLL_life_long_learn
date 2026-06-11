# Delivery Notes

Status: delivered

## Delivered

```text
Go backend (backend-go/) replacing iter-01 Node.js implementation
  - HTTP entry point (cmd/lll/main.go)
  - Layered packages: paths, httpx, server, workspace, projectindex,
    agentregistry, promptassembly, memorystore, sessionstore,
    claudelauncher, artifactwriter
  - Single-binary deployment via `go build -o dist/lll.exe`

Filesystem-first project model
  - projects/<slug>/ with project.md, state.json, memory/, 5 zone folders,
    runs/_index/, assets/, subprojects/
  - Atomic file writes via sibling temp + rename
  - Slug collision returns HTTP 409
  - Path traversal protection on every {id}/{zone} parameter

Five learning zones with dependency graph
  - Intro → Explain → Practice + Extend → Summary
  - ResolvePredecessorFiles exposes the graph to prompt assembly

Agent registry + first production agent (Explain Agent)
  - agents/registry/explain.json + agents/charters/explain.md
  - Loaded from disk; zones validated
  - POST /api/agents/{id}/invoke assembles prompt + launches real Claude

Session/turn model with append-only history
  - State machine: preparing → launching → running → completed|failed|cancelled
  - Turns persisted in-memory and indexed under runs/_index/
  - POST /api/sessions/{id}/follow-up re-launches Claude with prior
    result.md paths as context (transitional; see Open Risks)

Raw runs vs curated artifacts
  - runs/<timestamp>-<agent>/ contains prompt.md, package.json,
    stdout.log, stderr.log, result.md, run.json
  - artifactwriter promotes result.md to zone folders (e.g., explain/output.md)
  - summary/summary.md is learner-protected; runtime never passes force=true

Project memory
  - memory/project-memory.md and project-state.json created with project
  - Read-only this slice (no auto-update); paths surfaced in prompt

Extended SSE events
  - session-created, session-state, turn-created, terminal-output,
    artifact-updated (emitted on promote), session-completed, session-failed

Frontend (v2 design + new wiring)
  - index.html (dashboard), project.html (sidebar + reading panel + drawer),
    agents.html (registry), memory.html (file viewer)
  - frontend/js/{api,sse,tree,project-view,markdown-render,app-project}.js
  - frontend/css/styles.css shared
  - Lora Italic LifeLongLearn logo (orange Long) in topbar

Test suite (Go stdlib testing)
  - internal/workspace/*_test.go — skeleton, atomicity, summary refusal,
    predecessor resolution, slugify, slug validation
  - internal/agentregistry/*_test.go — load + zone validation
  - internal/promptassembly/*_test.go — happy path + zone mismatch
  - internal/sessionstore/*_test.go — append-only invariant
  - `go test ./...` is green
```

## Decision: Go rewrite

```text
iter-01 was Node.js. We rewrote in Go 1.26 for these reasons:
  - Single binary deployment (no Node runtime install needed)
  - Native Windows executable, simpler CLI orchestration
  - Superior concurrency for SSE fan-out via goroutines + channels
  - Type-safe filesystem operations (no callback discipline burden)
  - Long-term foundation for iter-03+ PTY integration

The tradeoff accepted: ~315 LOC of working iter-01 reference code was
discarded. We kept the Claude CLI invocation pattern (--verbose
stream-json --include-partial-messages --permission-mode) but rewrote
everything else from scratch.

Alternatives considered:
  - Python + FastAPI + asyncio: 7/10 confidence, ecosystem strong but
    deployment story weaker on Windows.
  - Node.js + TypeScript: 9/10 confidence, but type safety alone did not
    address the Windows child_process pain points that motivated the move.
```

## Open Risks

```text
Follow-up via re-launch (not real PTY continuity)
  iter-02 follow-ups spawn a new Claude process per turn, with prior
  result.md paths injected as prompt context. This is NOT true session
  continuity — Claude cannot maintain in-memory state across the
  boundary. iter-03 should upgrade to PTY-backed sessions
  (Windows conpty + node-pty equivalent in Go).

In-memory session state lost on server restart
  Sessions are persisted to runs/_index/<sessionId>.json on disk, but
  the index itself is not rebuilt from disk on startup. iter-03 should
  load runs/_index/*.json during New() to support full recovery.

SSE backpressure handling is minimal
  Buffered channel size 64; overflow drops silently. Acceptable for a
  single-user local tool. Would need proper flow control for multi-user.

No structured logging yet
  println() is used for request logs. iter-03 should adopt log/slog or
  similar.

CREATE_NEW_CONSOLE is Windows-only
  iter-02.1 introduced a separate PowerShell window (Windows flag
  0x10) for observing Claude execution. Linux/macOS equivalent
  (xterm / Terminal.app) is deferred to iter-03.
```

## iter-02.1 Patch (Mock Alignment + UX Repair)

```text
After iter-02 acceptance, user feedback exposed a top-level design
mismatch: the implementation was API-wired but did not match the
mock product spec. iter-02.1 closes that gap:

  - Wave 1 (overview page): sidebar with shortcut entries / recent
    projects / active sessions; dash-stats hero (projects / sessions
    / turns / active); grid-2 project tiles with empty-state honest
    guidance; running-card + recent-runs list. Backend gained
    GET /api/sessions?recent|active and stats field on /api/health.
  - Wave 2 (rich create): modal form with why / current / target /
    standard. POST /api/projects schema extended; project.md seeds
    these as full sections (not optional placeholders).
  - Wave 3 (core interaction):
    * P0-1: main panel now renders the persisted artifact
      (explain/output.md) instead of Claude's live terminal text.
      artifactwriter.Promote skips overwriting files Claude authored
      under acceptEdits (size >200B heuristic).
    * P0-2: separate PowerShell window opens per invocation, tailing
      stdout.log. Backends SysProcAttr.CreationFlags = 0x10 on
      Windows; non-Windows is a no-op (iter-03).
  - Wave 4 (agents + memory pages): three-pane agents.html with
    detail panel; memory.html gains sidebar + learner-owned edit
    (POST /files/projects/{id}/memory/<filename>, restricted).
  - Wave 5 (cleanup): removed all "deepthink" references from
    agents/ and backend code (1 line in explain.json + 1 comment);
    agents/charters/explain.md was already clean.

Emoji policy: deleted decorative 🏠🤖🧠📁📂🏃📎➕; replaced
buttons/state marks with line-style SVG icons (sprites in
frontend/img/icons.svg + inline <symbol> per page). Zone-step emoji
(🌱/📖/✏️/🔗/📝) preserved as LEARNING_PROJECT_STRUCTURE.md anchors.

Layout policy: project.html uses main-wide (no max-width) so AI
output fills the panel. Zone progress moved to sidebar as a vertical
dot-line timeline with done/current/pending states.

CSS + SVG sprite: styles.css grew by 11 shared component classes
(.dash-stats, .grid-2, .zone-timeline, .agent-detail, .mem-card,
.modal, etc.). icons.svg ships 10 line icons inheriting currentColor.

Tests still green: `go test ./...` passes after all changes.
```

## Deviations From API_CONTRACT.md

```text
- GET /api/projects/:id/zones/:zone returns { zone, predecessors };
  contract did not specify the response body shape beyond "key files,
  artifact references, and latest session references." Artifact +
  session references are deferred to iter-03.
- POST /api/agents/:id/invoke returns 201 with { session, runDir }
  before Claude exits. Contract specified the orchestration steps but
  not the synchronous response shape; this was the natural choice for
  streaming UX.
- POST /api/sessions/:id/follow-up returns 202 with { sessionId, runDir }
  and runs the actual Claude launch asynchronously.
- GET /files/projects/{id}/... was added (not in the original contract)
  as a minimal file-read endpoint for the frontend to display zone
  outputs and memory files.
```
