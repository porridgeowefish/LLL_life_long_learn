package workspace

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTempWorkspace swaps paths.PROJECTS_ROOT to a temp dir for the test.
// Returns a cleanup function.
func withTempWorkspace(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	old := projectsRootOverride
	projectsRootOverride = dir
	return dir, func() { projectsRootOverride = old }
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Recommender Systems": "recommender-systems",
		"  Rust  Ownership ":  "rust-ownership",
		"Multiple   Spaces":   "multiple-spaces",
		"场论 (Field Theory)":   "场论-field-theory",
		"UPPER--CASE":         "upper-case",
		"with/slash\\dot":     "with-slash-dot",
		"":                    "",
		"---":                 "",
	}
	for in, want := range cases {
		got := Slugify(in)
		if got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateSlug(t *testing.T) {
	good := []string{"a", "abc", "abc-123", "rust-ownership", "场论-笔记"}
	bad := []string{"", "ABC", "abc/def", "..", "abc.def", strings.Repeat("a", 200)}
	for _, s := range good {
		if !ValidateSlug(s) {
			t.Errorf("ValidateSlug(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if ValidateSlug(s) {
			t.Errorf("ValidateSlug(%q) = true, want false", s)
		}
	}
}

func TestCreateProjectSkeleton_TopLevel(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("recommender-systems", "Recommender Systems", ""); err != nil {
		t.Fatalf("CreateProjectSkeleton: %v", err)
	}

	// Verify folder tree.
	wantDirs := []string{"", "memory", "intro", "explain", "practice", "extend", "summary", "progress", "runs", "runs/_index", "assets", "subprojects"}
	for _, d := range wantDirs {
		p := filepath.Join(projectsRootOverride, "recommender-systems", d)
		if info, err := os.Stat(p); err != nil || !info.IsDir() {
			t.Errorf("expected dir %s missing or not a directory", p)
		}
	}

	// Verify state.json.
	stateBytes, err := os.ReadFile(filepath.Join(projectsRootOverride, "recommender-systems", "state.json"))
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}
	var state ProjectState
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		t.Fatalf("decode state.json: %v", err)
	}
	if state.Title != "Recommender Systems" || state.Slug != "recommender-systems" {
		t.Errorf("state mismatch: %+v", state)
	}

	// Verify summary/summary.md exists and is empty.
	summary, err := os.ReadFile(filepath.Join(projectsRootOverride, "recommender-systems", "summary", "summary.md"))
	if err != nil {
		t.Fatalf("read summary.md: %v", err)
	}
	if len(summary) != 0 {
		t.Errorf("summary.md should be empty, got %q", summary)
	}
}

