# Iteration 06 — Phase C: Claude Code Hooks + Reliable Completion — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add granular run progress and reliable session completion for Claude runs by injecting Claude Code hooks (`PostToolUse` / `Stop`) that report to a new run-status endpoint, and show a determinate progress bar from those reports.

**Architecture:** New `runprogress` package holds an in-memory `runId → Status` map plus per-run auth tokens. `POST /api/runs/{runId}/status` (`X-Run-Token`) updates it, emits `run-progress`, and on `done:true` marks the session completed + emits `session-completed` (fixing the "sessions stay running forever" bug for Claude runs). The launcher generates a token per run and injects run-scoped Claude Code hooks that POST activity on page writes and completion. The frontend subscribes to `run-progress` for a determinate bar. Non-Claude runtimes are unaffected (they keep the Phase-A fsnotify fallback). Builds on Phase A (`run-progress` concept, `RunProgressBar`) and Phase B.

**Tech Stack:** Go 1.25.0, net/http, fsnotify (Phase A), `internal/sessionstore`, `internal/claudelauncher`, React 18 + TanStack Query + Zustand + the Phase-A `useRunProgress`/`RunProgressBar`.

**Two pending unknowns (resolve on a real Windows machine in Task 3 before finalizing the hook command):**
1. Does the `claude` CLI accept `--settings <file>` to load a run-scoped settings file? (If not, fall back to a documented one-time global hook install.)
2. What shell does Claude Code run hook commands in on Windows (cmd vs PowerShell), and how to pass Chinese `activity` text safely (per the project's Windows + shell + UTF-8 rule: text via stdin/file, never argv)?

## Global Constraints

- `run-progress` is the event name (avoid the taken `progress`). Payload: `{ runId, phase?, activity?, pagesDone?, pagesPlanned?, done? }`.
- The status endpoint authenticates with a per-run random token (`X-Run-Token`); only the launched run may report.
- `done:true` transitions the session to `completed` and emits `session-completed` — the reliable completion signal Phase A lacked.
- Non-Claude runtimes get NO hooks; they rely on Phase A's fsnotify page-level progress + indeterminate bar. Do not break them.
- Backend tests: `go test ./backend-go/...` (repo root). Frontend: `cd frontend && npm run test`.

## File Structure

- Create `backend-go/internal/runprogress/store.go` (+ `_test.go`).
- Create `backend-go/internal/server/routes_runstatus.go`.
- Modify `backend-go/internal/server/router.go` (register route + store on Server).
- Modify `backend-go/internal/claudelauncher/launcher.go` (token + hook injection).
- Modify `backend-go/internal/sessionstore/...` only if `SetFinished`'s signature differs from the assumed `SetFinished(id, state, exitCode)` (validate in Task 0).
- Modify `frontend/src/lib/constants.ts` + `frontend/src/hooks/useSSE.ts` (add `runProgress`).
- Modify `frontend/src/hooks/useRunProgress.ts` + `RunProgressBar.tsx` (determinate mode).

---

### Task 0: Confirm sessionstore + launcher signatures (read-only)

**Files:** read only — `backend-go/internal/sessionstore/*.go`, `backend-go/internal/claudelauncher/launcher.go`.

- [ ] **Step 1:** Confirm `sessionstore.SetFinished` exists and its exact signature. The exploration found `SetFinished(id, state, exitCode)` called in `sessionstore/store_test.go:65` and `State` enum `preparing|launching|running|completed|failed|cancelled` (`session.go:13-21`). Record the real signature; if it differs, adjust Task 2's call.
- [ ] **Step 2:** Confirm how `claudelauncher.Launch` builds its args/wrapper (`launcher.go:90-225`) and where the `runId`/session id is known, so Task 3 can generate a token and inject `--settings <file>` (or the fallback) at the right point.
- [ ] **Step 3:** On a real Windows machine, confirm whether `claude --help` lists a `--settings` flag, and what shell a hook command runs in. Record findings in `DELIVERY_NOTES.md`.

---

### Task 1: `runprogress` store (TDD)

**Files:**
- Create: `backend-go/internal/runprogress/store.go`
- Test: `backend-go/internal/runprogress/store_test.go`

**Interfaces:**
- Produces: `type Status`, `type Store`, `func New() *Store`, `func (s *Store) Register(runId string) string` (returns token), `func (s *Store) Set(runId, token string, in Status) (Status, bool)` (validates token; merges non-zero fields; returns updated + ok), `func (s *Store) Get(runId string) (Status, bool)`.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/runprogress/store_test.go
package runprogress

import "testing"

func TestSetValidatesToken(t *testing.T) {
	s := New()
	tok := s.Register("r1")
	st, ok := s.Set("r1", "wrong", Status{Activity: "x"})
	if ok {
		t.Errorf("wrong token must be rejected")
	}
	st, ok = s.Set("r1", tok, Status{Activity: "wrote p", PagesDone: 2, PagesPlanned: 5})
	if !ok || st.Activity != "wrote p" || st.PagesDone != 2 {
		t.Errorf("valid set failed: %+v ok=%v", st, ok)
	}
	got, has := s.Get("r1")
	if !has || got.PagesPlanned != 5 {
		t.Errorf("get failed: %+v", got)
	}
	// unknown runId rejects even with empty token
	if _, ok := s.Set("nope", "", Status{}); ok {
		t.Errorf("unknown runId must reject")
	}
}
```

- [ ] **Step 2: Run to fail** — `go test ./backend-go/internal/runprogress/` → undefined.

- [ ] **Step 3: Implement**

```go
// backend-go/internal/runprogress/store.go
// Package runprogress holds in-memory, per-run progress reported by Claude Code
// hooks. It is deliberately separate from the gamification progressstore.
package runprogress

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Status is the latest reported state of one run.
type Status struct {
	Phase         string `json:"phase,omitempty"`
	Activity      string `json:"activity,omitempty"`
	PagesDone     int    `json:"pagesDone,omitempty"`
	PagesPlanned  int    `json:"pagesPlanned,omitempty"`
	Done          bool   `json:"done,omitempty"`
	Failed        bool   `json:"failed,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// Store maps runId -> *Status with per-run auth tokens.
type Store struct {
	mu     sync.Mutex
	m      map[string]*Status
	tokens map[string]string
}

func New() *Store {
	return &Store{m: map[string]*Status{}, tokens: map[string]string{}}
}

func newToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Register creates an entry for runId and returns the auth token the launcher
// injects into the run's hooks.
func (s *Store) Register(runId string) string {
	tok := newToken()
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[runId]; !ok {
		s.m[runId] = &Status{}
	}
	s.tokens[runId] = tok
	return tok
}

// Set validates the token and merges non-zero fields of in into the run's
// status. Returns the merged status and ok=false if the token is wrong or the
// run is unknown.
func (s *Store) Set(runId, token string, in Status) (Status, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	want, ok := s.tokens[runId]
	if !ok || want == "" || want != token {
		return Status{}, false
	}
	cur := s.m[runId]
	if in.Phase != "" {
		cur.Phase = in.Phase
	}
	if in.Activity != "" {
		cur.Activity = in.Activity
	}
	if in.PagesDone != 0 {
		cur.PagesDone = in.PagesDone
	}
	if in.PagesPlanned != 0 {
		cur.PagesPlanned = in.PagesPlanned
	}
	if in.Done {
		cur.Done = true
	}
	if in.Failed {
		cur.Failed = true
	}
	cur.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return *cur, true
}

// Get returns a copy of the status for runId.
func (s *Store) Get(runId string) (Status, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.m[runId]
	if !ok {
		return Status{}, false
	}
	return *st, true
}
```

- [ ] **Step 4: Run to pass** — `go test ./backend-go/internal/runprogress/` → PASS.
- [ ] **Step 5: Commit** — `git add backend-go/internal/runprogress/ && git commit -m "feat(runprogress): in-memory per-run status store with token auth"`.

---

### Task 2: `POST /api/runs/{runId}/status` handler + route (TDD)

**Files:**
- Create: `backend-go/internal/server/routes_runstatus.go`
- Modify: `backend-go/internal/server/router.go` (add `runProgress *runprogress.Store` field, init in `New()`, register route)

**Interfaces:**
- Consumes: `runprogress.Store`, `sessionstore.SetFinished` (signature from Task 0), `broadcaster.Emit`, `httpx.ReadJSON/WriteJSON/Error`.
- Produces: `POST /api/runs/{runId}/status` with `X-Run-Token`; on valid update emits `run-progress`; on `done:true` calls `sessionstore.SetFinished(runId, completed, 0)` + emits `session-completed`.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/server/routes_runstatus_test.go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/runprogress"
)

func TestRunStatusRejectsBadToken(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	srv.runProgress.Register("r1")
	body := `{"activity":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader(body))
	req.SetPathValue("runId", "r1")
	req.Header.Set("X-Run-Token", "wrong")
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: want 401, got %d", rec.Code)
	}
}

func TestRunStatusAcceptsAndDoneCompletes(t *testing.T) {
	srv := &Server{runProgress: runprogress.New()}
	tok := srv.runProgress.Register("r1")
	body := `{"activity":"wrote p","pagesDone":3,"pagesPlanned":5,"done":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/runs/r1/status", strings.NewReader(body))
	req.SetPathValue("runId", "r1")
	req.Header.Set("X-Run-Token", tok)
	rec := httptest.NewRecorder()
	srv.handleRunStatus(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Status runprogress.Status `json:"status"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Status.PagesDone != 3 || !resp.Status.Done {
		t.Errorf("status not reflected: %+v", resp.Status)
	}
	// sessionstore.SetFinished is called in production; here we only assert the
	// handler returns the merged status (sessionstore integration is exercised
	// by an end-to-end smoke).
}
```

> Note: if `sessionstore.SetFinished` cannot be called with a non-existent session id without error in unit tests, guard the `done` branch so a missing session doesn't fail the request (log + continue). Confirm the exact behavior in Task 0.

- [ ] **Step 2: Run to fail** — `go test ./backend-go/internal/server/ -run TestRunStatus` → `handleRunStatus` undefined.

- [ ] **Step 3: Implement the handler**

```go
// backend-go/internal/server/routes_runstatus.go
package server

import (
	"net/http"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	"github.com/xmz14/lll/backend-go/internal/runprogress"
	"github.com/xmz14/lll/backend-go/internal/sessionstore"
)

func (s *Server) handleRunStatus(w http.ResponseWriter, r *http.Request) {
	runId := r.PathValue("runId")
	token := r.Header.Get("X-Run-Token")
	var in runprogress.Status
	if err := httpx.ReadJSON(r, &in); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	updated, ok := s.runProgress.Set(runId, token, in)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "bad run token")
		return
	}
	broadcaster.Emit("run-progress", map[string]any{
		"runId":        runId,
		"phase":        updated.Phase,
		"activity":     updated.Activity,
		"pagesDone":    updated.PagesDone,
		"pagesPlanned": updated.PagesPlanned,
		"done":         updated.Done,
	})
	if in.Done {
		// Reliable completion for Claude runs (fixes "sessions stay running").
		// Confirm the SetFinished signature in Task 0; adjust if needed.
		_ = sessionstore.SetFinished(runId, sessionstore.StateCompleted, 0)
		broadcaster.Emit("session-completed", map[string]any{"runId": runId})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": updated})
}
```
(If `sessionstore.StateCompleted` / `SetFinished` names differ per Task 0, use the real names.)

- [ ] **Step 4: Wire the store + route**

In `router.go`: add import `"github.com/xmz14/lll/backend-go/internal/runprogress"`; add field `runProgress *runprogress.Store` to `Server`; in `New()` set `runProgress: runprogress.New()` in the returned struct; register the route:
```go
	mux.HandleFunc("POST /api/runs/{runId}/status", s.handleRunStatus)
```

- [ ] **Step 5: Run to pass** — `go test ./backend-go/internal/server/ -run TestRunStatus` → PASS; `go build ./backend-go/...` OK.
- [ ] **Step 6: Commit** — `git add backend-go/internal/server/routes_runstatus.go backend-go/internal/server/routes_runstatus_test.go backend-go/internal/server/router.go && git commit -m "feat(server): run-status endpoint with token auth + completion"`.

---

### Task 3: Launcher token + hook injection (validate-then-implement)

**Files:**
- Modify: `backend-go/internal/claudelauncher/launcher.go`

**Interfaces:**
- Consumes: the run's session/run id, the `runprogress.Store` (to register + get the token), the wrapper-script builder.
- Produces: when launching a Claude run, the launcher registers the run (gets a token), writes a run-scoped `claude-settings.json` with `PostToolUse` (Write/Edit on `**/explain/pages/*.md`, etc.) and `Stop` hooks that POST to `/api/runs/{runId}/status` with `X-Run-Token`, and passes the settings via `--settings <file>` (or the documented fallback).

- [ ] **Step 1: Validate the two unknowns** (Task 0 Step 3): does `claude --settings <file>` exist? What shell runs hooks on Windows? Record in DELIVERY_NOTES.
- [ ] **Step 2: Write the hook-config builder**

```go
// Inside launcher.go (exact insertion point per Task 0 Step 2). The runId and a
// runprogress.Store reference must be reachable here; if the launcher does not
// currently hold them, thread them through Launch()'s args (smallest change).

// writeHookSettings writes a run-scoped Claude Code settings file that reports
// progress to LLL, and returns its path + the generated token.
func writeHookSettings(runDir, runId string, store *runprogress.Store) (path, token string, err error) {
	token = store.Register(runId)
	base := "http://127.0.0.1:8787/api/runs/" + runId + "/status"
	// Hook command: curl.exe is present on Win10+. Chinese activity text is NOT
	// passed via argv (Windows + UTF-8 rule); only ASCII field names + the token
	// go on the command line, the JSON body is minimal ASCII here.
	post := `curl.exe -s -o /dev/null -X POST ` + base +
		` -H "X-Run-Token: ` + token + `" -H "Content-Type: application/json" -d ` +
		`"{\\"activity\\":\\"page write\\"}"`
	done := `curl.exe -s -o /dev/null -X POST ` + base +
		` -H "X-Run-Token: ` + token + `" -H "Content-Type: application/json" -d ` +
		`"{\\"done\\":true}"`
	settings := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []map[string]any{{
				"matcher": "Write|Edit",
				"hooks": []map[string]any{{"type": "command", "command": post}},
			}},
			"Stop": []map[string]any{{
				"hooks": []map[string]any{{"type": "command", "command": done}},
			}},
		},
	}
	path = filepath.Join(runDir, "claude-settings.json")
	data, _ := json.MarshalIndent(settings, "", "  ")
	err = os.WriteFile(path, data, 0o644)
	return path, token, err
}
```
> The `PostToolUse` matcher + exact hook schema must match the Claude Code hooks contract — confirm against current Claude Code docs during Step 1 and adjust the keys (e.g. `matcher` regex, event name casing) accordingly. If `--settings` is unsupported, instead document a one-time global hook in DELIVERY_NOTES and skip per-run injection (Phase A's fsnotify still gives page-level progress).

- [ ] **Step 3: Pass `--settings <file>` to the Claude invocation** at the point the wrapper builds the `claude` command (`launcher.go:~450` per exploration). Append `--settings <runDir>/claude-settings.json` to the args when the runtime is Claude and a settings file was written.
- [ ] **Step 4: Build + smoke** — `go build ./backend-go/...`; run a real Explain generation and confirm `run-progress` events arrive on `/api/events` and the session reaches `completed`. Record results + the resolved hook syntax in DELIVERY_NOTES.
- [ ] **Step 5: Commit** — `git add backend-go/internal/claudelauncher/launcher.go && git commit -m "feat(launcher): inject Claude Code hooks for run progress + completion"`.

---

### Task 4: Frontend `run-progress` event + determinate bar (TDD)

**Files:**
- Modify: `frontend/src/lib/constants.ts` (add `runProgress`)
- Modify: `frontend/src/hooks/useSSE.ts` (add to fanout)
- Modify: `frontend/src/hooks/useRunProgress.ts` (subscribe `run-progress`; expose `pagesDone`/`pagesPlanned`)
- Modify: `frontend/src/components/feature/project/RunProgressBar.tsx` (determinate mode when planned > 0)

**Interfaces:**
- Produces: `useRunProgress` returns `{ active, activity, pagesDone, pagesPlanned, dismiss }`; `RunProgressBar` renders a determinate bar "N / M 页" when `pagesPlanned > 0`, else the Phase-A indeterminate bar.

- [ ] **Step 1:** In `constants.ts` add `runProgress: 'run-progress'` to `SSE_EVENTS`. In `useSSE.ts` add `SSE_EVENTS.runProgress` to `fanoutNames`.
- [ ] **Step 2:** In `useRunProgress.ts`, subscribe to `run-progress` for the active session's runId and store `pagesDone`/`pagesPlanned`/`activity`; return them. (The runId correlation: the active session id from `useSessionStore` matches the `runId` in the payload.)
- [ ] **Step 3:** Update `RunProgressBar` to take `pagesDone`/`pagesPlanned` and render determinate when `pagesPlanned > 0`.
- [ ] **Step 4:** Test — extend `useRunProgress.test.tsx`: fire a `run-progress` event with `{runId, pagesDone:3, pagesPlanned:5}` for the active session → assert the hook exposes those; and `RunProgressBar.test.tsx`: determinate text "3 / 5 页" when planned > 0.
- [ ] **Step 5:** `cd frontend && npm run test && npm run build`.
- [ ] **Step 6:** Commit — `git add frontend/src/lib/constants.ts frontend/src/hooks/useSSE.ts frontend/src/hooks/useRunProgress.ts frontend/src/hooks/useRunProgress.test.tsx frontend/src/components/feature/project/RunProgressBar.tsx frontend/src/components/feature/project/RunProgressBar.test.tsx && git commit -m "feat(frontend): determinate run-progress bar from hooks"`.

---

## End-to-End Smoke (Phase C)

1. `go run ./backend-go/cmd/lll` + `cd frontend && npm run dev`.
2. Trigger an Explain generation. With hooks injected: `run-progress` events stream on `/api/events`; the bar shows "N / M 页" as pages are written; on Claude `Stop`, `session-completed` fires and the bar hides; the session is no longer stuck in `running`.
3. A non-Claude runtime (e.g. codex) still gets fsnotify page-level refresh + indeterminate bar (Phase A) — no regression.

## Notes / Honest Limitations (Phase C)

- Hook command + `--settings` are validated live in Task 3; the command shown is the best Windows-curl form and must be confirmed against current Claude Code hooks docs.
- Only Claude runs get hooks; other runtimes keep the Phase-A indeterminate fallback by design.
- `runprogress` is in-memory; status is lost on server restart (acceptable — it is transient).
