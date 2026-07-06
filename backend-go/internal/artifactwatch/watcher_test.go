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
