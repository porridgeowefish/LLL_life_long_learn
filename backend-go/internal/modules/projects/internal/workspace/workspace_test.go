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

func TestDeleteProjectRemovesWholeProjectRoot(t *testing.T) {
	dir, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("delete-me", "Delete Me", ""); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "delete-me", "runs", "example", "result.md")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte("associated content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteProject("delete-me"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "delete-me")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project root still exists: %v", err)
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
	wantDirs := []string{"", "intro", "explain", "practice", "progress", "runs", "runs/_index", "assets"}
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

}

func TestCreateProjectSkeletonDoesNotCreateRetiredZones(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("retired-zones", "Retired Zones", ""); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(projectsRootOverride, "retired-zones")
	for _, directory := range []string{"extend", "summary"} {
		if _, err := os.Stat(filepath.Join(root, directory)); !os.IsNotExist(err) {
			t.Fatalf("retired directory %s exists or returned unexpected error: %v", directory, err)
		}
	}
	if ValidateZoneName("Extend") || ValidateZoneName("Summary") {
		t.Fatal("retired zone names remain valid")
	}
}

func TestCreateDisciplineMapSkeletonHasOverviewAndNoZones(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeletonWithInput("physics", "物理学", "", ProjectInput{
		ProjectType: ProjectTypeDisciplineMap,
		Why:         "不应写入地图",
		Current:     "未接触",
		Target:      "融会贯通·能教别人",
		Standard:    "不应写入地图",
	}); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(projectsRootOverride, "physics")
	if _, err := os.Stat(filepath.Join(root, "overview.md")); err != nil {
		t.Fatalf("overview.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "learning-plan.json")); err != nil {
		t.Fatalf("learning-plan.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "discipline-topics.json")); err != nil {
		t.Fatalf("discipline-topics.json missing: %v", err)
	}
	for _, zone := range []string{"intro", "explain", "practice", "extend", "summary"} {
		if _, err := os.Stat(filepath.Join(root, zone)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("discipline map unexpectedly contains %s", zone)
		}
	}
	state, err := ReadProjectState("physics")
	if err != nil {
		t.Fatal(err)
	}
	if state.ProjectType != ProjectTypeDisciplineMap || state.ActiveZone != "" {
		t.Fatalf("unexpected map state: %+v", state)
	}
	projectBrief, err := os.ReadFile(filepath.Join(root, "project.md"))
	if err != nil {
		t.Fatalf("read map project.md: %v", err)
	}
	brief := string(projectBrief)
	for _, forbidden := range []string{"Active Phase", "Current ability", "Target ability", "Completion standard", "Intro", "不应写入地图"} {
		if strings.Contains(brief, forbidden) {
			t.Errorf("discipline-map project.md contains system-learning context %q:\n%s", forbidden, brief)
		}
	}
	for _, required := range []string{"## 项目形态", "学科地图", "## 总览目标", "## 范围备注"} {
		if !strings.Contains(brief, required) {
			t.Errorf("discipline-map project.md missing %q:\n%s", required, brief)
		}
	}
}

func TestCreateSystemLearningSkeletonStartsWithDraftLearningScope(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeletonWithInput("probability", "概率论", "", ProjectInput{
		ProjectType: ProjectTypeSystemLearning,
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(projectsRootOverride, "probability", "learning-scope.json"))
	if err != nil {
		t.Fatal(err)
	}
	var scope struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Source struct {
			Type string `json:"type"`
		} `json:"source"`
	}
	if err := json.Unmarshal(data, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Status != "draft" || scope.Title != "概率论" || scope.Source.Type != "standalone" {
		t.Fatalf("unexpected learning scope: %+v", scope)
	}
}

func TestReadLegacyProjectDefaultsToSystemLearning(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	root := filepath.Join(projectsRootOverride, "legacy")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `{"id":"legacy","slug":"legacy","title":"Legacy","status":"active","activeZone":"Explain"}`
	if err := os.WriteFile(filepath.Join(root, "state.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	state, err := ReadProjectState("legacy")
	if err != nil {
		t.Fatal(err)
	}
	if state.ProjectType != ProjectTypeSystemLearning {
		t.Fatalf("legacy project type = %q", state.ProjectType)
	}
}

func TestCreateProjectSkeleton_LegacyParentArgumentDoesNotCreateHierarchy(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()

	if err := CreateProjectSkeleton("recommender-systems", "Recommender Systems", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateProjectSkeleton("collaborative-filtering", "Collaborative Filtering", "recommender-systems"); err != nil {
		t.Fatalf("create peer project: %v", err)
	}

	// The project is top-level; the legacy argument creates no relationship.
	subPath := filepath.Join(projectsRootOverride, "collaborative-filtering", "state.json")
	if _, err := os.Stat(subPath); err != nil {
		t.Fatalf("peer project state.json missing: %v", err)
	}

	if _, err := ReadProjectState("collaborative-filtering"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projectsRootOverride, "recommender-systems", "subprojects")); !errors.Is(err, os.ErrNotExist) {
		t.Error("new project creation must not create subprojects directory")
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
	// Every indexed project is peer-level.
	var sub *ProjectMeta
	for i := range all {
		if all[i].Slug == "beta-sub" {
			sub = &all[i]
		}
	}
	if sub == nil {
		t.Fatal("beta-sub missing from index")
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

func TestProjectResolvesByOwnSlugAndFlatRoot(t *testing.T) {
	_, cleanup := withTempWorkspace(t)
	defer cleanup()
	if err := CreateProjectSkeleton("parent", "Parent", ""); err != nil {
		t.Fatal(err)
	}
	if err := CreateProjectSkeletonWithInput("child-topic", "Child", "", ProjectInput{
		Why: "prerequisite",
	}); err != nil {
		t.Fatal(err)
	}
	_, err := ReadProjectState("child-topic")
	if err != nil {
		t.Fatal(err)
	}
	root, err := ProjectRootForSlug("child-topic")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(root) != projectsRootOverride {
		t.Fatalf("project root %q is not flat", root)
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
		"intro/assessment.json": `{"schemaVersion":1}`,
		"explain/manifest.json": `{"pages":[{"id":"p001"}]}`,
		"practice/tasks.json":   `{"tasks":[{"id":"q1"}]}`,
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
