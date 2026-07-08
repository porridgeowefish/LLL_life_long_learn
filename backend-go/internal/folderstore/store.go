// Package folderstore manages workspace-global project folders — user-created
// named groupings that hold references (project slugs) to projects. File-backed,
// no DB. Membership is by reference only: projects are never moved on disk, so
// deleting a folder simply returns its projects to "uncategorized".
package folderstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xmz14/lll/backend-go/internal/paths"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Folder is one user-created grouping of project slugs (references only).
type Folder struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	SlugOrder []string `json:"slugOrder"`
}

// Layout is the full workspace-global folder layout.
type Layout struct {
	Folders []Folder `json:"folders"`
}

// Store is a file-backed, workspace-global folder store.
type Store struct {
	mu       sync.Mutex
	filePath string
	data     Layout
}

// New opens (or creates) the workspace folder store.
func New() (*Store, error) {
	s := &Store{filePath: filepath.Join(paths.WORKSPACE, "folders.json")}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.data = Layout{}
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		s.data = Layout{}
		return nil
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		_, _ = workspace.BackupCorruptFile(s.filePath, err)
		s.data = Layout{}
		return nil
	}
	return nil
}

func (s *Store) save() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	return workspace.AtomicWriteFile(s.filePath, raw, 0o644)
}

// Layout returns a copy of the current folder layout. Folders is always a
// non-nil slice so the JSON serializes as [] rather than null.
func (s *Store) Layout() Layout {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Layout{Folders: make([]Folder, len(s.data.Folders))}
	copy(out.Folders, s.data.Folders)
	return out
}

// Replace swaps the entire layout. It sanitizes: drops unnamed folders, caps
// name length, fills missing/duplicate IDs, and enforces tree-style membership
// (a project slug may appear in at most one folder — first occurrence wins).
func (s *Store) Replace(in Layout) (Layout, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := Layout{Folders: make([]Folder, 0, len(in.Folders))}
	seenID := map[string]bool{}
	seenSlug := map[string]bool{}
	for _, f := range in.Folders {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		if len([]rune(name)) > 60 {
			name = string([]rune(name)[:60])
		}
		id := strings.TrimSpace(f.ID)
		if id == "" || seenID[id] {
			id = newID()
		}
		seenID[id] = true

		slugs := make([]string, 0, len(f.SlugOrder))
		for _, sl := range f.SlugOrder {
			sl = strings.TrimSpace(sl)
			if sl == "" || seenSlug[sl] {
				continue
			}
			seenSlug[sl] = true
			slugs = append(slugs, sl)
		}
		out.Folders = append(out.Folders, Folder{ID: id, Name: name, SlugOrder: slugs})
	}

	s.data = out
	if err := s.save(); err != nil {
		return Layout{}, err
	}
	return out, nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "f_" + hex.EncodeToString(b[:])
}
