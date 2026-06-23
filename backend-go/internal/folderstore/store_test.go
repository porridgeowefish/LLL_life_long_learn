package folderstore

import (
	"testing"

	"github.com/xmz14/lll/backend-go/internal/paths"
)

// setTempWorkspace points the store at a per-test workspace dir and restores it
// afterwards. folderstore.New resolves paths.WORKSPACE at call time.
func setTempWorkspace(t *testing.T) {
	t.Helper()
	prev := paths.WORKSPACE
	paths.WORKSPACE = t.TempDir()
	t.Cleanup(func() { paths.WORKSPACE = prev })
}

func TestReplaceSanitizes(t *testing.T) {
	setTempWorkspace(t)
	s, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := s.Replace(Layout{Folders: []Folder{
		{ID: "a", Name: "  编程  ", SlugOrder: []string{"go", "rust", "go"}}, // trim name; dedupe within
		{Name: ""},                       // dropped (unnamed)
		{Name: "社科", SlugOrder: []string{"econ", "history"}},
		{ID: "a", Name: "冲突ID", SlugOrder: []string{"go"}}, // dup id -> new id; "go" already seen -> dropped
	}})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	if len(out.Folders) != 3 {
		t.Fatalf("want 3 folders (unnamed dropped), got %d", len(out.Folders))
	}
	if out.Folders[0].Name != "编程" {
		t.Errorf("name not trimmed: %q", out.Folders[0].Name)
	}
	if len(out.Folders[0].SlugOrder) != 2 || out.Folders[0].SlugOrder[0] != "go" {
		t.Errorf("folder0 slugOrder want [go rust], got %v", out.Folders[0].SlugOrder)
	}
	// Tree membership: the duplicate-id folder keeps none of the already-assigned slug.
	if len(out.Folders[2].SlugOrder) != 0 {
		t.Errorf("folder2 slugOrder want empty (go already claimed), got %v", out.Folders[2].SlugOrder)
	}
	if out.Folders[2].ID == "a" {
		t.Errorf("duplicate id 'a' should have been regenerated")
	}

	// Persisted: a fresh Store reloads the sanitized layout.
	s2, err := New()
	if err != nil {
		t.Fatalf("reload New: %v", err)
	}
	if got := len(s2.Layout().Folders); got != 3 {
		t.Errorf("reload want 3 folders, got %d", got)
	}
}

func TestEmptyLayoutRoundTripsAsArray(t *testing.T) {
	setTempWorkspace(t)
	s, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Layout() must expose a non-nil slice so JSON is [], not null.
	if got := s.Layout().Folders; got == nil {
		t.Errorf("Layout().Folders want non-nil slice, got nil")
	}
}
