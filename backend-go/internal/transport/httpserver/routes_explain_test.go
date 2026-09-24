package httpserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/httpx"
	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
)

// TestEmitInfographicReadyBroadcastsArtifactUpdated pins the SSE contract the
// frontend ExplainInfographic component subscribes to: on completion the
// infographic pipeline must emit "artifact-updated" carrying the project slug
// and the explain infographic artifact path. Drift here would silently break
// Stage 2 delivery.
func TestEmitInfographicReadyBroadcastsArtifactUpdated(t *testing.T) {
	s := &Server{broadcaster: httpx.NewBroadcaster()}
	ch, unsub := s.broadcaster.Subscribe()
	defer unsub()

	const slug = "test-proj"
	s.emitInfographicReady(slug)

	select {
	case ev := <-ch:
		if ev.Type != "artifact-updated" {
			t.Fatalf("event type = %q, want %q", ev.Type, "artifact-updated")
		}
		m, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload type = %T, want map[string]any", ev.Payload)
		}
		if m["slug"] != slug {
			t.Fatalf("payload slug = %v, want %q", m["slug"], slug)
		}
		if m["artifact"] != "explain/infographic.png" {
			t.Fatalf("payload artifact = %v, want %q", m["artifact"], "explain/infographic.png")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive artifact-updated event within 1s")
	}
}

// TestEmitInfographicReadyDoesNotBlockWithoutSubscribers guards the pipeline:
// it calls emit unconditionally on completion, so Emit must never block when no
// SSE client is subscribed (Broadcaster drops to a buffered channel / silently).
func TestEmitInfographicReadyDoesNotBlockWithoutSubscribers(t *testing.T) {
	s := &Server{broadcaster: httpx.NewBroadcaster()}
	done := make(chan struct{})
	go func() {
		s.emitInfographicReady("solo-proj")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("emitInfographicReady blocked with no subscribers")
	}
}

// TestInfographicScriptPathResolution pins ADR-0023: the backend-invoked image
// script resolves from the repository's scripts/ by default and from a
// workspace-local scripts/ override when one exists.
func TestInfographicScriptPathResolution(t *testing.T) {
	oldProject, oldWorkspace := paths.PROJECT_ROOT, paths.WORKSPACE
	t.Cleanup(func() {
		paths.PROJECT_ROOT, paths.WORKSPACE = oldProject, oldWorkspace
	})

	repo := t.TempDir()
	paths.PROJECT_ROOT = repo
	shipped := filepath.Join(repo, "scripts", "gen_infographic.py")

	t.Run("workspace equal to repository resolves shipped script", func(t *testing.T) {
		paths.WORKSPACE = repo
		if got := infographicScriptPath(); got != shipped {
			t.Fatalf("got %q, want %q", got, shipped)
		}
	})

	t.Run("workspace-local script wins when present", func(t *testing.T) {
		ws := t.TempDir()
		local := filepath.Join(ws, "scripts", "gen_infographic.py")
		if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(local, []byte("# custom"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths.WORKSPACE = ws
		if got := infographicScriptPath(); got != local {
			t.Fatalf("got %q, want %q", got, local)
		}
	})

	t.Run("missing workspace script falls back to repository", func(t *testing.T) {
		paths.WORKSPACE = t.TempDir()
		if got := infographicScriptPath(); got != shipped {
			t.Fatalf("got %q, want %q", got, shipped)
		}
	})
}
