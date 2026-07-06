# Iteration 06 — Phase A: Live Refresh Foundation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace frontend file-polling with event-driven refresh via an fsnotify watcher, and ship an indeterminate run-progress bar driven by those events — the runtime-agnostic, lowest-risk first slice of iteration-06.

**Architecture:** A new Go package `artifactwatch` watches `paths.PROJECTS_ROOT` with fsnotify, maps each zone-folder file write to `{projectSlug, zone, path}`, debounces it, and emits `artifact-updated` on the existing package-global SSE `Broadcaster`. The frontend subscribes to `artifact-updated`/`confusion-updated` in a single hook that invalidates the right TanStack Query keys, so the `refetchInterval` polls can be deleted. A presentational `RunProgressBar` + a `useRunProgress` hook surface live activity. Determinate per-page progress and reliable run-completion arrive in Phase C (Claude Code hooks); Phase A is intentionally indeterminate.

**Tech Stack:** Go 1.25.0 (`github.com/xmz14/lll`), `github.com/fsnotify/fsnotify` (new — first external Go dep), React 18 + TypeScript, TanStack Query, Zustand, Vitest + @testing-library/react.

## Global Constraints

- Module `github.com/xmz14/lll`; Go 1.25.0. `fsnotify` is the project's first external Go dependency — `go get` will create `go.sum`.
- File-first, no database (per `docs/00-product-and-architecture/REPOSITORY_MAP.md`).
- Zone folders are exactly `intro`, `explain`, `practice`, `extend`, `summary` (see `backend-go/internal/workspace/workspace.go:215`). `runs`, `memory`, `assets`, `progress` are NOT zones and must be ignored.
- The progress concept is named `run-progress` to avoid collision with the existing gamification `progress` (`backend-go/internal/progressstore`). Phase A does not emit `run-progress` yet (no hooks); it only adds the presentational bar.
- SSE event names live in `frontend/src/lib/constants.ts` (`SSE_EVENTS`); the fanout in `frontend/src/hooks/useSSE.ts` ALREADY routes `artifact-updated`, `confusion-updated`, `session-completed`, `session-failed` — Phase A adds subscribers, not fanout entries.
- The `artifact-updated` payload emitted by this phase is `{ projectSlug: string, zone: string, path: string }`. It coexists with the existing infographic emit (`{ slug, artifact }`, `routes_explain.go:173`) because each subscriber only reads the keys it cares about.
- Windows-primary; fsnotify works via ReadDirectoryChangesW. Use forward slashes in emitted `path` (`filepath.ToSlash`).
- Backend tests: `go test ./backend-go/...`. Frontend tests: `npm run test`. Build: `cd frontend && npm run build`.

## File Structure

Backend (new package + wiring):
- Create `backend-go/internal/artifactwatch/zone.go` — `parseZonePath` pure helper.
- Create `backend-go/internal/artifactwatch/watcher.go` — fsnotify `Watcher`, debounce, emit.
- Create `backend-go/internal/artifactwatch/zone_test.go`, `watcher_test.go`.
- Modify `backend-go/internal/server/router.go` — `watcher` field on `Server`, start in `New()`, `Close()`.
- Modify `backend-go/cmd/lll/main.go` — `defer srv.Close()`.
- Modify `go.mod` / create `go.sum` — add `github.com/fsnotify/fsnotify`.

Frontend (event-driven refresh + progress bar):
- Create `frontend/src/hooks/useArtifactRefresh.ts` (+ `.test.tsx`) — SSE→invalidate.
- Create `frontend/src/components/feature/project/RunProgressBar.tsx` (+ `.module.css`, `.test.tsx`) — presentational bar.
- Create `frontend/src/hooks/useRunProgress.ts` (+ `.test.tsx`) — activity + visibility.
- Modify `frontend/src/api/learningArtifacts.ts` — drop 5s poll.
- Modify `frontend/src/api/practice.ts` — drop evaluation poll.
- Modify `frontend/src/components/feature/practice/PracticeFlow.tsx` — drop tasks poll option + import.
- Modify `frontend/src/pages/ProjectPage.tsx` — mount hooks + bar.

---

### Task 1: `parseZonePath` pure helper (TDD)

**Files:**
- Create: `backend-go/internal/artifactwatch/zone.go`
- Test: `backend-go/internal/artifactwatch/zone_test.go`

**Interfaces:**
- Produces: `func parseZonePath(rel string) (slug, zone string, ok bool)` — `slug` uses `/` separators and matches the frontend project slug (nested subprojects are `parent/subprojects/child`); `ok=false` when no zone folder appears, or when an ignored structural folder (`runs`/`memory`/`assets`/`progress`) appears before the zone.

- [ ] **Step 1: Write the failing test**

