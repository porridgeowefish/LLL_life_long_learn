package filesystem

import (
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
