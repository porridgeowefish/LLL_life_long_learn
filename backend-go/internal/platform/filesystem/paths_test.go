package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigureWorkspaceUpdatesEveryRuntimeDataRoot(t *testing.T) {
	oldWorkspace, oldProjects, oldAgents := WORKSPACE, PROJECTS_ROOT, AGENTS_ROOT
	t.Cleanup(func() {
		WORKSPACE, PROJECTS_ROOT, AGENTS_ROOT = oldWorkspace, oldProjects, oldAgents
	})

	root := t.TempDir()
	if err := ConfigureWorkspace(root); err != nil {
		t.Fatal(err)
	}
	if WORKSPACE != root {
		t.Fatalf("workspace = %q, want %q", WORKSPACE, root)
	}
	if PROJECTS_ROOT != filepath.Join(root, "projects") {
		t.Fatalf("projects root = %q", PROJECTS_ROOT)
	}
	if AGENTS_ROOT != filepath.Join(root, "learning-agents") {
		t.Fatalf("agents root = %q", AGENTS_ROOT)
	}
}

func TestDefaultWorkspaceRootLivesInsideRepository(t *testing.T) {
	got := DefaultWorkspaceRoot(filepath.Join("D", "repo"))
	want := filepath.Join("D", "repo", WorkspaceDirName)
	if got != want {
		t.Fatalf("DefaultWorkspaceRoot = %q, want %q", got, want)
	}
}

func TestConfigureWorkspaceCreatesMissingRoot(t *testing.T) {
	oldWorkspace, oldProjects, oldAgents := WORKSPACE, PROJECTS_ROOT, AGENTS_ROOT
	t.Cleanup(func() {
		WORKSPACE, PROJECTS_ROOT, AGENTS_ROOT = oldWorkspace, oldProjects, oldAgents
	})

	root := filepath.Join(t.TempDir(), "nested", WorkspaceDirName)
	if err := ConfigureWorkspace(root); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(root)
	if err != nil || !fi.IsDir() {
		t.Fatalf("workspace root not created: %v", err)
	}
}

func TestIsTestBinaryDetectsTestBinariesOnly(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })

	os.Args = []string{"/tmp/httpserver.test", "-test.run=x"}
	if !isTestBinary() {
		t.Fatal("expected .test binary to be detected")
	}
	os.Args = []string{`C:\build\pkg.test.exe`, "-test.v"}
	if !isTestBinary() {
		t.Fatal("expected .test.exe binary to be detected")
	}
	os.Args = []string{`C:\lll\dist\lll.exe`}
	if isTestBinary() {
		t.Fatal("production binary must not be treated as a test binary")
	}
}
