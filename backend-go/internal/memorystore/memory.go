// Package memorystore reads project memory files for prompt context.
// This slice is read-only — no inference, no auto-update.
package memorystore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Snapshot captures the memory file paths for a project plus whether each exists.
type Snapshot struct {
	ProjectMemoryPath    string `json:"projectMemoryPath"`
	ProjectMemoryExists  bool   `json:"projectMemoryExists"`
	ProjectStatePath     string `json:"projectStatePath"`
	ProjectStateExists   bool   `json:"projectStateExists"`
}

// Read returns the snapshot for a project. Memory files are at
// <project>/memory/project-memory.md and project-state.json.
func Read(slug string) (*Snapshot, error) {
	// Use a workspace helper to resolve the project root.
	// Since projectRoot is unexported, we use ReadArtifact on a sentinel.
	// Simpler: recompute paths using paths package directly here, but we
	// lack access to activeProjectsRoot. Instead, use workspace.ResolvePredecessorFiles
	// with an unrelated zone just to validate the slug — overkill.
	// Cleanest: add a public ProjectRoot in workspace. For now we route via
	// a side path.
	root, err := workspace.ProjectRootForSlug(slug)
	if err != nil {
		return nil, err
	}
	memPath := filepath.Join(root, "memory", "project-memory.md")
	statePath := filepath.Join(root, "memory", "project-state.json")
	out := &Snapshot{
		ProjectMemoryPath: memPath,
		ProjectStatePath:  statePath,
	}
	if _, err := os.Stat(memPath); err == nil {
		out.ProjectMemoryExists = true
	}
	if _, err := os.Stat(statePath); err == nil {
		out.ProjectStateExists = true
	}
	return out, nil
}

// ErrProjectNotFound is returned when memory resolution fails because the
// project slug does not exist on disk.
var ErrProjectNotFound = errors.New("project not found")

// ReadProjectMemoryText returns the raw text of project-memory.md, or empty
// string if the file does not exist.
func ReadProjectMemoryText(slug string) (string, error) {
	snap, err := Read(slug)
	if err != nil {
		return "", err
	}
	if !snap.ProjectMemoryExists {
		return "", nil
	}
	b, err := os.ReadFile(snap.ProjectMemoryPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
