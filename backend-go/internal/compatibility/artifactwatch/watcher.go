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
	// Skip directory events. On Windows, writing a file into a watched
	// directory also fires a Write event whose Name is the directory itself
	// (its metadata changed); without this guard that directory event would
	// clobber the real file event during debounce and emit the dir path.
	if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
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
