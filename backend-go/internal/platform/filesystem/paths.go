// Package filesystem exposes canonical filesystem roots for LLL.
// All other packages resolve project, agent, and workspace paths through here.
package filesystem

import (
	"os"
	"path/filepath"
)

// PROJECT_ROOT is the repository root (where go.mod lives).
// WORKSPACE is the user's learning workspace (configurable via WORKSPACE env).
// PROJECTS_ROOT holds all learning project folders.
// AGENTS_ROOT holds agent registry + charter files.
var (
	PROJECT_ROOT   string
	WORKSPACE      string
	PROJECTS_ROOT  string
	AGENTS_ROOT    string
	FRONTEND_ROOT  string
	FRONTEND_INDEX string
)

func init() {
	// Resolve PROJECT_ROOT by walking up from the working directory until we find go.mod.
	wd, err := os.Getwd()
	if err != nil {
		panic("paths: cannot get working directory: " + err.Error())
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			PROJECT_ROOT = dir
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("paths: go.mod not found walking up from " + wd)
		}
		dir = parent
	}

	if env := os.Getenv("WORKSPACE"); env != "" {
		if err := ConfigureWorkspace(env); err != nil {
			panic("paths: configure workspace: " + err.Error())
		}
	} else {
		if err := ConfigureWorkspace(PROJECT_ROOT); err != nil {
			panic("paths: configure workspace: " + err.Error())
		}
	}
	// FRONTEND_ROOT points at the Vite build output (frontend/dist) so a
	// single Go binary can serve the React SPA in production. In dev the
	// frontend runs on Vite :5173 and proxies API calls here.
	FRONTEND_ROOT = filepath.Join(PROJECT_ROOT, "frontend", "dist")
	FRONTEND_INDEX = filepath.Join(FRONTEND_ROOT, "index.html")
}

// ConfigureWorkspace applies one resolved workspace root to every mutable
// runtime data path. Call it after loading CLI/environment/file configuration;
// changing WORKSPACE in the process environment after package initialization
// does not update Go package variables by itself.
func ConfigureWorkspace(root string) error {
	if root == "" {
		return os.ErrInvalid
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	WORKSPACE = filepath.Clean(abs)
	PROJECTS_ROOT = filepath.Join(WORKSPACE, "projects")
	AGENTS_ROOT = filepath.Join(WORKSPACE, "learning-agents")
	return nil
}