func TestCreateProjectSkeleton_Subproject(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("recommender-systems", "Recommender Systems", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateProjectSkeleton("collaborative-filtering", "Collaborative Filtering", "recommender-systems"); err != nil {
		t.Fatalf("create subproject: %v", err)
	}

	// Subproject lives under parent/subprojects/.
	subPath := filepath.Join(projectsRootOverride, "recommender-systems", "subprojects", "collaborative-filtering", "state.json")
	if _, err := os.Stat(subPath); err != nil {
		t.Fatalf("subproject state.json missing: %v", err)
	}

	// Parent's state has childProjectIds updated.
	parent, err := ReadProjectState("recommender-systems")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range parent.ChildProjectIDs {
		if c == "collaborative-filtering" {
			found = true
		}
	}
	if !found {
		t.Errorf("parent missing childProjectId; got %v", parent.ChildProjectIDs)
	}
}

func TestCreateProjectSkeleton_CollisionReturns409(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	err := CreateProjectSkeleton("test", "Test", "")
	if !IsSlugConflict(err) {
		t.Errorf("expected SlugConflictError, got %v", err)
	}
}

func TestIndexAll(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("alpha", "Alpha", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateProjectSkeleton("beta", "Beta", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateProjectSkeleton("beta-sub", "Beta Sub", "beta"); err != nil {
		t.Fatal(err)
	}

	all, err := IndexAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 projects, got %d: %+v", len(all), all)
	}
	// Find the subproject; its parentProjectId should be set.
	var sub *ProjectMeta
	for i := range all {
		if all[i].Slug == "beta-sub" {
			sub = &all[i]
		}
	}
	if sub == nil {
		t.Fatal("beta-sub missing from index")
	}
	if sub.ParentProjectID != "beta" {
		t.Errorf("beta-sub parent = %q, want beta", sub.ParentProjectID)
	}
}

func TestResolvePredecessorFiles(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	// No predecessor files exist yet → Explain should report Intro as predecessor (exists:false).
	preds, err := ResolvePredecessorFiles("test", ZoneExplain)
	if err != nil {
		t.Fatal(err)
	}
	if len(preds) != 1 || preds[0].ZoneName != ZoneIntro {
		t.Errorf("Explain predecessors = %+v, want [Intro]", preds)
	}
	if preds[0].Exists {
		t.Errorf("Intro should not exist yet")
	}

	// Write intro/output.md and re-check.
	if err := SafeWriteArtifact("test", ZoneIntro, "output.md", []byte("# Intro\n")); err != nil {
		t.Fatal(err)
	}
	preds, _ = ResolvePredecessorFiles("test", ZoneExplain)
	if !preds[0].Exists {
		t.Errorf("Intro should exist after write")
	}

	// Summary pulls from all four zones.
	preds, _ = ResolvePredecessorFiles("test", ZoneSummary)
	if len(preds) != 4 {
		t.Errorf("Summary predecessors len = %d, want 4: %+v", len(preds), preds)
	}
}

func TestSafeWriteSummary_learnerProtected(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	// Initial summary.md is empty → first write OK.
	if err := SafeWriteSummary("test", []byte("# First\n"), false); err != nil {
		t.Fatalf("first write should succeed: %v", err)
	}
	// Now non-empty → second write without force should fail.
	err := SafeWriteSummary("test", []byte("# Second\n"), false)
	if err == nil {
		t.Error("second write without force should fail")
	}
	// With force → succeeds.
	if err := SafeWriteSummary("test", []byte("# Second\n"), true); err != nil {
		t.Errorf("forced write should succeed: %v", err)
	}
}

func TestSafeWriteArtifact_RefusesSummary(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("test", "Test", ""); err != nil {
		t.Fatal(err)
	}
	err := SafeWriteArtifact("test", ZoneSummary, "summary.md", []byte("nope"))
	if err == nil {
		t.Error("SafeWriteArtifact should refuse summary/summary.md")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	if err := AtomicWriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Errorf("got %q, want hello", got)
	}

	// Overwrite.
	if err := AtomicWriteFile(target, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(target)
	if string(got) != "world" {
		t.Errorf("got %q, want world", got)
	}

	// No leftover temp files.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestPathTraversalRejected(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	// Invalid slugs should error before touching disk.
	err := CreateProjectSkeleton("..\\escape", "Bad", "")
	if err == nil {
		t.Error("expected error for traversal slug")
	}
	// ValidateSlug catches uppercase.
	if ValidateSlug("UPPER") {
		t.Error("uppercase slug should be rejected")
	}
}

// sanity check that errors.Is works with our sentinel.
func TestSlugConflictErrorChain(t *testing.T) {
	wrapped := errors.New("wrapped: " + (&SlugConflictError{Slug: "x"}).Error())
	if !strings.Contains(wrapped.Error(), "x") {
		t.Errorf("error message lost slug")
	}
}

func TestNestedProjectResolvesByOwnSlug(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("parent", "Parent", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateSubprojectWithInput("parent", "child-topic", "Child", ProjectInput{
		Why: "prerequisite",
	}); err != nil {
		t.Fatal(err)
	}
	state, err := ReadProjectState("child-topic")
	if err != nil {
		t.Fatal(err)
	}
	if state.ParentProjectID != "parent" {
		t.Fatalf("parentProjectId = %q, want parent", state.ParentProjectID)
	}
	root, err := ProjectRootForSlug("child-topic")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(root)) != "subprojects" {
		t.Fatalf("child root %q is not nested under subprojects", root)
	}
}

func TestReadProjectStateDetectsGeneratedZones(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("generated-zones", "Generated Zones", ""); err != nil {
		t.Fatal(err)
	}
	root, err := ProjectRootForSlug("generated-zones")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"intro/assessment.json":    `{"schemaVersion":1}`,
		"explain/manifest.json":    `{"pages":[{"id":"p001"}]}`,
		"practice/tasks.json":      `{"tasks":[{"id":"q1"}]}`,
		"extend/relation-notes.md": "related topic",
		"summary/flashcards.json":  `{"version":1,"cards":[{"id":"fc-1"}]}`,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	state, err := ReadProjectState("generated-zones")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.GeneratedZones) != len(AllZones) {
		t.Fatalf("expected all zones generated, got %v", state.GeneratedZones)
	}
	for i, zone := range AllZones {
		if state.GeneratedZones[i] != zone {
			t.Fatalf("zone %d: expected %s, got %s", i, zone, state.GeneratedZones[i])
		}
	}
}

func TestReadProjectStateDetectsFlashcardVariants(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("flashcard-variants", "Flashcard Variants", ""); err != nil {
		t.Fatal(err)
	}
	root, err := ProjectRootForSlug("flashcard-variants")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "summary", "flashcards.json")
	if err := os.WriteFile(path, []byte("```json\n{\"flashcards\":[{\"id\":\"fc-1\"}]}\n```"), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := ReadProjectState("flashcard-variants")
	if err != nil {
		t.Fatal(err)
	}
	if len(state.GeneratedZones) != 1 || state.GeneratedZones[0] != ZoneSummary {
		t.Fatalf("expected Summary generated from flashcard variant, got %v", state.GeneratedZones)
	}
}