```go
// backend-go/internal/artifactwatch/zone_test.go
package artifactwatch

import "testing"

func TestParseZonePath(t *testing.T) {
	cases := []struct {
		rel      string
		wantSlug string
		wantZone string
		wantOK   bool
	}{
		{"myproj/explain/pages/01.md", "myproj", "explain", true},
		{"myproj/explain/manifest.json", "myproj", "explain", true},
		{"p/subprojects/c/practice/tasks.json", "p/subprojects/c", "practice", true},
		{"abc/intro/output.md", "abc", "intro", true},
		{"abc/summary/summary.md", "abc", "summary", true},
		// ignored structural folders before a zone -> not a zone artifact
		{"abc/runs/2026-x/explain/result.md", "", "", false},
		{"abc/memory/note.md", "", "", false},
		{"abc/progress/summary.json", "", "", false},
		// no zone at all
		{"README.md", "", "", false},
		// windows backslashes
		{"myproj\\explain\\pages\\01.md", "myproj", "explain", true},
	}
	for _, c := range cases {
		slug, zone, ok := parseZonePath(c.rel)
		if slug != c.wantSlug || zone != c.wantZone || ok != c.wantOK {
			t.Errorf("parseZonePath(%q) = (%q,%q,%t), want (%q,%q,%t)",
				c.rel, slug, zone, ok, c.wantSlug, c.wantZone, c.wantOK)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./backend-go/internal/artifactwatch/`
Expected: build failure — `parseZonePath undefined` (and `zone.go` does not exist yet).

- [ ] **Step 3: Write minimal implementation**

```go
// backend-go/internal/artifactwatch/zone.go
package artifactwatch

import "strings"

// zones is the set of folder names that hold generated artifacts.
var zones = map[string]bool{
	"intro":   true,
	"explain": true,
	"practice": true,
	"extend":  true,
	"summary": true,
}

// ignoreFolders are structural project folders that are NOT zones. If one
// appears before a zone segment, the path is inside that structural folder
// (e.g. runs/<ts>/explain/result.md) and must not be treated as a zone write.
var ignoreFolders = map[string]bool{
	"runs":     true,
	"memory":   true,
	"assets":   true,
	"progress": true,
}

// parseZonePath maps a path relative to the projects root to the owning project
// slug and the zone whose artifact changed. ok is false when no zone folder
// appears or when an ignored structural folder precedes the zone. Separators are
// normalized to "/", so slug uses "/" (matching the frontend slug convention).
func parseZonePath(rel string) (slug, zone string, ok bool) {
	rel = strings.ReplaceAll(rel, "\\", "/")
	parts := strings.Split(rel, "/")
	for i, seg := range parts {
		if ignoreFolders[seg] {
			return "", "", false
		}
		if zones[seg] {
			return strings.Join(parts[:i], "/"), seg, true
		}
	}
	return "", "", false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./backend-go/internal/artifactwatch/`
Expected: `ok` (PASS).

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/artifactwatch/zone.go backend-go/internal/artifactwatch/zone_test.go
git commit -m "feat(artifactwatch): add parseZonePath helper"
```

---

### Task 2: fsnotify watcher with debounce + emit (TDD)

**Files:**
- Create: `backend-go/internal/artifactwatch/watcher.go`
- Test: `backend-go/internal/artifactwatch/watcher_test.go`
- Modify: `go.mod`, `go.sum` (add `github.com/fsnotify/fsnotify`)

**Interfaces:**
- Consumes: `parseZonePath` from Task 1.
- Produces:
  - `type EmitFunc func(event string, payload any)`
  - `type ArtifactPayload struct { ProjectSlug, Zone, Path string }`
  - `func Start(root string, emit EmitFunc) (*Watcher, error)`
  - `func (w *Watcher) Close() error`
  - Emits SSE event `"artifact-updated"` with `ArtifactPayload` (JSON tags `projectSlug`,`zone`,`path`).

- [ ] **Step 1: Add the fsnotify dependency**

Run from the repo root (`D:/2_Study/lll`, where `go.mod` lives):
```bash
go get github.com/fsnotify/fsnotify@latest
```
Expected: the repo-root `go.mod` gains a `require` block; `go.sum` is created at the repo root. (This command is ASCII-only, so it is safe under git-bash on Windows.)

- [ ] **Step 2: Write the failing test**

```go
// backend-go/internal/artifactwatch/watcher_test.go
package artifactwatch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// shortenDebounce makes debounce deterministic and fast in tests.
func shortenDebounce(t *testing.T) {
	t.Helper()
	debounceWindow = 5 * time.Millisecond
}

