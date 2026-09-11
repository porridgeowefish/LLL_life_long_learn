package folderstore

import (
	"os"
	"path/filepath"
	"testing"

	paths "github.com/xmz14/lll/backend-go/internal/platform/filesystem"
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
		{Name: ""}, // dropped (unnamed)
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

func TestSyncMapFoldersPromotesExistingFolderAndKeepsChildren(t *testing.T) {
	setTempWorkspace(t)
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replace(Layout{Folders: []Folder{
		{ID: "math", Name: "应用数学", SlugOrder: []string{"monte-carlo", "applied-math"}},
	}}); err != nil {
		t.Fatal(err)
	}

	out, err := s.SyncMapFolders([]MapFolderSpec{{Slug: "applied-math", Title: "应用数学"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Folders) != 1 {
		t.Fatalf("same-name folder should be reused, got %d folders", len(out.Folders))
	}
	got := out.Folders[0]
	if got.MapProjectSlug != "applied-math" {
		t.Fatalf("map binding=%q", got.MapProjectSlug)
	}
	if len(got.SlugOrder) != 1 || got.SlugOrder[0] != "monte-carlo" {
		t.Fatalf("map project must not render as its own child: %v", got.SlugOrder)
	}
}

func TestSyncMapFoldersCreatesMissingFolder(t *testing.T) {
	setTempWorkspace(t)
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.SyncMapFolders([]MapFolderSpec{{Slug: "physics", Title: "物理学"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Folders) != 1 || out.Folders[0].MapProjectSlug != "physics" || out.Folders[0].Name != "物理学" {
		t.Fatalf("unexpected map folder: %+v", out.Folders)
	}
}

func TestRemoveProjectPrunesMembershipAndMapBinding(t *testing.T) {
	setTempWorkspace(t)
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Replace(Layout{Folders: []Folder{
		{ID: "map", Name: "Physics", MapProjectSlug: "physics", SlugOrder: []string{"mechanics", "optics"}},
	}}); err != nil {
		t.Fatal(err)
	}

	out, err := s.RemoveProject("physics")
	if err != nil {
		t.Fatal(err)
	}
	if out.Folders[0].MapProjectSlug != "" || len(out.Folders[0].SlugOrder) != 2 {
		t.Fatalf("map cleanup lost folder data: %+v", out.Folders[0])
	}
	out, err = s.RemoveProject("mechanics")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Folders[0].SlugOrder) != 1 || out.Folders[0].SlugOrder[0] != "optics" {
		t.Fatalf("membership not pruned: %+v", out.Folders[0])
	}
}

func TestLegacyRootFoldersJSONMigratesIntoProjects(t *testing.T) {
	root := t.TempDir()
	prev := paths.WORKSPACE
	paths.WORKSPACE = root
	t.Cleanup(func() { paths.WORKSPACE = prev })
	if err := os.MkdirAll(paths.PROJECTS_ROOT, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, "folders.json")
	if err := os.WriteFile(legacy, []byte(`{"folders":[{"id":"f_1","name":"编译原理","slug":"bianyi","sortOrder":1}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if len(store.data.Folders) != 1 || store.data.Folders[0].Name != "编译原理" {
		t.Fatalf("legacy folders not migrated: %#v", store.data)
	}
	if _, err := os.Stat(filepath.Join(root, "projects", "folders.json")); err != nil {
		t.Fatalf("projects/folders.json not created: %v", err)
	}
	// Legacy file stays untouched; new location is authoritative from now on.
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy file must not be deleted: %v", err)
	}
}
