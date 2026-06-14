// Package promptassembly builds the prompt package that gets sent to Claude.
package promptassembly

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/agentregistry"
	"github.com/xmz14/lll/backend-go/internal/memorystore"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

// Request carries the inputs to a prompt build.
type Request struct {
	ProjectSlug string
	ZoneName    workspace.ZoneName
	AgentID     string
	Intent      string

	// SourceRefs are confusion IDs to inject into the prompt as source references.
	SourceRefs []string

	// FollowupPriorResultPaths is set when this invocation is a follow-up.
	// Each path points to a prior run's result.md.
	FollowupPriorResultPaths []string
}

// Package is the assembled prompt package.
type Package struct {
	PromptMd       string    // the full prompt text passed to Claude
	RunDirName     string    // e.g. "2026-06-08T15-22-11-explain"
	PackageMeta    PackageMeta
}

// PackageMeta is written to run dir / package.json.
type PackageMeta struct {
	ProjectSlug       string                  `json:"projectSlug"`
	ZoneName          workspace.ZoneName      `json:"zoneName"`
	AgentID           string                  `json:"agentId"`
	PredecessorFiles  []workspace.PredecessorFile `json:"predecessorFiles"`
	OutputTargets     []agentregistry.OutputTarget `json:"outputTargets"`
	MemorySnapshot    *memorystore.Snapshot   `json:"memorySnapshot,omitempty"`
	FollowupPrior     []string                `json:"followupPriorResultPaths,omitempty"`
	GeneratedAt       time.Time               `json:"generatedAt"`
}

// Build constructs the prompt package. Returns error on validation failure.
func Build(req Request, reg *agentregistry.Registry) (*Package, error) {
	if !workspace.ValidateSlug(req.ProjectSlug) {
		return nil, fmt.Errorf("invalid project slug: %q", req.ProjectSlug)
	}
	if !workspace.ValidateZoneName(string(req.ZoneName)) {
		return nil, fmt.Errorf("invalid zone: %s", req.ZoneName)
	}
	agent, ok := reg.Get(req.AgentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", req.AgentID)
	}
	if !agentZoneAllowed(agent, req.ZoneName) {
		return nil, fmt.Errorf("agent %s not allowed in zone %s", agent.ID, req.ZoneName)
	}

	preds, err := workspace.ResolvePredecessorFiles(req.ProjectSlug, req.ZoneName)
	if err != nil {
		return nil, fmt.Errorf("resolve predecessors: %w", err)
	}
	memSnap, err := memorystore.Read(req.ProjectSlug)
	if err != nil {
		return nil, fmt.Errorf("read memory: %w", err)
	}

	promptMd := renderPrompt(agent, req, preds, memSnap)

	runDirName := timestampRunDir(req.AgentID, time.Now().UTC())

	return &Package{
		PromptMd:   promptMd,
		RunDirName: runDirName,
		PackageMeta: PackageMeta{
			ProjectSlug:      req.ProjectSlug,
			ZoneName:         req.ZoneName,
			AgentID:          agent.ID,
			PredecessorFiles: preds,
			OutputTargets:    agent.DefaultOutputTargets,
			MemorySnapshot:   memSnap,
			FollowupPrior:    req.FollowupPriorResultPaths,
			GeneratedAt:      time.Now().UTC(),
		},
	}, nil
}

// MarshalPackageMeta returns indented JSON for package.json.
func (p *Package) MarshalPackageMeta() ([]byte, error) {
	return json.MarshalIndent(p.PackageMeta, "", "  ")
}

func agentZoneAllowed(a *agentregistry.Agent, zone workspace.ZoneName) bool {
	for _, z := range a.AllowedZones {
		if z == zone {
			return true
		}
	}
	return false
}