func TestWatcherEmitsOnZoneFileWrite(t *testing.T) {
	shortenDebounce(t)
	root := t.TempDir()
	// Pre-create the project zone tree so Start's walk already covers it
	// (avoids the new-directory race).
	projDir := filepath.Join(root, "myproj", "explain", "pages")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var got []ArtifactPayload
	emit := func(event string, payload any) {
		if event != "artifact-updated" {
			return
		}
		mu.Lock()
		got = append(got, payload.(ArtifactPayload))
		mu.Unlock()
	}

	w, err := Start(root, emit)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Close()

	if err := os.WriteFile(filepath.Join(projDir, "01.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // past debounce + fsnotify latency

	mu.Lock()
	defer mu.Unlock()
	want := ArtifactPayload{ProjectSlug: "myproj", Zone: "explain", Path: "myproj/explain/pages/01.md"}
	var found bool
	for _, p := range got {
		if p == want {
			found = true
		}
	}
	if !found {
		t.Errorf("expected payload %+v among %+v", want, got)
	}
}

func TestWatcherDebouncesBursts(t *testing.T) {
	shortenDebounce(t)
	root := t.TempDir()
	projDir := filepath.Join(root, "p1", "practice")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var count int
	emit := func(event string, _ any) {
		if event == "artifact-updated" {
			mu.Lock()
			count++
			mu.Unlock()
		}
	}
	w, err := Start(root, emit)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Close()

	// Three rapid writes to the same (slug,zone) should collapse to one emit.
	for i := 0; i < 3; i++ {
		_ = os.WriteFile(filepath.Join(projDir, "tasks.json"), []byte("x"), 0o644)
		time.Sleep(1 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected exactly 1 debounced emit, got %d", count)
	}
}

func TestWatcherIgnoresNonZonePaths(t *testing.T) {
	shortenDebounce(t)
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "p1", "runs"), 0o755)
	var count int
	emit := func(event string, _ any) {
		if event == "artifact-updated" {
			count++
		}
	}
	w, err := Start(root, emit)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Close()

	_ = os.WriteFile(filepath.Join(root, "p1", "runs", "stdout.log"), []byte("x"), 0o644)
	time.Sleep(80 * time.Millisecond)
	if count != 0 {
		t.Errorf("expected no emit for runs/ write, got %d", count)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./backend-go/internal/artifactwatch/`
Expected: build failure — `Start`, `Watcher`, `ArtifactPayload`, `debounceWindow` undefined.

- [ ] **Step 4: Write minimal implementation**

```go
// backend-go/internal/artifactwatch/watcher.go
package artifactwatch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// debounceWindow collapses bursts of events for the same (slug,zone) into one
// emit. It is a package var so tests can shorten it.
var debounceWindow = 150 * time.Millisecond

// EmitFunc delivers an SSE event. It matches httpx.Broadcaster.Emit.
type EmitFunc func(event string, payload any)

// ArtifactPayload is the payload emitted on artifact-updated.
type ArtifactPayload struct {
	ProjectSlug string `json:"projectSlug"`
	Zone        string `json:"zone"`
	Path        string `json:"path"`
}

// Watcher watches a projects root and emits "artifact-updated" SSE events when
// files under a zone folder change. Runtime-agnostic: any process writing files
// (Claude, Codex, ...) triggers a refresh.
type Watcher struct {
	fw   *fsnotify.Watcher
	root string
	emit EmitFunc
	stop chan struct{}
	done chan struct{}

	mu      sync.Mutex
	pending map[string]*time.Timer // keyed by slug + "\x00" + zone
}

// Start creates a Watcher over root, adds root and all existing subdirectories
// recursively, and begins watching. New directories created later are added on
// the fly. Call Close to stop.
func Start(root string, emit EmitFunc) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		fw:      fw,
		root:    root,
		emit:    emit,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
		pending: make(map[string]*time.Timer),
	}
	if err := w.addDir(root); err != nil {
		_ = fw.Close()
		return nil, err
	}
	go w.loop()
	return w, nil
}

// addDir recursively adds dir and its subdirectories to the watcher. Read errors
// are skipped (best-effort; the projects root always exists at Start).
func (w *Watcher) addDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			_ = w.fw.Add(path)
		}
		return nil
	})
}

func (w *Watcher) loop() {
	defer close(w.done)
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.handle(ev)
		case _, ok := <-w.fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) handle(ev fsnotify.Event) {
	// A new directory (e.g. a freshly created project) -> watch it so its
	// future writes are caught.
	if ev.Has(fsnotify.Create) {
		if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
			_ = w.addDir(ev.Name)
			return
		}
	}
	// Only react to file writes/creates.
	if !(ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create)) {
		return
	}
	rel, err := filepath.Rel(w.root, ev.Name)
	if err != nil {
		return
	}
	slug, zone, ok := parseZonePath(rel)
	if !ok {
		return
	}
	w.schedule(slug, zone, filepath.ToSlash(rel))
}

