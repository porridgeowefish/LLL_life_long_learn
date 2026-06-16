// Package artifactwriter promotes session output into project zone files.
package artifactwriter

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// PromoteOptions configures artifact promotion.
type PromoteOptions struct {
	ProjectSlug string
	Agent       *agentregistry.Agent
	RunDirRel   string // e.g. "runs/2026-06-08T15-explain"
	SessionID   string
	Content     []byte // the result.md content
}

// Promote writes the agent's default output targets with Content.
// Returns the list of artifacts written (or skipped-because-already-written) and any error.
//
// Rules:
//   - summary/summary.md is NEVER overwritten here. The runtime never
//     passes force=true; learner must explicitly save via the summary API.
//   - If the target file already exists and is non-trivial (>200 bytes), it
//     means Claude authored the file directly during the agent run. In that
//     case we SKIP the overwrite but still record the ArtifactRef so the
//     audit trail shows the file was produced this session.
//   - All writes go through workspace.SafeWriteArtifact (atomic).
func Promote(opts PromoteOptions) ([]workspace.ArtifactRef, error) {
	if opts.Agent == nil {
		return nil, errors.New("agent is nil")
	}
	if len(opts.Content) == 0 {
		return nil, errors.New("no content to promote")
	}
	var written []workspace.ArtifactRef
	for _, t := range opts.Agent.DefaultOutputTargets {
		// summary/summary.md protection.
		if t.ZoneName == workspace.ZoneSummary && t.Filename == "summary.md" {
			continue
		}
		// P0-1: detect if Claude already authored the target file. If so, preserve it.
		if existing, err := workspace.ReadArtifact(opts.ProjectSlug, t.ZoneName, t.Filename); err == nil && len(existing) > 200 {
			written = append(written, workspace.ArtifactRef{
				ZoneName:  t.ZoneName,
				Filename:  t.Filename,
				SessionID: opts.SessionID,
				RunDirRel: opts.RunDirRel,
				WrittenAt: time.Now().UTC(),
			})
			continue
		}
		if err := workspace.SafeWriteArtifact(opts.ProjectSlug, t.ZoneName, t.Filename, opts.Content); err != nil {
			return written, fmt.Errorf("promote %s/%s: %w", t.ZoneName, t.Filename, err)
		}
		written = append(written, workspace.ArtifactRef{
			ZoneName:  t.ZoneName,
			Filename:  t.Filename,
			SessionID: opts.SessionID,
			RunDirRel: opts.RunDirRel,
			WrittenAt: time.Now().UTC(),
		})
	}
	// Append to project state.lastArtifacts for traceability.
	if state, err := workspace.ReadProjectState(opts.ProjectSlug); err == nil {
		state.LastArtifacts = append(state.LastArtifacts, written...)
		if len(state.LastArtifacts) > 50 {
			state.LastArtifacts = state.LastArtifacts[len(state.LastArtifacts)-50:]
		}
		_ = workspace.WriteProjectState(opts.ProjectSlug, state, "")
	}
	return written, nil
}

// RunResultPath returns the absolute path to result.md inside a run dir.
func RunResultPath(projectSlug, runDirRel string) (string, error) {
	root, err := workspace.ProjectRootForSlug(projectSlug)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, runDirRel, "result.md"), nil
}