func renderPrompt(agent *agentregistry.Agent, req Request, preds []workspace.PredecessorFile, mem *memorystore.Snapshot) string {
	var b strings.Builder
	b.WriteString("# Agent Identity\n\n")
	b.WriteString(fmt.Sprintf("**%s %s** — %s\n\n", agent.Icon, agent.Name, agent.Description))

	// # User Story — explicitly anchors the agent to the learner's story
	// before any mechanism content is loaded. Required by the registry
	// schema; if empty, validateAgent would have failed at registry load.
	if strings.TrimSpace(agent.UserStory) != "" {
		b.WriteString("# User Story\n\n")
		b.WriteString(strings.TrimSpace(agent.UserStory))
		b.WriteString("\n\n")
	}

	b.WriteString("# Charter\n\n")
	b.WriteString(agent.CharterText)
	b.WriteString("\n\n")

	// # Reasoning Primitives — mechanism files loaded from
	// agents/primitives/<name>.md. Required primitives are mandatory;
	// optional primitives are included unconditionally in v1 (see
	// primitives_loader.go for the policy).
	if len(agent.Primitives.Required) > 0 || len(agent.Primitives.Optional) > 0 {
		b.WriteString("# Reasoning Primitives\n\n")
		expanded, err := ExpandPrimitives(agent.Primitives.Required, agent.Primitives.Optional)
		if err != nil {
			// This should not happen — registry validation should have
			// caught missing required primitives at server start. If we
			// reach here we still surface a clear note in the prompt
			// rather than silently shipping a broken prompt.
			fmt.Println("promptassembly: expand primitives warning:", err)
			b.WriteString(fmt.Sprintf("_(primitive expansion failed: %s)_\n\n", err))
		} else {
			b.WriteString(expanded)
		}
	}

	b.WriteString("# Behavior Rules\n\n")
	b.WriteString("- Cite predecessor files when building on prior zone output.\n")
	b.WriteString("- Do NOT edit `summary/summary.md`. That file is learner-owned.\n")
	b.WriteString("- Write Markdown that renders cleanly with GitHub-flavored Markdown + Mermaid.\n")
	b.WriteString("- If information is missing, say so explicitly rather than fabricating.\n")
	b.WriteString("- Each required reasoning primitive's output must appear under its own heading in the deliverable.\n\n")

	b.WriteString("# Project Context\n\n")
	b.WriteString(fmt.Sprintf("- Project slug: `%s`\n", req.ProjectSlug))
	b.WriteString(fmt.Sprintf("- Active zone: `%s`\n", req.ZoneName))

	b.WriteString("\n# Predecessor Files\n\n")
	if len(preds) == 0 {
		b.WriteString("_(none — this zone has no predecessors in the dependency graph)_\n")
	} else {
		for _, p := range preds {
			marker := " "
			if p.Exists {
				marker = "✓"
			} else {
				marker = "○"
			}
			b.WriteString(fmt.Sprintf("- %s `%s` — %s zone\n", marker, p.Path, p.ZoneName))
		}
	}

	b.WriteString("\n# Memory Snapshot\n\n")
	if mem != nil {
		if mem.ProjectMemoryExists {
			b.WriteString(fmt.Sprintf("- Project memory: `%s` (exists — read for context)\n", mem.ProjectMemoryPath))
		} else {
			b.WriteString(fmt.Sprintf("- Project memory: `%s` (not yet present)\n", mem.ProjectMemoryPath))
		}
		if mem.ProjectStateExists {
			b.WriteString(fmt.Sprintf("- Project state: `%s`\n", mem.ProjectStatePath))
		}
	}

	if len(req.FollowupPriorResultPaths) > 0 {
		b.WriteString("\n# Follow-up Context\n\n")
		b.WriteString("This invocation is a follow-up within an existing session. The user's prior turns produced the following result files — read them before responding:\n\n")
		for _, p := range req.FollowupPriorResultPaths {
			b.WriteString(fmt.Sprintf("- `%s`\n", p))
		}
	}

	b.WriteString("\n# Output Targets\n\n")
	for _, t := range agent.DefaultOutputTargets {
		b.WriteString(fmt.Sprintf("- `%s/%s`\n", strings.ToLower(string(t.ZoneName)), t.Filename))
	}

	// Source References — confusion markers the learner selected for batch ask.
	if len(req.SourceRefs) > 0 {
		b.WriteString("\n# Source References\n\n")
		b.WriteString("The learner marked the following confusion points. Address each one:\n\n")
		for _, ref := range req.SourceRefs {
			b.WriteString(fmt.Sprintf("- confusion ID: `%s`\n", ref))
		}
	}

	if intent := strings.TrimSpace(req.Intent); intent != "" {
		b.WriteString("\n# Additional Guidance\n\n")
		b.WriteString(intent)
		b.WriteString("\n")
	}

	return b.String()
}

func timestampRunDir(agentID string, at time.Time) string {
	// Filesystem-safe ISO timestamp (colons are forbidden in Windows folder names).
	stamp := at.Format("2006-01-02T15-04-05")
	return fmt.Sprintf("%s-%s", stamp, agentID)
}