// schedule debounces bursts of events for the same (slug, zone) into one emit.
func (w *Watcher) schedule(slug, zone, rel string) {
	key := slug + "\x00" + zone
	w.mu.Lock()
	defer w.mu.Unlock()
	if t := w.pending[key]; t != nil {
		t.Stop()
	}
	w.pending[key] = time.AfterFunc(debounceWindow, func() {
		w.mu.Lock()
		delete(w.pending, key)
		w.mu.Unlock()
		if w.emit != nil {
			w.emit("artifact-updated", ArtifactPayload{
				ProjectSlug: slug,
				Zone:        zone,
				Path:        rel,
			})
		}
	})
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	close(w.stop)
	err := w.fw.Close()
	<-w.done
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./backend-go/internal/artifactwatch/`
Expected: `ok` — all three tests PASS. (These are timing-based; if a CI box is very slow, bump the `time.Sleep` values, but do not change the assertions.)

- [ ] **Step 6: Commit**

```bash
git add backend-go/internal/artifactwatch/watcher.go backend-go/internal/artifactwatch/watcher_test.go go.mod go.sum
```
(The module root is the repo root, so `go.mod` and `go.sum` live at `D:/2_Study/lll/`.)
```bash
git commit -m "feat(artifactwatch): add fsnotify watcher with debounce + emit"
```

---

### Task 3: Wire the watcher into the server lifecycle

**Files:**
- Modify: `backend-go/internal/server/router.go` (Server struct ~line 22, `New()` ~line 38, add `Close()`)
- Modify: `backend-go/cmd/lll/main.go` (after line 17)

**Interfaces:**
- Consumes: `artifactwatch.Start`, `artifactwatch.Watcher.Close` from Task 2; the package-global `broadcaster` (`router.go:34`) and `paths.PROJECTS_ROOT`.
- Produces: `(*Server).Close()` called by `main` on shutdown.

- [ ] **Step 1: Add the watcher field + import**

In `backend-go/internal/server/router.go`, add the import alongside the others in the import block (lines 4–19):

```go
	"github.com/xmz14/lll/backend-go/internal/artifactwatch"
```

Add a field to the `Server` struct (after `ImageAvailable bool` at line 28):

```go
	ImageAvailable  bool
	watcher         *artifactwatch.Watcher
	shutdown        func()
```

- [ ] **Step 2: Start the watcher in `New()` and add `Close()`**

In `New()`, immediately before the `return &Server{` statement (around line 73), insert:

```go
	// Start the artifact file watcher (event-driven refresh; replaces polling).
	watcher, werr := artifactwatch.Start(paths.PROJECTS_ROOT, broadcaster.Emit)
	if werr != nil {
		println("artifactwatch: start warning:", werr.Error())
	}
```

Then add `watcher: watcher,` to the struct literal being returned (inside the `&Server{ ... }` block, alongside `ImageAvailable: imgAvailable,`):

```go
	return &Server{
		ClaudeBin:       bin,
		ClaudeAvailable: available,
		Runtime:         selectedRuntime,
		RuntimeOptions:  runtimeOptions,
		ImageConfig:     imgCfg,
		ImageAvailable:  imgAvailable,
		watcher:         watcher,
	}
```

Add a new method at the end of the file (after `SetShutdownFunc`):

```go
// Close releases background resources (the artifact file watcher).
func (s *Server) Close() {
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
}
```

- [ ] **Step 3: Call `Close()` from `main`**

In `backend-go/cmd/lll/main.go`, immediately after `srv := server.New()` (line 17), add:

```go
	defer srv.Close()
```

- [ ] **Step 4: Verify build + vet**

Run:
```bash
go build ./backend-go/...
go vet ./backend-go/...
go test ./backend-go/internal/artifactwatch/
```
Expected: build succeeds, vet clean, artifactwatch tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add backend-go/internal/server/router.go backend-go/cmd/lll/main.go
git commit -m "feat(server): start artifact watcher, close on shutdown"
```

---

### Task 4: `useArtifactRefresh` — SSE-driven query invalidation (TDD)

**Files:**
- Create: `frontend/src/hooks/useArtifactRefresh.ts`
- Test: `frontend/src/hooks/useArtifactRefresh.test.tsx`

**Interfaces:**
- Consumes: `subscribeToSSE(eventName, handler)` and `SSE_EVENTS.artifactUpdated`/`confusionUpdated` (both already exist and are fanned out by `useSSE`); `qk.confusions.all(slug)` from `@/api/queryKeys`.
- Produces: `function useArtifactRefresh(projectSlug: string): void` — mount once per active project (ProjectPage).
- Emits (TanStack invalidations): `['files', slug]` (covers all `qk.files.raw`), `['practice','tasks',slug]`, `['practice','evaluation',slug]`, `['practice','draft',slug]`, `['practice','attempt','latest',slug]`, `qk.confusions.all(slug)`.

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/hooks/useArtifactRefresh.test.tsx
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

import { useArtifactRefresh } from './useArtifactRefresh';

// Capture SSE handlers by event name.
let handlers: Record<string, (data: unknown) => void> = {};
vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (name: string, h: (data: unknown) => void) => {
    handlers[name] = h;
    return () => {
      delete handlers[name];
    };
  },
}));

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe('useArtifactRefresh', () => {
  beforeEach(() => {
    handlers = {};
  });

  it('invalidates files + practice on artifact-updated for the slug', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'practice' }));

    const keys = spy.mock.calls.map((c) => (c[0] as { queryKey: unknown }).queryKey);
    expect(keys).toContainEqual(['files', 'myproj']);
    expect(keys).toContainEqual(['practice', 'tasks', 'myproj']);
  });

  it('ignores artifact events for other projects', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['artifact-updated']({ projectSlug: 'other', zone: 'explain' }));
    expect(spy).not.toHaveBeenCalled();
  });

  it('invalidates confusions on confusion-updated', () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const spy = vi.spyOn(client, 'invalidateQueries');
    renderHook(() => useArtifactRefresh('myproj'), { wrapper: wrapper(client) });

    act(() => handlers['confusion-updated']({ projectSlug: 'myproj', id: 'c1' }));
    expect(spy).toHaveBeenCalledWith({ queryKey: ['confusions', 'myproj'] });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- useArtifactRefresh`
Expected: FAIL — module `./useArtifactRefresh` not found.

- [ ] **Step 3: Write minimal implementation**

```ts
// frontend/src/hooks/useArtifactRefresh.ts
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { qk } from '@/api/queryKeys';

function invalidatePractice(
  qc: ReturnType<typeof useQueryClient>,
  slug: string,
) {
  qc.invalidateQueries({ queryKey: ['practice', 'tasks', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'evaluation', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'draft', slug] });
  qc.invalidateQueries({ queryKey: ['practice', 'attempt', 'latest', slug] });
}

/**
 * Drive server-state refresh from SSE instead of polling. Mount once per active
 * project (in ProjectPage). On artifact-updated for this project, invalidate the
 * affected file/practice queries; on confusion-updated, invalidate the
 * confusions list. The 'files' prefix covers explain manifest/pages and the
 * intro/extend/summary outputs read via qk.files.raw.
 */
export function useArtifactRefresh(projectSlug: string): void {
  const qc = useQueryClient();
  useEffect(() => {
    const unsubArtifact = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const p = data as { projectSlug?: string; zone?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      qc.invalidateQueries({ queryKey: ['files', projectSlug] });
      if (p.zone === 'practice') invalidatePractice(qc, projectSlug);
    });
    const unsubConfusion = subscribeToSSE(SSE_EVENTS.confusionUpdated, (data) => {
      const p = data as { projectSlug?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      qc.invalidateQueries({ queryKey: qk.confusions.all(projectSlug) });
    });
    return () => {
      unsubArtifact();
      unsubConfusion();
    };
  }, [projectSlug, qc]);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- useArtifactRefresh`
Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/hooks/useArtifactRefresh.ts frontend/src/hooks/useArtifactRefresh.test.tsx
git commit -m "feat(frontend): add useArtifactRefresh SSE-driven invalidation"
```

---

### Task 5: Delete the polling `refetchInterval` sites

**Files:**
- Modify: `frontend/src/api/learningArtifacts.ts` (line 116)
- Modify: `frontend/src/api/practice.ts` (line 216)
- Modify: `frontend/src/components/feature/practice/PracticeFlow.tsx` (lines 22, 172–173)

**Interfaces:**
- Consumes: the event-driven invalidation from Task 4 (already live after Task 4).
- Produces: no API change — these queries now refresh purely via `artifact-updated`.

- [ ] **Step 1: Remove the explain manifest poll**

In `frontend/src/api/learningArtifacts.ts`, delete the `refetchInterval: 5000,` line inside `useExplainManifest` (line 116). The hook becomes:

```ts
export function useExplainManifest(projectSlug: string) {
  return useQuery({
    queryKey: qk.files.raw(projectSlug, 'explain/manifest.json'),
    retry: false,
    queryFn: () =>
      http.get<ExplainManifest>(
        `/files/projects/${encodeURIComponent(projectSlug)}/explain/manifest.json`,
      ),
  });
}
```

- [ ] **Step 2: Remove the practice evaluation poll**

In `frontend/src/api/practice.ts`, delete the `refetchInterval` line inside `usePracticeEvaluation` (line 216). The hook becomes:

```ts
export function usePracticeEvaluation(projectSlug: string | undefined, attempt: number) {
  return useQuery({
    queryKey: ['practice', 'evaluation', projectSlug, attempt],
    enabled: !!projectSlug && attempt > 0,
    retry: false,
    queryFn: async () => {
      const res = await http.get<{ evaluation: Evaluation }>(
        `/api/projects/${encodeURIComponent(projectSlug!)}/practice/evaluation?attempt=${attempt}`,
      );
      return res.evaluation;
    },
  });
}
```

- [ ] **Step 3: Remove the practice tasks poll at the call site**

In `frontend/src/components/feature/practice/PracticeFlow.tsx`, the call at lines 172–173 passes a poll option. Replace it with the no-option form (Task 4's `useArtifactRefresh` invalidates `['practice','tasks',slug]` on practice writes):

Before:
```tsx
  const tasksData = usePracticeTasks(projectSlug, {
    refetchInterval: state.phase === 'generating' ? PRACTICE_GEN_POLL_MS : false,
  });
```
After:
```tsx
  const tasksData = usePracticeTasks(projectSlug);
```

Then remove the now-unused import on line 22. Change:
```tsx
import { PERMISSION_MODES, PRACTICE_GEN_POLL_MS, PRACTICE_GEN_TIMEOUT_MS } from '@/lib/constants';
```
to:
```tsx
import { PERMISSION_MODES, PRACTICE_GEN_TIMEOUT_MS } from '@/lib/constants';
```
(`PRACTICE_GEN_POLL_MS` is used only at the call site just removed; `PRACTICE_GEN_TIMEOUT_MS` is still used elsewhere in PracticeFlow, so keep it. Confirm with a grep before deleting: `PRACTICE_GEN_POLL_MS` should have zero remaining references in this file.)

- [ ] **Step 4: Verify frontend tests + build**

Run:
```bash
cd frontend && npm run test
cd frontend && npm run build
```
Expected: all tests PASS; production build succeeds (an unused import would fail the TS build, confirming Step 3 cleanup is complete).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api/learningArtifacts.ts frontend/src/api/practice.ts frontend/src/components/feature/practice/PracticeFlow.tsx
git commit -m "refactor(frontend): drop file-polling, rely on SSE artifact-updated"
```

---

### Task 6: `RunProgressBar` presentational component (TDD)

**Files:**
- Create: `frontend/src/components/feature/project/RunProgressBar.tsx`
- Create: `frontend/src/components/feature/project/RunProgressBar.module.css`
- Test: `frontend/src/components/feature/project/RunProgressBar.test.tsx`

**Interfaces:**
- Produces: `function RunProgressBar(props: { active: boolean; activity: string | null; onDismiss?: () => void }): JSX.Element | null`. Renders `null` when `!active`; otherwise an indeterminate bar + `activity` text (fallback `运行中…`) + an optional dismiss button.

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/components/feature/project/RunProgressBar.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

import { RunProgressBar } from './RunProgressBar';

describe('RunProgressBar', () => {
  it('renders nothing when inactive', () => {
    const { container } = render(<RunProgressBar active={false} activity={null} />);
    expect(container.firstChild).toBeNull();
  });

  it('renders the activity text when active', () => {
    render(<RunProgressBar active={true} activity={'最近更新：explain'} />);
    expect(screen.getByText('最近更新：explain')).toBeTruthy();
  });

  it('falls back to the running label when activity is null', () => {
    render(<RunProgressBar active={true} activity={null} />);
    expect(screen.getByText('运行中…')).toBeTruthy();
  });

  it('calls onDismiss when the dismiss button is clicked', () => {
    const onDismiss = vi.fn();
    render(<RunProgressBar active={true} activity={'x'} onDismiss={onDismiss} />);
    fireEvent.click(screen.getByRole('button', { name: '收起进度' }));
    expect(onDismiss).toHaveBeenCalledOnce();
  });
});
```
(Add `import { vi } from 'vitest';` at the top — needed for `vi.fn()`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- RunProgressBar`
Expected: FAIL — module `./RunProgressBar` not found.

- [ ] **Step 3: Write the component + CSS**

```tsx
// frontend/src/components/feature/project/RunProgressBar.tsx
import { Icon } from '@/components/primitive/Icon';

import s from './RunProgressBar.module.css';

export interface RunProgressBarProps {
  active: boolean;
  activity: string | null;
  onDismiss?: () => void;
}

/**
 * Indeterminate run-progress indicator. Shown while a generation is producing
 * artifacts. Determinate per-page/phase progress and reliable completion land in
 * Phase C (Claude Code hooks); for now the bar is indeterminate with the latest
 * activity text. A dismiss (×) lets the learner clear a lingering bar in Phase A
 * (sessions don't reach "completed" until hooks are wired).
 */
export function RunProgressBar({ active, activity, onDismiss }: RunProgressBarProps) {
  if (!active) return null;
  return (
    <div className={s.bar} role="status" aria-live="polite">
      <div className={s.track} aria-hidden="true">
        <div className={s.fill} />
      </div>
      <span className={s.label}>{activity ?? '运行中…'}</span>
      {onDismiss && (
        <button
          type="button"
          className={s.dismiss}
          onClick={onDismiss}
          title="收起"
          aria-label="收起进度"
        >
          <Icon name="x" size={12} />
        </button>
      )}
    </div>
  );
}
```

```css
/* frontend/src/components/feature/project/RunProgressBar.module.css */
.bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  margin-top: 8px;
  background: #f4f5f7;
  border: 1px solid #e6e8eb;
  border-radius: 8px;
  font-size: 12px;
  color: #555;
}
.track {
  position: relative;
  width: 72px;
  height: 4px;
  border-radius: 2px;
  background: rgba(0, 0, 0, 0.08);
  overflow: hidden;
}
.fill {
  position: absolute;
  top: 0;
  left: -40%;
  width: 40%;
  height: 100%;
  border-radius: 2px;
  background: #4c6ef5;
  animation: runpulse 1.1s ease-in-out infinite;
}
.label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dismiss {
  border: none;
  background: transparent;
  cursor: pointer;
  color: inherit;
  display: inline-flex;
  align-items: center;
  padding: 2px;
  border-radius: 4px;
}
.dismiss:hover {
  background: rgba(0, 0, 0, 0.06);
}
@keyframes runpulse {
  0% { left: -40%; }
  100% { left: 100%; }
}
```
(These use literal colors so the bar renders without depending on design-token names; align to the project's tokens in a later polish pass if desired.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- RunProgressBar`
Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/feature/project/RunProgressBar.tsx frontend/src/components/feature/project/RunProgressBar.module.css frontend/src/components/feature/project/RunProgressBar.test.tsx
git commit -m "feat(frontend): add RunProgressBar presentational component"
```

---

### Task 7: `useRunProgress` hook + mount in ProjectPage (TDD)

**Files:**
- Create: `frontend/src/hooks/useRunProgress.ts`
- Test: `frontend/src/hooks/useRunProgress.test.tsx`
- Modify: `frontend/src/pages/ProjectPage.tsx` (imports + header render)

**Interfaces:**
- Consumes: `subscribeToSSE`, `SSE_EVENTS.artifactUpdated`/`sessionCompleted`/`sessionFailed`, `useSessionStore.activeSessionId`, `RunProgressBar` (Task 6).
- Produces: `function useRunProgress(projectSlug: string): { active: boolean; activity: string | null; dismiss: () => void }`. `active` is true only when a session is active, not completed, not dismissed, AND at least one `artifact-updated` for this slug has arrived (so the bar appears only when generation is actually producing output here). Mounted in ProjectPage next to `RunProgressBar`.

- [ ] **Step 1: Write the failing test**

```tsx
// frontend/src/hooks/useRunProgress.test.tsx
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';

import { useRunProgress } from './useRunProgress';
import { useSessionStore } from '@/store/slices/session';

let handlers: Record<string, (data: unknown) => void> = {};
vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (name: string, h: (data: unknown) => void) => {
    handlers[name] = h;
    return () => {
      delete handlers[name];
    };
  },
}));

describe('useRunProgress', () => {
  beforeEach(() => {
    handlers = {};
    useSessionStore.setState({ activeSessionId: null });
  });

  it('is inactive until an artifact event arrives for this slug', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(false);
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    expect(result.current.active).toBe(true);
    expect(result.current.activity).toContain('explain');
  });

  it('ignores artifact events for other projects', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'other', zone: 'explain' }));
    expect(result.current.active).toBe(false);
  });

  it('hides on session-completed', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    expect(result.current.active).toBe(true);
    act(() => handlers['session-completed']({}));
    expect(result.current.active).toBe(false);
  });

  it('dismiss hides the bar', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    act(() => result.current.dismiss());
    expect(result.current.active).toBe(false);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npm run test -- useRunProgress`
Expected: FAIL — module `./useRunProgress` not found.

- [ ] **Step 3: Write minimal implementation**

```ts
// frontend/src/hooks/useRunProgress.ts
import { useEffect, useState } from 'react';

import { subscribeToSSE } from '@/hooks/useSSE';
import { SSE_EVENTS } from '@/lib/constants';
import { useSessionStore } from '@/store/slices/session';

export interface RunProgressState {
  active: boolean;
  activity: string | null;
  dismiss: () => void;
}

/**
 * Phase A progress source: indeterminate, driven by artifact-updated activity
 * for the current project. The bar appears once an artifact event arrives for
 * this slug (generation is actually producing output here), updates its activity
 * text as pages/files land, and hides on session-completed/failed (emitted
 * reliably from Phase C hooks) or on user dismiss. Per-page/phase granularity
 * arrives in Phase C via the run-progress event.
 */
export function useRunProgress(projectSlug: string): RunProgressState {
  const activeSessionId = useSessionStore((s) => s.activeSessionId);
  const [activity, setActivity] = useState<string | null>(null);
  const [completed, setCompleted] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Reset run-scoped state when the active session changes.
  useEffect(() => {
    setActivity(null);
    setCompleted(false);
    setDismissed(false);
  }, [activeSessionId]);

  useEffect(() => {
    const unsubArtifact = subscribeToSSE(SSE_EVENTS.artifactUpdated, (data) => {
      const p = data as { projectSlug?: string; zone?: string } | undefined;
      if (!p || p.projectSlug !== projectSlug) return;
      setActivity(`最近更新：${p.zone ?? '产物'}`);
      setDismissed(false);
    });
    const doneEvents = [SSE_EVENTS.sessionCompleted, SSE_EVENTS.sessionFailed];
    const unsubsDone = doneEvents.map((evt) =>
      subscribeToSSE(evt, () => setCompleted(true)),
    );
    return () => {
      unsubArtifact();
      unsubsDone.forEach((u) => u());
    };
  }, [projectSlug]);

  const active = !!activeSessionId && !completed && !dismissed && activity !== null;
  return { active, activity, dismiss: () => setDismissed(true) };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd frontend && npm run test -- useRunProgress`
Expected: 4 tests PASS.

- [ ] **Step 5: Mount the hooks + bar in ProjectPage**

In `frontend/src/pages/ProjectPage.tsx`, add imports near the other hook/component imports (after the AgentInvokePanel import, line 9):

```tsx
import { useArtifactRefresh } from '@/hooks/useArtifactRefresh';
import { useRunProgress } from '@/hooks/useRunProgress';
import { RunProgressBar } from '@/components/feature/project/RunProgressBar';
```

Inside `ProjectPage()`, after the `zone` is resolved (after line 49, where `setZone` is used in the effect) and before the early returns is too early (slug/project not validated). Call both hooks right after the `const project = data?.project;` line is fine, but to keep them unconditionally called (Rules of Hooks), place them after the `slug` is known and before the first early `return`. Add after line 36 (`setSummaryPanelCollapsed`):

```tsx
  useArtifactRefresh(slug);
  const runProgress = useRunProgress(slug);
```

Then render the bar inside the header, after the `headerTop` `<div>` closes (line 164) and before `</header>` (line 165). The header block becomes:

```tsx
        <header className={s.header}>
          <div className={s.headerTop}>
            <div className={s.headerBody}>
              <h1 className={s.title}>{project.title}</h1>
              <p className={s.subtitle}>
                当前阶段：<strong>{ZONE_DISPLAY[zone]}</strong>
              </p>
            </div>
            {zone !== 'Practice' && (
              <div className={s.headerActions}>
                <AgentInvokePanel slug={slug} zone={zone} />
              </div>
            )}
          </div>
          <RunProgressBar
            active={runProgress.active}
            activity={runProgress.activity}
            onDismiss={runProgress.dismiss}
          />
        </header>
```

- [ ] **Step 6: Verify the full frontend test + build**

Run:
```bash
cd frontend && npm run test
cd frontend && npm run build
```
Expected: all tests PASS; production build succeeds.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/hooks/useRunProgress.ts frontend/src/hooks/useRunProgress.test.tsx frontend/src/pages/ProjectPage.tsx
git commit -m "feat(frontend): add useRunProgress + mount RunProgressBar in ProjectPage"
```

---

## End-to-End Smoke (after all 7 tasks)

1. `go run ./backend-go/cmd/lll` (per the project's run workflow — the prebuilt `dist/lll.exe` will NOT include these routes; always `go run`).
2. `cd frontend && npm run dev`.
3. Open a project, trigger an Explain generation (AgentInvokePanel → 调用). Watch the terminal write pages: each page should appear in the reader within ~1s (no 5s poll lag), and the RunProgressBar should show "最近更新：explain" while pages stream in, then be dismissible.
4. Trigger Practice generation: tasks should appear via the event path (no 3s poll); the bar should reflect "最近更新：practice".
5. Edit a confusion (保存摘要): the ConfusionPanel should refresh without a manual reload.
6. `go test ./backend-go/...` and `cd frontend && npm run test` — all green.

## Notes / Honest Limitations (Phase A)

- The RunProgressBar is **indeterminate** and its reliable hide signal (`session-completed`) only fires once Phase C wires Claude Code `Stop` hooks. In Phase A, dismiss (×) or navigating away clears a lingering bar. This is intentional and documented in the iteration README.
- `run-progress` event name and determinate `pagesDone`/`pagesPlanned` are Phase C; Phase A emits only `artifact-updated`.
- fsnotify tests are timing-based; the debounce window is a package var (`debounceWindow`) so tests shorten it to 5ms.
